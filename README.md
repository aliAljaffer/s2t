# s2t

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="images/s2t-logo-dark.png">
  <img src="images/s2t-logo-light.png" alt="s2t — secret to text">
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

```sh
git clone https://github.com/aljaffer/s2t.git
cd s2t
make install
```

`make install` builds the binary and copies it to `~/.local/bin/s2t`. Make
sure that directory is on your `PATH`.

## Usage

```
s2t -h
```

```
Examples:
  s2t -f secret.yaml                            decode a saved manifest file
  cat secret.json | s2t                         decode piped stdin (format auto-detected)
  s2t -s db-creds -n prod                       fetch and decode a live secret via kubectl
  s2t -f secret.yaml --only username,password   only print specific keys
  s2t -f secret.yaml -o env                     print as KEY=value pairs
  s2t -s db-creds -n prod -o yaml               re-encode a live secret as a patch-ready manifest
```

### Flags

| Flag                | Description                                                                                                                                     |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `-f`, `--file`      | Path to a file containing secret data; omit to read from stdin                                                                                  |
| `-t`, `--format`    | Input format: `yaml`, `json`, or `kv` (default `any`, auto-detected)                                                                            |
| `-s`, `--secret`    | Name of the secret to fetch live via `kubectl`                                                                                                  |
| `-n`, `--namespace` | Kubernetes namespace (used with `--secret`; defaults to the kubeconfig's current context if omitted)                                            |
| `--kubeconfig`      | Path to the kubeconfig file to use (default `~/.kube/config`)                                                                                   |
| `--only`            | Comma-separated list of keys to print                                                                                                           |
| `-o`, `--output`    | Output format: empty (plain), `env`, `json`, `jsonc`, or `yaml` (json/jsonc/yaml produce a patch-ready `stringData` manifest; jsonc is compact) |
| `-h`, `--help`      | Print usage                                                                                                                                     |

## Development

```sh
make build   # go vet + go test + build to bin/
make test    # go test ./...
make vet     # go vet ./...
make clean   # remove bin/
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on contributing.

## License

[MIT](LICENSE)
