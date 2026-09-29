# Security Policy

## Supported versions

Security fixes are applied to the **latest tagged release** on the default branch (1.0.0 and newer). Older tags are not maintained separately.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security problems.

Prefer a **private vulnerability report** on the GitHub repository
([Sopandgo/sopandgo](https://github.com/Sopandgo/sopandgo)): **Security** → **Report a vulnerability**
(enable private vulnerability reporting in the repo settings if it is not already on).

Include:

- A short description of the issue and impact
- Steps to reproduce or a proof of concept if you have one
- Affected version / commit if known

You should receive an acknowledgment within a reasonable time. Please give us time to investigate and ship a fix before any public disclosure.

## Trust model (summary)

sopandgo is a **self-hosted** tool. Operators are responsible for host hardening, network exposure, TLS termination, and backups. The application focuses on authentication, RBAC, session revocation, and tamper-evident audit trails — not enterprise SSO or QMS validation.

See [`docs/concepts/security-model.md`](docs/concepts/security-model.md) for assumptions and limitations.
