#!/usr/bin/env bash
# check_vps_deploy.sh — Diagnostic permissions/état VPS avant déploiement LevelUp
#
# À copier/exécuter directement sur le VPS :
#   bash /opt/levelup/scripts/check_vps_deploy.sh
#   bash /opt/levelup/scripts/check_vps_deploy.sh /opt/levelup
#
# Peut aussi être lancé via SSH depuis le poste local :
#   ssh deploy@<VPS_HOST> "bash /opt/levelup/scripts/check_vps_deploy.sh"

set -euo pipefail

DEPLOY_DIR="${1:-/opt/levelup}"
DATA_DIR="${DEPLOY_DIR}/data"
APPUSER_UID=10001

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RESET='\033[0m'

pass() { echo -e "${GREEN}  ✅ $1${RESET}"; }
fail() { echo -e "${RED}  ❌ $1${RESET}"; ERRORS=$((ERRORS + 1)); }
warn() { echo -e "${YELLOW}  ⚠️  $1${RESET}"; WARNINGS=$((WARNINGS + 1)); }
info() { echo -e "${BLUE}  ℹ️  $1${RESET}"; }
header() { echo -e "\n${BLUE}=== $1 ===${RESET}"; }

ERRORS=0
WARNINGS=0

echo "======================================"
echo " LevelUp — Diagnostic VPS Déploiement"
echo " Cible : ${DEPLOY_DIR}"
echo " Date  : $(date '+%Y-%m-%d %H:%M:%S')"
echo "======================================"

# === [1/6] Répertoire de déploiement ===
header "[1/6] Répertoire de déploiement"

if [[ -d "$DEPLOY_DIR" ]]; then
    pass "DEPLOY_DIR existe : $DEPLOY_DIR"
    owner=$(stat -c '%u:%g' "$DEPLOY_DIR" 2>/dev/null || stat -f '%u:%g' "$DEPLOY_DIR")
    perms=$(stat -c '%a' "$DEPLOY_DIR" 2>/dev/null || stat -f '%OLp' "$DEPLOY_DIR")
    info "Propriétaire : $owner — Permissions : $perms"
else
    fail "DEPLOY_DIR introuvable : $DEPLOY_DIR"
    echo "  Vérifier que le repo est cloné dans $DEPLOY_DIR"
    exit 1
fi

# === [2/6] Permissions répertoires data/ ===
header "[2/6] Propriétaires et permissions data/"

for dir in "$DATA_DIR" "$DATA_DIR/logs" "$DATA_DIR/demo" "$DATA_DIR/warehouse" "$DATA_DIR/players"; do
    if [[ -d "$dir" ]]; then
        owner=$(stat -c '%u:%g' "$dir" 2>/dev/null || stat -f '%u:%g' "$dir")
        perms=$(stat -c '%a' "$dir" 2>/dev/null || stat -f '%OLp' "$dir")
        uid=$(echo "$owner" | cut -d: -f1)
        label="${dir#"$DEPLOY_DIR/"}"
        if [[ "$uid" == "$APPUSER_UID" ]]; then
            pass "$label — owner ${owner}, perms ${perms} (UID ${APPUSER_UID} : OK)"
        else
            warn "$label — owner ${owner}, perms ${perms} (PAS UID ${APPUSER_UID} — risque permission)"
        fi
    else
        warn "Absent : ${dir#"$DEPLOY_DIR/"} (sera créé au prochain déploiement)"
    fi
done

# === [3/6] Test écriture UID 10001 via Docker ===
header "[3/6] Test écriture UID ${APPUSER_UID} dans data/ et sous-répertoires (via Docker)"

_test_uid_write() {
    local dir="$1"
    local label="$2"
    if [[ ! -d "$dir" ]]; then
        warn "$label : répertoire absent — impossible de tester"
        return
    fi
    if docker run --rm -u "${APPUSER_UID}" \
        -v "${dir}:/tmp/testdir:rw" \
        alpine sh -c "touch /tmp/testdir/.__uid_write_test__ && rm /tmp/testdir/.__uid_write_test__" \
        2>/dev/null; then
        pass "UID ${APPUSER_UID} peut écrire dans $label"
    else
        fail "UID ${APPUSER_UID} NE PEUT PAS écrire dans $label"
        echo -e "    ${YELLOW}→ Fix : sudo chown -R ${APPUSER_UID}:${APPUSER_UID} ${dir}${RESET}"
    fi
}

