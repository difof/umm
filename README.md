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

---

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

---

## Usage

```bash
umm                          # Interactive search
umm -p "function"            # Start with pattern
umm -r ~/projects            # Search directory
umm -r ~/projects -p "TODO"  # Both
umm -p "error" -n            # Open first match (no UI)
umm -d 3                     # Limit depth
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

---

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
3. **Gitignore respect** - Skips `node_modules`, `.git` automatically
4. **Depth limiting** - Optional `--max-depth`
5. **Smart case** - Case-insensitive unless uppercase in query

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

---

## Contributing

1. Fork and create branch: `git checkout -b feat/name`
2. Make changes: `source umm.sh && umm --help`
3. Commit: `git commit -m "feat: description"` using conventional commits
4. Push and create PR

**Commit prefixes:** `feat:` `fix:` `docs:` `refactor:` `perf:`

---

## Acknowledgments

- [ripgrep](https://github.com/BurntSushi/ripgrep) - @BurntSushi
- [fzf](https://github.com/junegunn/fzf) - @junegunn
- [bat](https://github.com/sharkdp/bat) - @sharkdp

