  printf '  [3/3] Installing the CLI...\n' >&2
  install_dir=${OUTFIT_INSTALL_DIR:-"$HOME/.local/bin"}
  mkdir -p "$install_dir"
  install_dir=$(cd "$install_dir" && pwd -P)
  [ ! -L "$install_dir/outfit" ] || { echo 'Refusing to replace a symlink at the Outfit install path.' >&2; exit 1; }
  staged=$(mktemp "$install_dir/.outfit-XXXXXXXX")
  cp "$binary" "$staged"
  chmod 755 "$staged"
  mv -f "$staged" "$install_dir/outfit"

  # Quote paths as literal shell words, including spaces and apostrophes.
  outfit_quote_path() {
    _outfit_value=$1
    printf "'"
    while :; do
      case $_outfit_value in
        *"'"*)
          printf '%s' "${_outfit_value%%"'"*}" "'\''"
          _outfit_value=${_outfit_value#*"'"}
          ;;
        *) printf "%s'" "$_outfit_value"; break ;;
      esac
    done
  }

  if [ -t 1 ] && [ "${TERM:-dumb}" != dumb ] && [ -z "${NO_COLOR+x}" ]; then
    printf '\n  \033[32m[OK] Outfit @VERSION@ installed successfully.\033[0m\n'
  else
    printf '\n  [OK] Outfit @VERSION@ installed successfully.\n'
  fi
  printf '       Binary: %s/outfit\n' "$install_dir"
  printf '       Download verified with SHA-256.\n\n'
  resolved=$(command -v outfit || true)
  if [ -n "$resolved" ] && [ "$resolved" -ef "$install_dir/outfit" ]; then
    printf '  PATH is ready. Start here:\n'
  else
    if [ -n "$resolved" ]; then
      printf '  Another Outfit is earlier on PATH: %s\n' "$resolved"
    fi
    quoted_dir=$(outfit_quote_path "$install_dir")
    login_shell=${SHELL:-sh}
    login_shell=${login_shell##*/}
    case $login_shell in
      fish)
        printf '  Make Outfit available in fish now and in future terminals:\n\n'
        printf '    fish_add_path --move %s\n\n' "$quoted_dir"
        ;;
      *)
        path_line=$(printf 'export PATH=%s:"$PATH"' "$quoted_dir")
        printf '  Enable Outfit in your terminal:\n\n    %s\n\n' "$path_line"
        case $login_shell in
          zsh) startup_file=${ZDOTDIR:-$HOME}/.zshrc ;;
          bash)
            if [ "$os" = darwin ]; then startup_file=$HOME/.bash_profile
            else startup_file=$HOME/.bashrc; fi
            ;;
          sh|dash|ksh) startup_file=$HOME/.profile ;;
          *) startup_file='' ;;
        esac
        if [ -n "$startup_file" ]; then
          printf '  For future terminals, add that line to %s.\n' "$(outfit_quote_path "$startup_file")"
        else
          printf '  This export command is for POSIX-compatible shells. Add it to your shell startup file.\n'
        fi
        printf '  Your shell files have not been changed.\n\n'
        ;;
    esac
    printf '  Then start here:\n'
  fi
  printf '\n    outfit version      # Check the CLI\n'
  printf '    outfit setup        # Choose a profile and review software\n'
  printf '    outfit doctor       # Check tooling and credential setup\n\n'
  printf '  Dotfiles are separate: review with outfit dotfiles before applying.\n'
  printf '  Help and profiles: https://getoutfit.dev/\n'
