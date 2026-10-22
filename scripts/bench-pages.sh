#!/usr/bin/env bash
# bench-pages.sh — banc de mesure des pages sur une COPIE des bases de prod (plan
# .ai/V7.5/PLAN_PERF_PAGES_V801_2026-10-21.md, étape 1).
#
# Isolation (décision D-5 du plan) :
#   - serveur du worktree sur le port 8010, jamais celui des autres sessions (8000) ;
#   - copie des bases de prod par flux ssh en lecture seule (`fetch`), rien n'est écrit sur le VPS ;
#   - AUCUN fichier de data/auth (jetons) : LEVELUP_AUTH_DIR pointe sur un dossier vide ;
#   - app_settings.json assaini (webhook Discord vidé, notifications, watcher, auto-sync,
#     surveillance médias et sauvegardes coupés) ;
#   - HTTP(S)_PROXY vers un port fermé : toute tentative de sortie échoue (127.0.0.1 exclu) ;
#   - réglages DuckDB de prod (défauts du code : 2 fils, 512 Mo).
#
# Usage (depuis la racine du worktree) :
#   scripts/bench-pages.sh fetch     copie les bases de prod (liste : scripts/bench-pages.files)
#   scripts/bench-pages.sh serve     lance le serveur de banc (tmp/bench-server.exe) et attend /health
#   scripts/bench-pages.sh pass      serveur redémarré à neuf + mesure + mémoire + arrêt (usage normal)
#   scripts/bench-pages.sh measure   mesure les pages, écrit tmp/bench-results/<horodatage>.tsv
#   scripts/bench-pages.sh check     contrôles d'isolation (auth absent, sorties, fichiers suivis)
#   scripts/bench-pages.sh stop      arrête le serveur de banc
#   scripts/bench-pages.sh clean     supprime les copies listées et tmp/bench-* (jamais data/ entier)
#
# Prérequis : tmp/bench-server.exe construit depuis apps/go-api (`go build -o ../../tmp/bench-server.exe
# ./cmd/server`, CGO), `ssh lvelup`, jq, curl.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
PORT=8010
BASE="http://127.0.0.1:${PORT}/api/v1"
FILES="scripts/bench-pages.files"
PROD=/opt/levelup
RUNS=5
PIDFILE=tmp/bench-server.pid
LOGDIR=tmp/bench-logs
AUTHDIR=tmp/bench-auth-empty

# Comptes et match de référence (un match avec artefact de rejeu par compte, 2026-10-21).
PLAYERS=(JGtm Nuzzles Madina97294)
declare -A MATCH=(
  [JGtm]=8ba9251a-53f3-4367-aa55-9c5df983120f
  [Nuzzles]=37c6e818-fc07-4b2e-90ed-f2f84c904933
  [Madina97294]=a853258c-3027-4144-85c5-7309b9dc0031
)

copies() { grep -v '^#' "$FILES" | grep -v '^[[:space:]]*$'; }

