#!/usr/bin/env bash
# test_deploy_dry_run.sh — Simulation locale du workflow GitHub Actions deploy-demo
#
# Détecte les problèmes AVANT de lancer une vraie GitHub Action.
# Couvre ~90% des cas de défaillance rencontrés en production.
#
# Usage (depuis la racine du repo) :
#   bash scripts/test_deploy_dry_run.sh           # mode diagnostic seul
#   bash scripts/test_deploy_dry_run.sh --fix      # tente de corriger automatiquement
#
# Prérequis : Docker, Git Bash (Windows) ou bash (Linux/macOS)

set -euo pipefail

FIX_MODE=false
for arg in "$@"; do
    [[ "$arg" == "--fix" ]] && FIX_MODE=true
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
APPUSER_UID=10001
ERRORS=0
WARNINGS=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RESET='\033[0m'

pass()   { echo -e "${GREEN}  ✅ $1${RESET}"; }
fail()   { echo -e "${RED}  ❌ $1${RESET}"; ERRORS=$((ERRORS + 1)); }
warn()   { echo -e "${YELLOW}  ⚠️  $1${RESET}"; WARNINGS=$((WARNINGS + 1)); }
info()   { echo -e "${BLUE}  ℹ️  $1${RESET}"; }
header() { echo -e "\n${BLUE}=== $1 ===${RESET}"; }

echo "======================================"
echo " LevelUp — Deploy Dry-Run (local)"
echo " Repo : ${REPO_ROOT}"
echo " Mode : $(${FIX_MODE} && echo 'FIX' || echo 'diagnostic seul')"
echo "======================================"

# ──────────────────────────────────────────────────────────────
# [1/7] Pré-requis système
# ──────────────────────────────────────────────────────────────
header "[1/7] Pré-requis système"

if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    pass "Docker disponible : $(docker --version 2>&1 | head -1)"
else
    fail "Docker non disponible ou daemon non démarré"
    info "Démarrer Docker Desktop (Windows) puis relancer ce script"
    exit 1
fi

if docker compose version &>/dev/null 2>&1; then
    pass "docker compose plugin v2 disponible"
elif command -v docker-compose &>/dev/null; then
    warn "docker-compose v1 trouvé — le workflow utilise 'docker compose' (v2)"
else
    fail "docker compose non trouvé"
fi

if command -v python3 &>/dev/null || command -v python &>/dev/null; then
    PY=$(command -v python3 || command -v python)
    pass "Python disponible : $($PY --version 2>&1)"
else
    warn "Python absent — certaines validations seront ignorées"
    PY=""
fi

# ──────────────────────────────────────────────────────────────
# [2/7] Validation YAML — workflows GitHub Actions
# ──────────────────────────────────────────────────────────────
header "[2/7] Validation YAML workflows GitHub Actions"

WF_DIR="${REPO_ROOT}/.github/workflows"
if [[ -d "$WF_DIR" ]] && [[ -n "$PY" ]]; then
    yaml_errors=0
    while IFS= read -r -d '' wf; do
        bname=$(basename "$wf")
        if $PY -c "import yaml; yaml.safe_load(open('$wf'))" 2>/dev/null; then
            pass "YAML valide : $bname"
        else
            fail "YAML invalide : $bname"
            yaml_errors=$((yaml_errors + 1))
        fi
    done < <(find "$WF_DIR" -name "*.yml" -print0)
    [[ $yaml_errors -eq 0 ]] || echo -e "  ${RED}→ Corriger les YAML avant de pusher${RESET}"
else
    warn "Validation YAML ignorée (python absent ou dossier .github/workflows manquant)"
fi

# ──────────────────────────────────────────────────────────────
# [3/7] Validation docker-compose.yml
# ──────────────────────────────────────────────────────────────
header "[3/7] Validation docker-compose.yml"

# Fichiers requis pour que docker compose config ne plante pas
COMPOSE_FILE="${REPO_ROOT}/docker-compose.yml"

if [[ ! -f "$COMPOSE_FILE" ]]; then
    fail "docker-compose.yml introuvable : $COMPOSE_FILE"
