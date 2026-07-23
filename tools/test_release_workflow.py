#!/usr/bin/env python3
"""Static security contract for the release workflow.

This intentionally avoids executing release input. It guards the properties that
prevent GitHub input and annotated-tag text from becoming shell source.
"""

from __future__ import annotations

import re
import secrets
import sys
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
            "frontend",
            "docs-site",
            "embedded-web",
            "compose-smoke",
            "golangci-lint",
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
        self.assertGreaterEqual(BACKEND_CI.count("if-no-files-found: error"), 4)

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

    def test_disconnected_compose_smoke_exercises_fallback(self) -> None:
        self.assertIn("internal: true", SMOKE_COMPOSE)
        self.assertIn("UPDATE_PROXY_URL: http://127.0.0.1:1", SMOKE_COMPOSE)
        self.assertIn("env_file: !reset []", SMOKE_COMPOSE)
        self.assertIn("ports: !reset []", SMOKE_COMPOSE)
        self.assertIn("SMOKE_APP_CONTAINER_NAME", SMOKE_COMPOSE)
        self.assertIn("/readyz", E2E_SCRIPT)
        self.assertIn("/api/v1/auth/login", E2E_SCRIPT)
        self.assertIn("/api/v1/admin/compliance", E2E_SCRIPT)
        self.assertIn("/api/v1/admin/compliance/accept", E2E_SCRIPT)
        self.assertIn("/model-pricing?model=", E2E_SCRIPT)
        self.assertIn("offline pricing fallback", E2E_SCRIPT)


if __name__ == "__main__":
    unittest.main()
