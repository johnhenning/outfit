# Contributing

Start with AGENTS.md, README.md, and docs/profile-format.md. Open an issue for substantial provider/schema changes; small fixes can go straight to a focused PR.

Use Go 1.26+ and Python 3.12+. Build with `go build -o bin/outfit ./cmd/outfit`. Run `go test ./...`, `go vet ./...`, and `python3 -m unittest discover -s tests/bootstrap -v`. Native Windows CI additionally runs PowerShell/Pester bootstrap tests. Tests use disposable homes and synthetic credentials; package installs and live vault access are not development checks.

Keep the resolver deterministic and independent of machine mutation. New providers need tests for actual argument boundaries, failure detection, cancellation and reruns. New profile schema fields need strict validation and compatibility tests. New software defaults belong in outfit-profiles.

PRs should explain the user-visible change, why it is needed, and how it was verified. Include unresolved platform/manual steps. See docs/releases.md for publication and docs/hosting.md for domain routing.
