#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
SKIP_DIRS = {".git", "dist", "build", "release"}
SKIP_FILES = {"rsrc_amd64.syso"}
PATTERNS = {
    "GitHub personal access token": re.compile(r"gh[pousr]_[A-Za-z0-9_]{20,}"),
    "OpenAI-style secret": re.compile(r"\bsk-[A-Za-z0-9_-]{20,}"),
    "private key": re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    "Windows user profile path": re.compile(r"[A-Za-z]:\\Users\\[^\\\r\n]+", re.I),
    "Unix home path": re.compile(r"/(?:home|Users)/[^/\s]+/"),
}
FORBIDDEN_NAMES = {"config.json", "MouseButtonMapper.log", ".env"}

errors = []
for path in ROOT.rglob("*"):
    if not path.is_file() or any(part in SKIP_DIRS for part in path.parts):
        continue
    if path.name in FORBIDDEN_NAMES:
        errors.append(f"forbidden user-state file: {path.relative_to(ROOT)}")
        continue
    if path.name in SKIP_FILES:
        continue
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue
    for label, pattern in PATTERNS.items():
        if pattern.search(text):
            errors.append(f"{label}: {path.relative_to(ROOT)}")

if errors:
    print("Public repository audit failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)
print("Public repository audit passed.")
