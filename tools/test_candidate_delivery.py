#!/usr/bin/env python3
"""Regression gates for candidate SQL and clean immutable image delivery."""

from __future__ import annotations

import json
import os
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.dont_write_bytecode = True
from migration_manifest import build_manifest  # noqa: E402


ROOT = Path(__file__).resolve().parents[1]
CANDIDATE_DIR = ROOT / "deploy" / "candidate"
BUILD_SCRIPT = ROOT / "tools" / "build_candidate_image.sh"
FORWARD_SCHEMA_SCRIPT = ROOT / "deploy" / "tests" / "candidate-forward-schema-test.sh"
IDENTITY_FILTER = ROOT / "deploy" / "tests" / "extract-candidate-database-identity.awk"
E2E_SCRIPT = ROOT / "backend" / "scripts" / "e2e-test.sh"
CANDIDATE_COMPOSE = ROOT / "deploy" / "docker-compose.candidate-ci.yml"
EMBED_TEST = ROOT / "backend" / "internal" / "web" / "embed_test.go"
OBSOLETE_MARKERS = {
    "candidate-c305b0376a-final.sh",
    "773aa8d60602f473bac2a7e76a1bcddca80599169926bf104157d0fa6b805964",
    "c305b0376aaa51b6149d073c16fa9c7d12dc2db0",
    "codex-deploy-c305b0376a",
    "cand_c305b0376a",
}


def candidate_sources() -> dict[Path, str]:
    paths = sorted(CANDIDATE_DIR.rglob("*")) + [
        BUILD_SCRIPT,
        FORWARD_SCHEMA_SCRIPT,
        E2E_SCRIPT,
        CANDIDATE_COMPOSE,
    ]
    return {
        path: path.read_text(encoding="utf-8")
        for path in paths
        if path.is_file()
    }


