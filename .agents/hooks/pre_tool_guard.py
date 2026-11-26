#!/usr/bin/env python3
"""Antigravity 2.0 PreToolUse Zero-Trust Policy & Anti-Habituation Hook for New Vision Gaming.

Executes outside the LLM reasoning loop:
  - HARD DENY: secrets/keys access, unsafe/CGO/go:linkname injection under app/, destructive shell commands
  - FORCE_ASK: production deploys, git push, or modifying high-trust policy/cleanroom/workflow/hook files
  - AUTO_APPROVE: routine safe local read/build/verify commands (eliminates "Yes-click" fatigue)
  - ALLOW: defer everything else to Antigravity's native sandbox / user preset
"""

import json
import os
import re
import sys

SECRET_PATH_PATTERNS = [
    r"(^|/)\.factory-state/.*\.key\.json$",
    r"(^|/)\.env(\..+)?$",
    r"(^|/)id_rsa(\.pub)?$",
    r"\.(pem|key|p12|pfx)$",
    r"^/opt/antigravity/tokens/",
]

HIGH_TRUST_WRITE_PATTERNS = [
    r"(^|/)policy/",
    r"(^|/)cleanroom/",
    r"(^|/)\.github/workflows/",
    r"(^|/)\.agents/hooks(\.json|/)",
    r"(^|/)firebase/firestore\.rules$",
]

FORBIDDEN_APP_CODE_PATTERNS = [
    (r'import\s+(?:\(\s*)?["`]unsafe["`]', 'Importing "unsafe" violates CISA/NSA memory-safety policy'),
    (r'import\s+(?:\(\s*)?["`]C["`]', 'Importing cgo ("C") violates memory-safety & CGO_ENABLED=0 policy'),
    (r"//go:(linkname|cgo_|noescape|nosplit)\b", "Forbidden Go compiler directive under app/"),
]

DENY_CMD_PATTERNS = [
    (r"\brm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+(/(\*|\s|$)|~|\.\.)", "Destructive recursive delete blocked by repo policy"),
    (r"\bgit\s+push\b[^\n]*(--force|-f\b)", "Force-push is prohibited by 2PI branch integrity policy"),
    (r"\bgit\s+reset\s+--hard\b", "git reset --hard blocked by repo policy"),
    (r"\bcurl\b[^\n]*\|\s*(sh|bash)\b", "Piping remote script directly to shell is forbidden"),
    (r"/opt/antigravity/tokens", "Direct access to plugin token files is forbidden"),
]

FORCE_ASK_CMD_PATTERNS = [
    (r"\bgit\s+push\b", "Pushing commits to remote requires explicit human review"),
    (r"\bfirebase\s+deploy\b", "Firebase production/staging deploy requires explicit human approval"),
    (r"\bgcloud\s+run\s+deploy\b", "Cloud Run deployment requires explicit human approval"),
    (r"\bdocker\s+push\b", "Container registry push requires explicit human approval"),
]

SAFE_AUTO_APPROVE_PREFIXES = (
    "make fast-check",
    "make dast",
    "make test",
    "make build",
    "go test ",
    "go vet ",
    "git status",
    "git diff",
    "git log",
    "/tmp/tailwindcss ",
)


def emit(decision: str, reason: str = "") -> None:
    payload = {"decision": decision}
    if reason:
        payload["reason"] = reason
    sys.stdout.write(json.dumps(payload) + "\n")
    sys.exit(0)


def check_file_tool(name: str, args: dict) -> None:
    path = args.get("TargetFile") or args.get("AbsolutePath") or ""
    norm = path.replace("\\", "/")

    for pat in SECRET_PATH_PATTERNS:
        if re.search(pat, norm):
            emit("deny", f"Zero-trust policy: access to secret/key path blocked ({os.path.basename(norm)})")

    if name in ("write_to_file", "replace_file_content", "multi_replace_file_content"):
        # Check forbidden memory-safety / directive escapes inside app/
        if re.search(r"(^|/)app/", norm):
            content_parts = [
                args.get("CodeContent", ""),
                args.get("ReplacementContent", ""),
            ]
            for chunk in args.get("ReplacementChunks", []) or []:
                if isinstance(chunk, dict):
                    content_parts.append(chunk.get("ReplacementContent", ""))
            blob = "\n".join(content_parts)
            for pat, msg in FORBIDDEN_APP_CODE_PATTERNS:
                if re.search(pat, blob):
                    emit("deny", f"Zero-trust memory-safety hook: {msg}")

        # Require explicit human approval for high-trust boundary edits
        for pat in HIGH_TRUST_WRITE_PATTERNS:
            if re.search(pat, norm):
                emit("force_ask", f"High-trust governance/cleanroom file edit requires explicit human confirmation: {norm}")

    emit("allow")


def check_command_tool(args: dict) -> None:
    cmd = (args.get("CommandLine") or "").strip()

    for pat, msg in DENY_CMD_PATTERNS:
        if re.search(pat, cmd):
            emit("deny", f"Zero-trust command policy: {msg}")

    for pat, msg in FORCE_ASK_CMD_PATTERNS:
        if re.search(pat, cmd):
            emit("force_ask", f"Repo policy: {msg}")

    # Only auto-approve single simple commands (no command chaining or subshells)
    if not re.search(r"[;&|`$<>]", cmd):
        for prefix in SAFE_AUTO_APPROVE_PREFIXES:
            if cmd == prefix.strip() or cmd.startswith(prefix):
                emit("auto_approve", f"Routine verified repo command auto-approved: {prefix.strip()}")

    emit("allow")


def main() -> None:
    try:
        data = json.load(sys.stdin)
    except Exception:
        emit("force_ask", "Policy hook could not parse stdin JSON; requiring manual approval")

    tc = data.get("toolCall") or {}
    name = tc.get("name", "")
    args = tc.get("args") or {}

    if name == "run_command":
        check_command_tool(args)
    elif name in ("view_file", "write_to_file", "replace_file_content", "multi_replace_file_content"):
        check_file_tool(name, args)
    else:
        emit("allow")


if __name__ == "__main__":
    main()
