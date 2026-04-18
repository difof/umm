# umm

`umm` is a Go CLI for interactive and non-interactive search across:

- file contents
- file paths
- directory names
- git objects

It uses `ripgrep` for search, `fzf` for the interactive picker, and optional preview tools such as `delta` and `bat`.

## Status

The Go binary is the primary interface.

The old `umm.sh` shell implementation is now legacy reference code and should not be treated as the main installation target.

## Features

- live file search with `fzf` reloads
- content and path matching by default
- file-only and dirname-only modes
- non-interactive `--no-ui` flows
- git search across commits, branches, tags, reflog, stashes, and tracked files
- file, directory, and git previews through `umm preview`
- editor opening with line targeting
- system-open actions
- Bubble Tea action picker via `--open-ask`
- stat output modes for file and directory results

## Requirements

Required tools depend on the mode you use.

Always required for normal file search:

- `rg`

Required for interactive search:

- `fzf`

Required for git mode:

- `git`

Required when opening in an editor:

- `$EDITOR` or the default `nvim`

Optional preview tools:

- `delta`
- `bat`

Fallback chains:

- diff preview: `delta -> bat -> cat -> internal`
- file preview: `bat -> cat -> internal`
- dir preview: internal tree preview

## Installation

### Build from source

```bash
git clone https://github.com/difof/umm.git
cd umm
go build -o umm .
```

### Install with Go

```bash
go install github.com/difof/umm@latest
```

## Usage

### Normal search

```bash
umm
umm --root ~/src --pattern TODO
umm --root ~/src --pattern root\.go --only-filename
umm --root ~/src --pattern cmd --only-dirname
umm --root ~/src --pattern TODO --exclude vendor/**
umm --root ~/src --pattern TODO --hidden
umm --root ~/src --pattern TODO --no-ui
umm --root ~/src --pattern TODO --only-stat lite
```

### Git search

```bash
umm --root ~/repo --git
umm --root ~/repo --git --git-mode commit,tracked
umm --root ~/repo --git --pattern 'tag:\s+v1'
umm --root ~/repo --git --no-ui --pattern 'branch:.*main'
```

### Action flags

```bash
umm --root ~/src --pattern TODO --open-ask
umm --root ~/src --pattern TODO --open-sys
umm --root ~/src --pattern TODO --only-stat full
```

## Flag Reference

Public v1 root flags:

- `-r, --root`
- `-p, --pattern`
- `-e, --exclude`
- `-a, --hidden`
- `--no-filename`
- `-f, --only-filename`
- `-d, --only-dirname`
- `-g, --git`
- `--git-mode`
- `-m, --max-depth`
- `-n, --no-ui`
- `-s, --no-multi`
- `-q, --open-ask`
- `-o, --open-sys`
- `--only-stat`
- `-h, --help`
- `-v, --version`

## Semantics

### Default file search

- searches file contents
- also searches file paths
- interactive default action opens selected files in `$EDITOR`

### `--hidden`

`--hidden` means include hidden files and ignored files.

### `--no-filename`

Search file contents only.

### `--only-filename`

Search file paths only.

This mode returns file results only, not directory results.

### `--only-dirname`

Search directory names only.

This mode returns the directory path itself. The default action is to print the selected directory path instead of opening it in the editor.

### `--no-ui`

`--no-ui` disables the interactive search picker.

Rules:

- `--pattern` is required
- normal file modes open the first compatible match by default
- dirname mode prints the first compatible directory path by default
- `--only-stat` prints all matching stat outputs
- git mode prints all matching git summaries by default

### `--open-ask`

After selection, `umm` shows a Bubble Tea action picker with these actions:

- `editor`
- `system`
- `stat`
- `cancel`

Directory results do not offer the `editor` action.

### `--only-stat`

Normal file and directory modes support:

- `full`
- `lite`
- `list`

In git mode, summary/stat output is already the default behavior, so `--only-stat` is accepted for consistency but does not change the output format in v1.

## Git Mode

Git mode searches a unified typed list containing:

- `commit:`
- `branch:`
- `tag:`
- `reflog:`
- `stash:`
- `file:`

`--git-mode` accepts repeated values and comma-separated values. If omitted, all git modes are enabled.

Example:

```bash
umm --root ~/repo --git --git-mode commit,branch
umm --root ~/repo --git --git-mode tracked --git-mode stash
```

Default git behavior:

- interactive mode prints git summaries for the selected items
- no-ui mode prints git summaries for all matches
- open actions only apply to tracked-file selections

Interactive git shortcut:

- `Ctrl+O` opens the tracked-file subset directly in `$EDITOR`

## Interactive Keys

Common:

- `Ctrl+G` / `Ctrl+B`: jump to bottom/top of result list
- `Alt+G` / `Alt+B`: jump to top/bottom of preview
- `Shift+Up` / `Shift+Down`: scroll preview by one line
- `Alt+U` / `Alt+D`: scroll preview by half a page
- `Ctrl+U` / `Ctrl+D`: scroll result list by half a page

Normal search:

- `Tab` / `Shift+Tab`: toggle multi-select and move

Git search:

- `Ctrl+/`: toggle preview pane
- `Ctrl+O`: open tracked-file selections in `$EDITOR`

## Editor Support

Supported editor argument styles:

- `vim`
- `vi`
- `nvim`
- `nano`
- `micro`
- `emacs`
- `emacsclient`
- `code`
- `code-insiders`
- `cursor`
- `agy`
- `subl`
- `sublime_text`

## Shell Completions

Generate completions with Cobra's built-in command.

### Bash

```bash
umm completion bash > /etc/bash_completion.d/umm
```

### Zsh

```bash
mkdir -p "${fpath[1]}"
umm completion zsh > "${fpath[1]}/_umm"
```

## Development

Useful local commands:

```bash
go test ./...
go build ./...
task test
task build
```

## Platform Support

Supported in v1:

- macOS
- Linux

Not part of v1:

- Windows
- Homebrew packaging
- YAML config
- worktree search
- git blame mode
- file-history search
