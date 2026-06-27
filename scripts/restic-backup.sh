#!/usr/bin/env bash
# restic-backup.sh -- sauvegarde quotidienne coherente des DuckDB + tokens + config.
#
# Strategie : arret bref du service `levelup` (fichiers DuckDB au repos = coherents,
# restaurables), snapshot restic, redemarrage des que le snapshot est fait, puis prune
# (app deja relancee). Un trap garantit le redemarrage meme si le backup plante.
#
# Perimetre : DuckDB de TOUS les titres (titles/*, incl. halo_5) + tokens OAuth
# (data/auth) + config JSON. MEDIAS EXCLUS (volumineux, peu sujets a corruption,
# disque VPS serre).
#
# Installation (VPS, one-shot) :
#   apt-get install -y restic
#   umask 077; openssl rand -base64 24 > /opt/levelup/.restic-password; chmod 600 ...
#   RESTIC_REPOSITORY=/opt/levelup/restic-repo RESTIC_PASSWORD_FILE=/opt/levelup/.restic-password restic init
#   cp scripts/restic-backup.sh /opt/levelup/scripts/ && chmod +x ...
#   cp scripts/systemd/levelup-restic-backup.* /etc/systemd/system/
#   systemctl daemon-reload && systemctl enable --now levelup-restic-backup.timer
#
# Restauration (serveur arrete) : cf. scripts/RESTIC_BACKUP.md
set -uo pipefail

export RESTIC_REPOSITORY=/opt/levelup/restic-repo
export RESTIC_PASSWORD_FILE=/opt/levelup/.restic-password
cd /opt/levelup

LOG=/opt/levelup/data/logs/restic-backup.log
exec >>"$LOG" 2>&1
echo "=== $(date -u +%FT%TZ) restic backup START ==="

restart_app() { docker compose start levelup >/dev/null 2>&1 || true; }
trap restart_app EXIT   # filet de securite : redemarre meme si plantage

# 1. Arret bref pour coherence DuckDB.
docker compose stop levelup >/dev/null 2>&1 || true

# 2. Snapshot : DuckDB de tous les titres + tokens auth + config (medias/cache/logs exclus).
rc=0
restic backup --tag auto --host levelup-vps \
  data/titles \
  data/auth \
  db_profiles.json app_settings.json .env.local || rc=$?

# 3. Redemarrage ASAP (minimise le downtime) puis on desarme le filet.
docker compose start levelup >/dev/null 2>&1 || true
trap - EXIT

# 4. Prune (app relancee, ne touche pas les fichiers live) si le backup a reussi.
if [ "$rc" -eq 0 ]; then
  restic forget --tag auto --keep-daily 7 --keep-weekly 4 --prune >/dev/null 2>&1 || true
  echo "=== $(date -u +%FT%TZ) restic backup OK ==="
else
  echo "=== $(date -u +%FT%TZ) restic backup FAILED rc=$rc ==="
fi
exit "$rc"
