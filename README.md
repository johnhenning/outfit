# Outfit

Equip a workstation, development environment, compute node, or media node from reusable profiles.

Outfit is the Go CLI/TUI and bootstrap runtime. Public software catalogs and dotfile templates live in [outfit-profiles](https://github.com/johnhenning/outfit-profiles); optional personal profiles are loaded as a separate authenticated overlay. The binary contains provider implementations and a schema, not personal software defaults.

## Entry points

Planned custom-domain endpoints (not live until hosting/DNS is configured):

```sh
# Install only the Outfit CLI.
curl -fsSL https://johnhenning.dev/tools/outfit/install.sh | sh

# Run the interactive wizard using a temporary CLI.
curl -fsSL https://johnhenning.dev/tools/outfit/run.sh | sh

# Start from a known profile; no prior Outfit, Python, Go or Git installation.
curl -fsSL https://johnhenning.dev/tools/outfit/profiles/compute.sh | sh
```

Windows equivalents end in `.ps1`:

```powershell
irm https://johnhenning.dev/tools/outfit/install.ps1 | iex
irm https://johnhenning.dev/tools/outfit/profiles/workstation.ps1 | iex
```

Known-profile scripts pin the CLI and config bundle independently, verify downloads, and open a review before applying. Unix unattended invocation is `curl .../compute.sh | sh -s -- --yes`. Inspect the script and profile before using unattended mode. Download first and run the local file if you prefer reviewing it before execution. Checksums alongside release assets provide integrity checks; they do not independently establish publisher authenticity.

## Development and local use

Build with Go 1.26 or newer:

```sh
go build -o bin/outfit ./cmd/outfit
bin/outfit plan --source ../outfit-profiles --profile compute --target debian-amd64
bin/outfit setup --source ../outfit-profiles
bin/outfit dotfiles --source ../outfit-profiles --profile dev
```

`plan` never installs software or writes state unless you explicitly pass `--output`. Simulated targets work only for previews. `setup` offers a TUI with profile selection, Advanced software categories, preferred alternatives, role overrides and additional presets. Press `p` to mark an app as preferred. `--plain` uses text review/confirmation; `--yes` approves unattended application.

```sh
outfit plan --source ./profiles --profile dev --output plan.json
outfit apply --source ./profiles --plan plan.json --yes
outfit export --source ./profiles --profile dev --output my-machine.json
outfit setup --source ./profiles --config my-machine.json
outfit update --source ./profiles
outfit doctor
```

Reusing a plan requires identical config-bundle contents, selection and target. Package detection is rerun before each install. Results and nonsecret selections live in the user configuration directory under `outfit/`; failed installs and manual steps are reported separately. `update` reconciles selections and does not upgrade the entire system.

## Personal configuration and credentials

Start from a public template. Once Git and GitHub CLI are installed:

```sh
outfit personalize --repo johnhenning/outfit-personal
# The command opens GitHub sign-in if needed and prints the private local directory.
outfit setup --source ./public-profiles --personal /printed/private/directory --profile john-dev
```

GitHub authentication protects private profile data. Secrets remain in 1Password; private repositories should contain references, never private keys or token values. `outfit doctor` explains desktop CLI/SSH-agent setup and the separate headless path. Authorize vault access in 1Password when needed.

```sh
outfit secrets run --env-file refs.env -- your-command
```

`refs.env` may contain only `NAME=op://vault/item/field` entries. Credentials are injected into the child process by `op run`.

## Dotfiles and platform boundaries

Dotfile manifests explicitly allowlist source/target pairs in a chezmoi-compatible static source layout. This first version applies static files itself so remote bundles cannot run chezmoi hooks, external downloads, scripts or secret template functions. Review with `outfit dotfiles`; apply separately with `--apply --yes`, and add `--replace` only after reviewing nonprivate conflicts. Originals are backed up with private permissions. Existing private files require manual merging and are never printed or backed up by the tool.

Windows and WSL have separate homes and package managers. Run native PowerShell setup for desktop apps; run the Unix profile bootstrap inside each chosen WSL distro for Linux tooling. WSL enablement/restarts, GPU drivers and media-server service/storage settings require explicit manual configuration in this alpha. No filesystem formatting, service exposure, or private-key migration is automatic.

## Release and verification

`python3 scripts/release.py --version v0.1.0-alpha.1 --profiles ../outfit-profiles` builds six OS/architecture binaries, a deterministic public profile archive, checksums, and self-contained install/run/profile scripts under `dist/site/tools/outfit/`. Config versions can differ from engine versions using `--profiles-version`.

```sh
go test ./...
go vet ./...
python3 -m unittest discover -s tests/bootstrap -v
```

Cross-compilation does not prove fresh Windows/WSL or Linux desktop installation. Consult the release notes for actual validation and remaining manual installers.
