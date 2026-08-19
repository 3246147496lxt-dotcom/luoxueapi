#!/usr/bin/env python3
"""Static security contract for the release workflow.

This intentionally avoids executing release input. It guards the properties that
prevent GitHub input and annotated-tag text from becoming shell source.
"""

from __future__ import annotations

import json
import re
import secrets
import sys
import tomllib
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RELEASE = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
BACKEND_CI = (ROOT / ".github/workflows/backend-ci.yml").read_text(encoding="utf-8")
SECURITY = (ROOT / ".github/workflows/security-scan.yml").read_text(encoding="utf-8")
GORELEASER = (ROOT / ".goreleaser.yaml").read_text(encoding="utf-8")
GORELEASER_SIMPLE = (ROOT / ".goreleaser.simple.yaml").read_text(encoding="utf-8")
ROOT_DOCKERFILE = (ROOT / "Dockerfile").read_text(encoding="utf-8")
DEPLOY_DOCKERFILE = (ROOT / "deploy/Dockerfile").read_text(encoding="utf-8")
GORELEASER_DOCKERFILE = (ROOT / "Dockerfile.goreleaser").read_text(
    encoding="utf-8"
)
FRONTEND_PACKAGE = (ROOT / "frontend/package.json").read_text(encoding="utf-8")
QUOTA_VIEWER_PACKAGE = (ROOT / "quota-viewer/package.json").read_text(
    encoding="utf-8"
)
QUOTA_VIEWER_PACKAGE_JSON = json.loads(QUOTA_VIEWER_PACKAGE)
QUOTA_VIEWER_TAURI_CONFIG = json.loads(
    (ROOT / "quota-viewer/src-tauri/tauri.conf.json").read_text(encoding="utf-8")
)
QUOTA_VIEWER_CARGO = tomllib.loads(
    (ROOT / "quota-viewer/src-tauri/Cargo.toml").read_text(encoding="utf-8")
)
QUOTA_VIEWER_CARGO_LOCK = tomllib.loads(
    (ROOT / "quota-viewer/src-tauri/Cargo.lock").read_text(encoding="utf-8")
)
WINDOWS_RELEASE_PREPARE_SCRIPT = (
    ROOT / "quota-viewer/scripts/prepare-windows-release.ps1"
).read_text(encoding="utf-8")
WINDOWS_RELEASE_VERSION_SCRIPT = (
    ROOT / "quota-viewer/scripts/verify-windows-release-version.ps1"
).read_text(encoding="utf-8")
WINDOWS_AUTHENTICODE_SCRIPT = (
    ROOT / "quota-viewer/scripts/verify-windows-authenticode.ps1"
).read_text(encoding="utf-8")
WINDOWS_FILE_SIGNING_SCRIPT = (
    ROOT / "quota-viewer/scripts/sign-windows-file.ps1"
).read_text(encoding="utf-8")
WINDOWS_SIGNING_SCRIPT = (
    ROOT / "quota-viewer/scripts/sign-windows-release.ps1"
).read_text(encoding="utf-8")
E2E_SCRIPT = (ROOT / "backend/scripts/e2e-test.sh").read_text(encoding="utf-8")
SMOKE_COMPOSE = (ROOT / "deploy/docker-compose.smoke.yml").read_text(encoding="utf-8")
INSTALL_SCRIPT = (ROOT / "deploy/install.sh").read_text(encoding="utf-8")
DOCKER_DEPLOY_SCRIPT = (ROOT / "deploy/docker-deploy.sh").read_text(
    encoding="utf-8"
)
DEPLOY_README = (ROOT / "deploy/README.md").read_text(encoding="utf-8")
ROOT_README = (ROOT / "README.md").read_text(encoding="utf-8")
ROOT_README_CN = (ROOT / "README_CN.md").read_text(encoding="utf-8")
ROOT_README_JA = (ROOT / "README_JA.md").read_text(encoding="utf-8")
sys.dont_write_bytecode = True
sys.path.insert(0, str(ROOT / "tools"))
from release_semver import is_release_semver  # noqa: E402


