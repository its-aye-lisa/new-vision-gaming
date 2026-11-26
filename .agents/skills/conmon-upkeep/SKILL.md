---
name: conmon-upkeep
description: Continuous Monitoring (ConMon), VRA triage, OpenVEX upkeep, and Red-Team regression verification for New Vision Gaming.
disable-model-invocation: true
---

# Skill: Autonomous Defense & ConMon Upkeep (`/conmon-upkeep`)

1. Run fast security preflight and red-team regression proof via the decoupled factory:
   ```bash
   make fast-check && make simulate-attacks
   ```
2. Check Go toolchain currency and run `govulncheck ./...` (if installed) to verify zero reachable CVEs against BOD 26-04 / FedRAMP 20x VDR SLAs (Critical <= 15d, High <= 30d).
3. Review `dist/vra-report.json` and `dist/vex.openvex.json`; if any CVE or control drift exists, draft a POA&M issue using `.github/ISSUE_TEMPLATE/security-poam-remediation.yml`.
