#!/usr/bin/env bash
# compact-passes-nuit.sh -- compactage automatique des passes supersedees, la nuit, HORS du
# processus serveur (unite systemd levelup-compact-passes.timer, chaque nuit a 03:30 UTC).
#
# Pourquoi hors du serveur : une compaction reconstruit chaque table entiere ; dans le serveur elle
# tiendrait l'ecrivain partage le temps de la reconstruction (dizaines de secondes pour les grosses
# tables apres une recuisson) -- le site repondrait 503 -- et la memoire du VPS (1,8 Go) ne laisse
# pas de marge au serveur pendant une reconstruction fenetree. La commande `levelup compact-passes`
# (serveur arrete, sauvegarde fichier prealable) fait deja le travail de facon sure : ce script ne
# decide que QUAND la jouer.
#
# Decision : la jauge « passes par match » (data/global/admin_state/passes_par_match.txt), ecrite
# par le controle de sante des donnees du serveur (au demarrage puis toutes les 24 h). Si aucune
# table d'aucun titre ne depasse le seuil (migration.SeuilPassesParMatch, porte par le fichier), le
# script sort sans rien toucher.
#
# Sinon : verifications (verrou, sauvegarde restic, place disque), arret du serveur, compaction
# avec sauvegarde fichier, redemarrage (TOUJOURS, meme si la compaction echoue), attente de
# /health. Les sauvegardes `*.avant-compaction-*` de plus de 24 h du dossier dedie sont supprimees
# seulement si le dernier passage qui a compacte a reussi.
#
# Codes de sortie : 0 rien a faire ou compaction reussie ; 1 echec (jauge absente ou perimee,
# disque insuffisant, compaction ou redemarrage en echec) ; 75 passage refuse sans rien toucher
# (verrou tenu, sauvegarde restic en cours).
#
# Installation (VPS) : scripts/systemd/README.md. Tests : apps/go-api/internal/archlint/
# compact_passes_nuit_script_test.go (faux docker/curl/df/systemctl/flock).
set -uo pipefail

LEVELUP_DIR="${LEVELUP_DIR:-/opt/levelup}"
JAUGE="${LEVELUP_COMPACT_JAUGE:-$LEVELUP_DIR/data/global/admin_state/passes_par_match.txt}"
LOG="${LEVELUP_COMPACT_LOG:-$LEVELUP_DIR/data/logs/compact-passes-nuit.log}"
ETAT="${LEVELUP_COMPACT_ETAT:-$LEVELUP_DIR/data/logs/compact-passes-nuit.etat}"
VERROU="${LEVELUP_COMPACT_VERROU:-/run/lock/levelup-compact-passes.lock}"
HEALTH_URL="${LEVELUP_COMPACT_HEALTH_URL:-http://127.0.0.1:8000/health}"

# Dossier des sauvegardes de compaction, vu de l'hote et du conteneur (volume ./data:/app/data).
# Hors de data/titles : la sauvegarde restic quotidienne ne l'embarque pas, et le menage ne peut
# toucher aucune base.
SAUVEGARDES_HOTE="$LEVELUP_DIR/data/compaction-backups"
SAUVEGARDES_CONTENEUR="/app/data/compaction-backups"

# Jauge plus vieille que deux cycles du controle de sante (24 h) : le serveur ne la mesure plus.
JAUGE_AGE_MAX_S=$((48 * 3600))
# Marge disque exigee au-dela de la taille cumulee des bases partagees (la sauvegarde les copie).
MARGE_DISQUE_KO=$((2 * 1024 * 1024))
# Age au-dela duquel une sauvegarde de compaction est supprimee (minutes).
SAUVEGARDE_AGE_MAX_MIN=1440
# Attente maximale de /health apres redemarrage (le deploiement en accorde 90 s).
HEALTH_ATTENTE_S="${LEVELUP_COMPACT_HEALTH_ATTENTE_S:-120}"
HEALTH_INTERVALLE_S=2
# Essais de redemarrage (start + /health) et pause entre deux : le site ne reste jamais arrete sur
# un seul echec de `docker compose start`, et la boucle est bornee.
START_ESSAIS=3
START_DELAI_S="${LEVELUP_COMPACT_START_DELAI_S:-15}"
# Unite de la sauvegarde restic : elle arrete et redemarre le serveur elle aussi.
UNITE_RESTIC=levelup-restic-backup.service

