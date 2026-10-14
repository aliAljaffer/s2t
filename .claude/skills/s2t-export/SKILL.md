---
name: s2t-export
description: Export or reshape a Kubernetes Secret or ConfigMap with s2t as a .env file, a kubectl patch payload, a single value, or a masked view. Use for .env export, patch payloads, or sharing output safely.
---

# s2t export

Source: `s2t <name> -n <ns>` (live), `-f file`, or stdin. Add `-k configmap` for ConfigMaps.

| Goal | Command |
|---|---|
| .env file | `s2t <name> -n <ns> -o env > .env` |
| Patch payload | `s2t -f secret.yaml -o jsonc` |
| Apply payload | `kubectl patch secret <name> -n <ns> --type merge -p "$(s2t -f secret.yaml -o jsonc)"` |
| Selected keys | `--only key1,key2` |
| Safe to share | `--mask` |
| Sealed Secret | `S2T_SEALED_SECRETS_KEY_FILE=<pem> s2t -f sealed.yaml -t sealed-secret` |

Rules:
- `--mask` cannot combine with `-o json|jsonc|yaml`. Masked values would overwrite the real secret.
- Write `.env` files to disk. Do not print them. Check `.gitignore` covers the file.
- Confirm before running `kubectl patch`.
