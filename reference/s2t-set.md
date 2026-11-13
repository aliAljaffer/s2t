# s2t set

Upsert plaintext keys into a live Secret or ConfigMap.

## Synopsis

```text
s2t set <name> [key=value ...] [flags]
s2t upsert <name> [key=value ...] [flags]
```

## Description

`s2t set` builds a JSON merge patch from plaintext values.
By default it prints the patch to stdout. With `--apply`, it runs `kubectl patch --type merge`.

You never base64-encode a value. A Secret patch uses `stringData`, which the API server encodes.
A ConfigMap patch uses `data`.

`s2t set` only adds and updates keys. It never removes a key. Use [`s2t edit`](s2t-edit.md) to remove keys.

### Input sources

`s2t set` merges values from these sources, in this order:

1. The file from `-f`. If you do not give `-f` and stdin is not a terminal, `s2t` reads stdin.
2. The `key=value` arguments. These override keys from step 1, with a warning on stderr.

> [!IMPORTANT]
> In a script or a CI job, stdin is often not a terminal. If you pass only arguments, redirect stdin from `/dev/null`. If you do not, `s2t set` waits for input on stdin.

### Nested keys

`s2t set` flattens nested JSON and YAML into flat keys:

- A map key is joined to its parent with the separator, `__` by default.
- An array item uses its index as the key, for example `AllowedHosts__0`.
- A `null` value becomes an empty string.

The `__` separator matches the .NET configuration convention, so an `appsettings.json` file maps directly to environment variable keys.

### Key rules

A key may contain only letters, digits, `-`, `_`, and `.`.

## Arguments

| Argument | Description |
|---|---|
| `name` | The live resource. Accepts `name`, `secret/name`, `configmap/name`, or `cm/name`. |
| `key=value` | A plaintext pair. One argument may hold several pairs separated by commas. |

`s2t` splits each pair at the first `=`, so a value may contain `=`.
A value in an argument cannot contain a comma. Put such values in a file.

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-f`, `--file` | string | stdin | Path to a plaintext JSON, YAML, or env file. |
| `-n`, `--namespace` | string | current context | Namespace of the resource. |
| `--kubeconfig` | string | `$KUBECONFIG`, then `~/.kube/config` | Path to a kubeconfig file. |
| `-o`, `--output` | string | `jsonc` | Patch format: `jsonc`, `json`, or `yaml`. |
| `--apply` | bool | `false` | Apply the patch with `kubectl patch`. |
| `--dry-run` | bool | `false` | With `--apply`, add `--dry-run=server`. Nothing is written. |
| `--separator` | string | `__` | Separator for flattened nested keys. |
| `-h`, `--help` | bool | `false` | Print the help text. |

### Inherited flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-t`, `--format` | string | `any` | Format of `-f` or stdin: `any`, `json`, `yaml`, `env`, or `kv` (same as `env`). Arguments are always `env`. |
| `-k`, `--kind` | string | `secret` | Kind of the resource, unless the name has a `kind/` prefix. |

## Examples

### Print a patch for one key

```bash
s2t set db-creds -n prod password=s3cret </dev/null
```

```text
{"stringData":{"password":"s3cret"}}
```

### Set several keys

Use separate arguments, or commas in one argument:

```bash
s2t set db-creds -n prod username=admin password=s3cret --apply
s2t set db-creds -n prod username=admin,password=s3cret --apply
```

### Use a value that contains `=`

```bash
s2t set db-creds 'url=postgres://u:p@h/db?a=1' </dev/null
```

```text
{"stringData":{"url":"postgres://u:p@h/db?a=1"}}
```

### Apply the patch (`--apply`)

```bash
s2t set db-creds -n prod password=s3cret --apply
```

`s2t` writes `applying merge patch to secret/db-creds in namespace prod` to stderr, then prints the `kubectl` output.

### Validate before you write (`--dry-run`)

```bash
s2t set db-creds -n prod password=s3cret --apply --dry-run
```

