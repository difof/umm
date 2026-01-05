# umm - Ultimate Multi-file Matcher

Interactive search tool combining **ripgrep**, **fzf**, and **bat** for fast code search with live preview.

**Compatible with:** bash, zsh

## What it does

- **Live search** - Results update as you type
- **Preview** - See file contents with syntax highlighting
- **Jump to line** - Opens your editor at the exact match
- **Multi-select** - Open multiple files at once

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
- `ripgrep` - Fast file search
- `fzf` - Interactive fuzzy finder
- A text editor (set via `$EDITOR`, defaults to `nvim`)

**Recommended:**
- `bat` - Syntax highlighting in preview (falls back to `sed` if not available)

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
git clone https://github.com/yourusername/umm.git

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
```

### Default Behavior

**umm uses ripgrep's smart defaults:**

- **Respects `.gitignore`** - Automatically excludes files/directories listed in `.gitignore`
- **Excludes `.git` directory** - Never searches inside `.git` by default
- **Excludes hidden files** - Files/directories starting with `.` are skipped (except `.gitignore` itself)
- **Skips binary files** - Binary files are automatically detected and excluded

To search **everything** (override all defaults), use the `--all` flag.

### Options

- `-p, --pattern REGEXP` - Initial search pattern
- `-e, --exclude PATTERN` - Exclude file/directory pattern (can be used multiple times)
- `-a, --all` - Search all files including .gitignore'd and hidden files
- `-n, --noui` - Non-interactive mode, open first match directly
- `-d, --max-depth N` - Maximum search depth
- `-h, --help` - Show help
- `-v, --version` - Show version

### Exclude Patterns

Exclude files or directories using gitignore-style glob patterns:

```bash
umm -e "*.log"                     # Exclude all .log files
umm -e "test" -e "vendor"          # Exclude multiple patterns
umm -e "**/node_modules/**"        # Exclude nested directories
umm -e "test\ dir"                 # Escape spaces in patterns
```

### Search All Files

By default, umm respects `.gitignore` and excludes hidden files (via ripgrep's defaults). Use `--all` to override:

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

The key to instant updates is fzf's `--disabled` mode:

```zsh
fzf --disabled \
  --bind "change:reload:sleep 0.05; rg {q} $root"
```

- `--disabled` - fzf delegates ALL search to ripgrep (no local filtering)
- `change:reload:` - Every keystroke triggers new ripgrep search
- `sleep 0.05` - Debounce to prevent system overload
- `{q}` - Current query from fzf input

Without `--disabled`, fzf would only filter pre-loaded results. This enables true live search.

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

Preview updates as you navigate because fzf re-runs the command with new values.

### Performance

1. **Debouncing** - Wait 50ms between keystrokes
2. **Binary skipping** - ripgrep skips binary files automatically
3. **Gitignore respect** - ripgrep respects `.gitignore` by default (use `--all` to override)
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

