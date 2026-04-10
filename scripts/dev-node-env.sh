#!/usr/bin/env bash
# Shared by run.sh and scripts/fe-dev.sh: put npm on PATH in non-interactive / IDE shells.
# nvm: sourcing nvm.sh alone does not activate a version; nvm use is required.

ensure_npm_on_path() {
  command -v npm >/dev/null 2>&1 && return 0

  local nvm_sh="${NVM_DIR:-$HOME/.nvm}/nvm.sh"
  if [[ -s "$nvm_sh" ]]; then
    # shellcheck source=/dev/null
    source "$nvm_sh"
    # Project targets Node 22; override with RUN_SH_NVM_NODE_VERSION (e.g. 20) if needed.
    local _nv="${RUN_SH_NVM_NODE_VERSION:-22}"
    nvm use "$_nv" 2>/dev/null || nvm use default 2>/dev/null || nvm use --lts 2>/dev/null || nvm use node 2>/dev/null || true
  fi
  command -v npm >/dev/null 2>&1 && return 0

  if command -v mise >/dev/null 2>&1; then
    eval "$(mise activate bash 2>/dev/null)" || eval "$(mise env -s bash 2>/dev/null)" || true
  fi
  command -v npm >/dev/null 2>&1 && return 0

  if [[ -f "$HOME/.asdf/asdf.sh" ]]; then
    # shellcheck source=/dev/null
    source "$HOME/.asdf/asdf.sh"
  fi
  command -v npm >/dev/null 2>&1 && return 0

  if command -v fnm >/dev/null 2>&1; then
    eval "$(fnm env 2>/dev/null)" || true
  fi
  command -v npm >/dev/null 2>&1 && return 0

  # Volta / common install locations (no manager hook ran)
  local volta="$HOME/.volta/bin"
  [[ -x "$volta/npm" ]] && PATH="$volta:$PATH" && export PATH && return 0

  return 1
}
