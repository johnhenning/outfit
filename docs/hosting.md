# Hosting johnhenning.dev/tools/outfit/

The bootstrap site is generated from a release lock and deployed by GitHub Pages from a published Outfit release. The public Pages origin is `https://johnhenning.github.io/outfit/`.

A small Cloudflare Worker routes only `/tools/outfit/` requests on johnhenning.dev to the matching Pages path. It rejects non-GET/HEAD requests and strips credentials/cookies before fetching public assets. Its route can be extended later for other tools or replaced when the main personal website is deployed. Root content is not provided by this tool.

`deploy/wrangler.jsonc` declares the Worker and custom domain. Initial deployment uses an authorized Cloudflare session; routine installer updates are handled by GitHub Pages and need no Cloudflare CI token. Check existing DNS/routing before redeploying, and preserve unrelated mail/verification records.

The exact named endpoints are install.sh/install.ps1, run.sh/run.ps1, profiles/{workstation,dev,compute,media}.{sh,ps1}, plus v1 aliases. Install persists only the CLI. Run/profile endpoints use a temporary CLI, verify checksums, and restore terminal input for review.