def shell_blocks(workflow: str) -> list[str]:
    lines = workflow.splitlines()
    blocks: list[str] = []
    index = 0
    while index < len(lines):
        line = lines[index]
        if not re.search(r"\brun:\s*\|\s*$", line):
            index += 1
            continue
        base_indent = len(line) - len(line.lstrip())
        index += 1
        body: list[str] = []
        while index < len(lines):
            candidate = lines[index]
            indent = len(candidate) - len(candidate.lstrip())
            if candidate.strip() and indent <= base_indent:
                break
            body.append(candidate)
            index += 1
        blocks.append("\n".join(body))
    return blocks


class ReleaseWorkflowSecurityTest(unittest.TestCase):
    def test_untrusted_expressions_never_enter_shell_source(self) -> None:
        for block in shell_blocks(RELEASE):
            self.assertNotIn("${{", block)

    def test_release_is_gated_by_reusable_ci_and_security(self) -> None:
        self.assertIn("uses: ./.github/workflows/backend-ci.yml", RELEASE)
        self.assertIn("uses: ./.github/workflows/security-scan.yml", RELEASE)
        self.assertIn(
            "needs: [resolve-release, quality-gates, security-gates]", RELEASE
        )
        self.assertNotIn("skip_tests", RELEASE)
        self.assertIn("group: release", RELEASE)
        self.assertIn("cancel-in-progress: false", RELEASE)
        self.assertIn(
            "github.event_name != 'workflow_dispatch' && vars.SIMPLE_RELEASE == 'true'",
            RELEASE,
        )

        for job in [
            "shell",
            "compose-contract",
            "release-contract",
            "test",
            "postgresql18-integration",
            "deployment-script-regression",
            "frontend",
            "quota-viewer",
            "quota-viewer-windows",
            "docs-site",
            "embedded-web",
            "compose-smoke",
            "golangci-lint",
            "candidate-image",
            "candidate-image-validation",
        ]:
            self.assertRegex(BACKEND_CI, rf"(?m)^  {re.escape(job)}:$")
        for job in ["backend-security", "frontend-security"]:
            self.assertRegex(SECURITY, rf"(?m)^  {re.escape(job)}:$")
        self.assertIn("workflow_call:", BACKEND_CI)
        self.assertIn("workflow_call:", SECURITY)
        self.assertIn(
            "go test -race ./internal/lifecycle ./internal/server ./internal/modules/...",
            BACKEND_CI,
        )
        self.assertIn(
            "go test -tags=embed ./internal/web ./cmd/server", BACKEND_CI
        )
        self.assertIn("docker-compose.release-smoke.yml", BACKEND_CI)
        self.assertIn("config --format json --no-env-resolution", BACKEND_CI)
        self.assertIn('.services.sub2api.build == null', BACKEND_CI)
        self.assertIn('.services.sub2api.pull_policy == "never"', BACKEND_CI)
        self.assertGreaterEqual(BACKEND_CI.count("if-no-files-found: error"), 4)

    def test_branch_candidate_image_is_immutable_and_quality_gated(self) -> None:
        candidate_job = BACKEND_CI.split("  candidate-image:", 1)[1]
        self.assertIn("github.event_name == 'push'", candidate_job)
        self.assertIn("refs/heads/codex/quota-viewer-macos-dmg", candidate_job)
        self.assertIn("packages: write", candidate_job)
        self.assertIn("./tools/build_candidate_image.sh", candidate_job)
        self.assertIn("steps.build.outputs.immutable_image", candidate_job)
        self.assertIn("candidate-forward-schema-test.sh", candidate_job)
        self.assertIn("candidate-migration-manifest.json", candidate_job)
        self.assertIn("clean_archive_forbidden_entries=0", candidate_job)
        self.assertNotIn(":latest", candidate_job)

        for dependency in [
            "test",
            "postgresql18-integration",
            "deployment-script-regression",
            "frontend",
            "embedded-web",
            "compose-smoke",
            "golangci-lint",
        ]:
            with self.subTest(dependency=dependency):
                self.assertRegex(candidate_job, rf"(?m)^      - {re.escape(dependency)}$")

        validation_job = BACKEND_CI.split("  candidate-image-validation:", 1)[1]
        self.assertIn("needs: candidate-image", validation_job)
        self.assertIn("needs.candidate-image.outputs.immutable_image", validation_job)
        self.assertIn("docker pull --platform linux/amd64", validation_job)
        self.assertIn("backend/scripts/e2e-test.sh", validation_job)
        self.assertIn("candidate-forward-schema-test.sh", validation_job)

    def test_only_release_job_has_write_permissions(self) -> None:
        self.assertEqual(RELEASE.count("contents: write"), 1)
        self.assertEqual(RELEASE.count("packages: write"), 1)
        self.assertNotIn("sync-version-file:", RELEASE)

    def test_release_uses_immutable_resolved_commit(self) -> None:
        resolve_checkout = RELEASE.split(
            "- name: Checkout repository history", 1
        )[1].split("- name: Resolve and validate release tag", 1)[0]
        resolve_job = RELEASE.split("  resolve-release:", 1)[1].split(
            "  quality-gates:", 1
        )[0]

        self.assertIn("fetch-depth: 0", resolve_checkout)
        self.assertIn("fetch-tags: true", resolve_checkout)
        self.assertIn("persist-credentials: false", resolve_checkout)
        self.assertNotIn("git fetch", resolve_job)
        self.assertNotIn("GITHUB_TOKEN", resolve_job)
        self.assertNotIn("GH_TOKEN", resolve_job)
        self.assertNotIn("github.token", resolve_job)
        self.assertNotIn("extraheader", resolve_job.lower())
        self.assertNotIn("x-access-token", resolve_job.lower())
        self.assertNotIn("credential.helper", resolve_job.lower())
        self.assertIn(
            "ref: ${{ needs.resolve-release.outputs.commit }}",
            RELEASE,
        )
        self.assertIn(
            "RELEASE_COMMIT: ${{ needs.resolve-release.outputs.commit }}",
            RELEASE,
        )
        self.assertIn('tag_commit="$(git rev-parse --verify', RELEASE)
        self.assertIn('"$tag_commit" != "$RELEASE_COMMIT"', RELEASE)
        self.assertIn("persist-credentials: false", RELEASE)

    def test_reusable_quality_artifacts_feed_release(self) -> None:
        for artifact_name in ["ci-frontend-dist", "ci-docs-dist"]:
            with self.subTest(artifact=artifact_name):
                self.assertIn(f"name: {artifact_name}", BACKEND_CI)
                self.assertIn(f"name: {artifact_name}", RELEASE)
        self.assertIn("needs: [resolve-release, quality-gates, security-gates]", RELEASE)

    def test_windows_installer_is_signed_verified_and_attached(self) -> None:
        ci_installer_artifact = (
            "luoxue-quota-viewer-windows-x64-nsis-release-unsigned"
        )
        release_app_artifact = (
            "luoxue-quota-viewer-windows-x64-release-app-unsigned"
        )
        signed_artifact = "luoxue-quota-viewer-windows-x64-nsis-release-signed"
        self.assertIn("  quota-viewer-windows-release:", RELEASE)
        self.assertIn(ci_installer_artifact, BACKEND_CI)
        self.assertNotIn(ci_installer_artifact, RELEASE)
        self.assertIn(release_app_artifact, BACKEND_CI)
        self.assertIn(release_app_artifact, RELEASE)
        self.assertIn(signed_artifact, RELEASE)
        self.assertIn(
            "needs: [resolve-release, quality-gates, security-gates, "
            "quota-viewer-windows-release]",
            RELEASE,
        )
        self.assertIn(
            "needs.resolve-release.outputs.simple_release != 'true'", RELEASE
        )
        self.assertIn(
            "SIMPLE_RELEASE: ${{ needs.resolve-release.outputs.simple_release }}",
            RELEASE,
        )
        self.assertIn(
            "secrets.WINDOWS_SIGNING_CERTIFICATE_PFX_BASE64", RELEASE
        )
        self.assertIn("secrets.WINDOWS_SIGNING_CERTIFICATE_PASSWORD", RELEASE)
        self.assertIn("sign-windows-release.ps1", RELEASE)
        self.assertIn("Verify signed Windows release asset checksum", RELEASE)
        self.assertIn("sha256sum --check", RELEASE)
        self.assertIn("gh release upload", RELEASE)
        self.assertIn("--clobber", RELEASE)

        for required in [
            "WINDOWS_SIGNING_CERTIFICATE_PFX_BASE64 is required",
            "WINDOWS_SIGNING_CERTIFICATE_PASSWORD is required",
            "[Convert]::FromBase64String",
            "$securePassword = $null",
            "Import-PfxCertificate",
            "Get-AuthenticodeSignature",
            "SignatureStatus]::NotSigned",
            "verify-windows-release-version.ps1",
            "verify-windows-authenticode.ps1",
            "sign-windows-file.ps1",
            "signCommand",
            "certificateThumbprint",
            "digestAlgorithm = 'sha256'",
            "timestampUrl = $TimestampUrl.AbsoluteUri",
            "tsp = $true",
            "pnpm run tauri:bundle:windows --config $signingConfigPath",
            "NSIS !uninstfinalize",
            "luoxue-quota-viewer.exe",
            "uninstall.exe",
            "Start-Process",
            "Get-FileHash -LiteralPath $signedInstaller -Algorithm SHA256",
            "finally {",
            "foreach ($temporaryFile in @($pfxPath, $signingConfigPath, $signingLogPath))",
            "if ($certificateItem.HasPrivateKey)",
            "Remove-Item -LiteralPath $certificatePath -DeleteKey -Force",
            "The imported signing certificate still exists after removal.",
            "Failed to clean Windows signing state",
        ]:
            with self.subTest(required=required):
                self.assertIn(required, WINDOWS_SIGNING_SCRIPT)

        self.assertNotIn("Write-Host $certificate", WINDOWS_SIGNING_SCRIPT)
        self.assertNotIn("Write-Output $certificate", WINDOWS_SIGNING_SCRIPT)
        self.assertNotIn(
            "pnpm run tauri:bundle:windows -- --config",
            WINDOWS_SIGNING_SCRIPT,
        )
        self.assertGreaterEqual(WINDOWS_SIGNING_SCRIPT.count("& $versionVerifier"), 4)
        self.assertLess(
            WINDOWS_SIGNING_SCRIPT.index("& $versionVerifier"),
            WINDOWS_SIGNING_SCRIPT.index("[Environment]::GetEnvironmentVariable"),
        )
        self.assertLess(
            WINDOWS_SIGNING_SCRIPT.index(
                "Remove-Item Env:WINDOWS_SIGNING_CERTIFICATE_PASSWORD"
            ),
            WINDOWS_SIGNING_SCRIPT.index("pnpm run tauri:bundle:windows"),
        )

    def test_tauri_authenticode_chain_signs_every_installable_executable(self) -> None:
        bundle_command = QUOTA_VIEWER_PACKAGE_JSON["scripts"][
            "tauri:bundle:windows"
        ]
        self.assertIn("tauri bundle", bundle_command)
        self.assertIn("--bundles nsis", bundle_command)
        self.assertNotIn("--no-sign", bundle_command)

        for required in [
            "signtool.exe",
            "Get-AuthenticodeSignature",
            "SignatureStatus]::NotSigned",
            "/sha1 $normalizedThumbprint",
            "/fd SHA256",
            "/tr $TimestampUrl.AbsoluteUri",
            "/td SHA256",
            "verify-windows-authenticode.ps1",
            "AppendAllText",
            "signer_thumbprint",
        ]:
            with self.subTest(file_signing_requirement=required):
                self.assertIn(required, WINDOWS_FILE_SIGNING_SCRIPT)

        for required in [
            "signtool.exe",
            "verify /pa /all /v",
            "Get-AuthenticodeSignature",
            "SignatureStatus]::Valid",
            "SignerCertificate.Thumbprint",
            "TimeStamperCertificate",
        ]:
            with self.subTest(authenticode_requirement=required):
                self.assertIn(required, WINDOWS_AUTHENTICODE_SCRIPT)

        release_job = RELEASE.split(
            "  quota-viewer-windows-release:", 1
        )[1].split("\n  release:", 1)[0]
        for required in [
            "node-version: '24'",
            "version: 9.15.9",
            "rustup toolchain install 1.96.0",
            "prepare-windows-release.ps1",
            "pnpm install --frozen-lockfile",
            "luoxue-quota-viewer.exe",
            "Build, sign and verify Windows x64 NSIS release",
        ]:
            with self.subTest(release_job_requirement=required):
                self.assertIn(required, release_job)

    def test_windows_release_version_is_stamped_and_verified(self) -> None:
        lock_packages = [
            package
            for package in QUOTA_VIEWER_CARGO_LOCK["package"]
            if package["name"] == "luoxue-quota-viewer"
        ]
        self.assertEqual(len(lock_packages), 1)
        self.assertEqual(
            {
                QUOTA_VIEWER_PACKAGE_JSON["version"],
                QUOTA_VIEWER_TAURI_CONFIG["version"],
                QUOTA_VIEWER_CARGO["package"]["version"],
                lock_packages[0]["version"],
            },
            {QUOTA_VIEWER_PACKAGE_JSON["version"]},
        )

        self.assertIn("quota_viewer_release_version:", BACKEND_CI)
        self.assertIn(
            "quota_viewer_release_version: "
            "${{ needs.resolve-release.outputs.simple_release != 'true' && "
            "needs.resolve-release.outputs.version || '' }}",
            RELEASE,
        )
        windows_job = BACKEND_CI.split("  quota-viewer-windows:", 1)[1].split(
            "\n  docs-site:", 1
        )[0]
        for required in [
            "QUOTA_VIEWER_RELEASE_VERSION: ${{ inputs.quota_viewer_release_version }}",
            "Stamp release version into Windows Tauri sources",
            "prepare-windows-release.ps1",
            "pnpm run tauri:build:windows",
            "Verify Windows Tauri release version metadata",
            "luoxue-quota-viewer.exe",
            "verify-windows-release-version.ps1",
            "*-setup.exe",
        ]:
            with self.subTest(workflow_requirement=required):
                self.assertIn(required, windows_job)
        self.assertLess(
            windows_job.index("prepare-windows-release.ps1"),
            windows_job.index("pnpm run tauri:build:windows"),
        )
        self.assertLess(
            windows_job.index("pnpm run tauri:build:windows"),
            windows_job.index("Verify Windows Tauri release version metadata"),
        )

        for required in [
            "package.json",
            "tauri.conf.json",
            "Cargo.toml",
            "Cargo.lock",
            "ConvertFrom-Json",
            "version sources have drifted",
            "[uint16]::MaxValue",
            "cargo metadata",
            "--locked",
            "--manifest-path $cargoManifestPath",
        ]:
            with self.subTest(prepare_script_requirement=required):
                self.assertIn(required, WINDOWS_RELEASE_PREPARE_SCRIPT)

        for crlf_safe_pattern in [
            r'(?m)^version[ \t]*=[ \t]*"(?<value>[^"\r\n]+)"[ \t]*(?=\r?$)',
            r'(?<suffix>"[ \t]*)(?=\r?$)',
            r'(?m)^name[ \t]*=[ \t]*"luoxue-quota-viewer"[ \t]*(?=\r?$)',
        ]:
            with self.subTest(crlf_safe_pattern=crlf_safe_pattern):
                self.assertIn(crlf_safe_pattern, WINDOWS_RELEASE_PREPARE_SCRIPT)

        for required in [
            "[Diagnostics.FileVersionInfo]::GetVersionInfo",
            ".FileVersion",
            ".ProductVersion",
            ".FileVersionRaw",
            ".ProductVersionRaw",
            "$fileVersion -cne $ReleaseVersion",
            "$productVersion -cne $ReleaseVersion",
            "[uint16]::MaxValue",
        ]:
            with self.subTest(version_script_requirement=required):
                self.assertIn(required, WINDOWS_RELEASE_VERSION_SCRIPT)

    def test_release_toolchain_versions_are_bounded(self) -> None:
        self.assertEqual(RELEASE.count("version: 'v2.17.0'"), 2)
        self.assertNotIn("version: '~> v2'", RELEASE)
        self.assertIn(
            "test -s resources/model-pricing/model_prices_and_context_window.json",
            RELEASE,
        )
        self.assertEqual(RELEASE.count("> backend/cmd/server/VERSION"), 1)
        for config in [GORELEASER, GORELEASER_SIMPLE]:
            self.assertIn("-X main.Version={{.Version}}", config)

    def test_release_executes_binary_and_published_images(self) -> None:
        self.assertIn("Verify GoReleaser binary metadata", RELEASE)
        self.assertIn("dist/artifacts.json", RELEASE)
        self.assertIn('.type == "Binary"', RELEASE)
        self.assertIn('.extra.ID == "sub2api"', RELEASE)
        self.assertIn('"${binary_paths[0]}" --version 2>&1', RELEASE)
        self.assertIn("Verify published GHCR images", RELEASE)
        self.assertIn("for image_name in luoxueapi sub2api", RELEASE)
        self.assertIn("platforms=(linux/amd64)", RELEASE)
        self.assertIn("platforms=(linux/arm64 linux/amd64)", RELEASE)
        self.assertIn('docker pull --platform "$platform" "$image_ref"', RELEASE)
        self.assertIn(
            'docker run --rm --platform "$platform" "$image_ref" --version 2>&1',
            RELEASE,
        )
        self.assertIn("for attempt in 1 2 3", RELEASE)
        self.assertIn("org.opencontainers.image.version", RELEASE)
        self.assertIn("org.opencontainers.image.revision", RELEASE)
        self.assertIn("Run published GHCR image smoke", RELEASE)
        self.assertIn('SMOKE_IMAGE="ghcr.io/${REGISTRY_OWNER}/${image_name}', RELEASE)
        self.assertIn("bash backend/scripts/e2e-test.sh", RELEASE)
        self.assertIn(
            'LuoxueAPI ${RELEASE_VERSION} (commit: ${RELEASE_COMMIT},', RELEASE
        )

    def test_remote_actions_are_immutable(self) -> None:
        action_pattern = re.compile(
            r"(?m)^\s*uses:\s+(?!\./)([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)@([^\s#]+)"
        )
        for workflow in [RELEASE, BACKEND_CI, SECURITY]:
            matches = action_pattern.findall(workflow)
            self.assertTrue(matches)
            for action, reference in matches:
                with self.subTest(action=action):
                    self.assertRegex(reference, r"^[0-9a-f]{40}$")

    def test_tag_message_uses_collision_resistant_output_and_jq(self) -> None:
        self.assertIn("openssl rand -hex 16", RELEASE)
        self.assertNotIn("message<<EOF", RELEASE)
        self.assertIn("jq -n", RELEASE)
        self.assertIn('--arg text "$message"', RELEASE)
        self.assertIn('ghcr_image="ghcr.io/${owner}/luoxueapi"', RELEASE)

        hostile_messages = [
            "$() ' `", "line one\nEOF\nline three", "单引号'与 Unicode ❄️"
        ]
        for message in hostile_messages:
            delimiter = f"TAG_MESSAGE_{secrets.token_hex(16)}"
            while delimiter in message.splitlines():
                delimiter = f"TAG_MESSAGE_{secrets.token_hex(16)}"
            self.assertNotIn(delimiter, message.splitlines())

    def test_hostile_tags_are_rejected_as_data(self) -> None:
        self.assertIn('export RELEASE_TAG_CANDIDATE="$tag"', RELEASE)
        self.assertIn("python3 tools/release_semver.py", RELEASE)

        accepted = [
            "v0.0.0",
            "v1.2.3",
            "v1.2.3-rc.1",
            "v10.20.30-hotfix.2",
        ]
        rejected = [
            "v1.2",
            "v01.2.3",
            "v1.02.3",
            "v1.2.03",
            "v1.2.3-01",
            "v1.2.3-",
            "v1.2.3-rc..1",
            "v1.2.3.hotfix",
            "v1.2.3+",
            "v1.2.3+build.7",
            "v1.2.3-rc.1+build.7",
            "v1.2.3;id",
            "v1.2.3$(id)",
            "v1.2.3'",
            "v1.2.3\nEOF",
            "v１.２.３",
        ]
        for tag in accepted:
            self.assertTrue(is_release_semver(tag), tag)
        for tag in rejected:
            self.assertFalse(is_release_semver(tag), tag)


