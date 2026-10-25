# Profile bundle contract (schema 1)

Outfit accepts a local directory with `outfit.yaml`, or an HTTPS gzip-compressed tar archive with that file at its root. Remote archives require an explicit SHA-256 digest. Archives cannot contain symlinks, absolute/traversal paths, duplicate paths, special files, excessive entries or more than 64 MiB of expanded content. The engine rejects unknown YAML fields and unsupported schema versions.

A bundle contains these data sections:

- `packages`: unique IDs, display name/category, desktop requirement, dependencies, official source URL and per-platform install mappings.
- `categories`: UI labels and package choices.
- `roles`: desktop/headless defaults, baseline packages and dotfile IDs.
- `presets`: additional package and dotfile groups.
- `profiles`: named combinations of role, presets, includes/excludes, preferred apps and dotfile sets.
- `dotfiles`: explicit IDs, relative static source/target paths, platform limits and private-content flags.

Platform keys are `macos`, `debian`, `fedora`, and `windows`. Architectures are `amd64` and `arm64`; per-package architecture constraints are optional. Providers are `brew`, `apt`, `dnf`, `winget`, `uv`, `vendor`, and `manual`. Vendor IDs are engine-owned recipes, currently uv and rust; a manifest cannot supply an executable URL or command. Native provider mappings must match their target platform.

```yaml
schema: 1
name: Example templates
packages:
  - id: git
    name: Git
    category: cli
    source: https://git-scm.com/
    installs:
      debian: {provider: apt, package: git}
roles:
  compute: {desktop: false, packages: [git]}
presets:
  default: {packages: []}
profiles:
  compute: {role: compute, presets: [default]}
```

An overlay is another schema-1 bundle. It may add uniquely named packages, roles, presets, profiles and file mappings. Duplicate definitions or dotfile targets fail explicitly. Personal profiles can reference the public bundle's definitions after merging.

```yaml
schema: 1
name: Private personal layer
profiles:
  my-compute: {role: compute, presets: [default]}
```

Resolution is deterministic: role plus profile presets plus extra presets, then explicit includes/preferences, then explicit exclusions. Excluding a required dependency is an error. Desktop-only defaults are omitted with a note for headless roles; explicit incompatible choices fail. Preferred apps replace inherited choices within their category, while explicit includes can retain additional apps.

Plans record target, selection, full bundle digest and resolved actions. Apply regenerates the plan and rejects mismatches before executing. Detection is repeated at execution so reruns do not rely solely on previous journals.

Dotfile sources use chezmoi-compatible static layout but no hooks, scripts, external downloads or secret functions execute. Canonical target paths are validated; private SSH key and credential destinations cannot be managed through the manifest. Existing private targets require manual merging. `.zshrc.local`, tmux local includes and SSH config.d keep machine overrides local.

Engine releases and profile releases are independent. An engine release pins a published public archive and its SHA-256; generated direct-profile URLs pin both. Changes to the schema require a version bump and explicit compatibility handling in the engine.
