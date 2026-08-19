#!/usr/bin/env python3
"""Emit the exact deterministic migration manifest used by the Go runner."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path


CANONICAL_NAME = re.compile(r"^[0-9]{3}[a-z]?_[a-z0-9][a-z0-9_]*\.sql$")


def build_manifest(root: Path) -> dict[str, object]:
    migrations_dir = root / "backend" / "migrations"
    entries: list[dict[str, str]] = []

    for path in sorted(migrations_dir.iterdir(), key=lambda item: item.name):
        if path.suffix.lower() != ".sql":
            continue
        if not CANONICAL_NAME.fullmatch(path.name):
            raise ValueError(f"invalid migration filename: {path.name}")
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"migration is not a regular file: {path.name}")
        # Every checked-in migration is UTF-8/ASCII SQL. str.strip() follows
        # the same leading/trailing Unicode whitespace contract as Go's
        # strings.TrimSpace for this repository's migration corpus.
        content = path.read_text(encoding="utf-8").strip().encode("utf-8")
        entries.append(
            {
                "filename": path.name,
                "sha256": hashlib.sha256(content).hexdigest(),
            }
        )

    set_digest = hashlib.sha256()
    for entry in entries:
        set_digest.update(entry["filename"].encode("utf-8"))
        set_digest.update(b"\0")
        set_digest.update(entry["sha256"].encode("ascii"))
        set_digest.update(b"\n")

    return {
        "contract": "sub2api-migration-manifest/v1",
        "set_sha256": set_digest.hexdigest(),
        "migrations": entries,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path.cwd())
    args = parser.parse_args()
    print(json.dumps(build_manifest(args.root.resolve()), separators=(",", ":")))


if __name__ == "__main__":
    main()
