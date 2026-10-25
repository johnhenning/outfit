# Security

Outfit can install packages and replace allowlisted configuration files. Review action plans and profile sources before applying them. Public profile bundles must never contain secrets; private repositories should hold only secret references.

For a suspected vulnerability, use GitHub's private vulnerability reporting on the outfit repository. Do not post credentials, private profile contents or exploitable secret-disclosure details in a public issue. If private reporting is unavailable, open an issue requesting a private contact without disclosing the vulnerability details.

The current alpha is the supported development line. Fixes are delivered through new release artifacts and independently versioned profile bundles. SHA-256 validation detects corruption or substitution relative to a trusted bootstrap; checksums downloaded from the same publisher are not independent proof of publisher identity.
