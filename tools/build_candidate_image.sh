#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: "${CANDIDATE_COMMIT:?CANDIDATE_COMMIT is required}"
: "${CANDIDATE_VERSION:?CANDIDATE_VERSION is required}"
: "${CANDIDATE_IMAGE_REPOSITORY:?CANDIDATE_IMAGE_REPOSITORY is required}"

expected_migration_count="${EXPECTED_MIGRATION_COUNT:-250}"
output_file="${CANDIDATE_OUTPUT_FILE:-}"
temp_root="${TMPDIR:-/tmp}"
temp_root="${temp_root%/}"
temp_dir=""

die() {
  printf 'candidate image build failed: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$1" | awk '{print $1}'
}

cleanup() {
  local exit_code=$?
  if [[ -n "$temp_dir" && -d "$temp_dir" && ! -L "$temp_dir" ]]; then
    case "$temp_dir" in
      "$temp_root"/sub2api-candidate.*)
        rm -rf -- "$temp_dir"
        ;;
      *)
        printf 'refusing to clean unexpected candidate temp path: %s\n' "$temp_dir" >&2
        exit_code=1
        ;;
    esac
  fi
  exit "$exit_code"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

for command_name in git tar python3 jq docker; do
  require_command "$command_name"
done

[[ "$CANDIDATE_COMMIT" =~ ^[0-9a-f]{40}$ ]] || die "candidate commit must be a full lowercase SHA"
[[ "$CANDIDATE_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$ ]] || die "candidate version must be an rc SemVer"
[[ "$CANDIDATE_IMAGE_REPOSITORY" =~ ^[a-z0-9.-]+/[a-z0-9._/-]+$ ]] || die "candidate image repository is invalid"
[[ "$expected_migration_count" =~ ^[0-9]+$ ]] || die "expected migration count must be numeric"

resolved_commit="$(git -C "$repo_dir" rev-parse --verify "${CANDIDATE_COMMIT}^{commit}")"
[[ "$resolved_commit" == "$CANDIDATE_COMMIT" ]] || die "candidate commit did not resolve exactly"
[[ "$(git -C "$repo_dir" rev-parse HEAD)" == "$CANDIDATE_COMMIT" ]] || die "HEAD is not the candidate commit"
git -C "$repo_dir" diff --quiet "$CANDIDATE_COMMIT" -- || die "tracked worktree differs from candidate commit"
git -C "$repo_dir" diff --cached --quiet "$CANDIDATE_COMMIT" -- || die "index differs from candidate commit"

temp_dir="$(mktemp -d "$temp_root/sub2api-candidate.XXXXXX")"
archive="$temp_dir/source.tar"
source_dir="$temp_dir/source"
metadata_file="$temp_dir/build-metadata.json"
image_manifest_file="$temp_dir/image-migration-manifest.json"

git -C "$repo_dir" archive --format=tar --prefix=source/ "$CANDIDATE_COMMIT" >"$archive"
[[ -s "$archive" ]] || die "git archive is empty"

if tar -tf "$archive" | awk '
  /^\// { bad=1 }
  /(^|\/)\._[^/]*$/ { bad=1 }
  /(^|\/)\.\.[/]/ { bad=1 }
  /(^|\/)\.codex-deploy-tmp([/]|$)/ { bad=1 }
  /(^|\/)backend\/cmd\/skill-market-quickprep([/]|$)/ { bad=1 }
  END { exit bad ? 0 : 1 }
'; then
  die "git archive contains an unsafe or forbidden entry"
fi

tar -xf "$archive" -C "$temp_dir"
[[ -d "$source_dir" && ! -L "$source_dir" ]] || die "archive did not produce one regular source root"
if find "$source_dir" \( -name '._*' -o -path '*/.codex-deploy-tmp/*' -o -path '*/backend/cmd/skill-market-quickprep/*' \) -print -quit | grep -q .; then
  die "extracted source contains an unsafe or forbidden path"
fi

manifest_json="$(python3 "$source_dir/tools/migration_manifest.py" --root "$source_dir")"
manifest_count="$(jq -er '.migrations | length' <<<"$manifest_json")"
manifest_set_sha256="$(jq -er '.set_sha256 | select(test("^[0-9a-f]{64}$"))' <<<"$manifest_json")"
[[ "$manifest_count" == "$expected_migration_count" ]] || die "unexpected migration count: $manifest_count"

archive_sha256="$(sha256_file "$archive")"
created_at="$(git -C "$repo_dir" show -s --format=%cI "$CANDIDATE_COMMIT")"
image_ref="${CANDIDATE_IMAGE_REPOSITORY}:sha-${CANDIDATE_COMMIT}"

docker buildx build \
  --platform linux/amd64 \
  --file "$source_dir/Dockerfile" \
  --tag "$image_ref" \
  --build-arg "VERSION=$CANDIDATE_VERSION" \
  --build-arg "COMMIT=$CANDIDATE_COMMIT" \
  --build-arg "DATE=$created_at" \
  --label "org.opencontainers.image.created=$created_at" \
  --label "org.opencontainers.image.version=$CANDIDATE_VERSION" \
  --label "org.opencontainers.image.revision=$CANDIDATE_COMMIT" \
  --label "org.luoxueapi.migration.count=$manifest_count" \
  --label "org.luoxueapi.migration.set_sha256=$manifest_set_sha256" \
  --provenance=mode=max \
  --sbom=true \
  --push \
  --metadata-file "$metadata_file" \
  "$source_dir"

image_digest="$(jq -er '."containerimage.digest" | select(test("^sha256:[0-9a-f]{64}$"))' "$metadata_file")"
immutable_image="${CANDIDATE_IMAGE_REPOSITORY}@${image_digest}"
docker pull --platform linux/amd64 "$immutable_image" >/dev/null

image_version="$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.version" }}' "$immutable_image")"
image_revision="$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$immutable_image")"
image_migration_count="$(docker image inspect --format '{{ index .Config.Labels "org.luoxueapi.migration.count" }}' "$immutable_image")"
image_migration_set="$(docker image inspect --format '{{ index .Config.Labels "org.luoxueapi.migration.set_sha256" }}' "$immutable_image")"
[[ "$image_version" == "$CANDIDATE_VERSION" ]] || die "OCI version label mismatch"
[[ "$image_revision" == "$CANDIDATE_COMMIT" ]] || die "OCI revision label mismatch"
[[ "$image_migration_count" == "$manifest_count" ]] || die "OCI migration count label mismatch"
[[ "$image_migration_set" == "$manifest_set_sha256" ]] || die "OCI migration set label mismatch"

