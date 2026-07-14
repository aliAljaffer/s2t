# s2t - secret2text

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="images/s2t-logo-transparent-dark.svg">
  <img src="images/s2t-logo-transparent-light.svg" alt="s2t — secret to text">
</picture>

[![CI](https://github.com/aljaffer/s2t/actions/workflows/ci.yml/badge.svg)](https://github.com/aljaffer/s2t/actions/workflows/ci.yml)

A small CLI that decodes Kubernetes Secrets into readable key/value pairs.
Reads a raw manifest (YAML or JSON), a custom `key: value` blob, or fetches a
live secret via `kubectl`, base64-decodes every value, and prints the result
in a few different shapes.

`kubectl` is only required for the `--secret`/`-s` live-fetch flag. Decoding a
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
  s2t -f secret.yaml                            decode a saved manifest file
  cat secret.json | s2t                         decode piped stdin (format auto-detected)
  s2t -s db-creds -n prod                       fetch and decode a live secret via kubectl
  s2t -f secret.yaml --only username,password   only print specific keys
  s2t -f secret.yaml -o env                     print as KEY=value pairs
  s2t -s db-creds -n prod -o yaml               re-encode a live secret as a patch-ready manifest
```

## Advanced Usage

### Flags

| Flag                | Description                                                                                                                                     |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `-f`, `--file`      | Path to a file containing secret data; omit to read from stdin                                                                                  |
| `-t`, `--format`    | Input format: `yaml`, `json`, `kv`, or `sealed-secret` (default `any`, auto-detected among yaml/json/kv; `sealed-secret` must be requested explicitly) |
| `-s`, `--secret`    | Name of the secret to fetch live via `kubectl`                                                                                                  |
| `-n`, `--namespace` | Kubernetes namespace (used with `--secret`; defaults to the kubeconfig's current context if omitted)                                            |
| `--kubeconfig`      | Path to the kubeconfig file to use (default `~/.kube/config`)                                                                                   |
| `--only`            | Comma-separated list of keys to print                                                                                                           |
| `-o`, `--output`    | Output format: empty (plain), `env`, `json`, `jsonc`, or `yaml` (json/jsonc/yaml produce a patch-ready `stringData` manifest; jsonc is compact) |
| `-h`, `--help`      | Print usage                                                                                                                                     |

### Sealed Secrets

`s2t -f sealed.yaml -t sealed-secret` decrypts a [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) manifest client-side (`spec.encryptedData`), given the sealed-secrets controller's private key. Set the `S2T_SEALED_SECRETS_KEY_FILE` env var to a path to that key's PEM file:

```bash
export S2T_SEALED_SECRETS_KEY_FILE=~/.config/s2t/sealed-secrets-key.pem
s2t -f sealed.yaml -t sealed-secret
```

The key is read from a file path, not the env var's value directly, so the key material never touches argv or the environment itself. This isn't auto-detected by `-t any` — it must be requested explicitly, since it depends on external key material that no other format needs.

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
