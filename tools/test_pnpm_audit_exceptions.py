#!/usr/bin/env python3
"""Unit contract for fail-closed pnpm audit report validation."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
sys.path.insert(0, str(ROOT / "tools"))

from check_pnpm_audit_exceptions import validate_audit_report  # noqa: E402


class AuditReportValidationTest(unittest.TestCase):
    def test_accepts_supported_complete_reports(self) -> None:
        self.assertEqual(
            validate_audit_report(
                {"vulnerabilities": {}, "metadata": {"vulnerabilities": {}}}
            ),
            [],
        )
        self.assertEqual(
            validate_audit_report(
                {"advisories": {}, "metadata": {"vulnerabilities": {}}}
            ),
            [],
        )

    def test_rejects_empty_or_registry_error_reports(self) -> None:
        self.assertTrue(validate_audit_report({}))
        self.assertTrue(
            validate_audit_report(
                {"error": {"code": "ENETUNREACH"}, "metadata": {}}
            )
        )

    def test_rejects_partial_reports_without_metadata(self) -> None:
        self.assertTrue(validate_audit_report({"vulnerabilities": {}}))


if __name__ == "__main__":
    unittest.main()
