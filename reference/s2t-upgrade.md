# s2t upgrade

Upgrade `s2t` to the latest GitHub release.

## Synopsis

```text
s2t upgrade [flags]
```

## Description

`s2t upgrade` runs these steps:

1. Read the latest release tag from the GitHub API.
2. Compare it with the running version. If the running version is the same or newer, print `s2t <version> is up to date.` and exit.
3. Ask `Upgrade? [y/N]`, unless you give `--yes`.
4. Install the new version:
   - Homebrew install (the binary path contains `/Cellar/`): run `brew upgrade alialjaffer/tap/s2t`.
   - Other installs: download the release binary for this OS and architecture, check its SHA-256 against `checksums.txt`, and replace the running binary.

A `dev` build always offers the upgrade. Run [`s2t version`](s2t-version.md) to see the asset name that `upgrade` downloads.

> [!IMPORTANT]
> `s2t` must be able to write to the directory of the binary. If it cannot, it stops with `cannot write next to <path> (try sudo)`.

On Windows, `s2t` renames the running binary to `s2t.exe.old` before it writes the new one.

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-y`, `--yes` | bool | `false` | Upgrade without a prompt. Required when stdin is not a terminal. |
| `-h`, `--help` | bool | `false` | Print the help text. |

The global flags `-t` and `-k` are ignored.

## Examples

### Upgrade with a prompt

```bash
s2t upgrade
```

```text
New version found! 0.2.3 -> 0.2.4
Upgrade? [y/N] y
Upgraded to 0.2.4.
```

### Upgrade without a prompt (`-y`, `--yes`)

```bash
s2t upgrade --yes
```

### Upgrade a binary in a system directory

```bash
sudo s2t upgrade --yes
```

### Upgrade a source checkout

`s2t upgrade` replaces the binary with a release build. To rebuild from source, run this in the clone:

```bash
make upgrade
```

## Errors

| Message | Cause |
|---|---|
| `not a terminal; pass --yes to upgrade without a prompt` | Stdin is not a terminal and `--yes` is not set. |
| `checking latest release: ...` | The GitHub API is not reachable or returned an error. |
| `no release binary for <os>/<arch>: ...` | The release has no binary for this platform. |
| `checksum mismatch for <asset>; refusing to install` | The download does not match `checksums.txt`. |

## See also

- [`s2t version`](s2t-version.md)
- [`s2t`](index.md)
