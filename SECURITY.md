# Security Policy

## Supported versions

Security fixes are applied to the latest release on the `main` branch. Older
deployments should upgrade to the latest tagged image.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Report it privately to
the maintainers through the organization's internal security channel, including:

- affected version or commit;
- a short description and impact;
- reproduction steps or a proof of concept;
- any suggested remediation.

We aim to acknowledge reports within 2 business days and to provide a status
update within 7 days. Please allow time for a fix before public disclosure.

## Scope

In scope: authentication and session handling, authorization bypass, injection,
unsafe file upload, secret leakage, CSP/header regressions, and dependency
vulnerabilities reachable at runtime.

Out of scope: reports requiring a compromised host or database, social
engineering, denial of service from an already authenticated administrator, and
issues in local-only development tooling.

## Hardening baseline

- Secrets come from environment variables; never commit real credentials.
- Production requires HTTPS `APP_URL`, secure cookies, CSP, and non-placeholder
  secrets.
- Run `just audit` and keep Renovate updates reviewed.
- Rotate `ACCESS_KEY`/`SECRET_KEY` and provider credentials after exposure.
