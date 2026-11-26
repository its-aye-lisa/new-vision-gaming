#!/usr/bin/env python3
"""Antigravity 2.0 Stop Pre-Completion Verification Gate for New Vision Gaming.

Prevents an agent from stopping with broken Go code, AST confinement violations,
Trojan Source BiDi characters, or failing exhaustive game invariants.
"""

import json
import os
import subprocess
import sys


def emit(decision: str, reason: str = "") -> None:
    payload = {"decision": decision}
    if reason:
        payload["reason"] = reason
    sys.stdout.write(json.dumps(payload) + "\n")
    sys.exit(0)


def main() -> None:
    try:
        data = json.load(sys.stdin)
    except Exception:
        emit("allow")

    # Guard against infinite stop-continue loops
    if (data.get("executionNum") or 0) >= 3:
        emit("allow")

    # Skip if background tasks are still running or termination was due to error
    if not data.get("fullyIdle", True) or data.get("terminationReason") != "model_stop":
        emit("allow")

    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))

    # Only run fast-check if git shows modifications under app/, spec/, policy/, cleanroom/, or go.mod
    try:
        status = subprocess.run(
            ["git", "status", "--porcelain", "--", "app", "spec", "policy", "cleanroom", "go.mod"],
            cwd=repo_root,
            capture_output=True,
            text=True,
            timeout=5,
        )
        if status.returncode == 0 and not status.stdout.strip():
            emit("allow")
    except Exception:
        pass

    try:
        res = subprocess.run(
            ["make", "fast-check"],
            cwd=repo_root,
            capture_output=True,
            text=True,
            timeout=20,
        )
    except Exception as exc:
        emit("allow", f"fast-check skipped: {exc}")

    if res.returncode != 0:
        tail = ((res.stdout or "") + "\n" + (res.stderr or "")).strip()[-1200:]
        emit(
            "continue",
            f"Pre-completion gate blocked stop: `make fast-check` failed. Fix all findings before finishing:\n{tail}",
        )

    emit("allow")


if __name__ == "__main__":
    main()