else
    # Créer les fichiers manquants temporairement pour valider la config
    TEMP_ENV=false
    TEMP_SETTINGS=false
    [[ ! -f "${REPO_ROOT}/.env.local" ]] && { touch "${REPO_ROOT}/.env.local"; TEMP_ENV=true; }
    [[ ! -f "${REPO_ROOT}/app_settings.json" ]] && { echo '{}' > "${REPO_ROOT}/app_settings.json"; TEMP_SETTINGS=true; }

    if docker compose -f "$COMPOSE_FILE" config --quiet 2>/dev/null; then
        pass "docker-compose.yml valide (docker compose config)"
    else
        warn "docker compose config a échoué (peut être dû à des variables manquantes)"
    fi

    $TEMP_ENV && rm -f "${REPO_ROOT}/.env.local"
    $TEMP_SETTINGS && rm -f "${REPO_ROOT}/app_settings.json"
fi

# Vérifier que les fichiers des volumes bind-mount FICHIERS existent (pas des répertoires)
info "Vérification volumes bind-mount fichiers (racine) :"
for f in "db_profiles.json" "app_settings.json"; do
    fp="${REPO_ROOT}/${f}"
    if [[ ! -e "$fp" ]]; then
        warn "$f absent (requis au runtime — créer depuis les exemples)"
    elif [[ -d "$fp" ]]; then
        fail "$f EST UN RÉPERTOIRE — volume Docker cassé à la racine"
        if $FIX_MODE; then
            rm -rf "$fp"
            pass "Fix appliqué : rm -rf $fp"
        else
            info "Corriger avec : rm -rf ${fp}"
        fi
    else
        pass "$f est un fichier (OK)"
    fi
done

# ──────────────────────────────────────────────────────────────
# [4/7] Détection répertoires fantômes dans data/demo/
# ──────────────────────────────────────────────────────────────
header "[4/7] Répertoires fantômes dans data/demo/"
# Bug documenté : Docker crée un répertoire quand le source d'un bind-mount
# fichier n'existe pas. Cela fait échouer le prochain docker compose up avec :
# "not a directory: Are you trying to mount a directory onto a file?"

DEMO_DIR="${REPO_ROOT}/data/demo"
mkdir -p "$DEMO_DIR"

for f in "db_profiles.json" "app_settings.json"; do
    fp="${DEMO_DIR}/${f}"
    if [[ ! -e "$fp" ]]; then
        info "data/demo/${f} : absent (sera créé lors du regen)"
    elif [[ -d "$fp" ]]; then
        fail "RÉPERTOIRE FANTÔME : data/demo/${f}"
        echo -e "    ${YELLOW}Cause : Docker a créé un répertoire car le fichier n'existait pas${RESET}"
        echo -e "    ${YELLOW}Impact : docker compose up levelup-demo va échouer${RESET}"
        if $FIX_MODE; then
            rm -rf "$fp"
            pass "Fix appliqué : rm -rf data/demo/${f}"
        else
            info "Corriger avec : rm -rf ${fp}   (ou relancer avec --fix)"
        fi
    else
        pass "data/demo/${f} est un fichier (OK)"
    fi
done

# ──────────────────────────────────────────────────────────────
# [5/7] Test permissions UID 10001 (appuser Docker)
# ──────────────────────────────────────────────────────────────
header "[5/7] Test permissions UID ${APPUSER_UID} (appuser Docker)"

_test_uid_write() {
    local dir="$1"
    local label="$2"
    mkdir -p "$dir"
    # Sur Windows/Docker Desktop, les permissions POSIX ne s'appliquent pas
    # de la même façon. Ce test est surtout utile en Linux/VPS.
    if docker run --rm -u "${APPUSER_UID}" \
        -v "$(cd "$dir" && pwd):/tmp/testdir:rw" \
        alpine sh -c "touch /tmp/testdir/.__uid_test__ 2>/dev/null && rm -f /tmp/testdir/.__uid_test__" \
        2>/dev/null; then
        pass "UID ${APPUSER_UID} peut écrire dans $label"
        return 0
    else
        fail "UID ${APPUSER_UID} NE PEUT PAS écrire dans $label"
        info "Fix (sur VPS) : sudo chown -R ${APPUSER_UID}:${APPUSER_UID} ${dir}"
        return 1
    fi
}

_test_uid_write "${REPO_ROOT}/data"       "data/"
_test_uid_write "${REPO_ROOT}/data/demo"  "data/demo/"
_test_uid_write "${REPO_ROOT}/data/logs"  "data/logs/"

# ──────────────────────────────────────────────────────────────
# [6/7] Simulation séquence deploy-demo (dry-run logique)
# ──────────────────────────────────────────────────────────────
header "[6/7] Simulation séquence deploy-demo (logique, sans SSH)"

