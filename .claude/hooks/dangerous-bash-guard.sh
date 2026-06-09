#!/usr/bin/env bash
# PreToolUse(Bash) : bloque une poignée de commandes destructrices évidentes.
# Exit 2 = bloque et renvoie le message à l'agent. Garde volontairement étroite
# pour ne pas gêner le workflow normal (git, docker, task, go, codegraph).
set -euo pipefail

input=$(cat)
cmd=$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null || true)
[ -z "$cmd" ] && exit 0

dangerous=(
  'rm[[:space:]]+-[a-zA-Z]*r[a-zA-Z]*f?[[:space:]]+(/|~|\$HOME)([[:space:]]|$)'  # rm -rf / ~ $HOME
  ':\(\)[[:space:]]*\{[[:space:]]*:\|:'      # fork bomb
  'mkfs\.'                                    # formatage FS
  'dd[[:space:]]+if=.*of=/dev/'               # écrasement disque
  '>[[:space:]]*/dev/sd[a-z]'                 # redirection vers disque
  'chmod[[:space:]]+-R[[:space:]]+777[[:space:]]+/'
  'git[[:space:]]+push[[:space:]].*--force.*[[:space:]](origin[[:space:]]+)?main'  # force-push main
)

for p in "${dangerous[@]}"; do
  if printf '%s' "$cmd" | grep -Eq "$p"; then
    echo "dangerous-bash-guard: commande bloquée (motif: $p)." >&2
    exit 2
  fi
done
exit 0
