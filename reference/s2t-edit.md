# s2t edit

Edit the values of a live Secret or ConfigMap in `$EDITOR`.

## Synopsis

```text
s2t edit <name> [flags]
```

## Description

`s2t edit` runs these steps:

1. Fetch the live resource with `kubectl`.
2. Decode the values and write only the data keys to a temporary file. Metadata, labels, and annotations are not shown.
3. Open the file in the editor.
4. Compare the edited keys with the original keys.
5. Print a JSON merge patch that holds only the changes.
6. Ask `Apply this patch? [y/N]:` on stderr.
7. Run `kubectl patch --type merge` if you answer `y` or `yes`.

If nothing changed, `s2t` prints `no changes` to stderr and exits.
Any other answer, an empty answer, or a closed stdin means No. The temporary file is always deleted.

### Patch shape

| Change | Secret patch | ConfigMap patch |
|---|---|---|
| Added or changed key | `stringData.<key>: <value>` | `data.<key>: <value>` |
| Removed key | `data.<key>: null` | `data.<key>: null` |

### Editor selection

`s2t` uses `$EDITOR`, then `$VISUAL`, then `vi`.
The value may hold arguments, for example `code --wait`.

### Limits

- `s2t edit` refuses a resource with a value that is not valid UTF-8.
- The `env` format refuses a value that contains a newline. Use `-o yaml` or `-o json`.
- A key may contain only letters, digits, `-`, `_`, and `.`.

## Arguments

| Argument | Description |
|---|---|
| `name` | The live resource. Accepts `name`, `secret/name`, `configmap/name`, or `cm/name`. |

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-n`, `--namespace` | string | current context | Namespace of the resource. |
| `--kubeconfig` | string | `$KUBECONFIG`, then `~/.kube/config` | Path to a kubeconfig file. |
| `-o`, `--output` | string | `env` | Format of the file in the editor: `env`, `yaml`, or `json`. |
| `-h`, `--help` | bool | `false` | Print the help text. |

### Inherited flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-k`, `--kind` | string | `secret` | Kind of the resource, unless the name has a `kind/` prefix. |
| `-t`, `--format` | string | `any` | Ignored. |

## Examples

### Edit a Secret

```bash
s2t edit db-creds -n prod
```

The editor opens with one `KEY=value` line per key:

```text
deprecated-key=old
password=s3cret
username=admin
```

Change `password` and remove `deprecated-key`, then save and close the editor:

```bash
s2t edit db-creds -n prod
```

```text
{
  "data": {
    "deprecated-key": null
  },
  "stringData": {
    "password": "NEWPASS"
  }
}
Apply this patch? [y/N]: y
secret/db-creds patched
```

### Edit multi-line values (`-o yaml`)

```bash
s2t edit db-creds -n prod -o yaml
```

```yaml
tls.crt: |
  -----BEGIN CERTIFICATE-----
  MIIC...
  -----END CERTIFICATE-----
username: admin
```

### Edit as JSON (`-o json`)

```bash
s2t edit db-creds -n prod -o json
```

```json
{
  "password": "s3cret",
  "username": "admin"
}
```

### Edit a ConfigMap (`cm/name`, `-k`)

```bash
s2t edit cm/app-config -n prod
s2t edit app-config -k configmap -n prod
```

### Use a different editor (`EDITOR`)

```bash
EDITOR=nano s2t edit db-creds -n prod
EDITOR="code --wait" s2t edit db-creds -n prod
```

### Use a different kubeconfig (`--kubeconfig`)

```bash
s2t edit db-creds -n prod --kubeconfig ~/.kube/prod-cluster.yaml
```

### Preview a patch without applying it

Answer `n`, or press Enter, at the prompt. `s2t` prints the patch and writes nothing.

### Edit without a terminal

Set `EDITOR` to a script that changes the file, and pipe the answer to the prompt:

```bash
cat > /tmp/drop-key.sh <<'EOF'
#!/bin/sh
sed -i '/^deprecated-key=/d' "$1"
EOF
chmod +x /tmp/drop-key.sh
echo y | EDITOR=/tmp/drop-key.sh s2t edit db-creds -n prod
```

## Errors

| Message | Cause |
|---|---|
| `key "X" contains a newline; re-run with -o yaml or -o json` | `env` format cannot hold the value. |
| `key "X" contains non-UTF8 binary data ...` | The resource holds binary data. |
| `invalid -o "X" for edit (want env, yaml, or json)` | `-o` has an unsupported value. |
| `running editor "X": ...` | The editor did not start or exited with an error. |
| `parsing edited values: ...` | The edited file is not valid for the chosen format. |

## See also

- [`s2t set`](s2t-set.md)
- [`s2t`](index.md)
