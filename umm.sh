#!/usr/bin/env bash
# umm - Ultimate Multi-file Matcher
# Interactive search tool with live preview and instant file opening
# Compatible with: bash, zsh
# Version: 1.0.0
# Author: difof
# License: MIT

UMM_VERSION="1.0.0"

# Color codes
if [[ -t 2 ]]; then
  C_RED='\033[0;31m'
  C_GREEN='\033[0;32m'
  C_YELLOW='\033[0;33m'
  C_BLUE='\033[0;34m'
  C_CYAN='\033[0;36m'
  C_RESET='\033[0m'
else
  C_RED=''
  C_GREEN=''
  C_YELLOW=''
  C_BLUE=''
  C_CYAN=''
  C_RESET=''
fi

_error() { echo -e "${C_RED}x${C_RESET} $*" >&2; }
_success() { echo -e "${C_GREEN}*${C_RESET} $*" >&2; }
_info() { echo -e "${C_CYAN}i${C_RESET} $*" >&2; }
_warn() { echo -e "${C_YELLOW}!${C_RESET} $*" >&2; }

# Diff pager detection with fallback chain: delta > bat > cat
# Returns the command to use for piping diff output
_get_diff_pager() {
  if command -v delta >/dev/null 2>&1; then
    echo "delta"
  elif command -v bat >/dev/null 2>&1; then
    echo "bat --style=numbers,changes --language=diff --color=always"
  else
    echo "cat"
  fi
}

# Check which diff pager is available (for debugging/testing)
_get_diff_pager_name() {
  if command -v delta >/dev/null 2>&1; then
    echo "delta"
  elif command -v bat >/dev/null 2>&1; then
    echo "bat"
  else
    echo "cat"
  fi
}

# Get editor type and build appropriate arguments
_build_editor_args() {
  local editor="$1"
  local file="$2"
  local linenum="$3"
  local -a args
  
  # Get base editor name without path
  local editor_name="${editor##*/}"
  
  # Different editors use different syntax for opening at line number
  case "$editor_name" in
    vim|vi|nvim|nano)
      # +linenum file
      if [[ -n "$linenum" ]]; then
        args=("+$linenum" "$file")
      else
        args=("$file")
      fi
      ;;
    micro|emacs|emacsclient)
      # +linenum file
      if [[ -n "$linenum" ]]; then
        args=("+$linenum" "$file")
      else
        args=("$file")
      fi
      ;;
    code|code-insiders|cursor|agy)
      # --goto file:linenum
      if [[ -n "$linenum" ]]; then
        args=("--goto" "$file:$linenum")
      else
        args=("$file")
      fi
      ;;
    subl|sublime_text)
      # file:linenum
      if [[ -n "$linenum" ]]; then
        args=("$file:$linenum")
      else
        args=("$file")
      fi
      ;;
    *)
      # Default: try +linenum file (works for most terminal editors)
      if [[ -n "$linenum" ]]; then
        args=("+$linenum" "$file")
      else
        args=("$file")
      fi
      ;;
  esac
  
  printf '%s\n' "${args[@]}"
}

# Validate if a path is within a git repository
_git_validate_repo() {
  local repo_path="$1"
  
  # Check if git is available
  if ! command -v git >/dev/null 2>&1; then
    _error "git is not installed"
    return 1
  fi
  
  # Check if path is in a git repository
  if ! git -C "$repo_path" rev-parse --git-dir >/dev/null 2>&1; then
    _error "Not a git repository: ${C_CYAN}$repo_path${C_RESET}"
    return 1
  fi
  
  return 0
}

# Aggregate all git objects into searchable format with prefixes
_git_aggregate_data() {
  local repo_path="$1"
  local pattern="$2"
  
  # Collect commits (limit to 1000 for performance)
  git -C "$repo_path" log --oneline --all --color=always -n 1000 2>/dev/null | \
    sed 's/^/commit:  /' || true
  
  # Collect branches (local and remote)
  git -C "$repo_path" branch -a --color=always 2>/dev/null | \
    sed 's/^/branch:  /' || true
  
  # Collect tags with annotations
  git -C "$repo_path" tag -l --format="%(refname:short) %(subject)" 2>/dev/null | \
    sed 's/^/tag:     /' || true
  
  # Collect reflog entries (limit to 100)
  git -C "$repo_path" reflog --color=always -n 100 2>/dev/null | \
    sed 's/^/reflog:  /' || true
  
  # Collect stashes
  git -C "$repo_path" stash list 2>/dev/null | \
    sed 's/^/stash:   /' || true
}

