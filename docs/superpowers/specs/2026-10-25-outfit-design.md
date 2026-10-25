# Dotfiles and machine setup

Status: approved by the user on 2026-10-25; implementation planning underway. Retain the proposed Go implementation and chezmoi integration as the build baseline.

## Purpose

Two independent GitHub repositories deliver Outfit: `outfit` holds the reusable Go CLI/TUI, package-provider implementations, schema, and bootstraps; `outfit-profiles` holds software catalogs, role/preset definitions, platform package mappings, and chezmoi dotfiles. A small downloadable bootstrap starts setup on a fresh machine. The same tool supports subsequent updates and configuration changes. The current Mac is evidence for a software catalog, not a requirement to reproduce every installed app.

## Confirmed requirements

- macOS, Debian-family Linux, Fedora-family Linux, native Windows, and WSL.
- Both native Windows and WSL setup, coordinated so desktop apps belong to Windows and Linux development tooling belongs to WSL.
- Combinable software presets including default, development, and entertainment.
- Different configuration for development environments, compute nodes, and media nodes.
- A short guided setup with an advanced configuration menu, including alternatives for browsers and other software categories.
- Browser alternatives include Firefox, Chromium, and Dia, as supported on the target platform.
- Desktop software including browsers, ChatGPT, Claude, and common personal apps.
- uv manages Python versions, environments, and Python tools. Do not install or activate mise in the managed configuration.
- Manage shell, tmux, editor, Git, SSH, and other appropriate configs.
- Reusable credentials backed by 1Password where appropriate.
- Discover existing tooling and configurations before deciding what to migrate.

## Proposed implementation choices

These are builder recommendations, not user-confirmed technology choices.

Use chezmoi for dotfile templates, differences between machines, and updates. Use a Go executable for the cross-platform orchestration and terminal interface; the exact maintained TUI library is left to implementation planning. Go avoids requiring Python, Node, or a preconfigured development environment to start setup. Publish release binaries for supported OS/architecture combinations, with small POSIX-shell and PowerShell bootstraps. Unsupported combinations stop with an actionable message rather than executing an incompatible artifact.

Use declarative manifests for software, categories, presets, platform package identifiers, detection, and configuration dependencies. Keep UI, planning, and execution separate so interactive and noninteractive modes use the same resolved plan. Prefer native package managers: Homebrew on macOS, apt on Debian-family Linux, dnf on Fedora-family Linux, and winget on Windows. Vendor repositories, runtime installers, Flatpak, and manual installs are explicit per-package providers rather than silent arbitrary fallbacks.

The existing workspace is empty apart from Git metadata and has no configured remote. Implement here after design approval; repo name, GitHub owner, and publication visibility can be settled at publication time. Do not invent a working GitHub bootstrap URL before publication.

## Setup experience

The default path is: detect platform and architecture; choose machine role; choose software presets; review recommended choices; optionally open Advanced; inspect the resulting action plan; apply; review results and required manual steps.

Advanced groups settings by category: browsers, terminals, editors, AI apps and CLIs, Python/Rust/Node/Java tooling, containers, cloud/ML, communication/productivity, media, remote access, shell/dotfiles, and credentials. Each category supports a recommended selection, alternatives, and None. Allow multiple choices where useful, but keep a distinct preferred app selection. Installing an app does not silently make it the OS default. Unsupported apps are visible with a reason and cannot be selected for installation.

Every action explains its target and provider. Show package installs, dotfile changes, service changes, privilege requirements, restarts, and manual steps before applying. Persist nonsecret selections locally; optionally export a sanitized named machine manifest for reuse. Provide plain prompts if rich terminal features are unavailable. A piped bootstrap must reconnect interactive input to the terminal; noninteractive execution requires explicit options and must never hang on prompts.

Proposed command surface: setup, plan, apply, update, doctor, config, and export. Exact flag spelling is left to the builder. Users can select a role, presets, and manifest without a TUI, and obtain machine-readable plans and results. Update does not implicitly upgrade every package on the system.

## Roles and presets

Role describes how the machine runs; presets describe installed capabilities. Keep them independent and permit a compute node to add development tools or a workstation to add entertainment.

- Workstation: interactive desktop with default personal apps and optional development/entertainment bundles.
- Development environment: CLI-first development baseline; ask whether it is a desktop, remote VM, container, or WSL environment. Desktop apps are included only when desktop mode is selected.
- Compute node: headless by default, minimal shell/Git/tmux/uv/monitoring and optional containers or ML tooling. No desktop browser, credential GUI, or media apps by default. GPU drivers/CUDA are advanced opt-ins with explicit compatibility checks and reboot/manual-step reporting.
- Media node: headless by default, media processing and optional media-server capability. FFmpeg can be selected; server products, libraries, mounts, storage paths, ports, credentials, and service activation require explicit advanced configuration. No automatic disk formatting or network exposure.

