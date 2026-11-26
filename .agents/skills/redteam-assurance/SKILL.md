---
name: redteam-assurance
description: Execute the decoupled Untrusted Developer Red-Team Attack Simulation suite against New Vision Gaming to maintain autonomous defense parity.
---

# Skill: Autonomous Defense & Red-Team Assurance

1. **Run 6-Attack Simulation Suite Against New Vision Gaming:**
   - Execute `make simulate-attacks` (runs `../verifiable-software-factory/bin/factory --target . simulate-attacks`).
   - Confirm `6 / 6 Untrusted Developer Attacks Detected & Blocked` (`ATK-01` through `ATK-06`).
2. **Adding New Attack Scenarios:**
   - New red-team attack simulations belong in the decoupled factory repository (`../verifiable-software-factory/factory/internal/redteam/simulator.go`), never mixed into `new-vision-gaming/app/`.
3. **Verify Zero False Negatives:**
   - Ensure `ATK-05` proves BOTH Clean-Room hash mismatch AND Gatekeeper binary-tamper rejection on `dist/nvg-arcade`.
   - Ensure `ATK-06` proves a clean-syntax logic backdoor is caught by Clean-Room Go/GCC/Clang differential verification.