# Generate preview based on selected git object type
# Uses diff pager (delta > bat > cat) for improved diff readability
_git_preview() {
  local repo_path="$1"
  local line="$2"
  
  # Extract type prefix
  local type="${line%%:*}"
  # Remove prefix and leading spaces
  local content="${line#*:}"
  content="${content#"${content%%[![:space:]]*}"}"
  
  # Get the diff pager command
  local pager_name=$(_get_diff_pager_name)
  local pager_cmd=$(_get_diff_pager)
  
  # Determine git color settings based on pager
  # delta: use no color (delta does its own coloring)
  # bat/cat: use git's color output
  local git_color="always"
  if [[ "$pager_name" == "delta" ]]; then
    git_color="never"
  fi
  
  case "$type" in
    commit)
      # Extract commit hash (first word after prefix)
      local hash=$(echo "$content" | awk '{print $1}')
      if [[ "$pager_name" == "cat" ]]; then
        # cat: show with --stat for compact view
        git -C "$repo_path" show --color=always --stat "$hash" 2>/dev/null || \
          echo "Error: Could not show commit $hash"
      else
        # delta/bat: show full diff (no --stat) for proper syntax highlighting
        git -C "$repo_path" show --color=$git_color "$hash" 2>/dev/null | \
          eval "$pager_cmd" 2>/dev/null || \
          echo "Error: Could not show commit $hash"
      fi
      ;;
    branch)
      # Extract branch name (remove leading * and spaces)
      local branch=$(echo "$content" | sed 's/^[* ]*//' | awk '{print $1}')
      echo "Recent commits on branch: $branch"
      echo "----------------------------------------"
      git -C "$repo_path" log --oneline --color=always -10 "$branch" 2>/dev/null || \
        echo "Error: Could not show branch $branch"
      ;;
    tag)
      # Extract tag name (first word)
      local tag=$(echo "$content" | awk '{print $1}')
      if [[ "$pager_name" == "cat" ]]; then
        # cat: show with --stat for compact view
        git -C "$repo_path" show --color=always --stat "$tag" 2>/dev/null || \
          echo "Error: Could not show tag $tag"
      else
        # delta/bat: show full diff (no --stat) for proper syntax highlighting
        git -C "$repo_path" show --color=$git_color "$tag" 2>/dev/null | \
          eval "$pager_cmd" 2>/dev/null || \
          echo "Error: Could not show tag $tag"
      fi
      ;;
    reflog)
      # Extract reflog entry (first word like HEAD@{0})
      local entry=$(echo "$content" | awk '{print $1}')
      if [[ "$pager_name" == "cat" ]]; then
        # cat: show with --stat for compact view
        git -C "$repo_path" show --color=always --stat "$entry" 2>/dev/null || \
          echo "Error: Could not show reflog entry $entry"
      else
        # delta/bat: show full diff (no --stat) for proper syntax highlighting
        git -C "$repo_path" show --color=$git_color "$entry" 2>/dev/null | \
          eval "$pager_cmd" 2>/dev/null || \
          echo "Error: Could not show reflog entry $entry"
      fi
      ;;
    stash)
      # Extract stash id (like stash@{0})
      local stash=$(echo "$content" | grep -o 'stash@{[0-9]*}' | head -n1)
      # Stash always shows full diff
      if [[ "$pager_name" == "cat" ]]; then
        git -C "$repo_path" stash show -p --color=always "$stash" 2>/dev/null || \
          echo "Error: Could not show stash $stash"
      else
        git -C "$repo_path" stash show -p --color=$git_color "$stash" 2>/dev/null | \
          eval "$pager_cmd" 2>/dev/null || \
          echo "Error: Could not show stash $stash"
      fi
      ;;
    *)
      echo "Unknown type: $type"
      ;;
  esac
}

