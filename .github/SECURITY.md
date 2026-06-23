# Security Policy

## Supported Versions

Security fixes are applied to the two most recent releases.

| Version | Supported |
|---------|-----------|
| Latest release | yes |
| Previous release | yes |
| Older releases | no |

## Reporting a Vulnerability

Please use GitHub's private vulnerability reporting to report security issues.
Navigate to the **Security** tab of this repository and select **Report a vulnerability**.
Do not open a public issue for security vulnerabilities.

Include as much of the following as possible:

- A description of the vulnerability and its potential impact
- The affected version(s)
- Steps to reproduce or a proof-of-concept
- Any suggested mitigation or fix

## Response Timeline

| Stage | Target |
|-------|--------|
| Acknowledgement | Within 7 days |
| Assessment and triage | Within 14 days |
| Fix and release | Within 90 days of confirmation |

We will keep you informed of progress throughout. If a vulnerability requires more than 90 days to resolve we will communicate this with an updated timeline.

## Scope

The following are in scope:

- The `goss` binary and its releases
- The published Docker image (`ghcr.io/uk-bolly/goss`)
- Integration test infrastructure where a vulnerability could affect users

The following are out of scope:

- Vulnerabilities in upstream [goss-org/goss](https://github.com/goss-org/goss) that have not been introduced by this fork -- please report those upstream
- Third-party dependencies where a fix requires an upstream release (we will update the dependency once a fix is available)
- Theoretical vulnerabilities without a realistic attack vector

## Disclosure

We follow coordinated disclosure. Once a fix is available we will:

1. Release a patched version
2. Publish a security advisory on GitHub
3. Credit the reporter in the advisory unless they prefer to remain anonymous
