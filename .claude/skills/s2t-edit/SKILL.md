---
name: s2t-edit
description: Edit or delete keys of a live Kubernetes Secret or ConfigMap non-interactively with s2t edit, using a scripted $EDITOR. Use when keys must be removed or several values changed in one patch.
---

# s2t edit

`s2t edit` opens the data keys in `$EDITOR`, prints a merge patch, then asks `Apply this patch? [y/N]`. Script both prompts.

1. Write an editor script that changes the file passed as `$1`. Deleting a line deletes the key.

```bash
cat > /tmp/ed.sh <<'SH'
#!/bin/sh
grep -v '^OLD_KEY=' "$1" > "$1.new" && mv "$1.new" "$1"
SH
chmod +x /tmp/ed.sh
```

2. Preview. Empty stdin answers No, so nothing is written.

```bash
EDITOR=/tmp/ed.sh s2t edit <name> -n <ns> </dev/null
```

3. Apply only after the user confirms the target.

```bash
EDITOR=/tmp/ed.sh s2t edit <name> -n <ns> <<< y
```

Rules:
- Use `-o yaml` or `-o json` for multi-line values. `env` refuses newlines.
- ConfigMap: `-k configmap` or `cm/<name>`.
- Never print the edited file or the patch values to the user.
- For adding or changing keys only, use `s2t-set`. It supports `--dry-run`.
