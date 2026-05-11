.PHONY: semantic-gate check-core e2e-smoke e2e-full

semantic-gate:
	@./scripts/semantic-gate.sh

check-core:
	@./scripts/check-core.sh

e2e-smoke:
	@npm run test:e2e:smoke

e2e-full:
	@npm run test:e2e:full
