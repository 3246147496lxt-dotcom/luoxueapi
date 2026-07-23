#!/usr/bin/env python3
"""Static contract for delivery-path runtime observability wiring."""

from __future__ import annotations

import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class DeliveryRuntimeContractTest(unittest.TestCase):
    def test_delivery_paths_record_runtime_signals(self) -> None:
        expected_calls = {
            "backend/internal/repository/migrations_runner.go": [
                "opsruntime.ObserveMigrationLock(",
            ],
            "backend/internal/repository/scheduler_outbox_repo.go": [
                "opsruntime.ObserveOutboxFence(",
                "opsruntime.ObserveOutboxBatch(",
            ],
            "backend/internal/repository/repository_tx.go": [
                "opsruntime.ObserveProducerTransactionRollback(",
            ],
            "backend/internal/service/scheduler_snapshot_service.go": [
                "opsruntime.ObserveOutboxBacklog(",
            ],
        }

        for relative_path, calls in expected_calls.items():
            source = (ROOT / relative_path).read_text(encoding="utf-8")
            for call in calls:
                with self.subTest(path=relative_path, call=call):
                    self.assertIn(call, source)

    def test_ops_dashboard_exposes_delivery_snapshot(self) -> None:
        dashboard = (ROOT / "backend/internal/service/ops_dashboard.go").read_text(
            encoding="utf-8"
        )
        models = (
            ROOT / "backend/internal/service/ops_dashboard_models.go"
        ).read_text(encoding="utf-8")

        self.assertIn(
            "overview.DeliveryRuntime = SnapshotDeliveryRuntimeStats()", dashboard
        )
        self.assertIn('json:"delivery_runtime"', models)


if __name__ == "__main__":
    unittest.main()