The API server checks RBAC, the schema, and admission webhooks. Nothing is stored.
`--dry-run` has no effect without `--apply`.

### Choose the patch format (`-o`, `--output`)

```bash
s2t set db-creds username=admin password=s3cret -o json </dev/null
```

```text
{
  "stringData": {
    "password": "s3cret",
    "username": "admin"
  }
}
```

```bash
s2t set db-creds password=s3cret -o yaml </dev/null
```

```text
stringData:
    password: s3cret
```

Pass the printed patch to `kubectl` yourself:

```bash
kubectl patch secret db-creds -n prod --type merge -p "$(s2t set db-creds password=s3cret </dev/null)"
```

### Read values from an env file (`-f`)

```bash
cat values.env
```

```text
API_KEY=abc123
REGION=eu-west-1
```

```bash
s2t set db-creds -n prod -f values.env
```

```text
{"stringData":{"API_KEY":"abc123","REGION":"eu-west-1"}}
```

### Read values from stdin

```bash
s2t set db-creds -n prod --apply < values.env
```

### Override a file value with an argument

```bash
s2t set db-creds -f values.env REGION=us-east-1
```

```text
warning: command-line REGION overrides value from file/stdin
{"stringData":{"API_KEY":"abc123","REGION":"us-east-1"}}
```

### Load a .NET `appsettings.json` file

```json
{
  "ConnectionStrings": { "Default": "Server=db;Database=app" },
  "Logging": { "LogLevel": { "Default": "Warning" } },
  "AllowedHosts": ["a.example.com", "b.example.com"]
}
```

```bash
s2t set db-creds -f appsettings.json -o json
```

```text
{
  "stringData": {
    "AllowedHosts__0": "a.example.com",
    "AllowedHosts__1": "b.example.com",
    "ConnectionStrings__Default": "Server=db;Database=app",
    "Logging__LogLevel__Default": "Warning"
  }
}
```

Add an override on top of the file:

```bash
cat appsettings.json | s2t set db-creds -n prod Feature__Enabled=true --apply
```

### Change the separator (`--separator`)

```bash
s2t set db-creds -f appsettings.json --separator . -o json
```

```text
{
  "stringData": {
    "AllowedHosts.0": "a.example.com",
    "AllowedHosts.1": "b.example.com",
    "ConnectionStrings.Default": "Server=db;Database=app",
    "Logging.LogLevel.Default": "Warning"
  }
}
```

The separator must give valid keys. For example, `--separator :` fails because `:` is not allowed in a key.

### Force the input format (`-t`, `--format`)

```bash
s2t set db-creds -n prod -f settings.yaml -t yaml --apply
s2t set db-creds -n prod -f values.txt -t env --apply
```

### Update a ConfigMap (`cm/name`, `-k`)

```bash
s2t set cm/app-config feature.flag=on </dev/null
```

```text
{"data":{"feature.flag":"on"}}
```

```bash
s2t set app-config -k configmap -n prod LOG_LEVEL=debug --apply
```

### Use a different kubeconfig (`--kubeconfig`)

```bash
s2t set db-creds -n prod password=s3cret --apply --kubeconfig ~/.kube/prod-cluster.yaml
```

### Use the `upsert` alias

```bash
s2t upsert db-creds -n prod password=s3cret --apply
```

## Errors

| Message | Cause |
|---|---|
| `no keys provided ...` | No arguments, no `-f`, and stdin was empty. |
| `invalid key "X": keys may only contain ...` | A key has a character that is not allowed. |
| `invalid --output "X" (want jsonc, json, or yaml)` | `-o` has an unsupported value, such as `env`. |
| `unknown plaintext format "X" ...` | `-t` has a value that `set` does not accept, such as `sealed-secret`. |
| `duplicate key "X"` | One env input has the same key twice. |
| `malformed pair "X" (want key=value)` | An env pair has no `=`. |

## See also

- [`s2t edit`](s2t-edit.md)
- [`s2t`](index.md)
