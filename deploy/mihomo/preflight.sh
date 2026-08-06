#!/usr/bin/env sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
MIHOMO_IMAGE_REF=${MIHOMO_IMAGE:-docker.io/metacubex/mihomo:v1.19.28@sha256:e6acd921addecfd59a8e2d38203f88356d635b54de6c0673db0e015139989312}

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

for shard in a b c; do
  config_dir="${SCRIPT_DIR}/instances/${shard}/etc"
  state_dir="${SCRIPT_DIR}/instances/${shard}/state"
  config_file="${config_dir}/config.yaml"
  provider_file="${config_dir}/providers/egress.yaml"

  if [ ! -f "$config_file" ] || [ ! -f "$provider_file" ] || [ ! -d "$state_dir" ]; then
    echo "mihomo-${shard}: missing config, provider, or state directory" >&2
    exit 1
  fi

  if grep -R -n 'REPLACE_WITH_' "$config_dir"; then
    echo "mihomo-${shard}: replace every credential placeholder before startup" >&2
    exit 1
  fi

  if ! grep -Eq '^[[:space:]]*-[[:space:]]+name:' "$provider_file"; then
    echo "mihomo-${shard}: providers/egress.yaml has no YAML proxy entry" >&2
    exit 1
  fi

  for secret_file in "$config_file" "$provider_file"; do
    mode=$(file_mode "$secret_file")
    case "$mode" in
      400|600) ;;
      *)
        echo "${secret_file}: expected mode 0600 or 0400, got ${mode}" >&2
        exit 1
        ;;
    esac
  done

  docker run --rm \
    --read-only \
    --tmpfs /tmp:size=16m,mode=1777,nosuid,nodev \
    --mount "type=bind,src=${config_dir},dst=/etc/mihomo,readonly" \
    --mount "type=bind,src=${state_dir},dst=/root/.config/mihomo" \
    --env SAFE_PATHS=/etc/mihomo \
    "$MIHOMO_IMAGE_REF" \
    -t -d /root/.config/mihomo -f /etc/mihomo/config.yaml

  echo "mihomo-${shard}: static configuration passed"
done

echo "Static checks passed. File providers are loaded only at runtime; run the Sub2API proxy and quality tests after startup."
