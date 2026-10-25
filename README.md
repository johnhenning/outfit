# Outfit

Equip a workstation, development environment, compute node, or media node from reusable profiles.

Outfit is the Go CLI/TUI and bootstrap runtime. Public software catalogs and dotfile templates live in [outfit-profiles](https://github.com/johnhenning/outfit-profiles); personal profiles are loaded separately and can be public nonsecret bundles or authenticated private overlays. The binary contains provider implementations and a schema, not personal software defaults.

## Entry points

Browse the public profiles at [getoutfit.dev/profiles/](https://getoutfit.dev/profiles/).
Choose a platform to inspect the exact included software, manual steps,
configuration files and optional presets, then copy a preview or setup command.
The page is generated with the checksum-verified released CLI and public bundle
from the release lock, so it follows the installer version rather than
the latest unpublished catalog edits. Advanced per-app choices remain in the CLI.

General Outfit endpoints (deployment status is recorded in the release notes):

```sh
# Install only the Outfit CLI.
curl -fsSL https://getoutfit.dev/install.sh | sh

# Run the interactive wizard using a temporary CLI.
curl -fsSL https://getoutfit.dev/run.sh | sh

# Preview a known profile without changing your machine.
curl -fsSL https://getoutfit.dev/profiles/workstation.sh | sh -s -- --dry-run

# Start from a known profile; no prior Outfit, Python, Go or Git installation.
curl -fsSL https://getoutfit.dev/profiles/compute.sh | sh
```

Windows equivalents end in `.ps1`:

```powershell
irm https://getoutfit.dev/install.ps1 | iex
irm https://getoutfit.dev/profiles/workstation.ps1 | iex
```

The CLI installer reports download, verification and installation progress, then
shows the installed version/path and the next commands. If Outfit is not already
reachable on PATH, it prints copyable instructions for zsh, bash, fish or
PowerShell, including how to keep the setting for future terminals. It does not
edit shell startup files or your Windows user PATH automatically. Color is used
only for interactive success output and respects `NO_COLOR`; progress goes to
stderr so profile previews can still emit clean JSON.

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

Public, nonsecret selections can be packaged with the public templates as a
standalone bundle. The personal Mac entry points use such a bundle and need no
GitHub authentication or preinstalled Outfit CLI:

```sh
curl -fsSL https://johnhenning.dev/tools/outfit/profiles/mac/everyday.sh | sh -s -- --dry-run
curl -fsSL https://johnhenning.dev/tools/outfit/profiles/mac/dev.sh | sh -s -- --dry-run
```

Those are John's personal selections; use getoutfit.dev for general templates.
Sign in to 1Password and apps afterward to access private credentials and synced
data. Publishing a profile never publishes or restores those private values.

Private overlays remain supported. Once Git and GitHub CLI are installed, use
`outfit personalize --repo OWNER/PRIVATE_REPO` to authenticate and fetch one,
then pass its printed path with `--personal` to setup or plan. Keep private keys
and tokens out of both public and private profile repos. `outfit doctor` explains
credential setup; authorize vault access separately when needed.

```sh
outfit secrets run --env-file refs.env -- your-command
```

`refs.env` may contain only `NAME=op://vault/item/field` entries. Credentials are injected into the child process by `op run`.

## Dotfiles and platform boundaries

Dotfile manifests explicitly allowlist source/target pairs in a chezmoi-compatible static source layout. This first version applies static files itself so remote bundles cannot run chezmoi hooks, external downloads, scripts or secret template functions. Review with `outfit dotfiles`; apply separately with `--apply --yes`, and add `--replace` only after reviewing nonprivate conflicts. Originals are backed up with private permissions. Existing private files require manual merging and are never printed or backed up by the tool.

Windows and WSL have separate homes and package managers. Run native PowerShell setup for desktop apps; run the Unix profile bootstrap inside each chosen WSL distro for Linux tooling. WSL enablement/restarts, GPU drivers and media-server service/storage settings require explicit manual configuration in this alpha. No filesystem formatting, service exposure, or private-key migration is automatic.

## Release and verification

`python3 scripts/release.py --version v0.1.0-alpha.3 --profiles ../outfit-profiles --profiles-version v0.1.0-alpha.2` builds six OS/architecture binaries, a deterministic public profile archive, checksums, and self-contained install/run/profile scripts under `dist/site/tools/outfit/`. Config versions can differ from engine versions using `--profiles-version`.

```sh
go test ./...
go vet ./...
python3 -m unittest discover -s tests/bootstrap -v
```

Cross-compilation does not prove fresh Windows/WSL or Linux desktop installation. Consult the release notes for actual validation and remaining manual installers.
