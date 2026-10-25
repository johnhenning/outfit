.PHONY: build test check demo
build:
	go build -o bin/outfit ./cmd/outfit

test:
	go test -race ./...
	python3 -m unittest discover -s tests/bootstrap -v
	python3 -m unittest discover -s tests/release -v

test-powershell:
	python3 scripts/test_bootstrap_fixtures.py
	pwsh -NoProfile -Command 'Invoke-Pester ./tests/bootstrap/bootstrap.Tests.ps1 -CI'

check:
	go vet ./...
	test -z "$$(gofmt -l cmd internal)"
	node --test tests/worker.test.mjs

demo: build
	bin/outfit setup --source ../outfit-profiles --profile workstation --dry-run
