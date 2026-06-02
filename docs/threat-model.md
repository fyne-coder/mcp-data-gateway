# Threat Model

## Assets

- OIDC tokens and group claims.
- Gateway configuration.
- Upstream bearer tokens.
- Tool arguments and results.
- Audit logs.
- Private connector credentials.
- Internal data exposed through connectors.

## Attacker Capabilities

- Stolen hosted-client token.
- Forged request headers.
- Confused-deputy tool invocation.
- Overbroad connector query.
- Compromised tunnel token.
- Malicious or careless tool result content.
- Supply-chain compromise of the gateway image.

## Required Mitigations

- Validate tokens using issuer, audience, expiry, signature, and required claims.
- Ignore client-supplied identity headers unless explicitly trusted behind a verified proxy.
- Enforce group-to-tool-pack policy before tool invocation.
- Add connector-level allowlists.
- Run result shaping before returning output.
- Redact or hash sensitive audit fields.
- Rotate upstream bearer credentials.
- Publish SBOMs and signed artifacts for releases.

## Explicit Non-Goals

- Preventing prompt injection inside arbitrary tool results.
- Managing IdP user lifecycle.
- Becoming a general-purpose network proxy.
- Claiming row or column-level governance before connector allowlists and result filters exist.
