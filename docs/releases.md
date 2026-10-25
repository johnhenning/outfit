# Releases

Runtime and public profiles use separate semantic versions. Alpha releases use
`v0.1.0-alpha.1` style tags. Nonsecret personal selections may also be public:
validate the overlay, package only allowlisted static sources, and publish a
separately pinned standalone personal bundle. Private host data, credentials,
SSH keys, sessions and authenticated synced data must never enter public assets.

1. Pass CI for the exact public profile commit. Tag it in outfit-profiles; its release workflow creates a draft with the deterministic profile archive and SHA256SUMS.
2. Review and publish that profile release.
3. Dispatch Outfit's release workflow with an engine tag, the exact profile commit SHA, and published profile tag. It runs native tests, builds all six binaries, verifies that the generated profile archive matches the published one, and creates a draft engine release with binaries, checksums and the release lock.
4. Review the draft and provenance, then publish it. The release event deploys the matching generated bootstrap site to GitHub Pages. No rebuild occurs between verified checksums and deployment.
5. Verify live endpoint status, script contents and the temporary-run flow in an isolated environment. Record actually exercised platforms in release notes.

The alpha.3 engine/installers reuse the alpha.2 general and personal bundles.
New installer UX does not require changing software selections. CLI installers
report verified success and shell-specific PATH instructions; they do not edit
startup files or Windows user PATH automatically.

For personal publishing, run outfit-personal's release workflow with the exact
general-profile commit, verify its standalone archive/checksums, and publish it
before the site workflow. The site workflow verifies the engine lock and personal
archive before generating the personal endpoints. GitHub sign-in is not needed
to fetch public bundles; app and 1Password authentication happen afterward.

Verify the actual response on both custom domains after every Pages deployment,
including the release version inside `install.sh`. A successful Actions result
alone is insufficient: deployments from the same source SHA can reuse the Pages
deployment identifier. Do not stage an older engine site from the same commit
immediately before publishing its newer release. If the uploaded artifact is
correct but the origin still serves the earlier site, deploy the verified
release from a fresh reviewed source commit and check the live contents again.

Build locally with `python3 scripts/release.py --version VERSION --profiles ../outfit-profiles --profiles-version PROFILE_VERSION`. Generation is deterministic for identical source/toolchain inputs. Built assets are ignored by Git; source, dependency lockfiles and release metadata are versioned.

GitHub Actions uses read permissions by default. Release jobs elevate only contents/attestation access, and Pages deployment uses pages/id-token permissions. Public archives contain only deliberately public nonsecret data. Do not use personal GitHub or Cloudflare OAuth tokens as workflow secrets.

The initial release is an alpha: full chezmoi templating, automatic WSL enable/reboot orchestration, GPU drivers, and media storage/service configuration remain manual. Some package providers may report manual installation; never describe those actions as completed installs.