Presets: default (small shared baseline), dev (build/runtime/editor tooling), entertainment (personal media and gaming apps on desktop machines), cloud-ml (optional heavier/cloud tooling), and custom. A role supplies recommendations and config defaults, not an inflexible bundle. Reject incompatible selections with an explanation rather than silently dropping them.

## Current workstation evidence

Read-only scan on 2026-10-25: application bundle names and metadata, Homebrew installed packages, uv tool directory names, config structure, shell integration names, and executable-name frequencies from recent zsh history. No private-key contents, credential values, browser profiles, or full history commands were collected into this document. Installed does not prove regularly used; command frequencies support only CLI usage inference.

Detected browser apps: Safari, Google Chrome, Arc, Dia, and Zen. Firefox and Chromium are requested alternatives rather than detected installs.

Detected developer apps: Cursor, Visual Studio Code, JetBrains Toolbox, CLion, IntelliJ IDEA, RustRover, Fleet, Gateway, Writerside, Insomnia, Docker Desktop, Podman Desktop, Ghostty, Warp, iTerm2, Graphite, DevUtils, Blender, Godot, and NVIDIA Nsight Deep Learning Designer. Preserve these as catalog candidates; verify current installer availability and maintenance before offering automated installation.

AI apps/tools: ChatGPT, Claude, Ollama, LM Studio, Unsloth Studio; CLI commands include claude, codex, cursor-agent, and agent. Determine each installer and configuration independently; do not transfer account sessions.

Communication/productivity: Slack, Discord, Signal, WhatsApp, Beeper, Zoom, Notion, Obsidian, Asana, Microsoft Office/Outlook/OneNote, OneDrive, Thunderbird, Proton Mail Bridge, Things, Reader, Kindle, iA Writer, Craft, and other personal productivity apps. They remain optional rather than all becoming defaults.

Entertainment/creative: Spotify, Steam, Stremio, Balatro, Baldur's Gate 3, Final Cut Pro, iMovie, GarageBand, Pocket Sync, and Udeler. Store-managed/licensed apps may require manual installation or sign-in; record that accurately rather than claiming automation.

Remote/network/VM tooling: Tailscale, AnyDesk, VNC Viewer, Windows App, Jump Desktop, Parallels Desktop, UTM, Raspberry Pi Imager, balenaEtcher. Keep agent/service activation explicit.

Mac utilities: Raycast, Alfred, Rectangle Pro, Bartender, Alcove, Caffeine, Setapp, CleanShot X, CleanMyMac, ForkLift, Paste, Session, Unclutter, Logitech tools, and Applite. These are optional macOS catalog entries; no need to invent Linux/Windows equivalents automatically.

Frequently observed CLI tooling: git, uv, cargo, gh, pnpm, aws, docker, Flyte, W&B, Claude, Codex, cmake, ninja, ruff, and tmux. Existing standalone uv tools include coiled, dstack, flyte, flytekit, google-colab-cli, marimo, mkdocs, pre-commit, ruff, wandb, and zenml. Keep specialized packages in optional groups; do not recreate the entire Homebrew dependency graph as explicitly requested packages.

Existing config families: Oh My Zsh with Powerlevel10k and git/mise/docker plugins; tmux under ~/.config/tmux; Neovim/NvChad with custom overrides and lockfile; Ghostty in its macOS Application Support location; Git ignore/config; Rust/Cargo; SSH. Inspect individual candidate files with secret redaction before migration. Retain useful customizations and licensing, remove mise activation only in the proposed managed files, and do not change the current workstation as a side effect of building the repo.

## Dotfile ownership and updates

Manage explicit allowlisted files only. Preserve machine-local settings through documented include files and template inputs. Do not recursively import ~/.config, entire application support folders, or nested Git repositories. Existing editor distributions should be managed by supported upstream install/config mechanisms with personal overrides kept separately, rather than copying a full vendored checkout without thought.

Before replacing an existing target, show a diff, make a private local backup, and record what changed. Reruns reconcile existing state; do not duplicate shell initialization or SSH blocks. Rollback restores managed config files when possible; package installs and service side effects are reported separately because a full machine rollback cannot be promised.

Windows and WSL have separate homes and dotfile targets. Native Windows gets a PowerShell profile, terminal/editor configs, and native SSH settings. WSL gets Unix shell/tmux/Python configs, with an explicit SSH integration strategy. Native tmux is not promised on Windows; offer it within WSL. Configure installed WSL distros explicitly; creating/enabling WSL is an opt-in privileged action that may require a restart and resume.

## Credentials and SSH

1Password is the preferred secret source. On interactive desktops install its app and CLI, then guide the user through sign-in and enabling CLI integration/SSH agent. Record uncompleted authorization steps honestly. Do not read or copy the user's existing 1Password account configuration as reusable dotfiles.

