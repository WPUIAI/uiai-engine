.PHONY: test browser-stress browser-soak browser-reliability release-browser-reliability

test:
	go test ./...

browser-stress:
	SESSIONS=$${SESSIONS:-4} ROUNDS=$${ROUNDS:-10} OUT=$${OUT:-/tmp/uiai-browser-diagnostics-4x10.json} scripts/stress-browser-diagnostics.sh

browser-soak:
	DURATION_SECONDS=$${DURATION_SECONDS:-30} CONCURRENCY=$${CONCURRENCY:-2} OUT=$${OUT:-/tmp/uiai-browser-flakiness-soak.json} scripts/soak-browser-flakiness.sh

browser-reliability: test browser-stress browser-soak

release-browser-reliability:
	DURATION_SECONDS=$${DURATION_SECONDS:-300} CONCURRENCY=$${CONCURRENCY:-2} OUT=$${OUT:-/tmp/uiai-browser-flakiness-soak-5m.json} scripts/soak-browser-flakiness.sh