class ReleaseArtifactContractTest(unittest.TestCase):
    def test_images_and_archives_ship_offline_pricing_data(self) -> None:
        self.assertGreaterEqual(
            GORELEASER.count("backend/resources/model-pricing"),
            4,
        )
        self.assertGreaterEqual(
            GORELEASER_SIMPLE.count("backend/resources/model-pricing"),
            2,
        )
        self.assertIn("/app/backend/resources /app/resources", ROOT_DOCKERFILE)
        self.assertIn("/app/backend/resources /app/resources", DEPLOY_DOCKERFILE)
        self.assertIn(
            "COPY backend/resources/model-pricing /app/resources/model-pricing",
            GORELEASER_DOCKERFILE,
        )
        archive_pricing = GORELEASER.split(
            "src: backend/resources/model-pricing/*", 1
        )[1].split("checksum:", 1)[0]
        self.assertIn("dst: resources/model-pricing", archive_pricing)
        self.assertIn("strip_parent: true", archive_pricing)
        pricing_file = (
            ROOT
            / "backend/resources/model-pricing/model_prices_and_context_window.json"
        )
        self.assertTrue(pricing_file.is_file())
        self.assertGreater(pricing_file.stat().st_size, 0)

    def test_registry_migration_contract(self) -> None:
        self.assertIn("/luoxueapi\"", GORELEASER)
        self.assertIn("/sub2api\"", GORELEASER)
        self.assertIn("id: dockerhub-luoxueapi", GORELEASER)
        self.assertIn("/luoxueapi{{ end }}", GORELEASER)

        ghcr_primary = GORELEASER.split("id: ghcr-luoxueapi", 1)[1].split(
            "id: ghcr-sub2api-compat", 1
        )[0]
        ghcr_compat = GORELEASER.split("id: ghcr-sub2api-compat", 1)[1].split(
            "id: dockerhub-luoxueapi", 1
        )[0]
        dockerhub_primary = GORELEASER.split(
            "id: dockerhub-luoxueapi", 1
        )[1].split("id: dockerhub-sub2api-compat", 1)[0]
        dockerhub_compat = GORELEASER.split(
            "id: dockerhub-sub2api-compat", 1
        )[1].split("release:", 1)[0]
        simple_compat = GORELEASER_SIMPLE.split(
            "id: ghcr-sub2api-compat", 1
        )[1].split("release:", 1)[0]
        for compat in [ghcr_compat, dockerhub_compat, simple_compat]:
            self.assertIn('- "{{ .Version }}"', compat)
            self.assertNotIn("- latest", compat)
            self.assertNotIn("{{ .Major }}", compat)

        stable_only_tags = [
            '"{{ if not .Prerelease }}latest{{ end }}"',
            '"{{ if not .Prerelease }}{{ .Major }}.{{ .Minor }}{{ end }}"',
            '"{{ if not .Prerelease }}{{ .Major }}{{ end }}"',
        ]
        for primary in [ghcr_primary, dockerhub_primary]:
            self.assertIn('- "{{ .Version }}"', primary)
            for tag in stable_only_tags:
                self.assertIn(f"- {tag}", primary)
            self.assertNotRegex(primary, r"(?m)^\s+- latest\s*$")


