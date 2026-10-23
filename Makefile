.PHONY: test desktop-contract-parity fpv-assets fpv-visual-smoke fpv-visual-baselines browser-stress browser-soak browser-reliability release-browser-reliability

test:
	go test ./...

desktop-contract-parity:
	python3 scripts/check-desktop-contract-parity.py
	go test ./internal/desktopcontract -count=1
	npm --prefix apps/cockpit test -- tests/unit/desktop-presentation-contract.test.ts
	cargo test --manifest-path tools/desktop-contract-check/Cargo.toml

fpv-assets:
	scripts/build-fpv-assets.sh

fpv-visual-smoke:
	scripts/smoke-fpv-visual-breakpoints.sh

fpv-visual-baselines:
	UPDATE_BASELINE=1 scripts/smoke-fpv-visual-breakpoints.sh

browser-stress:
	SESSIONS=$${SESSIONS:-4} ROUNDS=$${ROUNDS:-10} OUT=$${OUT:-/tmp/uiai-browser-diagnostics-4x10.json} scripts/stress-browser-diagnostics.sh

browser-soak:
	DURATION_SECONDS=$${DURATION_SECONDS:-30} CONCURRENCY=$${CONCURRENCY:-2} OUT=$${OUT:-/tmp/uiai-browser-flakiness-soak.json} scripts/soak-browser-flakiness.sh

browser-reliability: test browser-stress browser-soak

release-browser-reliability:
	DURATION_SECONDS=$${DURATION_SECONDS:-300} CONCURRENCY=$${CONCURRENCY:-2} OUT=$${OUT:-/tmp/uiai-browser-flakiness-soak-5m.json} scripts/soak-browser-flakiness.sh