version_output="$(docker run --rm --platform linux/amd64 --entrypoint /app/sub2api "$immutable_image" --version 2>&1)"
grep -Fq -- "LuoxueAPI ${CANDIDATE_VERSION} (commit: ${CANDIDATE_COMMIT}," <<<"$version_output" || die "binary version metadata mismatch"

docker run --rm --platform linux/amd64 --entrypoint /app/sub2api \
  "$immutable_image" --migration-manifest >"$image_manifest_file"
jq -e --argjson expected "$manifest_json" '. == $expected' "$image_manifest_file" >/dev/null || die "binary migration manifest differs from clean archive"

emit_output() {
  printf '%s=%s\n' "$1" "$2"
  if [[ -n "$output_file" ]]; then
    printf '%s=%s\n' "$1" "$2" >>"$output_file"
  fi
}

emit_output commit "$CANDIDATE_COMMIT"
emit_output version "$CANDIDATE_VERSION"
emit_output image_ref "$image_ref"
emit_output image_digest "$image_digest"
emit_output immutable_image "$immutable_image"
emit_output migration_count "$manifest_count"
emit_output migration_set_sha256 "$manifest_set_sha256"
emit_output clean_archive_sha256 "$archive_sha256"
emit_output clean_archive_forbidden_entries 0
