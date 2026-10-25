# Outfit contributor and agent guide

## Scope and boundaries
- This repo owns the reusable Go engine, provider implementations, schema, bootstraps and delivery pipeline.
- Public software defaults and dotfiles belong in outfit-profiles; private choices belong in outfit-personal.
- Read README.md and docs/profile-format.md before changing the source/profile contract. Schema changes require compatibility tests and release notes.
- Keep plan resolution pure. All side effects go through explicit executor/provider boundaries; pass executable/argument arrays, never a shell command from profile data.
- Remote bundles require HTTPS, SHA-256 verification, bounded extraction and supported schema validation before use. Do not execute profile hooks or templates.

## Working safely
- Never apply configs, install packages, access real vault items or run media/GPU/WSL provisioning on the developer workstation as a test.
- Use temporary homes, synthetic credentials and recording command runners. Never copy current ~/.config or auth caches into fixtures.
- Dotfile targets are allowlisted, canonicalized and checked for symlinks/conflicts. Private content must not appear in output, journals or backups.
- Keep terminal consent intact. Install-only bootstraps do not run setup; temporary profile bootstraps do not persist the CLI.
- Do not put personal GitHub/Cloudflare OAuth tokens in CI. Pages deployment uses GitHub's native token; domain routing is a separate infrastructure change.

## Development checks
- Go 1.26+; run gofmt on changed Go files.
- Run `go test ./...`, `go vet ./...`, and `python3 -m unittest discover -s tests/bootstrap -v`.
- Windows bootstrap tests: `Invoke-Pester ./tests/bootstrap/bootstrap.Tests.ps1 -CI` in PowerShell with Pester 5+.
- Before a behavior fix, add a regression test that demonstrates the failing contract. Test user-visible behavior, not source-text fragments.
- Validate public and personal profile combinations with catalog/plan commands; do not call apply in profile CI.

## Releases and review
- Keep README examples, CLI help, profile schema and bootstrap generator consistent.
- Use scripts/release.py; do not hand-edit generated installers or checksums.
- Pin public profiles independently of the engine; upload the exact built artifacts without rebuilding after checksums.
- CI evidence and clean-machine evidence are different. Report untested platform/service behavior explicitly.
- Keep changes focused and reviewable; include validation and manual steps in PR descriptions.
