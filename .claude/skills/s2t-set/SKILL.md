---
name: s2t-set
description: Upsert plaintext keys into a live Kubernetes Secret or ConfigMap with s2t set, including appsettings.json files. Use when asked to update, add, rotate, or change a secret or configmap value.
---

# s2t set

Run `s2t set <name> [key=value ...] -n <ns>`. It prints a merge patch and writes nothing.

1. Build the patch without `--apply`. Check keys and namespace.
2. Run again with `--apply --dry-run`.
3. Run with `--apply` only after the user confirms the target namespace.

Rules:
- ConfigMap: `-k configmap` or `cm/<name>`.
- File input: `-f file` or stdin (JSON, YAML, env). Nested keys flatten with `__`.
- A command-line pair overrides the same key from a file.
- Commas separate pairs. For a value with a comma, use JSON or YAML input.
- Never echo secret values back to the user.
- Do not use `s2t edit`. It needs an interactive `$EDITOR`.