if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    mkdir -p "${DATA_DIR}/demo" "${DATA_DIR}/logs" "${DATA_DIR}/warehouse"
    _test_uid_write "$DATA_DIR"           "data/"
    _test_uid_write "$DATA_DIR/demo"      "data/demo/"
    _test_uid_write "$DATA_DIR/logs"      "data/logs/"
    _test_uid_write "$DATA_DIR/warehouse" "data/warehouse/"
else
    warn "Docker non disponible — test d'écriture ignoré"
    info "Vérifier manuellement : ls -la ${DATA_DIR}"
fi

# === [4/6] Détection répertoires fantômes (bind-mount Docker) ===
header "[4/6] Détection répertoires fantômes dans data/demo/"

# Quand Docker monte un fichier inexistant en bind-mount, il crée un répertoire.
# Ces répertoires fantômes font échouer le prochain docker compose up.
DEMO_DIR="${DATA_DIR}/demo"
PHANTOM_FOUND=false

for f in "db_profiles.json" "app_settings.json"; do
    fp="${DEMO_DIR}/${f}"
    if [[ ! -e "$fp" ]]; then
        info "data/demo/${f} : absent (normal si regen jamais exécuté)"
    elif [[ -d "$fp" ]]; then
        fail "RÉPERTOIRE FANTÔME : data/demo/${f}"
        echo -e "    ${YELLOW}Cause : bind-mount Docker sur source fichier inexistante = Docker crée un répertoire${RESET}"
        echo -e "    ${YELLOW}Fix   : rm -rf ${fp}${RESET}"
        PHANTOM_FOUND=true
    else
        pass "data/demo/${f} est un fichier (OK)"
    fi
done

if $PHANTOM_FOUND; then
    echo ""
    warn "Commande de nettoyage complète :"
    echo -e "    sudo rm -rf ${DEMO_DIR}/db_profiles.json ${DEMO_DIR}/app_settings.json"
    echo -e "    sudo mkdir -p ${DEMO_DIR}"
    echo -e "    sudo chown -R ${APPUSER_UID}:${APPUSER_UID} ${DATA_DIR}"
fi

# === [5/6] État des containers ===
header "[5/6] État des containers Docker"

if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    echo "  Containers LevelUp actifs :"
    docker ps --filter "name=levelup" \
        --format "  {{.Names}}: {{.Status}} (image={{.Image}})" 2>/dev/null || \
        warn "Aucun container LevelUp en cours"
    echo ""
    echo "  Healthcheck :"
    for svc in levelup levelup-demo; do
        status=$(docker inspect --format='{{.State.Health.Status}}' "levelup-${svc}-1" 2>/dev/null || \
                 docker inspect --format='{{.State.Health.Status}}' "${svc}" 2>/dev/null || \
                 echo "n/a")
        info "  $svc : $status"
    done
else
    warn "Docker non disponible"
fi

# === [6/6] Résumé et correctif suggéré ===
header "[6/6] Résumé"

echo ""
if [[ $ERRORS -eq 0 && $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}✅ Tout est OK — le déploiement devrait fonctionner${RESET}"
elif [[ $ERRORS -eq 0 ]]; then
    echo -e "${YELLOW}⚠️  ${WARNINGS} avertissement(s) — vérifier avant déploiement${RESET}"
else
    echo -e "${RED}❌ ${ERRORS} erreur(s) critique(s) + ${WARNINGS} avertissement(s)${RESET}"
    echo ""
    echo "  Correctif recommandé (à exécuter en tant que root/sudo) :"
    echo -e "  ${BLUE}sudo chown -R ${APPUSER_UID}:${APPUSER_UID} ${DATA_DIR}${RESET}"
    echo -e "  ${BLUE}sudo rm -rf ${DEMO_DIR}/db_profiles.json ${DEMO_DIR}/app_settings.json${RESET}"
    echo -e "  ${BLUE}sudo mkdir -p ${DEMO_DIR}${RESET}"
fi

echo ""
echo "======================================"
exit $ERRORS
