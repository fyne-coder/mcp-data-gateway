# Publishing

## Private execution artifacts

Agent prompts, review logs, verification bundles, and the repo-local `executor_tasks.md` task ledger are private by default. Keep them outside public Git unless you intentionally publish them.

`.gitignore` excludes `artifacts/` and `executor_tasks.md`. Archive copies to a private store before removing them from a public tree. Do not stage these paths for a public push.

## First GitHub Push

Before pushing:

```bash
git status --short
make ci
git diff --check
git check-ignore -v artifacts/ executor_tasks.md
```

Confirm `.gitignore` excludes:

- `artifacts/` and `executor_tasks.md`.
- `.env` and local secret files.
- Build outputs in `bin/` and `dist/`.
- Temporary files in `tmp/`.
- Coverage outputs.
- Local OS files such as `.DS_Store`.

## Pre-Alpha Source Release Bar

The `0.1.x` line may be published as source-only pre-alpha releases when:

- CI is green.
- Negative auth tests exist.
- Threat model is current.
- Audit schema is stable enough for early adopters.
- README claims match implemented behavior.

Do not describe these tags as production-ready.

## Production Release Bar

Do not cut a production release until:

- CI is green.
- Negative auth tests exist.
- Threat model is current.
- Audit schema is stable enough for early adopters.
- Container image has SBOM and signature.
- README claims match implemented behavior.

## GitHub Metadata

Recommended repository description:

> Self-hosted MCP gateway for governed access to internal tools from hosted AI assistants.

Recommended topics:

- `mcp`
- `model-context-protocol`
- `gateway`
- `oidc`
- `policy`
- `audit`
- `self-hosted`
