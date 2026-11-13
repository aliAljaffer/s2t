# s2t version

Print the version, the platform, and the release asset that `s2t upgrade` downloads.

## Synopsis

```text
s2t version
```

## Description

`s2t version` prints four lines:

| Line | Content |
|---|---|
| `s2t version` | The `s2t` version. A local build without a version shows `dev`. |
| `OS/Arch` | The operating system and CPU architecture of the binary. |
| `Release asset` | The file name that [`s2t upgrade`](s2t-upgrade.md) downloads. |
| `Go version` | The Go toolchain that built the binary. |

For the version alone, use `s2t --version`.

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-h`, `--help` | bool | `false` | Print the help text. |

The global flags `-t` and `-k` are ignored.

## Examples

### Print the version details

```bash
s2t version
```

```text
s2t version 0.2.4
 OS/Arch:       linux/amd64
 Release asset: s2t-linux-amd64
 Go version:    go1.27.4
```

### Print only the version number

```bash
s2t --version
```

```text
s2t version 0.2.4
```

### Read the version in a script

```bash
S2T_VERSION=$(s2t --version | awk '{print $3}')
```

## See also

- [`s2t upgrade`](s2t-upgrade.md)
- [`s2t`](index.md)
