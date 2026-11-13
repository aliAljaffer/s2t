# s2t completion

Generate a shell completion script for `s2t`.

## Synopsis

```text
s2t completion bash [--no-descriptions]
s2t completion zsh [--no-descriptions]
s2t completion fish [--no-descriptions]
s2t completion powershell [--no-descriptions]
```

## Description

`s2t completion` prints a completion script for the given shell to stdout.

With completion loaded, the shell completes:

| What | Source |
|---|---|
| Subcommands | Built in. |
| `-n`, `--namespace` | Live namespaces from `kubectl get namespace`. |
| Resource names (root, `set`, `edit`, `--name`) | Live Secrets or ConfigMaps in the namespace from `-n`, or the current context. |
| `kind/` prefixes, such as `cm/app-` | Live names of that kind. |
| `-k`, `--kind` | `secret` and `configmap`. |

Each `kubectl` call for completion has a 2-second timeout. If `kubectl` fails or times out, the shell shows no suggestions.
With an empty word and no `-n`, the root command suggests only subcommands.

## Subcommands

| Subcommand | Description |
|---|---|
| `bash` | Generate a script for Bash. Needs the `bash-completion` package. |
| `zsh` | Generate a script for Zsh. |
| `fish` | Generate a script for fish. |
| `powershell` | Generate a script for PowerShell. |

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--no-descriptions` | bool | `false` | Leave out the description next to each suggestion. |
| `-h`, `--help` | bool | `false` | Print the help text. |

## Examples

### Bash

Load completion in the current session:

```bash
source <(s2t completion bash)
```

Load completion in every session on Linux:

```bash
s2t completion bash | sudo tee /etc/bash_completion.d/s2t > /dev/null
```

Load completion in every session on macOS with Homebrew:

```bash
s2t completion bash > "$(brew --prefix)/etc/bash_completion.d/s2t"
```

### Zsh

Turn on completion once, if it is not on already:

```bash
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

Load completion in the current session:

```zsh
source <(s2t completion zsh)
```

Load completion in every session on Linux:

```zsh
s2t completion zsh > "${fpath[1]}/_s2t"
```

Load completion in every session on macOS with Homebrew:

```zsh
s2t completion zsh > "$(brew --prefix)/share/zsh/site-functions/_s2t"
```

### fish

Load completion in the current session:

```fish
s2t completion fish | source
```

Load completion in every session:

```fish
s2t completion fish > ~/.config/fish/completions/s2t.fish
```

### PowerShell

Load completion in the current session:

```powershell
s2t completion powershell | Out-String | Invoke-Expression
```

Load completion in every session:

```powershell
s2t completion powershell >> $PROFILE
```

### Leave out descriptions (`--no-descriptions`)

```bash
source <(s2t completion bash --no-descriptions)
```

### Use completion

```bash
s2t -n ar<TAB>
```

```text
argocd     arms-prom
```

Start a new shell after you install a script for every session.

## See also

- [`s2t`](index.md)
