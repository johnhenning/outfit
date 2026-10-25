# Releases

Runtime and public profiles use separate semantic versions. Alpha releases use `v0.1.0-alpha.1` style tags. Personal configuration is private data; validate it in CI and version with private tags without publishing public assets.

1. Pass CI for the exact public profile commit. Tag it in outfit-profiles; its release workflow creates a draft with the deterministic profile archive and SHA256SUMS.
2. Review and publish that profile release.
3. Dispatch Outfit's release workflow with an engine tag, the exact profile commit SHA, and published profile tag. It runs native tests, builds all six binaries, verifies that the generated profile archive matches the published one, and creates a draft engine release with binaries, checksums and the release lock.
4. Review the draft and provenance, then publish it. The release event deploys the matching generated bootstrap site to GitHub Pages. No rebuild occurs between verified checksums and deployment.
5. Verify live endpoint status, script contents and the temporary-run flow in an isolated environment. Record actually exercised platforms in release notes.

Build locally with `python3 scripts/release.py --version VERSION --profiles ../outfit-profiles --profiles-version PROFILE_VERSION`. Generation is deterministic for identical source/toolchain inputs. Built assets are ignored by Git; source, dependency lockfiles and release metadata are versioned.

GitHub Actions uses read permissions by default. Release jobs elevate only contents/attestation access, and Pages deployment uses pages/id-token permissions. Profile archives are public but never include the private overlay. Do not use personal GitHub or Cloudflare OAuth tokens as workflow secrets.

The initial release is an alpha: full chezmoi templating, automatic WSL enable/reboot orchestration, GPU drivers, and media storage/service configuration remain manual. Some package providers may report manual installation; never describe those actions as completed installs.
