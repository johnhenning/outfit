# Outfit execution ledger

Plan: docs/superpowers/plans/2026-10-25-outfit.md

- Ruling: independent repos replace the original single-repo plan, as explicitly requested by the user. External schema-versioned profile bundles replace embedded production catalogs.
- Ruling: work directly in the two new repos. There is no prior code/history to protect with linked worktrees.
- Pre-flight: platform.Target flows into resolver, providers and TUI; catalog.Load takes an external root; named Profile resolves to role/presets before action resolution; saved plans include bundle digest.
- Bootstrap contract: install is persistent CLI-only; run and known-profile scripts use a temporary binary. All downloads finish and verify before execution.
- Publication pending repo visibility and authorized hosting/DNS access.

- Tasks 1–4: core platform detection, external bundle/schema validation, profile resolution, saved-plan validation, explicit package execution and journals implemented; Go tests pass in isolated fixtures.
- Tasks 5–6: safe static dotfile preview/apply with conflict detection, private-file protection and backups implemented; personal overlay fetching and process-scoped 1Password reference commands implemented.
- Ruling: use a static chezmoi-compatible source subset applied by Outfit directly in the initial version. This prevents profile bundles from executing hooks/templates. Cost: full chezmoi templating and sync commands remain a later feature.
- Ruling: Windows and WSL use separate entry points in this alpha. Automatic WSL provisioning/reboot orchestration remains manual; native and Linux profile resolution are covered independently. Cost: one additional user step on new Windows machines.
- Task 8: TUI role/profile/software category menus and plain review added; event tests pass.
- Task 9: self-contained shell/PowerShell generators and release builder added; five shell bootstrap tests prove temporary execution, install-only, checksum/download failures and truncation behavior.
- User revision: public outfit runtime + public outfit-profiles templates + private outfit-personal overlay; authentication is deferred until private config/secrets are requested.

- Independent review: five Important findings fixed with regressions: canonical private-key path rejection, scrollable complete plan review, effective inherited-package checkbox state, Windows WinGet tool discovery after installs, and atomic PowerShell replacement. Go suite and bootstrap suites passed after the fixes. No deferred minor findings.
- Native CI verified green on Linux, macOS and Windows; public/private data validation green.
- Interactive PTY smoke: profile selection, Advanced categories and inherited checkboxes rendered correctly; cancelling exited without creating machine state.
- Local container daemons unavailable; a disposable Ubuntu container install/rerun/dotfile smoke workflow is running on GitHub instead.
- Organization creation/transfers explicitly deferred by the user. Keep all repositories under johnhenning.
- Automatic approval rejected the initial whole-host Worker custom domain. Resolved by deploying only the /tools/outfit/* path route; no whole-host Worker is attached.
