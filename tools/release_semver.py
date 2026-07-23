#!/usr/bin/env python3
"""Strict SemVer validator for release tags.

The workflow passes the candidate through the environment so no tag text is
ever interpolated into shell source.  The accepted grammar is SemVer 2.0.0
core plus optional prerelease, with the repository's required leading ``v``.
Build metadata is deliberately excluded because ``+`` is not valid in the
version portion of the OCI/Docker tag published by this same release.
"""

from __future__ import annotations

import os
import re
import sys


_NUMERIC = r"(?:0|[1-9][0-9]*)"
_PRERELEASE_IDENTIFIER = rf"(?:{_NUMERIC}|[0-9]*[A-Za-z-][0-9A-Za-z-]*)"
_SEMVER_TAG = re.compile(
    rf"^v{_NUMERIC}\.{_NUMERIC}\.{_NUMERIC}"
    rf"(?:-{_PRERELEASE_IDENTIFIER}(?:\.{_PRERELEASE_IDENTIFIER})*)?"
    r"$"
)


def is_release_semver(tag: str) -> bool:
    return _SEMVER_TAG.fullmatch(tag) is not None


def main() -> int:
    tag = os.environ.get("RELEASE_TAG_CANDIDATE", "")
    if not is_release_semver(tag):
        print("Release tag is not valid SemVer", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
