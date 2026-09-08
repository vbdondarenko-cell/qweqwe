#!/usr/bin/env python3
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "LinkUp"
FILES = {
    "en": APP / "en.lproj" / "Localizable.strings",
    "uk": APP / "uk.lproj" / "Localizable.strings",
}

LINE = re.compile(r'^\s*("(?:\\.|[^"\\])*")\s*=\s*("(?:\\.|[^"\\])*")\s*;\s*$')
UI_PATTERNS = [
    re.compile(r'Text\("([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'Label\("([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'Button\("([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'LinkUpButton\(title:\s*"([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'navigationTitle\("([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'accessibilityLabel\("([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'TextField\("([^"\\]*(?:\\.[^"\\]*)*)"'),
    re.compile(r'SecureField\("([^"\\]*(?:\\.[^"\\]*)*)"'),
]


def parse(path: Path) -> dict[str, str]:
    result: dict[str, str] = {}
    for number, raw in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        if not raw.strip() or raw.lstrip().startswith("//"):
            continue
        match = LINE.match(raw)
        if not match:
            raise ValueError(f"{path}:{number}: invalid .strings syntax")
        key = json.loads(match.group(1))
        value = json.loads(match.group(2))
        if key in result:
            raise ValueError(f"{path}:{number}: duplicate key {key!r}")
        if not value:
            raise ValueError(f"{path}:{number}: empty value for {key!r}")
        result[key] = value
    return result


def main() -> int:
    try:
        tables = {language: parse(path) for language, path in FILES.items()}
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(error, file=sys.stderr)
        return 1

    en_keys = set(tables["en"])
    uk_keys = set(tables["uk"])
    missing_uk = sorted(en_keys - uk_keys)
    missing_en = sorted(uk_keys - en_keys)
    if missing_uk or missing_en:
        print(f"missing uk keys: {missing_uk}", file=sys.stderr)
        print(f"missing en keys: {missing_en}", file=sys.stderr)
        return 1

    missing_literals: list[str] = []
    for path in sorted(APP.rglob("*.swift")):
        if path.name == "LinkUpPlusView.swift":  # v1.2 surface, not v1.0 release gate
            continue
        text = path.read_text(encoding="utf-8")
        for pattern in UI_PATTERNS:
            for value in pattern.findall(text):
                if "\\(" in value:
                    continue
                if re.fullmatch(r"[a-z0-9.]+", value):  # SF Symbol / machine token, not user copy
                    continue
                if value not in en_keys:
                    missing_literals.append(f"{path.relative_to(ROOT)}: {value!r}")
    if missing_literals:
        print("active iOS UI literals missing localization keys:", file=sys.stderr)
        print("\n".join(missing_literals), file=sys.stderr)
        return 1

    print(f"localization OK: {len(en_keys)} keys; en/uk parity; active v1.0 static UI covered")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