class CandidateDeliveryContractTest(unittest.TestCase):
    def test_shell_syntax_and_optional_shellcheck(self) -> None:
        shell_scripts = [
            BUILD_SCRIPT,
            FORWARD_SCHEMA_SCRIPT,
            E2E_SCRIPT,
            *sorted(CANDIDATE_DIR.rglob("*.sh")),
        ]
        for script in shell_scripts:
            with self.subTest(script=script.relative_to(ROOT)):
                subprocess.run(["bash", "-n", str(script)], check=True)

        shellcheck = shutil.which("shellcheck")
        if shellcheck is None:
            self.skipTest("shellcheck is not installed")
        subprocess.run([shellcheck, *map(str, shell_scripts)], check=True)

    def test_server_awk_compatibility_and_no_obsolete_payloads(self) -> None:
        for path, source in candidate_sources().items():
            with self.subTest(path=path.relative_to(ROOT)):
                for line_number, line in enumerate(source.splitlines(), start=1):
                    stripped = line.lstrip()
                    self.assertFalse(
                        stripped.startswith("||") or stripped.startswith("&&"),
                        f"{path}:{line_number} begins with a shell connector",
                    )
                for marker in OBSOLETE_MARKERS:
                    self.assertNotIn(marker, source)
                self.assertNotRegex(source, r"\br(?:[1-9]|10)\b")

    def test_snapshot_restore_uses_unambiguous_saved_run_relation(self) -> None:
        suspend = (
            CANDIDATE_DIR / "sql" / "suspend_skill_import_review.sql"
        ).read_text(encoding="utf-8")
        restore = (
            CANDIDATE_DIR / "sql" / "restore_skill_import_review.sql"
        ).read_text(encoding="utf-8")

        self.assertIn("skill_import_runs also has", restore)
        self.assertIn("FROM candidate_validation.review_run_snapshot AS saved_run", restore)
        self.assertIn("SET status = saved_run.status", restore)
        executable_sql = "\n".join(
            line for line in restore.splitlines() if not line.lstrip().startswith("--")
        )
        self.assertNotRegex(executable_sql, r"\bAS\s+snapshot\b")
        self.assertNotIn("snapshot.status", executable_sql)
        self.assertIn("SET status = 'cancelled'", suspend)
        self.assertIn("AS saved_run", suspend)

    def test_content_fingerprint_excludes_only_stars_runtime_projection(self) -> None:
        source = (
            CANDIDATE_DIR / "sql" / "skill_content_fingerprint.sql"
        ).read_text(encoding="utf-8")
        excluded = set(re.findall(r"'(repository_stars[^']*)'", source))
        self.assertEqual(
            excluded,
            {
                "repository_stars",
                "repository_stars_fetched_at",
                "repository_stars_refresh_after",
            },
        )
        exclusion_array = source.split("ARRAY[", 1)[1].split("]::TEXT[]", 1)[0]
        self.assertNotIn("updated_at", exclusion_array)

    def test_manifest_contract_is_exact_and_deterministic(self) -> None:
        first = build_manifest(ROOT)
        second = build_manifest(ROOT)
        self.assertEqual(first, second)
        self.assertEqual(first["contract"], "sub2api-migration-manifest/v1")
        self.assertEqual(len(first["migrations"]), 269)
        self.assertEqual(
            first["set_sha256"],
            "49acdb5ff514eb8729bf6d84dd4c3e7d9d03400521b2361413a6ff12e1597e6d",
        )
        self.assertEqual(
            first["migrations"][-22:],
            [
                {
                    "filename": "201_library_files.sql",
                    "sha256": "03f6a53d92e93fbfee37b9e5dd78dc11cae49dd921812253d0425154f1a9c23e",
                },
                {
                    "filename": "201a_library_alias_unique_index_notx.sql",
                    "sha256": "ba15a71ce63180c21f8addda85351b13171a0e6c22f7427bcb4b8c955499e564",
                },
                {
                    "filename": "201b_library_alias_constraints.sql",
                    "sha256": "f52ac96a80583b4e7a3c7c5f9923eee5d95a47c4a2b2d9844c864f31abe83833",
                },
                {
                    "filename": "202_chat_message_activities.sql",
                    "sha256": "e2ee8b4480af916327f132d378eb70b2291c85efba0ced4555452147b56fdb8f",
                },
                {
                    "filename": "231_add_users_email_alias_dedup_index_notx.sql",
                    "sha256": "dca6d92a4567ab9fabc3550062acbec57ba89e4e452a1d7e2f17c3cf97e2d556",
                },
                {
                    "filename": "232_add_users_email_normalized_index_notx.sql",
                    "sha256": "052a61bf4bdc89a5215970059a61096f4eaea5c244b6781f3ec42d6ac8e8bb5d",
                },
                {
                    "filename": "233_group_profit_control.sql",
                    "sha256": "b39b90d72d8869dc46beeb426f5db112ff04235c89ddb6d0ecee61a9bea95381",
                },
                {
                    "filename": "234_add_usage_log_upstream_response_model.sql",
                    "sha256": "cad520cbfcf7af7ea9acae92e5bcbe27501fd9e3ad5b02e306f4f97be4410a82",
                },
                {
                    "filename": "235_add_usage_log_upstream_model_mismatch_index_notx.sql",
                    "sha256": "692f2a75f0c62670b4d68986912bf24eb92f6377ec904d3806ff7d62b0da8355",
                },
                {
                    "filename": "236_projects.sql",
                    "sha256": "050ad388c07995c4167ebd5ef52211f5cc2f04dfb74d6ab6655403d03f9936ce",
                },
                {
                    "filename": "237_skill_catalog_localizations.sql",
                    "sha256": "89f58ab9526f6eb21175f22ab44d073fada7c4601dc62060687f83f93c8ac1d3",
                },
                {
                    "filename": "238_skill_catalog_zh_001_167.sql",
                    "sha256": "13f8328f3a4da795351cf742908b761cc406cfc6a102eee4a6291ca984d47a91",
                },
                {
                    "filename": "239_skill_catalog_zh_168_334.sql",
                    "sha256": "b5de647a5ad20292ab538a556df71ad57c4932d070293f6804272d894f15a62d",
                },
                {
                    "filename": "240_skill_catalog_zh_335_500.sql",
                    "sha256": "05a8f15c65ce5e2f81c58fdeb3c270ebb7d1519e05209bc3f9c3e242bb0e33f1",
                },
                {
                    "filename": "241_skill_catalog_zh_batch_1.sql",
                    "sha256": "ee1d129d4f2ec009ca9a751b1db3f90e944269670a812926fc2b827fc0834401",
                },
                {
                    "filename": "242_skill_catalog_zh_batch_2.sql",
                    "sha256": "b4f3f4210bc2ce171ce4cd682bf739520f83d4e691475162c08f9454f2d882db",
                },
                {
                    "filename": "243_skill_catalog_zh_batch_3.sql",
                    "sha256": "ac5b08667cd780ea2c87de7f4ac0c8aa0de4aa2367aa8a81d4a38f1a18633cd6",
                },
                {
                    "filename": "244_skill_catalog_zh_batch_4.sql",
                    "sha256": "54635128c55355fed279ef97fefa51a9d71e07670567fca60abb156a1f2b9e57",
                },
                {
                    "filename": "245_skill_catalog_zh_batch_5.sql",
                    "sha256": "1a4fe1650914b06427e66f64654b8aae4f2dadebafc80db65663e46a77517ad1",
                },
                {
                    "filename": "246_skill_catalog_zh_batch_6.sql",
                    "sha256": "26af86008459fffc6aca2a3d73224c74b930b074fd3519e8df211898a0c36d97",
                },
                {
                    "filename": "247_user_platform_quotas_add_deepseek.sql",
                    "sha256": "6c6816fadf6ea30f2cfd0fabfc7852ddf2270c6da4dd49691f1764f16019d8c4",
                },
                {
                    "filename": "248_channel_monitor_deepseek_provider.sql",
                    "sha256": "03f90a36eec0e53e524d32479bfe8a377302afd259900516a12aee93dbaa10b7",
                },
                {
                    "filename": "249_zhipu_provider.sql",
                    "sha256": "3091d6c40ceaa24be39f94c81235d743e751727780f0cd4c8b0fb818860182e4",
                },
            ],
        )

    def test_build_script_never_inspects_env_or_mutates_production_resources(self) -> None:
        source = BUILD_SCRIPT.read_text(encoding="utf-8")
        self.assertNotIn(".Config.Env", source)
        self.assertNotIn("docker inspect", source)
        self.assertNotIn("docker container rm", source)
        self.assertNotIn("docker volume rm", source)
        self.assertNotIn("docker network rm", source)
        self.assertNotIn("docker system prune", source)
        self.assertNotIn("docker builder prune", source)
        self.assertNotIn("docker compose", source)
        self.assertIn("docker image inspect --format", source)
        self.assertIn("git -C \"$repo_dir\" archive", source)
        self.assertIn(".codex-deploy-tmp", source)
        self.assertIn("backend/cmd/skill-market-quickprep", source)
        self.assertIn(":sha-${CANDIDATE_COMMIT}", source)
        self.assertNotIn(":latest", source)

    def test_runtime_entrypoint_mode_is_explicit_and_non_root_is_exercised(self) -> None:
        for dockerfile in [
            ROOT / "Dockerfile",
            ROOT / "Dockerfile.goreleaser",
            ROOT / "deploy" / "Dockerfile",
        ]:
            with self.subTest(dockerfile=dockerfile.relative_to(ROOT)):
                source = dockerfile.read_text(encoding="utf-8")
                self.assertIn("chmod 0755 /app/docker-entrypoint.sh", source)
                self.assertNotIn("chmod +x /app/docker-entrypoint.sh", source)

        build_script = BUILD_SCRIPT.read_text(encoding="utf-8")
        self.assertIn("--user sub2api", build_script)
        self.assertIn("non-root default entrypoint", build_script)

    def test_forward_schema_runtime_is_isolated_and_workers_are_disabled(self) -> None:
        script = FORWARD_SCHEMA_SCRIPT.read_text(encoding="utf-8")
        compose = CANDIDATE_COMPOSE.read_text(encoding="utf-8")
        self.assertIn("@sha256:", script)
        self.assertIn("EXPECTED_OLD_MIGRATIONS:-246", script)
        self.assertIn("EXPECTED_CANDIDATE_MIGRATIONS:-269", script)
        self.assertIn("--migrate-only", script)
        self.assertIn("for replay in first second", script)
        self.assertIn("--volumes --remove-orphans", script)
        self.assertIn("candidate-forward-", script)
        self.assertIn("Reproduce normal Git checkout readability", script)
        self.assertRegex(script, r"\(\n  umask 022\n  tar -xf")
        self.assertIn("identity_stdout_file", script)
        self.assertIn("extract-candidate-database-identity.awk", script)
        self.assertNotIn("docker system prune", script)
        self.assertNotIn("docker builder prune", script)
        self.assertNotIn(".Config.Env", script)
        for setting in [
            'BATCH_IMAGE_ENABLED: "false"',
            'BATCH_IMAGE_QUEUE_ENABLED: "false"',
            'BATCH_IMAGE_VERTEX_ENABLED: "false"',
            'SKILL_IMPORT_ENABLED: "false"',
            'SKILL_IMPORT_WORKER_ENABLED: "false"',
            "LIBRARY_STORAGE_DRIVER: local",
            "LIBRARY_STORAGE_DIR: /app/data/library-files",
        ]:
            with self.subTest(setting=setting):
                self.assertIn(setting, compose)

    def test_database_identity_filter_rejects_missing_or_ambiguous_json(self) -> None:
        identity = {
            "contract": "sub2api-database-identity/v2",
            "database": "candidate",
            "system_identifier": "123456789",
            "in_recovery": False,
        }
        identity_line = json.dumps(identity, separators=(",", ":"), sort_keys=True)
        progress = " Container candidate-forward-app Creating\n"
        completed = " Container candidate-forward-app Removed\n"

        result = subprocess.run(
            ["awk", "-f", str(IDENTITY_FILTER)],
            input=progress + identity_line + "\n" + completed,
            text=True,
            capture_output=True,
            check=False,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout), identity)

        for invalid in [progress, identity_line + "\n" + identity_line + "\n"]:
            with self.subTest(invalid=invalid):
                rejected = subprocess.run(
                    ["awk", "-f", str(IDENTITY_FILTER)],
                    input=invalid,
                    text=True,
                    capture_output=True,
                    check=False,
                )
                self.assertNotEqual(rejected.returncode, 0)
                self.assertEqual(rejected.stdout, "")

    def test_failed_build_cleans_only_its_owned_temp_directory(self) -> None:
        with tempfile.TemporaryDirectory(prefix="candidate-delivery-test-") as root:
            test_root = Path(root)
            repo = test_root / "repo"

            def ignore(directory: str, names: list[str]) -> set[str]:
                ignored = {
                    ".git",
                    ".codex-deploy-tmp",
                    "node_modules",
                    "target",
                    "coverage",
                }
                if Path(directory) == ROOT / "backend" / "cmd":
                    ignored.add("skill-market-quickprep")
                if Path(directory) == ROOT / "backend" / "internal" / "web":
                    ignored.add("dist")
                return set(names) & ignored

            shutil.copytree(ROOT, repo, ignore=ignore)
            subprocess.run(["git", "init", "-q"], cwd=repo, check=True)
            subprocess.run(
                ["git", "config", "user.email", "candidate-test@example.invalid"],
                cwd=repo,
                check=True,
            )
            subprocess.run(
                ["git", "config", "user.name", "Candidate Test"],
                cwd=repo,
                check=True,
            )
            subprocess.run(["git", "add", "-A"], cwd=repo, check=True)
            subprocess.run(["git", "commit", "-qm", "fixture"], cwd=repo, check=True)
            commit = subprocess.check_output(
                ["git", "rev-parse", "HEAD"], cwd=repo, text=True
            ).strip()

            fake_bin = test_root / "fake-bin"
            fake_bin.mkdir()
            fake_docker = fake_bin / "docker"
            fake_docker.write_text("#!/bin/sh\nexit 73\n", encoding="utf-8")
            fake_docker.chmod(fake_docker.stat().st_mode | stat.S_IXUSR)

            owned_tmp = test_root / "tmp"
            owned_tmp.mkdir()
            sentinel = test_root / "must-survive"
            sentinel.write_text("preserve", encoding="utf-8")
            env = os.environ.copy()
            env.update(
                {
                    "PATH": f"{fake_bin}:{env['PATH']}",
                    "TMPDIR": str(owned_tmp),
                    "CANDIDATE_COMMIT": commit,
                    "CANDIDATE_VERSION": "2.0.0-rc.5",
                    "CANDIDATE_IMAGE_REPOSITORY": "ghcr.io/example/luoxueapi",
                }
            )
            result = subprocess.run(
                ["bash", str(repo / "tools" / "build_candidate_image.sh")],
                cwd=repo,
                env=env,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 73, result.stderr)
            self.assertEqual(sentinel.read_text(encoding="utf-8"), "preserve")
            self.assertEqual(list(owned_tmp.glob("sub2api-candidate.*")), [])

    def test_failed_forward_schema_run_cleans_only_its_owned_project(self) -> None:
        with tempfile.TemporaryDirectory(prefix="candidate-forward-test-") as root:
            test_root = Path(root)
            fake_bin = test_root / "fake-bin"
            fake_bin.mkdir()
            docker_log = test_root / "docker.log"
            fake_docker = fake_bin / "docker"
            fake_docker.write_text(
                "#!/bin/sh\n"
                f"printf '%s\\n' \"$*\" >> {docker_log!s}\n"
                "if [ \"${1:-}\" = buildx ]; then exit 73; fi\n"
                "exit 0\n",
                encoding="utf-8",
            )
            fake_docker.chmod(fake_docker.stat().st_mode | stat.S_IXUSR)

            owned_tmp = test_root / "tmp"
            owned_tmp.mkdir()
            sentinel = test_root / "must-survive"
            sentinel.write_text("preserve", encoding="utf-8")
            available_old_commit = subprocess.check_output(
                ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True
            ).strip()
            env = os.environ.copy()
            env.update(
                {
                    "PATH": f"{fake_bin}:{env['PATH']}",
                    "TMPDIR": str(owned_tmp),
                    "CANDIDATE_IMAGE": "ghcr.io/example/luoxueapi@sha256:"
                    + "a" * 64,
                    "CANDIDATE_COMMIT": "b" * 40,
                    "CANDIDATE_VERSION": "2.0.0-rc.5",
                    "OLD_APPLICATION_COMMIT": available_old_commit,
                }
            )
            result = subprocess.run(
                ["bash", str(FORWARD_SCHEMA_SCRIPT)],
                cwd=ROOT,
                env=env,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 73, result.stderr)
            self.assertEqual(sentinel.read_text(encoding="utf-8"), "preserve")
            self.assertEqual(list(owned_tmp.glob("candidate-forward.*")), [])
            invocations = docker_log.read_text(encoding="utf-8")
            self.assertRegex(
                invocations,
                r"compose --project-name candidate-forward-[0-9]+-[0-9]+ .* "
                r"down --volumes --remove-orphans",
            )
            self.assertNotIn("system prune", invocations)
            self.assertNotIn("builder prune", invocations)

    def test_recursive_embedded_asset_gate_uses_positive_size_not_100_bytes(self) -> None:
        source = EMBED_TEST.read_text(encoding="utf-8")
        self.assertIn("TestEmbeddedFrontendServesEveryBuiltAssetExactly", source)
        self.assertIn("require.NotEmpty(t, body", source)
        self.assertNotRegex(source, r"(?:Len|length|size)[^\n]{0,80}(?:100|>\s*100)")


if __name__ == "__main__":
    unittest.main()
