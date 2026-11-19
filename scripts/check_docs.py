#!/usr/bin/env python3
"""Validate local Markdown links and required ADR headings."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
LINK = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
REQUIRED_ADR_HEADINGS = ("## Status", "## Context", "## Decision", "## Alternatives considered", "## Consequences")


def main() -> int:
    errors = []
    for markdown in ROOT.rglob("*.md"):
        if ".git" in markdown.parts or "node_modules" in markdown.parts:
            continue
        text = markdown.read_text(encoding="utf-8")
        for target in LINK.findall(text):
            target = target.strip().split()[0].strip("<>")
            if not target or target.startswith(("http://", "https://", "mailto:", "#")):
                continue
            path_part, _, fragment = target.partition("#")
            destination = (markdown.parent / path_part).resolve() if path_part else markdown.resolve()
            if not destination.exists():
                errors.append(f"{markdown.relative_to(ROOT)}: missing link target {target}")
            elif fragment and destination.is_file() and destination.suffix == ".md":
                content = destination.read_text(encoding="utf-8")
                anchors = {
                    re.sub(r"[^\w -]", "", heading.lstrip("#").strip().lower()).replace(" ", "-")
                    for heading in re.findall(r"(?m)^#{1,6}\s+.+$", content)
                }
                if fragment.lower() not in anchors:
                    errors.append(f"{markdown.relative_to(ROOT)}: missing heading anchor {target}")
    for adr in (ROOT / "docs/architecture/decisions").glob("ADR-*.md"):
        text = adr.read_text(encoding="utf-8")
        for heading in REQUIRED_ADR_HEADINGS:
            if heading not in text:
                errors.append(f"{adr.relative_to(ROOT)}: missing required section {heading}")
    if errors:
        print("\n".join(f"FAIL: {error}" for error in errors), file=sys.stderr)
        return 1
    print("PASS: Markdown local links and ADR sections")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
