#!/usr/bin/env python3
"""Static contract checks for deploy/.env.example and Compose files."""

from __future__ import annotations

import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
ENV_EXAMPLE = ROOT / "deploy/.env.example"
COMPOSE_FILES = [
    ROOT / "deploy/docker-compose.yml",
    ROOT / "deploy/docker-compose.local.yml",
    ROOT / "deploy/docker-compose.dev.yml",
    ROOT / "deploy/docker-compose.standalone.yml",
]
SMOKE_COMPOSE = ROOT / "deploy/docker-compose.smoke.yml"
E2E_SCRIPT = ROOT / "backend/scripts/e2e-test.sh"
ROOT_MAKEFILE = ROOT / "Makefile"
BACKEND_MAKEFILE = ROOT / "backend/Makefile"
CONFIG_SOURCE = ROOT / "backend/internal/config/config.go"
SETUP_SOURCE = ROOT / "backend/internal/setup/setup.go"
VALID_CATEGORIES = {"application", "postgresql", "redis", "compose"}
KEY_PATTERN = re.compile(r"^([A-Z][A-Z0-9_]*)=")
CATEGORY_PATTERN = re.compile(r"^# contract-category: ([a-z]+)$")


def classified_environment_keys() -> dict[str, str]:
    category: str | None = None
    classified: dict[str, str] = {}
    for line in ENV_EXAMPLE.read_text(encoding="utf-8").splitlines():
        if match := CATEGORY_PATTERN.fullmatch(line):
            category = match.group(1)
            continue
        if match := KEY_PATTERN.match(line):
            if category is None:
                raise AssertionError(f"{match.group(1)} has no contract category")
            if match.group(1) in classified:
                raise AssertionError(f"{match.group(1)} is classified more than once")
            classified[match.group(1)] = category
    return classified