Keep reusable host aliases and public-key selectors in SSH templates; use local-only/private overlays for sensitive host metadata. Keep private keys in 1Password where supported. Existing on-disk keys are not automatically uploaded or removed; migration is a separate explicit operation. Do not globally copy current permissive host-key checking rules. Preserve host-specific behavior when importing and validate effective SSH configuration without connecting to remote hosts.

Use op:// secret references, op run for process-scoped credentials, and op inject only when software requires rendered files. Rendered secrets must have restrictive permissions, be excluded from Git/logs/backups, and have an explicit lifetime. Chezmoi templates must not casually materialize secrets during previews. Prefer native cloud SSO for AWS/Azure rather than moving cached sessions or long-lived credentials wholesale.

Headless nodes cannot assume the 1Password desktop SSH agent exists. Offer no-secret setup by default, interactive CLI authentication when suitable, or an explicitly configured narrowly scoped service-account integration. Do not provision a broad personal-vault service token or enable SSH agent forwarding by default. A service-account token must enter through a local protected channel, never a committed machine manifest.

Windows/WSL SSH handling follows the selected integration's actual config location, rather than assuming Linux SSH config controls ssh.exe. Validate configuration resolution and availability without triggering secret disclosure or remote login.

## Bootstrap and release trust

The approved name is Outfit, with repos `outfit` and `outfit-profiles`. Use `/tools/outfit/` for documentation and scripts:

- `/tools/outfit/install.sh` and `install.ps1` install only the CLI.
- `/tools/outfit/run.sh` and `run.ps1` download a temporary CLI and launch interactive setup without permanently installing Outfit.
- `/tools/outfit/profiles/dev.sh`, `workstation.sh`, `compute.sh`, and `media.sh` (and PowerShell equivalents) run a known named profile directly. These scripts contain an exact engine release and config-bundle checksum; they do not require Git, Go, Python, or an installed Outfit CLI. They still preview changes and ask before applying unless explicit unattended flags are given.
- `/tools/outfit/v1/` reserves stable major-version entry points.

A profile is a named composition of role, presets, package choices and dotfile sets. Profile bundles are independent, versioned data with a supported schema version and a content digest. The engine accepts a local bundle or an HTTPS URL plus SHA-256 digest; it rejects incompatible schemas and untrusted archive paths/symlinks before any action. User-owned executable scripts and package-provider command implementations are not taken from profile data. Profiles may define only supported declarative actions and templates; chezmoi script/external/hook execution is disabled for untrusted profile bundles. Allow an advanced user to select a different profile source, with its exact source/digest shown in review. Engine and profile versions are recorded separately in machine state and saved plans. Development/test engine fixtures are synthetic; personal production defaults stay in the profiles repo.

Leave the root domain for a personal site and `/tools/<future-tool>/` for unrelated tools. Treat these as planned URLs until deployed and verified.

Public DNS inspection on 2026-10-25 found Cloudflare nameservers (rosemary and noah), with no apex A/AAAA answers or www CNAME answer in that check; HTTPS could not resolve. DNS hosting is identified, but registrar, account access, and a website hosting destination are not yet established. Existing email and unrelated DNS records must be preserved. Hosting selection and DNS configuration remain part of delivery.

Support shell bootstrap on Unix and PowerShell bootstrap on Windows. Download a versioned release into a temporary directory, verify it against pinned release metadata/checksums, and run it as the user. HTTPS and checksum validation detect download corruption; checksums hosted with the same release alone do not establish publisher authenticity. Document that limitation and provide an inspect-before-run path. Never pipe downloaded content to an elevated shell. Request privilege only for individual actions that need it.

A public bootstrap/repo is the recommended initial publication model, containing nonsecret configs and examples only; private host overlays remain local or vault-backed. Private repository delivery is also possible but needs an authentication bootstrap and is a publication-time decision. Actual URLs are generated only after the destination exists.

## Validation and completion boundaries

Validate manifest resolution for platform/architecture/role combinations, package-provider mappings, conflict and unsupported-option behavior, dry-run nonmutation, repeat application, backup/conflict handling, and secret redaction. Exercise TUI navigation and cancellation. Use Linux containers/VMs for safe package-provider checks where possible and native Windows CI for PowerShell/planning checks. Linux containers cannot prove desktop integration, reboot behavior, or system services; Windows CI alone cannot prove a fresh native/WSL installation.

Do not run package installs, service activation, WSL provisioning, or dotfile application on the current workstation merely to test this repo. Use isolated temporary homes and explicit test environments. Release completion must distinguish checked plans and local tests from actual clean-machine installs. Manual license/sign-in/reboot steps are part of the result.

## Decisions left to implementation

Pick the Go TUI library, manifest schema, executable name, release automation, precise package versions/providers, and small default app selection. Choose conservative defaults and make them editable. Media-server product and GPU stack remain unselected until the user chooses them in advanced setup. GitHub destination and visibility are resolved before publishing; no credential migration is implied by repo implementation.