# Main git search function
_git_search() {
  local repo_path="$1"
  local pattern="$2"
  
  # Get absolute path for repo
  repo_path=$(cd "$repo_path" && pwd)
  
  # Create a temporary wrapper script for preview
  # This is needed because fzf preview needs to call our function
  local preview_script=$(mktemp)
  cat > "$preview_script" << 'PREVIEW_EOF'
#!/usr/bin/env bash
source "$UMM_SCRIPT_PATH"
_git_preview "$UMM_REPO_PATH" "$1"
PREVIEW_EOF
  chmod +x "$preview_script"
  
  # Export variables for preview script
  export UMM_SCRIPT_PATH="${BASH_SOURCE[0]:-${(%):-%x}}"
  export UMM_REPO_PATH="$repo_path"
  
  # Aggregate git data
  local git_data=$(_git_aggregate_data "$repo_path" "$pattern")
  
  if [[ -z "$git_data" ]]; then
    _error "No git objects found in repository"
    rm -f "$preview_script"
    return 1
  fi
  
  # Get pager info for header
  local pager_name=$(_get_diff_pager_name)
  
  # Build fzf options
  local fzf_opts=(
    --ansi
    --no-sort
    --tiebreak=index
    --query="$pattern"
    --delimiter=':'
    --prompt="> Git: "
    --info=inline
    --preview="$preview_script {}"
    --preview-window="top:60%"
    --bind "ctrl-/:toggle-preview"
    --header="COMMITS | BRANCHES | TAGS | REFLOG | STASHES | Pager: $pager_name"
  )
  
  # Run fzf
  local selected=$(echo "$git_data" | fzf "${fzf_opts[@]}")
  
  # Clean up
  rm -f "$preview_script"
  unset UMM_SCRIPT_PATH
  unset UMM_REPO_PATH
  
  # Check if selection was made
  if [[ -z "$selected" ]]; then
    _info "Search cancelled"
    return 0
  fi
  
  # Strip prefix and output
  local content="${selected#*:}"
  content="${content#"${content%%[![:space:]]*}"}"
  echo "$content"
}

umm() {
  # Use EDITOR environment variable, default to nvim
  local UMM_EDITOR="${EDITOR:-nvim}"
  
  local root="."
  local pattern=""
  local noui=false
  local max_depth=""
  local -a exclude_patterns=()
  local scan_all=false
  local positional_set=false
  local git_mode=false
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      --help|-h)
        cat << EOF
umm - Ultimate Multi-file Matcher

USAGE:
  umm [OPTIONS] [root_path]

OPTIONS:
  -p, --pattern REGEXP   Initial search pattern
  -e, --exclude PATTERN  Exclude file/directory pattern (gitignore-style glob)
                         Can be used multiple times
                         Examples: '*.log', 'test/', '**/tmp/**'
  -a, --all              Search all files (ignore .gitignore, include hidden)
  -g, --git              Search git repository (commits, branches, tags, etc.)
                         Combines all git objects into one searchable list
                         Use prefixes to filter: commit:, branch:, tag:, etc.
  -n, --noui             Non-interactive mode, open first match
  -d, --max-depth N      Maximum search depth
  -h, --help             Show this help
  -v, --version          Show version

ARGUMENTS:
  root_path              Directory to search (default: current directory)

ENVIRONMENT:
  EDITOR                 Editor to use (default: nvim)
                         Supported: vim, vi, nvim, nano, micro, emacs,
                         code, subl, and more

EXAMPLES:
  umm                                # Interactive search in current directory
  umm ~/projects                     # Search in ~/projects
  umm -p "function" ~/projects       # Search with initial pattern
  umm -e "*.log" -e "test"           # Exclude log files and test directories
  umm -a                             # Search all files (ignore .gitignore)
  umm -p "TODO" -n ~/src             # Open first match directly
  
  Git Mode:
  umm -g                             # Search all git objects (commits, branches, etc.)
  umm -g -p "fix"                    # Search git objects with initial pattern
  umm -g ~/projects/repo             # Search git objects in specific repository
  umm -g -p "commit:" | cut -d' ' -f1 | xargs git show  # Pipe to git commands
