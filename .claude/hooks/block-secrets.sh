#!/usr/bin/env bash
# PreToolUse(Edit|Write) : refuse l'écriture si le contenu contient un secret
# manifeste (clé privée, token cloud, clé API hardcodée). Exit 2 = bloque.
# Volontairement précis pour ne pas bloquer les placeholders de dev.
set -euo pipefail

input=$(cat)
file=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)
content=$(printf '%s' "$input" | jq -r '(.tool_input.content // "") + "\n" + (.tool_input.new_string // "")' 2>/dev/null || true)

# .env (hors .example) ne doit jamais être écrit dans le repo.
case "$file" in
  *.env|*/.env|*.env.local|*/.env.local)
    echo "block-secrets: refus d'écrire un fichier .env (utilise .env.example)." >&2
    exit 2
    ;;
esac

patterns=(
  '-----BEGIN [A-Z ]*PRIVATE KEY-----'   # clés privées PEM
  'AKIA[0-9A-Z]{16}'                     # AWS access key id
  'ghp_[0-9A-Za-z]{30,}'                 # GitHub PAT
  'xox[baprs]-[0-9A-Za-z-]{10,}'         # Slack token
  'sk-[0-9A-Za-z]{20,}'                  # clé style OpenAI/Stripe
)

for p in "${patterns[@]}"; do
  if printf '%s' "$content" | grep -Eq -e "$p"; then
    echo "block-secrets: secret potentiel détecté (motif: $p). Écriture bloquée." >&2
    exit 2
  fi
done
exit 0
