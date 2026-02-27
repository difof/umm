# umm - Ultimate Multi-file Matcher

Interactive search tool for both file content and [Git](https://github.com/git/git) objects, powered by **[ripgrep](https://github.com/BurntSushi/ripgrep)**, **[fzf](https://github.com/junegunn/fzf)**, and optional preview tools like **[bat](https://github.com/sharkdp/bat)**/**[delta](https://github.com/dandavison/delta)**.

**Compatible with:** bash, zsh

## What it does

- **Live search** - Results update as you type
- **Path + content matching** - Searches filenames/paths and file contents by default
- **Preview** - See file contents with syntax highlighting
- **Jump to line** - Opens your editor at the exact match
- **Multi-select** - Open multiple files at once
- **[Git](https://github.com/git/git) mode** - Search commits, branches, tags, reflog, and stashes in one view
- **Diff pager fallback** - Uses [`delta`](https://github.com/dandavison/delta), then [`bat`](https://github.com/sharkdp/bat), then [`cat`](https://www.gnu.org/software/coreutils/)

```bash
# Old way
$ grep -r "searchTerm" .
$ cd path/to/file
$ vim file.js
# ... search again ...

# With umm
$ umm
# type, see results, hit enter
# opens at exact line
```

## Installation

**Required:**
- [`ripgrep` (`rg`)](https://github.com/BurntSushi/ripgrep) - Fast file search
- [`fzf`](https://github.com/junegunn/fzf) - Interactive fuzzy finder
- A text editor (set via `$EDITOR`, defaults to `nvim`)

**Required for [Git](https://github.com/git/git) mode:**
- [`git`](https://github.com/git/git) - Repository search and previews

**Recommended:**
- [`delta`](https://github.com/dandavison/delta) - Recommended for the best [Git](https://github.com/git/git) diff preview (`delta` -> `bat` -> `cat`)
- [`bat`](https://github.com/sharkdp/bat) - Syntax highlighting in file previews (falls back to [`sed`](https://www.gnu.org/software/sed/) + line numbers if unavailable)

```bash
# macOS
brew install ripgrep fzf bat nvim

# Ubuntu/Debian
apt install ripgrep fzf bat neovim

# Arch
pacman -S ripgrep fzf bat neovim
```

**Install:**
```bash
# Clone
git clone https://github.com/difof/umm.git

# For zsh (includes tab completions)
echo 'source /path/to/umm/umm.sh' >> ~/.zshrc
source ~/.zshrc

# For bash
echo 'source /path/to/umm/umm.sh' >> ~/.bashrc
source ~/.bashrc
```

## Usage

```bash
umm                                # Interactive search in current directory
umm ~/projects                     # Search in specific directory
umm -p "function"                  # Start with pattern
umm -p "TODO" ~/projects           # Search with pattern in directory
umm -e "*.log" -e "test"           # Exclude patterns (gitignore-style globs)
umm -a                             # Search all files (ignore .gitignore, include hidden)
umm -p "error" -n                  # Open first match (no UI)
umm -d 3                           # Limit search depth

umm -g                             # Search git objects (commits/branches/tags/reflog/stashes)
umm -g -p "fix"                    # Start git mode with pattern
umm --git ~/projects/repo          # Git search in specific repository
```

### Default Behavior

**umm uses [ripgrep](https://github.com/BurntSushi/ripgrep)'s smart defaults:**

- **Respects `.gitignore`** - Automatically excludes files/directories listed in `.gitignore`
- **Excludes `.git` directory** - Never searches inside `.git` by default
- **Excludes hidden files** - Files/directories starting with `.` are skipped (except `.gitignore` itself)
- **Skips binary files** - Binary files are automatically detected and excluded

To search **everything** (override all defaults), use the `--all` flag.

### Options

- `-p, --pattern REGEXP` - Initial search pattern
- `-e, --exclude PATTERN` - Exclude file/directory pattern (can be used multiple times)
- `-a, --all` - Search all files including .gitignore'd and hidden files
- `-g, --git` - Search [Git](https://github.com/git/git) objects in a unified list
- `-n, --noui` - Non-interactive mode, open first match directly
- `-d, --max-depth N` - Maximum search depth
- `-h, --help` - Show help
- `-v, --version` - Show version

### Git Mode

When `-g/--git` is enabled, umm shows a single searchable list with [Git](https://github.com/git/git) type-prefixed entries:

- `commit:` recent commit history (up to 1000 entries)
- `branch:` local and remote branches
- `tag:` tags with subjects
- `reflog:` recent reflog entries (up to 100)
- `stash:` stash entries

Preview is context-aware by type and uses this diff rendering fallback chain:

- [`delta`](https://github.com/dandavison/delta) (recommended)
- [`bat`](https://github.com/sharkdp/bat) (`--style=numbers,changes --language=diff`)
- [`cat`](https://www.gnu.org/software/coreutils/) (plain output fallback)

Selection output strips the type prefix, so results are easy to pipe:

```bash
umm -g -p "commit:" | cut -d' ' -f1 | xargs git show
umm -g -p "branch:" | sed 's/^[* ]*//' | xargs git checkout
```

### Keybindings

Common (file mode and [Git](https://github.com/git/git) mode):

- `Shift+Up` / `Shift+Down` - Scroll preview one line up/down
- `Alt+U` / `Alt+D` - Scroll preview half-page up/down
- `Ctrl+U` / `Ctrl+D` - Scroll result list half-page up/down

Mode-specific:

- File mode: `Tab` / `Shift+Tab` toggle multi-select and move
- Git mode: `Ctrl+/` toggle preview pane

### Exclude Patterns

Exclude files or directories using gitignore-style glob patterns:

```bash
umm -e "*.log"                     # Exclude all .log files
umm -e "test" -e "vendor"          # Exclude multiple patterns
umm -e "**/node_modules/**"        # Exclude nested directories
umm -e "test\ dir"                 # Escape spaces in patterns
```

### Search All Files

By default, umm respects `.gitignore` and excludes hidden files (via [ripgrep](https://github.com/BurntSushi/ripgrep)'s defaults). Use `--all` to override:

```bash
umm -a                             # Search everything (ignore .gitignore, include hidden)
umm -a -e ".git"                   # Search all but exclude .git directory
umm -a -p "SECRET" ~/project       # Find sensitive data in all files
```

### Editor Support

umm respects the `$EDITOR` environment variable and automatically uses the correct syntax for different editors:

```bash
# Use your preferred editor
EDITOR=vim umm              # Opens with: vim +linenum file
EDITOR=code umm             # Opens with: code --goto file:linenum
EDITOR=nano umm             # Opens with: nano +linenum file
EDITOR=micro umm            # Opens with: micro +linenum file

# Set default in your shell config
export EDITOR=nvim          # Add to ~/.bashrc or ~/.zshrc
```

**Supported editors:** vim, vi, nvim, nano, micro, emacs, code (VSCode), cursor, subl (Sublime Text), and more.

## How it works

### Live reload mechanism

The key to instant updates is [fzf](https://github.com/junegunn/fzf)'s `--disabled` mode:

```zsh
fzf --disabled \
  --bind "change:reload:sleep 0.05; rg {q} $root"
```

- `--disabled` - [fzf](https://github.com/junegunn/fzf) delegates ALL search to [ripgrep](https://github.com/BurntSushi/ripgrep) (no local filtering)
- `change:reload:` - Every keystroke triggers new [ripgrep](https://github.com/BurntSushi/ripgrep) search
- `sleep 0.05` - Debounce to prevent system overload
- `{q}` - Current query from [fzf](https://github.com/junegunn/fzf) input

Without `--disabled`, [fzf](https://github.com/junegunn/fzf) would only filter pre-loaded results. This enables true live search.

### Preview system

```zsh
bat --color=always \
  --highlight-line {2} \
  --line-range {2}::15 \
  {1}
```

- `{1}` = file path (from `--delimiter=:`)
- `{2}` = line number
- `--line-range {2}::15` = show line with 15 lines of context

Preview updates as you navigate because [fzf](https://github.com/junegunn/fzf) re-runs the command with new values.

### Performance

1. **Debouncing** - Wait 50ms between keystrokes
2. **Binary skipping** - [ripgrep](https://github.com/BurntSushi/ripgrep) skips binary files automatically
3. **Gitignore respect** - [ripgrep](https://github.com/BurntSushi/ripgrep) respects `.gitignore` by default (use `--all` to override)
4. **Hidden files excluded** - Hidden files/directories excluded by default (use `--all` to include)
5. **Depth limiting** - Optional `--max-depth` flag
6. **Smart case** - Case-insensitive unless uppercase in query

## Troubleshooting

**Command not found:**
```bash
# Reload your shell config
source ~/.zshrc   # for zsh
source ~/.bashrc  # for bash
```

**No syntax highlighting:**
```bash
brew install bat
```

**Slow on large projects:**
```bash
umm -d 3  # Limit depth
```

## Contributing

1. Fork and create branch: `git checkout -b feat/name`
2. Make changes: `source umm.sh && umm --help`
3. Commit: `git commit -m "feat: description"` using conventional commits
4. Push and create PR

**Commit prefixes:** `feat:` `fix:` `docs:` `refactor:` `perf:`

## Acknowledgments

- [ripgrep](https://github.com/BurntSushi/ripgrep) - @BurntSushi
- [fzf](https://github.com/junegunn/fzf) - @junegunn
- [bat](https://github.com/sharkdp/bat) - @sharkdp
- [delta](https://github.com/dandavison/delta) - @dandavison
