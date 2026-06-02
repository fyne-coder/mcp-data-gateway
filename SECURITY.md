# Security Policy

## Supported Versions

This project is pre-alpha. Security reports are still welcome, but no production support window exists yet.

## Reporting a Vulnerability

Please do not open a public issue for a suspected vulnerability. Email the maintainer or use GitHub private vulnerability reporting once the repository is published.

Useful reports include:

- Impacted version or commit.
- Reproduction steps.
- Expected and observed behavior.
- Whether the issue affects authentication, authorization, audit integrity, tunnel handling, connector isolation, or sensitive data exposure.

## Security Posture

The gateway is intended to sit in a high-trust path between hosted AI clients and private systems. The current repository is pre-alpha and should be evaluated as a proof path, not a production release. Production release work must include:

- OIDC validation with mature libraries.
- Negative authorization tests.
- Payload-safe audit logging.
- Result-shaping interfaces.
- Supply-chain attestations, SBOMs, and signed release artifacts.
