# s2t

Decode a Kubernetes Secret or ConfigMap into readable key/value pairs.

This reference describes every `s2t` command, argument, and flag. Each page gives the synopsis, the flags, the exit status, and examples.

## Synopsis

```text
s2t [flags]
s2t [kind/]name [flags]
s2t kind name [flags]
```

## Description

`s2t` reads a Secret or ConfigMap from one of three sources:

1. A live resource, fetched with `kubectl get -o json`, when you give a name.
2. A file, when you give `-f`.
3. Stdin, when you give neither.

The source order above is the precedence order. A name wins over `-f`, and `-f` wins over stdin.

`s2t` base64-decodes every value and prints the result in the format that `-o` selects.
If a value is not valid base64, `s2t` prints the raw value and writes a warning to stderr.
Output keys are sorted.

If you give no name, no `-f`, and stdin is a terminal, `s2t` prints the usage and exits with status `1`.

## Arguments

| Argument | Description |
|---|---|
| `name` | Name of a live resource. Accepts `name`, `secret/name`, `configmap/name`, or `cm/name`. |
| `kind name` | Kind and name as two arguments. `kind` is `secret`, `configmap`, or `cm`. |

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-f`, `--file` | string | stdin | Path to a manifest or `kv` file. |
| `-t`, `--format` | string | `any` | Input format: `any`, `json`, `yaml`, `kv`, or `sealed-secret`. Ignored for a live fetch. |
| `-k`, `--kind` | string | `secret` | Resource kind: `secret` or `configmap`. |
| `-n`, `--namespace` | string | current context | Namespace of the live resource. |
| `--kubeconfig` | string | `$KUBECONFIG`, then `~/.kube/config` | Path to a kubeconfig file. A leading `~` is expanded. |
| `--name` | string | none | Same as the `name` argument. |
| `--only` | string | all keys | Comma-separated list of keys to print. |
| `-o`, `--output` | string | plain | Output format: plain (empty), `env`, `json`, `jsonc`, or `yaml`. |
| `--mask` | bool | `false` | Replace every value with `********`. |
| `-v`, `--version` | bool | `false` | Print `s2t version <version>` and exit. |
| `-h`, `--help` | bool | `false` | Print the help text. |

## Output formats

| `-o` value | Shape | Use |
|---|---|---|
| (empty) | `key: value` | Read values in a terminal. |
| `env` | `KEY=value` | Write a `.env` file. |
| `json` | Indented merge patch | Review a patch before you apply it. |
| `jsonc` | Single-line merge patch | Pass to `kubectl patch -p`. |
| `yaml` | YAML merge patch | Pass to `kubectl patch --patch-file`. |

For a Secret, the patch formats wrap the values in `stringData`.
For a ConfigMap, they wrap the values in `data`.
The API server encodes `stringData`, so the values stay plaintext in the patch.

## Subcommands

| Command | Description |
|---|---|
| [`s2t diff`](s2t-diff.md) | Compare the decoded contents of two Secrets or ConfigMaps key by key. |
| [`s2t set`](s2t-set.md) | Upsert plaintext keys into a live Secret or ConfigMap. Alias: `s2t upsert`. |
| [`s2t edit`](s2t-edit.md) | Edit the values of a live Secret or ConfigMap in `$EDITOR`. |
| [`s2t upgrade`](s2t-upgrade.md) | Upgrade `s2t` to the latest GitHub release. |
| [`s2t version`](s2t-version.md) | Print the version, platform, and release asset name. |
| [`s2t completion`](s2t-completion.md) | Generate a shell completion script. |
| `s2t help [command]` | Print the help text for a command. Same as `s2t [command] --help`. |

## Global flags

All commands accept these flags. Some commands ignore them. Each command page states how it uses them.

| Flag | Type | Default | Description |
|---|---|---|---|
| `-t`, `--format` | string | `any` | Input format of a file or stdin. |
| `-k`, `--kind` | string | `secret` | Resource kind: `secret` or `configmap`. |
| `-h`, `--help` | bool | `false` | Print the help text for the command. |

### Input formats (`-t`, `--format`)

The accepted values depend on the command.

| Value | `s2t`, `s2t diff` (encoded input) | `s2t set` (plaintext input) |
|---|---|---|
| `any` | Try `json`, then `yaml`, then `kv`. | Try `json`, then `yaml`, then `env`. |
| `json` | A Kubernetes manifest in JSON. | A JSON object of plaintext values. |
| `yaml`, `yml` | A Kubernetes manifest in YAML. | A YAML mapping of plaintext values. |
| `kv` | `key: base64value` pairs, separated by commas or newlines. | Same as `env`. |
| `env` | Not accepted. | `KEY=value` pairs, separated by commas or newlines. |
| `sealed-secret` | A SealedSecret manifest. Needs `S2T_SEALED_SECRETS_KEY_FILE`. | Not accepted. |

With `--kind configmap`, `s2t` and `s2t diff` accept only `any`, `json`, and `yaml`.

### Resource names

Commands that fetch a live resource accept a name in one of these forms:

| Form | Example | Kind used |
|---|---|---|
| `name` | `db-creds` | The value of `--kind`. |
| `secret/name` | `secret/db-creds` | `secret` |
| `configmap/name` | `configmap/app-config` | `configmap` |
| `cm/name` | `cm/app-config` | `configmap` |

The root command also accepts the kind and the name as two arguments, for example `s2t cm app-config`.
If you set `--kind` and also give a different kind in the name, `s2t` stops with an error.

## Environment variables

| Variable | Used by | Effect |
|---|---|---|
| `KUBECONFIG` | All live fetches and patches | `kubectl` reads it when `--kubeconfig` is not set. Colon-separated paths are merged. |
| `S2T_SEALED_SECRETS_KEY_FILE` | `-t sealed-secret` | Path to the PEM private key of the sealed-secrets controller. |
| `EDITOR`, `VISUAL` | `s2t edit` | The editor to open. `EDITOR` wins. The fallback is `vi`. |
| `NO_COLOR` | `s2t`, `s2t diff` | Any non-empty value turns off colored output. |

Colored output also turns off when stdout is not a terminal or `TERM=dumb`.

## Requirements

`kubectl` must be on `PATH` for any command that reads or writes a live resource.
Decoding a file or stdin does not need `kubectl`.

## Exit status

All commands use the same exit status:

| Code | Meaning |
|---|---|
| `0` | The command completed. |
| `1` | The command failed. `s2t` writes `error: <message>` to stderr. |

## Examples

The examples use this manifest, saved as `secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-creds
type: Opaque
data:
  username: YWRtaW4=
  password: czNjcmV0
  host: ZGIucHJvZC5pbnRlcm5hbA==