SORTIE_REFUS=75

log() { echo "$(date -u +%FT%TZ) $*"; }

mkdir -p "$(dirname "$LOG")"
exec >>"$LOG" 2>&1
log "=== compact-passes-nuit START ==="

# Verrou : jamais deux passages en parallele (lance a la main pendant le timer).
exec 9>"$VERROU"
if ! flock -n 9; then
  log "REFUS : un autre passage tient le verrou $VERROU"
  exit "$SORTIE_REFUS"
fi

# L'unite restic est `Type=oneshot` sans `RemainAfterExit` : pendant la sauvegarde son etat est
# `activating` (et `is-active` rend 3). On lit donc l'etat reel.
etat_restic="$(systemctl show -p ActiveState --value "$UNITE_RESTIC" 2>/dev/null)"
case "$etat_restic" in
  activating | active | reloading | deactivating)
    log "REFUS : sauvegarde restic en cours (etat $etat_restic), elle arrete et redemarre le serveur elle aussi"
    exit "$SORTIE_REFUS"
    ;;
esac

# lire_cle CLE : la valeur de `CLE=` dans le fichier de jauge (vide si absente).
lire_cle() { sed -n "s/^$1=//p" "$JAUGE" | head -n 1; }

# menage_sauvegardes : supprime les sauvegardes de compaction de plus de 24 h, seulement si le
# dernier passage qui a compacte a reussi (sinon elles sont la seule voie de restauration).
menage_sauvegardes() {
  if [ "$(cat "$ETAT" 2>/dev/null)" != "ok" ]; then
    log "menage : dernier passage non reussi ou inconnu, sauvegardes conservees"
    return 0
  fi
  [ -d "$SAUVEGARDES_HOTE" ] || return 0
  find "$SAUVEGARDES_HOTE" -maxdepth 1 -type f -name '*.avant-compaction-*' \
    -mmin "+$SAUVEGARDE_AGE_MAX_MIN" -print -delete | sed "s/^/menage : supprimee /" \
    || log "ERREUR : menage des sauvegardes de $SAUVEGARDES_HOTE en echec"
}

# --- 1. Jauge -----------------------------------------------------------------------------------
if [ ! -f "$JAUGE" ]; then
  log "ERREUR : jauge absente ($JAUGE) -- le controle de sante du serveur ne l'a jamais publiee"
  exit 1
fi
mesure="$(lire_cle mesure_utc)"
au_dessus="$(lire_cle tables_au_dessus_du_seuil)"
seuil="$(lire_cle seuil)"
mesure_s="$(date -u -d "$mesure" +%s 2>/dev/null)" || mesure_s=""
if [ -z "$mesure_s" ] || ! [[ "$au_dessus" =~ ^[0-9]+$ ]]; then
  log "ERREUR : jauge illisible (mesure_utc='$mesure', tables_au_dessus_du_seuil='$au_dessus')"
  exit 1
fi
age_s=$(($(date -u +%s) - mesure_s))
if [ "$age_s" -gt "$JAUGE_AGE_MAX_S" ]; then
  log "ERREUR : jauge perimee (mesuree le $mesure, il y a ${age_s} s > ${JAUGE_AGE_MAX_S} s)"
  exit 1
fi
if [ "$au_dessus" -eq 0 ]; then
  log "rien a faire : aucune table au-dela du seuil $seuil (jauge du $mesure)"
  menage_sauvegardes
  log "=== compact-passes-nuit FIN (rien a faire) ==="
  exit 0
fi
log "a compacter : $au_dessus table(s) au-dela du seuil $seuil (jauge du $mesure)"
grep '^table=' "$JAUGE" | sed 's/^/jauge : /'

