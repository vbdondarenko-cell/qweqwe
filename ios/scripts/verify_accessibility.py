#!/usr/bin/env python3
from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "LinkUp"


def require(path: str, snippets: list[str]) -> list[str]:
    text = (ROOT / path).read_text(encoding="utf-8")
    return [f"{path}: missing {snippet!r}" for snippet in snippets if snippet not in text]


def forbid(path: str, snippets: list[str]) -> list[str]:
    text = (ROOT / path).read_text(encoding="utf-8")
    return [f"{path}: forbidden accessibility regression {snippet!r}" for snippet in snippets if snippet in text]


def main() -> int:
    failures: list[str] = []
    typography = (APP / "Core/Design/LinkUpTypography.swift").read_text(encoding="utf-8")
    if typography.count("relativeTo:") != 3:
        failures.append("LinkUpTypography must scale all three custom font families with Dynamic Type")

    failures += require("LinkUp/Features/Chat/ChatView.swift", [
        "@Environment(\\.accessibilityReduceMotion) private var reduceMotion",
        'accessibilityLabel(L10n.text("Send message"))',
        ".frame(width: 44, height: 44)",
    ])
    failures += require("LinkUp/Features/Create/CreateLinkView.swift", [
        'accessibilityLabel(L10n.text("Close Create LINK"))',
        '"Decrease capacity" : "Increase capacity"',
    ])
    failures += require("LinkUp/Features/Social/EditSlotView.swift", [
        '"Decrease capacity" : "Increase capacity"',
    ])
    failures += require("LinkUp/Features/Map/MapView.swift", [
        'accessibilityLabel(L10n.text("Close map selection"))',
    ])
    failures += require("LinkUp/Core/Design/LinkUpStates.swift", [
        ".frame(width: 44, height: 44)",
        'accessibilityLabel("Dismiss error")',
    ])
    failures += require("LinkUp/Features/Social/HostManagementView.swift", [
        ".frame(minHeight: 44)",
    ])

    failures += forbid("LinkUp/Features/Create/CreateLinkView.swift", [".frame(width: 36, height: 36)"])
    failures += forbid("LinkUp/Features/Pulse/PulseView.swift", [".frame(width: 40, height: 40)", ".frame(height: 42)"])
    failures += forbid("LinkUp/Features/Map/MapView.swift", [".frame(width: 32, height: 32)"])
    failures += forbid("LinkUp/Features/Me/MeView.swift", [".frame(width: 38, height: 38)"])
    failures += forbid("LinkUp/Core/Design/LinkUpStates.swift", [".frame(width: 24, height: 24)"])
    failures += forbid("LinkUp/Features/Pulse/SlotCardView.swift", [".frame(height: 42)"])

    if failures:
        print("\n".join(failures), file=sys.stderr)
        return 1
    print("accessibility source baseline OK: Dynamic Type, Reduce Motion, labels, known 44pt targets")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