```

### Decode a file (`-f`, `--file`)

```bash
s2t -f secret.yaml
```

```text
host: db.prod.internal
password: s3cret
username: admin
```

### Decode stdin

```bash
cat secret.yaml | s2t
```

```text
host: db.prod.internal
password: s3cret
username: admin
```

```bash
kubectl get secret db-creds -n prod -o yaml | s2t
```

### Fetch a live Secret (`name`, `-n`, `--namespace`)

```bash
s2t db-creds -n prod
```

Omit `-n` to use the namespace of the current kubeconfig context:

```bash
s2t db-creds
```

### Fetch a live ConfigMap (`kind/name`, `kind name`, `-k`, `--kind`)

These three commands fetch the same ConfigMap:

```bash
s2t cm/app-config -n prod
s2t cm app-config -n prod
s2t app-config -n prod -k configmap
```

### Name the resource with a flag (`--name`)

Use `--name` when the resource name is the same as a subcommand, such as `diff` or `set`:

```bash
s2t --name diff -n prod
s2t --name cm/edit -n prod
```

### Use a different kubeconfig (`--kubeconfig`)

```bash
s2t db-creds -n prod --kubeconfig ~/.kube/prod-cluster.yaml
```

### Choose the input format (`-t`, `--format`)

Force YAML parsing:

```bash
s2t -f secret.yaml -t yaml
```

Decode the `kv` format, `key: base64value` pairs separated by commas or newlines:

```bash
cat creds.kv
```

```text
username: YWRtaW4=,password: czNjcmV0
```

```bash
s2t -f creds.kv -t kv
```

```text
password: s3cret
username: admin
```

### Decrypt a SealedSecret (`-t sealed-secret`)

> [!IMPORTANT]
> Set `S2T_SEALED_SECRETS_KEY_FILE` to the path of the controller's private key first. `-t any` never selects this format.

```bash
export S2T_SEALED_SECRETS_KEY_FILE=~/.config/s2t/sealed-secrets-key.pem
s2t -f sealed.yaml -t sealed-secret
```

The key file must hold a PEM-encoded RSA key in PKCS#1 or PKCS#8 form.
`s2t` honors the `cluster-wide` and `namespace-wide` scope annotations.

### Decode a ConfigMap file (`-k configmap`)

`s2t` reads plain text from `data` and base64 from `binaryData`, then merges both into one output:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  LOG_LEVEL: info
  feature.flag: "on"
```