EOF
        return 0
        ;;
      --version|-v)
        echo "umm version $UMM_VERSION"
        return 0
        ;;
      --pattern|-p)
        if [[ -z "$2" || "$2" == -* ]]; then
          _error "Option --pattern requires a value"
          return 1
        fi
        pattern="$2"
        shift 2
        ;;
      --max-depth|-d)
        if [[ -z "$2" || "$2" == -* ]]; then
          _error "Option --max-depth requires a value"
          return 1
        fi
        if [[ ! "$2" =~ ^[0-9]+$ ]]; then
          _error "Option --max-depth must be a number"
          return 1
        fi
        max_depth="$2"
        shift 2
        ;;
      --exclude|-e)
        if [[ -z "$2" || "$2" == -* ]]; then
          _error "Option --exclude requires a value"
          return 1
        fi
        exclude_patterns+=("$2")
        shift 2
        ;;
      --all|-a)
        scan_all=true
        shift
        ;;
      --git|-g)
        git_mode=true
        shift
        ;;
      --noui|-n)
        noui=true
        shift
        ;;
      *)
        if [[ "$1" == -* ]]; then
          _error "Unknown option: $1"
          echo "Use ${C_CYAN}--help${C_RESET} for usage" >&2
          return 1
        fi
        # Positional argument (root path)
        if [[ "$positional_set" == true ]]; then
          _error "Too many arguments. Expected: umm [OPTIONS] [root_path]"
          return 1
        fi
        root="$1"
        positional_set=true
        shift
        ;;
    esac
  done
  
  # Check dependencies
  local missing_deps=()
  command -v rg >/dev/null 2>&1 || missing_deps+=("ripgrep (rg)")
  command -v fzf >/dev/null 2>&1 || missing_deps+=("fzf")
  command -v "$UMM_EDITOR" >/dev/null 2>&1 || missing_deps+=("$UMM_EDITOR")
  
  if [[ ${#missing_deps[@]} -gt 0 ]]; then
    _error "Missing required dependencies:"
    printf "  ${C_RED}-${C_RESET} %s\n" "${missing_deps[@]}" >&2
    _info "Install with: ${C_CYAN}brew install ripgrep fzf bat neovim${C_RESET}"
    return 1
  fi
  
  # Check if bat is available
  local has_bat=false
  command -v bat >/dev/null 2>&1 && has_bat=true
  
  # Check if root directory exists
  if [[ ! -d "$root" ]]; then
    _error "Directory '$root' does not exist"
    return 1
  fi
  
  # Git mode branch
  if [[ "$git_mode" == true ]]; then
    # Validate git repository
    if ! _git_validate_repo "$root"; then
      return 1
    fi
    
    # Run git search
    _git_search "$root" "$pattern"
    return $?
  fi
  
  # Check pattern required for noui mode
  if [[ -z "$pattern" && "$noui" == true ]]; then
    _error "Option --pattern is required when using --noui"
    return 1
  fi
  
  # Build ripgrep options
  local rg_opts=(
    --line-number
    --no-heading
    --smart-case
  )
  
  # Add color only for interactive mode
  if [[ "$noui" != true ]]; then
    rg_opts+=(--color=always)
  fi
  
  [[ -n "$max_depth" ]] && rg_opts+=(--max-depth "$max_depth")
  
  # Add exclude patterns
  for exclude_pattern in "${exclude_patterns[@]}"; do
    rg_opts+=(--glob "!$exclude_pattern")
  done
  
  # Add --all flag options
  if [[ "$scan_all" == true ]]; then
    rg_opts+=(--no-ignore --hidden)
  fi
  
  local selected
  
  if [[ "$noui" == true ]]; then
    # Non-interactive mode
    selected=$(rg "${rg_opts[@]}" "$pattern" "$root" 2>/dev/null | head -n1)
    
    if [[ -z "$selected" ]]; then
      _error "No matches found for pattern: ${C_YELLOW}$pattern${C_RESET}"
      return 1
    fi
    
    _success "Found match, opening in $UMM_EDITOR..."
  else
    # Interactive mode
    local root_escaped=$(printf %q "$root")
    local rg_command="rg ${rg_opts[*]} {q} $root_escaped 2>/dev/null || true"
    
    # Build preview command
    local preview_cmd
    if [[ "$has_bat" == true ]]; then
      preview_cmd="bat --color=always --style=numbers,header --highlight-line {2} --line-range {2}::15 {1} 2>/dev/null"
    else
      preview_cmd='f={1}; l={2}; s=$((l > 10 ? l - 10 : 1)); e=$((l + 20)); sed -n "${s},${e}p" "$f" 2>/dev/null | nl -ba -s" " -w4 -v"$s"'
    fi
    
    # Build fzf options
    local fzf_opts=(
      --ansi
      --disabled
      --query="$pattern"
      --delimiter=:
      --prompt="> Search: "
      --info=inline
      --preview="$preview_cmd"
      --preview-window="top:60%"
      --bind "change:reload:sleep 0.05; $rg_command"
      --bind "start:reload:$rg_command"
      --multi
      --bind "tab:toggle+down,shift-tab:toggle+up"
    )
    
    # Run fzf
    selected=$(FZF_DEFAULT_COMMAND="rg ${rg_opts[*]} $(printf %q "$pattern") $root_escaped 2>/dev/null || true" \
      fzf "${fzf_opts[@]}")
  fi
  
  # Check if selection was made
  if [[ -z "$selected" ]]; then
    [[ "$noui" != true ]] && _info "Search cancelled"
    return 0
  fi
  
  # Parse and open selected file(s)
  local files_opened=0
  local -a all_files
  local first_file=""
  local first_linenum=""
  
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    
    # Parse file:line from selected result
    local file="${line%%:*}"
    local rest="${line#*:}"
    local linenum="${rest%%:*}"
    
    # Validate file
    if [[ -z "$file" ]]; then
      _warn "Could not parse file from selection: $line"
      continue
    fi
    
    if [[ ! -f "$file" ]]; then
      _warn "File does not exist: $file"
      continue
    fi
    
    # Store first file with line number
    if [[ $files_opened -eq 0 ]]; then
      first_file="$file"
      if [[ -n "$linenum" && "$linenum" =~ ^[0-9]+$ ]]; then
        first_linenum="$linenum"
      fi
    else
      all_files+=("$file")
    fi
    
    ((files_opened++))
  done <<< "$selected"
  
  # Open editor
  if [[ $files_opened -gt 0 ]]; then
    local -a editor_args
    
    # Build args for first file (with line number support)
    if [[ -n "$first_file" ]]; then
      local -a first_args
      while IFS= read -r arg; do
        first_args+=("$arg")
      done < <(_build_editor_args "$UMM_EDITOR" "$first_file" "$first_linenum")
      
      editor_args=("${first_args[@]}")
    fi
    
    # Add remaining files
    editor_args+=("${all_files[@]}")
    
    # Display message and open
    if [[ $files_opened -eq 1 ]]; then
      _success "Opening ${C_CYAN}$first_file${C_RESET} in $UMM_EDITOR"
    else
      _success "Opening ${C_CYAN}$files_opened files${C_RESET} in $UMM_EDITOR"
    fi
    $UMM_EDITOR "${editor_args[@]}"
  else
    _error "No valid files to open"
    return 1
  fi
}

# Completion
if [[ -n "${ZSH_VERSION:-}" ]]; then
  compdef _umm umm 2>/dev/null || true
  _umm() {
    local -a opts
    opts=(
      '(-h --help)'{-h,--help}'[Show help]'
      '(-v --version)'{-v,--version}'[Show version]'
      '(-p --pattern)'{-p,--pattern}'[Search pattern]:pattern:'
      '*'{-e,--exclude}'[Exclude pattern (gitignore-style glob)]:pattern:'
      '(-a --all)'{-a,--all}'[Search all files (ignore .gitignore, include hidden)]'
      '(-g --git)'{-g,--git}'[Search git repository (commits, branches, tags, etc.)]'
      '(-n --noui)'{-n,--noui}'[Non-interactive mode]'
      '(-d --max-depth)'{-d,--max-depth}'[Maximum depth]:depth:'
      '1:root path:_files -/'
    )
    _arguments $opts
  }
fi