cmd_fetch() {
  [ -d data/auth ] && { echo "data/auth présent dans le worktree : refus" >&2; exit 1; }
  local avail
  avail=$(df -Pk . | awk 'NR==2 {print $4}')
  [ "$avail" -ge $((10 * 1024 * 1024)) ] || { echo "moins de 10 Go libres : refus" >&2; exit 1; }
  local data_paths=()
  while read -r p; do
    case "$p" in
      data/*) data_paths+=("${p#data/}") ;;
    esac
  done < <(copies)
  # Les rejeux : seuls les artefacts des matchs de référence, jamais tout le dossier (5 Go).
  local replay_files=()
  local m
  for m in "${MATCH[@]}"; do
    replay_files+=("cache/replays/halo_infinite/${m:0:8}.json" "cache/replays/halo_infinite/${m:0:8}.derived.json")
  done
  local tar_paths=()
  for p in "${data_paths[@]}"; do
    [ "$p" = "cache/replays" ] && continue
    [ "$p" = "titles/halo_infinite/warehouse/snapshots" ] && continue
    tar_paths+=("$p")
  done
  echo "copie depuis la prod ($(date +%H:%M:%S))"
  # shellcheck disable=SC2029
  ssh lvelup "cd $PROD/data && tar -cf - ${tar_paths[*]} ${replay_files[*]} 2>/dev/null" | tar -xf - -C data
  local snap
  snap=$(ssh lvelup "cat $PROD/data/titles/halo_infinite/warehouse/snapshots/CURRENT.json" | jq -r '.manifest_rel' | cut -d/ -f1 | cut -d'\' -f1)
  ssh lvelup "cd $PROD/data/titles/halo_infinite/warehouse && tar -cf - snapshots/CURRENT.json snapshots/$snap" \
    | tar -xf - -C data/titles/halo_infinite/warehouse
  ssh lvelup "cat $PROD/app_settings.json" | sed -E \
    -e 's/("discord_webhook_url"[[:space:]]*:[[:space:]]*)"[^"]*"/\1""/' \
    -e 's/("media_captures_base_dir"[[:space:]]*:[[:space:]]*)"[^"]*"/\1""/' \
    -e 's/("(discord_notifications_enabled|discord_notify_[a-z_]+|watcher_presence_enabled|spnkr_auto_sync_enabled|media_watcher_enabled|backup_enabled|tailscale_enabled)"[[:space:]]*:[[:space:]]*)true/\1false/g' \
    > app_settings.json
  ssh lvelup "cat $PROD/db_profiles.json" \
    | jq '.profiles.halo_infinite |= with_entries(select(.key=="JGtm" or .key=="Nuzzles" or .key=="Madina97294"))' \
    > db_profiles.json
  echo "copie terminée ($(date +%H:%M:%S))"
}

cmd_serve() {
  [ -x tmp/bench-server.exe ] || { echo "tmp/bench-server.exe absent" >&2; exit 1; }
  [ -d data/auth ] && { echo "data/auth présent : refus" >&2; exit 1; }
  mkdir -p "$LOGDIR" "$AUTHDIR"
  LEVELUP_REPO_ROOT="$ROOT" \
  LEVELUP_API_PORT=$PORT \
  LEVELUP_AUTH_MODE=none \
  LEVELUP_AUTH_DIR="$ROOT/$AUTHDIR" \
  LEVELUP_DUCKDB_THREADS=2 \
  LEVELUP_DUCKDB_MEMORY_LIMIT=512MB \
  LEVELUP_SLOW_REQUEST_MS=1 \
  LEVELUP_LOGS_DIR="$ROOT/$LOGDIR" \
  HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 NO_PROXY=127.0.0.1,localhost \
    tmp/bench-server.exe > "$LOGDIR/stdout.log" 2>&1 &
  echo $! > "$PIDFILE"
  local i
  for i in $(seq 1 120); do
    if curl -sf "http://127.0.0.1:${PORT}/health" > /dev/null; then
      echo "serveur de banc prêt (pid $(cat "$PIDFILE"), ${i} s)"
      return 0
    fi
    sleep 1
  done
  echo "serveur de banc non prêt après 120 s (voir $LOGDIR/stdout.log)" >&2
  exit 1
}

# mesure <compte> <libellé> <méthode> <chemin> [corps JSON]
mesure() {
  local player=$1 label=$2 method=$3 path=$4 body=${5:-}
  local args=(-s -o /dev/null -w '%{http_code} %{time_total} %{size_download}' -X "$method" "$BASE$path")
  # Origine admise par le CSRF par origine (serveur de dev : vite sur 5173).
  [ -n "$body" ] && args+=(-H 'Origin: http://localhost:5173' -H 'Content-Type: application/json' --data "$body")
  curl "${args[@]}" > /dev/null || true # chauffe
  local times=() sizes=() code=""
  local i out
  for i in $(seq 1 $RUNS); do
    out=$(curl "${args[@]}" || echo "000 0 0")
    code=${out%% *}
    times+=("$(echo "$out" | awk '{print $2}')")
    sizes+=("$(echo "$out" | awk '{print $3}')")
  done
  local med max
  med=$(printf '%s\n' "${times[@]}" | sort -g | awk '{a[NR]=$1} END {print a[int((NR+1)/2)]}')
  max=$(printf '%s\n' "${times[@]}" | sort -g | tail -1)
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$player" "$label" "$code" "$med" "$max" "${sizes[-1]}"
}

cmd_measure() {
  mkdir -p tmp/bench-results
  local out
  out="tmp/bench-results/$(date +%Y%m%d-%H%M%S).tsv"
  printf 'compte\tpage\tcode\tmediane_s\tmax_s\toctets\n' > "$out"
  local p m
  for p in "${PLAYERS[@]}"; do
    m=${MATCH[$p]}
    {
      mesure "$p" accueil GET "/players/$p/pages/home"
      mesure "$p" vue_match GET "/players/$p/matches/$m"
      mesure "$p" rejeu_artefact GET "/players/$p/matches/$m/replay"
      mesure "$p" rejeu_contexte GET "/players/$p/matches/$m/replay/context"
      mesure "$p" explorer POST "/players/$p/pages/explorer/matches-query" \
        '{"filters":{},"pagination":{"page":1,"page_size":10000},"include_export_hint":true,"include_briefing":true}'
      mesure "$p" relations POST "/players/$p/pages/palmares/relations" '{}'
      mesure "$p" carriere GET "/players/$p/pages/career"
      mesure "$p" carriere_rencontres GET "/players/$p/pages/career/top-encounters"
      mesure "$p" carriere_rivaux GET "/players/$p/pages/career/rivals"
      mesure "$p" timeseries_tout POST "/players/$p/pages/timeseries" '{}'
      mesure "$p" escouade_tout POST "/players/$p/pages/teammates" '{}'
    } | tee -a "$out"
  done
  mesure JGtm comparaison POST "/players/JGtm/pages/compare" '{"target_gamertag":"Madina97294"}' | tee -a "$out"
  echo "résultats : $out"
  grep -h '"http_timings"' "$LOGDIR"/*.log 2>/dev/null | tail -n 200 > "${out%.tsv}.timings.jsonl" || true
  echo "sections : ${out%.tsv}.timings.jsonl"
}

cmd_check() {
  local ok=0
  if [ -d data/auth ]; then echo "ÉCHEC : data/auth présent"; ok=1; else echo "ok : data/auth absent"; fi
  # La migration du groupe par défaut écrit groups.json au boot (sans secret) ; aucun jeton ne doit y être.
  if ls "$AUTHDIR"/watcher_tokens* > /dev/null 2>&1; then
    echo "ÉCHEC : jeton présent dans $AUTHDIR"; ok=1
  else
    echo "ok : aucun jeton dans le dossier d'auth du banc ($(ls "$AUTHDIR" 2>/dev/null | tr '\n' ' '))"
  fi
  if git status --porcelain --untracked-files=no -- data | grep -q .; then
    echo "ÉCHEC : fichiers suivis modifiés sous data/"; git status --short --untracked-files=no -- data; ok=1
  else
    echo "ok : aucun fichier suivi modifié sous data/"
  fi
  local hook
  hook=$(jq -r '.discord_webhook_url' app_settings.json)
  if [ -n "$hook" ]; then echo "ÉCHEC : webhook non vide"; ok=1; else echo "ok : webhook vide"; fi
  echo "tentatives de sortie journalisées (bloquées par le proxy fermé) :"
  grep -hoE '"msg":"halo_api[^"]*"' "$LOGDIR"/*.log 2>/dev/null | sort | uniq -c || echo "  aucune"
  curl -s "http://127.0.0.1:${PORT}/debug/vars" | jq -c '[paths(scalars) as $p | {k: ($p | map(tostring) | join(".")), v: getpath($p)} | select(.k | test("halo_api"))]' 2>/dev/null || true
  return $ok
}

cmd_stop() {
  [ -f "$PIDFILE" ] || { echo "pas de serveur de banc"; return 0; }
  local pid
  pid=$(cat "$PIDFILE")
  kill "$pid" 2>/dev/null || true
  rm -f "$PIDFILE"
  echo "serveur de banc arrêté (pid $pid)"
}

cmd_clean() {
  cmd_stop
  local p
  while read -r p; do
    if git ls-files --error-unmatch "$p" > /dev/null 2>&1; then
      echo "refus : $p est suivi par git" >&2
      continue
    fi
    rm -rf -- "$p"
  done < <(copies)
  rm -rf tmp/bench-logs tmp/bench-auth-empty tmp/bench-results tmp/bench-server.pid tmp/bench-backup tmp/bench-*.out
  echo "copies du banc supprimées"
}

# Un passage = serveur redémarré à neuf, mesure, mémoire du processus relevée, arrêt : deux passages
# successifs partent du même état (un serveur gardé entre deux passages ralentissait de 6 à 15 % par
# passage, constaté le 2026-10-21).
cmd_pass() {
  cmd_stop
  cmd_serve
  cmd_measure
  powershell -NoProfile -Command \
    "Get-Process bench-server -ErrorAction SilentlyContinue | ForEach-Object { 'memoire privee du serveur de banc : {0} Mo' -f [int](\$_.PrivateMemorySize64/1MB) }" \
    || true
  cmd_stop
}

case "${1:-}" in
  pass) cmd_pass ;;
  fetch) cmd_fetch ;;
  serve) cmd_serve ;;
  measure) cmd_measure ;;
  check) cmd_check ;;
  stop) cmd_stop ;;
  clean) cmd_clean ;;
  *) sed -n '2,25p' "$0"; exit 2 ;;
esac