```bash
s2t -f app-config.yaml -k configmap
```

```text
LOG_LEVEL: info
feature.flag: on
```

### Print selected keys (`--only`)

```bash
s2t -f secret.yaml --only username,password
```

```text
password: s3cret
username: admin
```

A missing key is an error:

```bash
s2t -f secret.yaml --only token
```

```text
error: key "token" not found
```

### Print as a `.env` file (`-o env`)

```bash
s2t -f secret.yaml -o env
```

```text
host=db.prod.internal
password=s3cret
username=admin
```

```bash
s2t db-creds -n staging -o env > .env
```

### Read one value into a variable (`--only` with `-o env`)

```bash
PASSWORD=$(s2t db-creds -n prod --only password -o env | cut -d= -f2-)
```

### Print an indented patch (`-o json`)

```bash
s2t -f secret.yaml -o json
```

```text
{
  "stringData": {
    "host": "db.prod.internal",
    "password": "s3cret",
    "username": "admin"
  }
}
```

### Print a single-line patch (`-o jsonc`)

```bash
s2t -f secret.yaml -o jsonc
```

```text
{"stringData":{"host":"db.prod.internal","password":"s3cret","username":"admin"}}
```

Apply a local file to a live Secret:

```bash
kubectl patch secret db-creds -n prod --type merge -p "$(s2t -f secret.yaml -o jsonc)"
```

For a ConfigMap, the patch uses `data`:

```bash
s2t -f app-config.yaml -k configmap -o jsonc
```

```text
{"data":{"LOG_LEVEL":"info","feature.flag":"on"}}
```

### Print a YAML patch (`-o yaml`)

```bash
s2t -f secret.yaml -o yaml
```

```text
stringData:
    host: db.prod.internal
    password: s3cret
    username: admin
```

Copy a Secret from one namespace to another:

```bash
s2t db-creds -n staging -o yaml > patch.yaml
kubectl patch secret db-creds -n prod --type merge --patch-file patch.yaml
```

### Hide values (`--mask`)

```bash
s2t -f secret.yaml --mask
```

```text
host: ********
password: ********
username: ********
```

The placeholder has a fixed length, so it does not show the length of the real value.

> [!WARNING]
> `--mask` does not work with `-o json`, `jsonc`, or `yaml`. A masked patch would overwrite the real values with `********`, so `s2t` stops with an error.

`--mask` works with `-o env`:

```bash
s2t db-creds -n prod -o env --mask
```

### Print the version (`-v`, `--version`)

```bash
s2t --version
```

```text
s2t version 0.2.4
```

## Errors

| Message | Cause |
|---|---|
| `key "X" not found` | A key in `--only` is not in the resource. |
| `unknown resource kind "X" (want secret, configmap, or cm)` | The first of two arguments is not a known kind. |
| `--kind X conflicts with resource kind "Y" ...` | `--kind` and the kind in the name differ. |
| `conflicting name: positional argument "X" and --name "Y" differ` | The `name` argument and `--name` differ. |
| `--kind configmap is only supported with --format json, yaml, or any` | `-k configmap` was used with `kv` or `sealed-secret`. |
| `S2T_SEALED_SECRETS_KEY_FILE must be set ...` | `-t sealed-secret` was used without the key file. |
| `<kind> "X" not found in namespace "Y"` | The live resource does not exist. |
