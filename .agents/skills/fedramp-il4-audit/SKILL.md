---
name: fedramp-il4-audit
description: Audit New Vision Gaming code, Firebase/container IaC, and generated evidence against FedRAMP Moderate (Rev 5 & 20x KSIs), DoD CC SRG IL4, DISA STIGs, and NIST SP 800-218 SSDF.
---

# Skill: FedRAMP Moderate / 20x / DoD IL4 Compliance Audit

1. **Run Decoupled Verifiable Factory & Admission Control:**
   - Execute `make factory && make admit` (invokes `../verifiable-software-factory/bin/factory --target .`).
   - Confirm all 9 Gatekeeper checks pass.
2. **Verify Machine-Readable Evidence Artifacts in `dist/`:**
   - `dist/fedramp-il4-evidence.json`: confirm `overallCompliant == true`, `fips140Enabled == true`, `thirdPartyDepCount == 0`.
   - `dist/aegis-findings.sarif`: confirm `runs[0].results` is empty.
   - `dist/vex.openvex.json` & `dist/vra-report.json`: confirm `releaseBlocked == false`.
3. **Verify Policy-as-Code & IaC Hardening:**
   - Inspect `Dockerfile` (`FROM scratch`, `GOFIPS140=v1.0.0`, `USER 65532:65532`, `GODEBUG=fips140=on`).
   - Inspect `firebase/firestore.rules` (fail-closed) and `infra/k8s/` (`runAsNonRoot`, `readOnlyRootFilesystem`, `drop: [ALL]`, `egress: []`, Kyverno Cosign/SLSA).
   - If `conftest` is installed, run `conftest test dist/fedramp-il4-evidence.json infra/k8s/deployment.yaml -p ../verifiable-software-factory/policy/opa`.
4. **Report Concise Audit Summary:** Highlight any drift, SLA timers, or open POA&M items.
