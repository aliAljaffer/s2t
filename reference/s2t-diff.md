# s2t diff

Compare the decoded contents of two Secrets or ConfigMaps key by key.

## Synopsis

```text
s2t diff <fileA|nameA> <fileB|nameB> [flags]
```

## Description

`s2t diff` decodes two sources and prints one line for each key that differs.
It compares decoded values, not raw manifest lines.

Each argument is a file or a live resource:

1. If the argument is the path of an existing regular file, `s2t` reads the file.
2. If not, `s2t` fetches the argument as a live resource with `kubectl`.

You can compare two files, two live resources, or one of each.

Output lines use these prefixes, sorted by key:

| Prefix | Color | Meaning |
|---|---|---|
| `+` | green | The key is only in the second source. |
| `-` | red | The key is only in the first source. |
| `~` | yellow | The key is in both sources with different values. |

Keys with equal values are not printed. If the sources are equal, the output is empty.

> [!NOTE]
> `s2t diff` exits with status `0` when the sources differ. Check for empty output to detect a difference in a script.

## Arguments

| Argument | Description |
|---|---|
| `fileA`, `nameA` | The first source. A file path, or a live resource name. |
| `fileB`, `nameB` | The second source. A file path, or a live resource name. |

A live resource name accepts `name`, `secret/name`, `configmap/name`, or `cm/name`.
A `kind/` prefix sets the kind for that argument only.

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--show-values` | bool | `false` | Print values next to each key. |
| `-n`, `--namespace` | string | current context | Namespace for each live argument. |
| `-A`, `--namespace-a` | string | none | Namespace for the first argument. Overrides `-n`. |
| `-B`, `--namespace-b` | string | none | Namespace for the second argument. Overrides `-n`. |
| `--kubeconfig` | string | `$KUBECONFIG`, then `~/.kube/config` | Path to a kubeconfig file. |
| `-h`, `--help` | bool | `false` | Print the help text. |

### Inherited flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-t`, `--format` | string | `any` | Input format of file arguments. Live arguments are always JSON. |
| `-k`, `--kind` | string | `secret` | Kind for both arguments, unless a `kind/` prefix overrides it. |

## Examples

The file examples use `staging.yaml` and `prod.yaml`:

```yaml
# staging.yaml
apiVersion: v1
kind: Secret
data:
  username: YWRtaW4=
  password: czNjcmV0
  host: ZGIucHJvZC5pbnRlcm5hbA==
```

```yaml
# prod.yaml
apiVersion: v1
kind: Secret
data:
  username: YWRtaW4=
  password: cHJvZC1wYXNz
  feature-flag: ZW5hYmxlZA==
```

### Compare two files

```bash
s2t diff staging.yaml prod.yaml
```

```text
+ feature-flag
- host
~ password
```

### Show the values (`--show-values`)

> [!WARNING]
> `--show-values` prints secret values. Do not paste this output into a ticket or a chat.

```bash
s2t diff --show-values staging.yaml prod.yaml
```

```text
+ feature-flag: enabled
- host: db.prod.internal
~ password: s3cret -> prod-pass
```

### Compare one live Secret in two namespaces (`-n`, `-B`)

```bash
s2t diff db-creds db-creds -n staging -B prod
```

### Set each namespace on its own (`-A`, `-B`)

```bash
s2t diff db-creds db-creds -A staging -B prod
```

### Compare two Secrets in one namespace (`-n`)

```bash
s2t diff old-db-creds db-creds -n prod
```

### Compare a file with a live Secret

```bash
s2t diff staging.yaml db-creds -n prod
```

### Compare live ConfigMaps (`kind/name`, `-k`)

```bash
s2t diff cm/app-config cm/app-config -n staging -B prod
s2t diff app-config app-config -k configmap -n staging -B prod
```

### Compare a ConfigMap file with a live ConfigMap

```bash
s2t diff -k configmap app-config.yaml app-config -n prod
```

### Use a different kubeconfig (`--kubeconfig`)

```bash
s2t diff db-creds db-creds -n staging -B prod --kubeconfig ~/.kube/shared.yaml
```

### Compare `kv` files (`-t kv`)

```bash
s2t diff -t kv old.kv new.kv
```

### Compare SealedSecrets (`-t sealed-secret`)

```bash
export S2T_SEALED_SECRETS_KEY_FILE=~/.config/s2t/sealed-secrets-key.pem
s2t diff -t sealed-secret sealed-staging.yaml sealed-prod.yaml
```

### Fail a CI job on drift

```bash
if [ -n "$(s2t diff expected.yaml db-creds -n prod)" ]; then
  echo "db-creds drifted from expected.yaml" >&2
  exit 1
fi
```

## See also

- [`s2t`](index.md)
