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

umm() {
  # Use EDITOR environment variable, default to nvim
  local UMM_EDITOR="${EDITOR:-nvim}"
  
  local root="."
  local pattern=""
  local noui=false
  local max_depth=""
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      --help|-h)
        cat << EOF
umm - Ultimate Multi-file Matcher

USAGE:
  umm [OPTIONS]

OPTIONS:
  -r, --root PATH       Search directory (default: current directory)
  -p, --pattern REGEXP  Initial search pattern
  -n, --noui            Non-interactive mode, open first match
  -d, --max-depth N     Maximum search depth
  -h, --help            Show this help
  -v, --version         Show version

ENVIRONMENT:
  EDITOR                Editor to use (default: nvim)
                        Supported: vim, vi, nvim, nano, micro, emacs,
                        code, subl, and more

EXAMPLES:
  umm                        # Interactive search
  umm -p "function"          # Search for pattern
  umm -r ~/projects          # Search in directory
  umm -p "TODO" -n           # Open first match directly
  EDITOR=micro umm           # Use micro editor
EOF
        return 0
        ;;
      --version|-v)
        echo "umm version $UMM_VERSION"
        return 0
        ;;
      --root|-r)
        if [[ -z "$2" || "$2" == -* ]]; then
          _error "Option --root requires a value"
          return 1
        fi
        root="$2"
        shift 2
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
      --noui|-n)
        noui=true
        shift
        ;;
      *)
        _error "Unknown option: $1"
        echo "Use ${C_CYAN}--help${C_RESET} for usage" >&2
        return 1
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
      '(-r --root)'{-r,--root}'[Search directory]:directory:_files -/'
      '(-p --pattern)'{-p,--pattern}'[Search pattern]:pattern:'
      '(-n --noui)'{-n,--noui}'[Non-interactive mode]'
      '(-d --max-depth)'{-d,--max-depth}'[Maximum depth]:depth:'
    )
    _arguments $opts
  }
fi
