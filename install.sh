#!/usr/bin/env bash
# Build the skills command from this checkout (or a fresh clone) and install
# it to ~/.local/bin. The checkout path is baked into the binary so `skills`
# finds its skills/ directory from anywhere.
set -euo pipefail

die() { printf 'Install failed: %s\n' "$*" >&2; exit 1; }

command -v go >/dev/null 2>&1 || die 'Go is required. Install Go, then run this script again.'
: "${HOME:?HOME must be set}"

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

# A checkout installs itself. A downloaded or piped script uses
# $SKILLS_REPO or ~/src/skills, cloning it when absent and pulling otherwise,
# because the installed links must point at a checkout that stays put.
repo=''
if [[ -n ${BASH_SOURCE[0]:-} ]]; then
  script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
  if [[ -f "$script_dir/go.mod" && -f "$script_dir/main.go" && -d "$script_dir/skills" ]]; then
    repo=$script_dir
  fi
fi
if [[ -z "$repo" ]]; then
  command -v git >/dev/null 2>&1 || die 'Git is required to download the source.'
  repo=${SKILLS_REPO:-$HOME/src/skills}
  if [[ -d "$repo/.git" ]]; then
    printf 'Updating %s…\n' "$repo"
    git -C "$repo" pull --ff-only --quiet || die "could not fast-forward $repo; pull it by hand and retry"
  else
    printf 'Cloning into %s…\n' "$repo"
    mkdir -p "$(dirname -- "$repo")"
    git clone --quiet https://github.com/jorden-parker/skills.git "$repo"
  fi
fi

printf 'Building skills from %s…\n' "$repo"
(
  cd "$repo"
  go build -ldflags "-X github.com/jorden-parker/skills/internal/skills.DefaultDir=$repo" -o "$work_dir/skills" .
)

install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"
install -m 755 "$work_dir/skills" "$install_dir/skills"

add_path() {
  local config=$1 line=$2
  mkdir -p "$(dirname -- "$config")"
  if ! grep -Fqx -- "$line" "$config" 2>/dev/null; then
    printf '\n# Added by skills installer\n%s\n' "$line" >> "$config"
  fi
  printf 'PATH configured in %s\n' "$config"
}

path_line='case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) export PATH="$HOME/.local/bin:$PATH" ;; esac'
user_shell=${SHELL:-sh}
case "${user_shell##*/}" in
  zsh)
    add_path "${ZDOTDIR:-$HOME}/.zshrc" "$path_line"
    ;;
  bash)
    add_path "$HOME/.bashrc" "$path_line"
    # Bash reads only the first existing login profile.
    profile="$HOME/.profile"
    if [[ -f "$HOME/.bash_profile" ]]; then
      profile="$HOME/.bash_profile"
    elif [[ -f "$HOME/.bash_login" ]]; then
      profile="$HOME/.bash_login"
    fi
    add_path "$profile" "$path_line"
    ;;
  fish)
    add_path "${XDG_CONFIG_HOME:-$HOME/.config}/fish/conf.d/skills.fish" \
      'fish_add_path "$HOME/.local/bin"'
    ;;
  *)
    add_path "$HOME/.profile" "$path_line"
    printf 'For other shells, add ~/.local/bin to PATH in your shell configuration.\n'
    ;;
esac

printf '\nInstalled skills in %s, using the checkout at %s\n' "$install_dir" "$repo"
printf 'Open a new terminal, then run: skills --help\n'
printf 'Or run now: "%s/skills" --help\n' "$install_dir"
