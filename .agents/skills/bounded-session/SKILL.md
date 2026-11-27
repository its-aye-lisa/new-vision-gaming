---
name: bounded-session
description: Bounded AI engineering workflow using a Daily Orchestration Session plus disposable Ephemeral Working Sessions (Explore, Execute, Clean-Eyes Review) tailored for New Vision Gaming.
---

# Skill: Bounded Session Workflow (`/bounded-session`)

Use this skill to keep game development, UI/ad updates, and infrastructure work contained, predictable, and free of scope creep or conversational drift.

## 1. Two-Tier Cadence

1. **Daily Orchestration Session (`.session/daily-plan.md`, `.gitignore`'d):**
   - Sets 1–3 bounded objectives (e.g., a single GitHub Intent Issue), explicit deferred items, and stopping points for the day (`Plan → Act → Review → Repeat`).
   - **Commit-Boundary Gate:** Promote only human-approved **Verified Facts** (backed by `make fast-check`, `go test`, `make dast`, or commit SHAs) into the daily plan; park tangential ideas separately.
2. **Ephemeral Working Sessions:**
   - **Explore (read-only):** Clarify requirements, run `/grill-me`, and lock `spec/*.spec.json` decisions before coding.
   - **Execute (bounded build):** Edit only `Allowed Paths`, then run the cheapest-sufficient verification tier:
     ```bash
     make fast-check          # Tier 1 (< 1s)
     go test -v ./app/...     # Tier 2 (< 1s)
     make dast                # Tier 3 (~2s, when routes/headers/templates change)
     ```
   - **Clean-Eyes Review (fresh diff-only):** Review `git diff` against the spec and `OUT OF SCOPE` list in a fresh context (or via `cleanroom-ivv-auditor` / `ad-cwv-auditor`) with zero implementation transcript.

## 2. Ephemeral Working Session Brief Template

```markdown
## Ephemeral Working Session Brief
- **Mode:** Explore (read-only) | Execute (code + verify) | Clean-Eyes Review (diff-only)
- **Single Objective:** <one concrete, verifiable outcome>
- **Allowed Paths / Files:** <explicit list or glob>
- **Explicitly OUT OF SCOPE:** <what NOT to touch, refactor, or expand>
- **Done Condition (Verifiable):** `make fast-check && go test ./app/...` (plus `make dast` if HTTP/UI changed)
- **Circuit Breaker:** Stop and report back after 2 failed attempts or if scope needs to expand.
- **Return Artifact:** 1) Verified Facts (command/SHA proof), 2) Decisions Made, 3) Parked Ideas.
```

## 3. Handover File (`.handoff/latest.md`, `.gitignore`'d)

Before resetting a worn session or switching phases, write `.handoff/latest.md` with:
- Current objective & mode
- Locked decisions & verified command outputs
- Pointers to modified files / commits (do not paste whole files)
- Exact next step & remaining `OUT OF SCOPE` guardrails
