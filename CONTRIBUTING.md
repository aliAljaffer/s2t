# Contributing

Thanks for considering a contribution to `s2t`.

## Getting started

```sh
git clone https://github.com/alialjaffer/s2t.git
cd s2t
make test
```

## Workflow

1. Fork the repo and create a branch off `main` for your change.
2. Keep changes focused — one logical change per PR.
3. Add or update tests for any behavior change. This project favors
   table-driven tests (see `*_test.go` files for examples).
4. Before opening a PR, make sure the following all pass:
   ```sh
   gofmt -l .
   go vet ./...
   go test ./...
   ```
5. Open a PR describing what changed and why.

`main` is protected: direct pushes are rejected, and every change lands through
a PR. The `go-check` CI job is a required status check, so a PR cannot merge
until it passes. No approving review is required (the maintainer merges their
own PRs), but review threads must be resolved first.

## Code style

- Run `gofmt` before committing; CI checks `go vet` and `go test` on every
  push and PR.
- Keep `main.go` thin — parsing/formatting logic belongs in its own file
  (see `models.go`, `decode.go`, `formatters.go`, `fetch.go` for the existing
  split).
- Avoid adding dependencies for things the standard library already covers.

## Reporting bugs

Open a GitHub issue with the command you ran, the input (redact any real
secret values), and what you expected vs. what happened.
