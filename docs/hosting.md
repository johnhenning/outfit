# Hosting general and personal Outfit endpoints

The bootstrap site is generated from a release lock and deployed by GitHub Pages from a published Outfit release. The deployment overlays the current landing page from `site/tools/outfit/index.html`, so documentation updates can be deployed against an existing release without rebuilding its installers. The public Pages origin is `https://johnhenning.github.io/outfit/`.

A Cloudflare Worker serves the general site at `https://getoutfit.dev/` from the
Pages `/tools/outfit/` directory. The path `https://johnhenning.dev/tools/outfit/`
serves the personal site from the Pages `/personal/` directory. Requests elsewhere
on johnhenning.dev are outside its route. Both sites reject non-GET/HEAD requests
and fetch public assets without forwarding credentials or cookies.

`deploy/wrangler.jsonc` declares a Worker custom domain for the dedicated
getoutfit.dev apex, and only the `/tools/outfit` namespace on johnhenning.dev.
Cloudflare provisions DNS/certificates for the custom domain; the path route
still uses johnhenning.dev's existing proxied apex record. Preserve unrelated
web, mail and verification records. Initial deployment uses the authorized
Cloudflare session; Pages updates use GitHub's native token, with no personal
Cloudflare token in CI.

General endpoints are `/install.sh`, `/run.sh`, `/profiles/dev.sh`, etc. Personal
Mac endpoints are `/tools/outfit/profiles/mac/everyday.sh` and
`/tools/outfit/profiles/mac/dev.sh` on johnhenning.dev. Personal `install.sh` and
`run.sh` both start Everyday setup; general `install.sh` installs only the CLI.
The general site's old `/tools/outfit/*` URLs redirect to root-relative endpoints.

Personal bundles contain public nonsecret selections and allowlisted configs,
merged with the validated public catalog at release time. The standalone bundle
does not require GitHub authentication. App sign-in, 1Password authorization,
private SSH hosts and synced account data are separate follow-up steps. Never
proxy an authenticated request or publish a token/private config in the site.

The site workflow downloads the checksum-verified engine release and personal
bundle, then generates personal scripts with the exact engine checksums and
personal archive SHA-256. Publish the personal bundle before deploying the site,
and verify Pages assets before switching the Worker routes. Existing published
release assets remain immutable; new general defaults require a new coordinated
profile/engine release.

The exact named endpoints are install.sh/install.ps1, run.sh/run.ps1, profiles/{workstation,dev,compute,media}.{sh,ps1}, plus v1 aliases. Install persists only the CLI. Run/profile endpoints use a temporary CLI, verify checksums, and restore terminal input for review.
