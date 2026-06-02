# Contributing

This project is early. Contributions should keep the MVP narrow and security-focused.

## Local Checks

```bash
make ci
```

## Contribution Guidelines

- Keep gateway, connector, policy, and audit boundaries separate.
- Add tests for new behavior.
- Add negative auth tests for authorization changes.
- Avoid broad connector abstractions until at least three in-tree connectors prove the shape.
- Do not add telemetry that can capture raw tool arguments or raw tool results by default.