class DeliveryPipelineContractTest(unittest.TestCase):
    def test_current_repository_is_the_delivery_baseline(self) -> None:
        current_repository = "3246147496lxt-dotcom/luoxueapi"
        for delivery_source in [INSTALL_SCRIPT, DOCKER_DEPLOY_SCRIPT, DEPLOY_README]:
            self.assertIn(current_repository, delivery_source)
            self.assertNotIn("Wei-Shaw/sub2api", delivery_source)
        for readme in [ROOT_README, ROOT_README_CN, ROOT_README_JA]:
            self.assertIn(current_repository, readme)
            self.assertNotIn(
                "raw.githubusercontent.com/Wei-Shaw/sub2api", readme
            )

    def test_node_and_pnpm_contract_is_reproducible(self) -> None:
        self.assertIn('"packageManager": "pnpm@9.15.9"', FRONTEND_PACKAGE)
        for workflow in [BACKEND_CI, SECURITY]:
            self.assertIn("node-version: '24'", workflow)
            self.assertIn("version: 9.15.9", workflow)
            self.assertIn("pnpm install --frozen-lockfile", workflow)
            self.assertIn('test "$(pnpm --version)" = "9.15.9"', workflow)
        self.assertIn("pnpm audit --audit-level=high --json", SECURITY)
        self.assertNotIn("pnpm audit --prod", SECURITY)

    def test_quota_viewer_is_a_reproducible_release_quality_gate(self) -> None:
        self.assertIn('"packageManager": "pnpm@9.15.9"', QUOTA_VIEWER_PACKAGE)
        quota_job = BACKEND_CI.split("  quota-viewer:", 1)[1].split(
            "\n  docs-site:", 1
        )[0]

        self.assertIn("runs-on: macos-15", quota_job)
        self.assertIn("timeout-minutes: 45", quota_job)
        self.assertIn("node-version: '24'", quota_job)
        self.assertIn("version: 9.15.9", quota_job)
        self.assertIn(
            "cache-dependency-path: quota-viewer/pnpm-lock.yaml", quota_job
        )
        self.assertIn('test "$(pnpm --version)" = "9.15.9"', quota_job)
        self.assertIn(
            "test \"$(rustc --version | awk '{print $2}')\" = \"1.96.0\"",
            quota_job,
        )
        self.assertIn("pnpm install --frozen-lockfile", quota_job)
        for command in [
            "pnpm run typecheck",
            "pnpm run test",
            "pnpm run build",
            "Production quota viewer bundle contains demo quota data",
            "cargo fmt --all -- --check",
            "cargo check --locked --all-targets",
            "cargo clippy --locked --all-targets -- -D warnings",
            "cargo test --locked --all-targets",
        ]:
            with self.subTest(command=command):
                self.assertIn(command, quota_job)
        self.assertNotIn("tauri build", quota_job)
        self.assertIn("quota-viewer-windows-release", RELEASE)

    def test_disconnected_compose_smoke_exercises_fallback(self) -> None:
        self.assertIn("internal: true", SMOKE_COMPOSE)
        self.assertIn("UPDATE_PROXY_URL: http://127.0.0.1:1", SMOKE_COMPOSE)
        self.assertIn("env_file: !reset []", SMOKE_COMPOSE)
        self.assertIn("ports: !reset []", SMOKE_COMPOSE)
        self.assertIn("SMOKE_APP_CONTAINER_NAME", SMOKE_COMPOSE)
        self.assertIn("/readyz", E2E_SCRIPT)
        self.assertIn('.status == "ready"', E2E_SCRIPT)
        for check in ["draining", "components", "database", "redis", "scheduler"]:
            with self.subTest(readiness_check=check):
                self.assertIn(f'.checks.{check}.status == "ready"', E2E_SCRIPT)
        self.assertIn("/api/v1/auth/login", E2E_SCRIPT)
        self.assertIn("/api/v1/admin/compliance", E2E_SCRIPT)
        self.assertIn("/api/v1/admin/compliance/accept", E2E_SCRIPT)
        self.assertIn("/model-pricing?model=", E2E_SCRIPT)
        self.assertIn("offline pricing fallback", E2E_SCRIPT)


if __name__ == "__main__":
    unittest.main()
