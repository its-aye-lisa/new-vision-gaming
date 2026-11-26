---
name: compliance-gate
description: Full FedRAMP Moderate / 20x / DoD IL4 / NIST SSDF compliance verification, Gatekeeper admission, and Red-Team attack proof via the decoupled verifiable-software-factory.
disable-model-invocation: true
---

# Skill: FedRAMP Moderate & DoD IL4 Compliance Gate (`/compliance-gate`)

1. Execute the full decoupled Verifiable Software Factory and Fail-Closed Admission Gatekeeper:
   ```bash
   make factory && make admit
   ```
2. Execute all 6 Untrusted Developer Red-Team Attack Simulations:
   ```bash
   make simulate-attacks
   ```
3. Inspect `dist/fedramp-il4-evidence.json`, `dist/aegis-findings.sarif`, `dist/vex.openvex.json`, and `dist/vra-report.json` to confirm:
   - `overallCompliant == true`
   - `fips140Enabled == true`
   - `thirdPartyDepCount == 0`
   - `0` SARIF findings and `releaseBlocked == false`
4. If `conftest` is installed, run OPA policy evaluation against `../verifiable-software-factory/policy/opa`.
5. Summarize audit readiness for 3PAO / AO / cATO review.
