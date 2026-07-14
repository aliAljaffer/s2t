# s2t - secret2text

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="images/s2t-logo-transparent-dark.svg">
  <img src="images/s2t-logo-transparent-light.svg" alt="s2t — secret to text">
</picture>

[![CI](https://github.com/aliAljaffer/s2t/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/aliAljaffer/s2t/actions/workflows/ci.yml)

A small CLI that decodes Kubernetes Secrets into readable key/value pairs.
Reads a raw manifest (YAML or JSON), a custom `key: value` blob, or fetches a
live secret via `kubectl`, base64-decodes every value, and prints the result
in a few different shapes.

`kubectl` is only required for the `--name` live-fetch flag. Decoding a
file (`-f`) or piped stdin doesn't need it installed at all.

## Install

Requires Go 1.26+.

```bash
git clone https://github.com/aljaffer/s2t.git
cd s2t
make install
```

`make install` builds the binary and copies it to `~/.local/bin/s2t`. Make
sure that directory is on your `PATH`.

## Usage

```bash
s2t -h
```

```bash
Examples:
  s2t -f secret.yaml                              decode a saved manifest file
  cat secret.json | s2t                           decode piped stdin (format auto-detected)
  s2t --name db-creds --namespace prod            fetch and decode a live secret via kubectl
  s2t -f secret.yaml --only username,password     only print specific keys
  s2t -f secret.yaml -o env                       print as KEY=value pairs
  s2t --name db-creds --namespace prod -o yaml    re-encode a live secret as a patch-ready manifest
  s2t -f app.yaml -k configmap                    decode a ConfigMap manifest instead of a Secret
  s2t --name cm/app-config --namespace prod       fetch a ConfigMap live; kind is derived from the cm/ prefix
  s2t diff a.yaml b.yaml                          compare two secrets' decoded contents key by key
```

### Example outputs:

```bash
$ s2t -s ar # Auto-completion using default kubeconfig (-s is --namespace's shorthand)
argocd     arms-prom
$ s2t -s argocd --name argocd-initial-admin-secret # given the namespace and secret, print plain
password: CLG31TzP3S31XX5j
$ s2t -s argocd --name argocd-initial-admin-secret -oenv # print as .env file
password=CLG31TzP3S31XX5j
$ s2t -s argocd --name argocd-initial-admin-secret -oyaml # print as YAML string data
stringData:
    password: CLG31TzP3S31XX5j
$ s2t -s argocd --name argocd-initial-admin-secret -ojsonc # print as compact json (single line)
{"stringData":{"password":"CLG31TzP3S31XX5j"}}
```

### Flags

| Flag                | Description                                                                                                                                     | Default                                                 |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `-f`, `--file`      | Path to a file containing secret data                                                                                                           | `stdin`                                                 |
| `-t`, `--format`    | Input format: `yaml`, `json`, `kv`, or `sealed-secret` (`sealed-secret` must be requested explicitly), if empty, use `any` which auto-detects   | `any`                                                   |
| `-k`, `--kind`      | Resource kind: `secret` or `configmap`                                                                                                          | `secret`                                                |
| `-n`, `--name`      | Name of the resource to fetch live via `kubectl` — a plain name (uses `--kind`), or `kind/name` (e.g. `secret/my-secret`, `configmap/my-cm`, `cm/my-cm`) | -                                                       |
| `-s`, `--namespace` | Kubernetes namespace (used with `--name`)                                                                                                       | defaults to the kubeconfig's current context if omitted |
| `--kubeconfig`      | Path to the kubeconfig file to use                                                                                                              | `~/.kube/config`                                        |
| `--only`            | Comma-separated list of keys to print                                                                                                           | -                                                       |
| `-o`, `--output`    | Output format: empty (plain), `env`, `json`, `jsonc`, or `yaml` (json/jsonc/yaml produce a patch-ready manifest; jsonc is compact)              | empty                                                   |
| `--mask`            | Replace every value with a fixed-length placeholder; cannot be combined with `-o json`/`jsonc`/`yaml`                                           | `false`                                                 |
| `-h`, `--help`      | Print usage                                                                                                                                     | `false`                                                 |

### Real Use Cases

**Patch a live secret** from an edited local file — `-o jsonc` produces a compact, single-line `stringData` payload that's directly usable as a `kubectl patch` argument (the API server base64-encodes it server-side):

```bash
kubectl patch secret db-creds -n prod --type merge -p "$(s2t -f secret.yaml -o jsonc)"
```

**Export a live secret as a `.env` file** for local development:

```bash
s2t --name db-creds --namespace staging -o env > .env
```

**Diff two secret files' decoded values**, key by key, with `s2t diff` (see [Diffing secrets](#diffing-secrets) below):

```bash
s2t diff staging.yaml prod.yaml
```

Comparing two *live* secrets still works via the shell trick (`s2t diff` is file-only for now):

```bash
diff <(s2t --name db-creds --namespace staging) <(s2t --name db-creds --namespace prod)
```

**Grab a single value for scripting**, combining `--only` with `-o env`:

```bash
PASSWORD=$(s2t -f secret.yaml --only password -o env | cut -d= -f2-)
```

### Sealed Secrets

`s2t -f sealed.yaml -t sealed-secret` decrypts a [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) manifest client-side (`spec.encryptedData`), given the sealed-secrets controller's private key. Set the `S2T_SEALED_SECRETS_KEY_FILE` env var to a path to that key's PEM file:

```bash
export S2T_SEALED_SECRETS_KEY_FILE=~/.config/s2t/sealed-secrets-key.pem
s2t -f sealed.yaml -t sealed-secret
```

The key is read from a file path, not the env var's value directly, so the key material never touches argv or the environment itself. This isn't auto-detected by `-t any` — it must be requested explicitly, since it depends on external key material that no other format needs.

### ConfigMaps

`-k configmap` decodes a ConfigMap instead of a Secret. ConfigMaps store `data` as plain text (not base64) plus an optional `binaryData` for actual binary content — the opposite of a Secret's always-base64 `data` — so `s2t` handles both fields correctly and merges them into one output, whether they're used separately or together in the same manifest:

```bash
s2t -f app-config.yaml -k configmap
s2t --name app-config --namespace prod -k configmap        # live fetch
s2t -f app-config.yaml -k configmap -o jsonc       # patch payload wraps as {"data": ...}, not stringData
```

`-k configmap` only combines with `--format json`, `yaml`, or `any` — `kv` and `sealed-secret` have no defined ConfigMap shape and are rejected with a clear error.

The kind can also be embedded directly in `--name`, kubectl's `TYPE/NAME` style, instead of passing `-k` separately:

```bash
s2t --name secret/db-creds --namespace prod
s2t --name configmap/app-config --namespace prod
s2t --name cm/app-config --namespace prod           # cm is a short alias for configmap
```

If `--name` carries a `kind/` prefix and `-k` is also passed with a different kind, `s2t` rejects the conflict rather than silently picking one.

### Masking values

`--mask` replaces every value with a fixed-length placeholder (`********`, same length regardless of the real value's length, so it doesn't leak how long the secret actually is) — for sharing output in a screen-share, chat message, or CI log without exposing the real value:

```bash
s2t -f secret.yaml --mask
```

`--mask` cannot be combined with `-o json`, `jsonc`, or `yaml`: those produce a `kubectl patch`-ready payload, and a masked value would silently overwrite the real secret with the literal string `"********"` if applied. `s2t` refuses this combination outright rather than risk it.

### Diffing secrets

`s2t diff <fileA> <fileB>` compares two manifests' decoded contents key by key — added, removed, and changed keys, not a raw line diff:

```bash
s2t diff staging.yaml prod.yaml
```

```
- deprecated-key
+ new-feature-flag
~ password
```

Values are hidden by default (diff output is more likely to end up pasted into a ticket or chat than a single decode). Pass `--show-values` to see the actual old/new values for changed keys:

```bash
s2t diff --show-values staging.yaml prod.yaml
```

```
- deprecated-key: old-value
+ new-feature-flag: enabled
~ password: staging-pass -> prod-pass
```

`s2t diff` shares `--format`/`-t` and `--kind`/`-k` with the root command (both files are assumed to be the same format/kind), and is file-only for now — see the shell-trick example above for comparing two live secrets.

## Development

```bash
make build   # go vet + go test + build to bin/
make test    # go test ./...
make vet     # go vet ./...
make clean   # remove bin/
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on contributing.

## License

[MIT](LICENSE)
