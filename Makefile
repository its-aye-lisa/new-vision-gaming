.PHONY: fast-check check-wasm build-css factory dast admit simulate-attacks test serve serve-ads clean

FACTORY_DIR ?= ../verifiable-software-factory
FACTORY_BIN := $(FACTORY_DIR)/bin/factory

$(FACTORY_BIN):
	@$(MAKE) -C $(FACTORY_DIR) build

# Tier 1 (<1s): Sub-second preflight via external reusable Software Factory
fast-check: $(FACTORY_BIN)
	@$(FACTORY_BIN) fast-check --target .
	@$(MAKE) --no-print-directory check-wasm

# Phase 2 App Store Guard: guarantees every pure game kernel compiles cleanly to offline WASM
check-wasm:
	@GOOS=js GOARCH=wasm go build ./app/pkg/...
	@echo "[PASS] Offline Mobile WASM Portability Check (GOOS=js GOARCH=wasm go build ./app/pkg/...)"

build-css:
	@/tmp/tailwindcss -i app/web/styles/input.css -o app/web/static/tailwind.css --minify

# Tier 4: Full 9-Gate Verifiable Software Factory + Gatekeeper
factory: $(FACTORY_BIN)
	@$(FACTORY_BIN) run-all --target .

dast: $(FACTORY_BIN)
	@$(FACTORY_BIN) dast --target .

admit: $(FACTORY_BIN)
	@$(FACTORY_BIN) admit --target .

simulate-attacks: $(FACTORY_BIN)
	@$(FACTORY_BIN) simulate-attacks --target .

test:
	@go test -v -count=1 ./app/...

# Launch Verified Arcade (Strict Zero-Inline CSP, FIPS 140-3)
serve:
	@GODEBUG=fips140=on ./dist/nvg-arcade \
		--addr 127.0.0.1:8080 \
		--require-fips \
		--bundle dist/release-bundle.signed.json \
		--trust-roots .factory-state/keys/trust-roots.json

# Launch Verified Arcade with AdSense Nonce Strict CSP Enabled
serve-ads:
	@GODEBUG=fips140=on ADS_ENABLED=true ./dist/nvg-arcade \
		--addr 127.0.0.1:8080 \
		--require-fips \
		--bundle dist/release-bundle.signed.json \
		--trust-roots .factory-state/keys/trust-roots.json

clean:
	@rm -rf dist .factory-state