info "Reproduction des étapes du job deploy-demo dans deploy.yml :"
echo ""
echo "  Étape 1 : rm -rf data/demo/warehouse data/demo/db_profiles.json ..."
echo "  Étape 2 : docker compose run --rm levelup python scripts/prepare_demo_data.py ..."
echo "  Étape 3 : docker compose stop levelup-demo"
echo "  Étape 4 : docker compose up -d levelup-demo"
echo ""

# Simuler étape 1
info "→ Étape 1 (simulation rm -rf) — vérification de l'état post-clean"
for item in "warehouse" "db_profiles.json" "app_settings.json" "players/DEMO/stats.duckdb"; do
    # Après le rm -rf, ces items ne doivent pas être des répertoires fantômes
    fp="${DEMO_DIR}/${item}"
    if [[ -d "$fp" && "$item" == *.json ]]; then
        fail "Post-rm : ${item} toujours un répertoire fantôme (rm n'a pas tout nettoyé)"
    fi
done
pass "Étape 1 : état cohérent"

# Simuler étape 2 : vérifier que le script Python est importable
info "→ Étape 2 (validation script) — vérifie que prepare_demo_data.py est syntaxiquement correct"
PREPARE_SCRIPT="${SCRIPT_DIR}/prepare_demo_data.py"
if [[ -f "$PREPARE_SCRIPT" ]] && [[ -n "$PY" ]]; then
    if $PY -m py_compile "$PREPARE_SCRIPT" 2>/dev/null; then
        pass "prepare_demo_data.py : syntaxe Python OK"
    else
        fail "prepare_demo_data.py : erreur de syntaxe Python"
    fi
else
    warn "prepare_demo_data.py introuvable ou Python absent — validation ignorée"
fi

# Simuler étape 4 : vérifier que docker compose config pour levelup-demo passerait
info "→ Étape 4 (validation config levelup-demo) — vérifie les volumes"
TEMP_ENV2=false
TEMP_PROFILES=false
[[ ! -f "${REPO_ROOT}/.env.local" ]] && { touch "${REPO_ROOT}/.env.local"; TEMP_ENV2=true; }
[[ ! -f "${REPO_ROOT}/db_profiles.json" ]] && {
    echo '{"version":"2.1","warehouse_path":"data/warehouse","profiles":{}}' > "${REPO_ROOT}/db_profiles.json"
    TEMP_PROFILES=true
}

if docker compose -f "$COMPOSE_FILE" config --quiet 2>/dev/null; then
    pass "Configuration levelup-demo valide dans docker-compose.yml"
else
    warn "docker compose config levelup-demo — vérifier les variables d'environnement"
fi

$TEMP_ENV2     && rm -f "${REPO_ROOT}/.env.local"
$TEMP_PROFILES && rm -f "${REPO_ROOT}/db_profiles.json"

# ──────────────────────────────────────────────────────────────
# [7/7] Validation syntaxe scripts Bash
# ──────────────────────────────────────────────────────────────
header "[7/7] Validation syntaxe scripts Bash/Shell"

for sh in "${REPO_ROOT}/deploy.sh" "${REPO_ROOT}/scripts/docker_init.sh"; do
    if [[ -f "$sh" ]]; then
        bname=$(basename "$sh")
        if bash -n "$sh" 2>/dev/null; then
            pass "Syntaxe bash OK : $bname"
        else
            fail "Erreur syntaxe bash : $bname"
        fi
    fi
done

# ──────────────────────────────────────────────────────────────
# Résumé
# ──────────────────────────────────────────────────────────────
echo ""
echo "======================================"
echo " Résumé"
echo "======================================"

if [[ $ERRORS -eq 0 && $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}✅ Tout est OK — le déploiement devrait fonctionner${RESET}"
elif [[ $ERRORS -eq 0 ]]; then
    echo -e "${YELLOW}⚠️  ${WARNINGS} avertissement(s) — vérifier avant déploiement${RESET}"
else
    echo -e "${RED}❌ ${ERRORS} erreur(s) + ${WARNINGS} avertissement(s) — corriger avant de pusher${RESET}"
    if ! $FIX_MODE; then
        echo ""
        echo -e "  ${BLUE}Relancer avec --fix pour corriger automatiquement :${RESET}"
        echo -e "  ${BLUE}  bash scripts/test_deploy_dry_run.sh --fix${RESET}"
    fi
fi

echo ""
exit $ERRORS