# --- 2. Place disque ----------------------------------------------------------------------------
bases_ko=0
for base in "$LEVELUP_DIR"/data/titles/*/warehouse/shared_matches_v2.duckdb; do
  [ -f "$base" ] || continue
  bases_ko=$((bases_ko + $(stat -c %s "$base") / 1024 + 1))
done
libre_ko="$(df -Pk "$LEVELUP_DIR/data" | awk 'NR==2 {print $4}')"
requis_ko=$((bases_ko + MARGE_DISQUE_KO))
if ! [[ "$libre_ko" =~ ^[0-9]+$ ]] || [ "$libre_ko" -lt "$requis_ko" ]; then
  log "ERREUR : place disque insuffisante (libre ${libre_ko:-?} Ko < requis $requis_ko Ko = bases $bases_ko Ko + marge $MARGE_DISQUE_KO Ko), serveur non arrete"
  exit 1
fi

# --- 3. Arret, compaction, redemarrage ----------------------------------------------------------
cd "$LEVELUP_DIR" || { log "ERREUR : dossier $LEVELUP_DIR introuvable"; exit 1; }
# serveur_arrete : 1 de l'arret jusqu'a un redemarrage REUSSI (start + /health). Le filet ne
# s'arme jamais avant l'arret et ne se desarme qu'apres un redemarrage reussi ou apres epuisement
# des essais de redemarrer (le filet ne rejoue pas les memes essais une seconde fois).
serveur_arrete=0
essais_epuises=0

# demarrer_une_fois : un `docker compose start` puis l'attente bornee de /health.
demarrer_une_fois() {
  if ! docker compose start levelup; then
    log "ERREUR : docker compose start levelup a echoue"
    return 1
  fi
  local limite=$(($(date -u +%s) + HEALTH_ATTENTE_S))
  until curl -sf "$HEALTH_URL" >/dev/null; do
    if [ "$(date -u +%s)" -ge "$limite" ]; then
      log "ERREUR : /health ne repond pas apres ${HEALTH_ATTENTE_S} s"
      return 1
    fi
    sleep "$HEALTH_INTERVALLE_S"
  done
}

# redemarrer : jusqu'a START_ESSAIS essais, START_DELAI_S entre deux. Rend 1 si tous echouent.
redemarrer() {
  local essai
  for ((essai = 1; essai <= START_ESSAIS; essai++)); do
    if demarrer_une_fois; then
      serveur_arrete=0
      log "serveur redemarre, /health OK (essai $essai/$START_ESSAIS)"
      return 0
    fi
    log "ERREUR : redemarrage, essai $essai/$START_ESSAIS en echec"
    [ "$essai" -lt "$START_ESSAIS" ] && sleep "$START_DELAI_S"
  done
  essais_epuises=1
  log "ERREUR : serveur NON redemarre apres $START_ESSAIS essais -- intervention requise"
  return 1
}
# shellcheck disable=SC2329 # invoquee par le trap EXIT ci-dessous
filet() {
  if [ "$serveur_arrete" -eq 1 ] && [ "$essais_epuises" -eq 0 ]; then
    log "filet : redemarrage du serveur"
    redemarrer
  fi
}
trap filet EXIT
# Un arret par signal (timeout systemd, Ctrl-C) passe par exit, donc par le filet.
trap 'log "signal recu, arret du script"; exit 143' TERM INT HUP

echo "echec" >"$ETAT"
serveur_arrete=1
if ! docker compose stop levelup; then
  log "ERREUR : docker compose stop levelup a echoue, compaction annulee"
  exit 1
fi
log "serveur arrete, compaction (sauvegardes dans $SAUVEGARDES_HOTE)"
rc=0
docker compose run --rm --no-deps -u appuser --entrypoint levelup levelup \
  compact-passes --backup-dir "$SAUVEGARDES_CONTENEUR" || rc=$?
if [ "$rc" -ne 0 ]; then
  log "ERREUR : compaction en echec (code $rc) -- sauvegardes conservees, voir ci-dessus"
fi

if ! redemarrer; then
  exit 1
fi
if [ "$rc" -ne 0 ]; then
  exit 1
fi
echo "ok" >"$ETAT"
menage_sauvegardes
log "=== compact-passes-nuit FIN (compaction reussie) ==="
exit 0