class ComposeEnvironmentContractTest(unittest.TestCase):
    def test_every_example_key_has_one_supported_category(self) -> None:
        classified = classified_environment_keys()
        self.assertGreater(len(classified), 100)
        self.assertEqual(set(classified.values()) - VALID_CATEGORIES, set())

        self.assertEqual(classified["LUOXUEAPI_VERSION"], "compose")
        self.assertEqual(classified["BIND_HOST"], "compose")
        self.assertEqual(classified["SERVER_PORT"], "compose")
        self.assertEqual(classified["COMPOSE_STOP_GRACE_PERIOD"], "compose")
        self.assertEqual(classified["APPLE_CONTAINER_SUB2API_IMAGE"], "compose")
        self.assertEqual(classified["APPLE_CONTAINER_POSTGRES_IMAGE"], "compose")
        self.assertEqual(classified["APPLE_CONTAINER_REDIS_IMAGE"], "compose")
        self.assertEqual(classified["LOG_LEVEL"], "application")
        self.assertEqual(classified["SERVER_SHUTDOWN_GRACE_SECONDS"], "application")
        self.assertEqual(classified["SERVER_SHUTDOWN_FORCE_WAIT_SECONDS"], "application")
        self.assertEqual(classified["POSTGRES_MAX_CONNECTIONS"], "postgresql")
        self.assertEqual(classified["DATABASE_HOST"], "postgresql")
        self.assertEqual(classified["DATABASE_PASSWORD"], "postgresql")
        self.assertEqual(classified["DATABASE_MAX_OPEN_CONNS"], "postgresql")
        self.assertEqual(classified["REDIS_HOST"], "redis")
        self.assertEqual(classified["REDIS_MAXCLIENTS"], "redis")

    def test_compose_uses_complete_optional_env_contract_and_readyz(self) -> None:
        for compose_file in COMPOSE_FILES:
            text = compose_file.read_text(encoding="utf-8")
            with self.subTest(compose_file=compose_file.name):
                self.assertIn("env_file:", text)
                self.assertRegex(text, r"path:\s*\.env")
                self.assertRegex(text, r"required:\s*false")
                self.assertIn("/readyz", text)
                self.assertIn(
                    "stop_grace_period: ${COMPOSE_STOP_GRACE_PERIOD:-60s}", text
                )
                self.assertNotIn("weishaw/sub2api:latest", text.lower())

    def test_representative_application_keys_match_loader_contract(self) -> None:
        config = CONFIG_SOURCE.read_text(encoding="utf-8")
        setup = SETUP_SOURCE.read_text(encoding="utf-8")
        expected_config_keys = [
            "server.enable_server_timing",
            "server.max_request_body_size",
            "database.max_open_conns",
            "redis.pool_size",
            "gateway.max_body_size",
            "gateway.max_conns_per_host",
            "gateway.scheduling.outbox_poll_interval_seconds",
            "gateway.scheduling.outbox_backlog_rebuild_rows",
            "gateway.scheduling.shadow_comparison_enabled",
            "pricing.fallback_file",
        ]
        for key in expected_config_keys:
            with self.subTest(config_key=key):
                self.assertIn(key, config)
        self.assertIn('"SETUP_MIGRATION_TIMEOUT_SECONDS"', setup)

    def test_database_and_redis_server_tuning_reaches_server_commands(self) -> None:
        for compose_file in COMPOSE_FILES[:3]:
            text = compose_file.read_text(encoding="utf-8")
            with self.subTest(compose_file=compose_file.name):
                self.assertIn("max_connections=${POSTGRES_MAX_CONNECTIONS", text)
                self.assertIn("shared_buffers=${POSTGRES_SHARED_BUFFERS", text)
                self.assertIn(
                    '--maxclients "$${REDIS_MAXCLIENTS:-50000}"', text
                )
                self.assertIn(
                    "REDIS_MAXCLIENTS=${REDIS_MAXCLIENTS:-50000}", text
                )
                self.assertIn(
                    "set -- redis-server \\\n"
                    "          --save 60 1 \\\n"
                    "          --appendonly yes \\\n"
                    "          --appendfsync everysec \\\n",
                    text,
                )
                self.assertNotIn("command: >\n", text)

    def test_smoke_and_make_targets_cover_disconnected_runtime(self) -> None:
        smoke = SMOKE_COMPOSE.read_text(encoding="utf-8")
        e2e = E2E_SCRIPT.read_text(encoding="utf-8")
        root_make = ROOT_MAKEFILE.read_text(encoding="utf-8")
        backend_make = BACKEND_MAKEFILE.read_text(encoding="utf-8")

        self.assertIn("internal: true", smoke)
        self.assertIn("UPDATE_PROXY_URL: http://127.0.0.1:1", smoke)
        self.assertIn("env_file: !reset []", smoke)
        self.assertIn("ports: !reset []", smoke)
        self.assertIn("SMOKE_APP_CONTAINER_NAME", smoke)
        self.assertIn("SMOKE_POSTGRES_CONTAINER_NAME", smoke)
        self.assertIn("SMOKE_REDIS_CONTAINER_NAME", smoke)
        self.assertIn("docker-compose.smoke.yml", e2e)
        self.assertIn('export SMOKE_APP_CONTAINER_NAME="${project_name}-app"', e2e)
        self.assertIn('export SMOKE_POSTGRES_CONTAINER_NAME="${project_name}-postgres"', e2e)
        self.assertIn('export SMOKE_REDIS_CONTAINER_NAME="${project_name}-redis"', e2e)
        self.assertIn('base_url="http://127.0.0.1:8080"', e2e)
        self.assertGreaterEqual(e2e.count('exec -T sub2api'), 4)
        self.assertIn("/readyz", e2e)
        self.assertIn("/api/v1/admin/compliance", e2e)
        self.assertIn("/api/v1/admin/compliance/accept", e2e)
        self.assertIn("ack_phrase_en", e2e)
        self.assertIn("offline pricing fallback", e2e)
        self.assertRegex(root_make, r"(?m)^test-e2e:")
        self.assertRegex(backend_make, r"(?m)^test-e2e:")


if __name__ == "__main__":
    unittest.main()
