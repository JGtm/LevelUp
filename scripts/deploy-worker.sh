#!/usr/bin/env bash
# deploy-worker.sh — Mise a jour de l'OUVRIER de rejeu 2D sur le VPS de calcul (csstat)
#
# Usage (depuis le VPS de calcul) :
#   /opt/levelup/scripts/deploy-worker.sh
#
# Usage (depuis le poste local) :
#   ssh deploy@<VPS2_HOST> 'bash /opt/levelup/scripts/deploy-worker.sh'
#
# Declenche automatiquement par le job `deploy-worker` de .github/workflows/deploy.yml
# sur push main, APRES le deploiement du serveur web.
#
# CE QUE CE SCRIPT DEPLOIE, ET CE QU'IL NE DEPLOIE PAS. Il ne touche NI docker, NI le
# serveur web, NI aucune base : l'ouvrier est un binaire Go seul qui TIRE son travail en
# HTTPS (`POST /api/v1/internal/build-queue/claim`). Il n'a aucun token Halo, aucun acces
# a une DuckDB, aucun port entrant. Voir docs/RUNBOOK_REPLAY_WORKER.md.
#
# PRE-REQUIS SUR LA MACHINE (poses par le superviseur, hors de ce script) :
#   - compte `deploy`, proprietaire de /opt/levelup (clone https public du depot) ;
#   - toolchain Go dans /usr/local/go (le build est CGO : DuckDB est une dependance
#     transitive du module go-api) et gcc ;
#   - /etc/levelup-worker.env (root 600) portant LEVELUP_BUILD_WORKER_TOKEN ;
#   - unite systemd installee PAR LIEN (cf. bloc « unite » plus bas) ;
#   - sudoers restreint a `systemctl daemon-reload` et `systemctl restart levelup-worker`.
#
# PHASE PRE-ACTIVATION : tant que l'unite n'est pas activee, ce script met le binaire a
# jour et SORT EN 0 sans rien redemarrer. C'est le chemin NOMINAL, pas une erreur.
#
# NB (meme piege que scripts/deploy.sh) : une modification de CE script ne prend effet
# qu'au deploiement suivant — bash retient l'ancien inode pendant que le git reset
# remplace le fichier.

set -euo pipefail

REPO_DIR="/opt/levelup"
BIN_DIR="$REPO_DIR/bin"
BIN_PATH="$BIN_DIR/replay-worker"
UNIT_NAME="levelup-worker"
UNIT_SRC="$REPO_DIR/packaging/systemd/${UNIT_NAME}.service"
UNIT_DST="/etc/systemd/system/${UNIT_NAME}.service"
GO_BIN_DIR="/usr/local/go/bin"

echo "[deploy-worker] Repertoire : $REPO_DIR"
cd "$REPO_DIR"

# 0. Environnement de build. Une session SSH non interactive n'herite PAS du PATH d'un
# shell de login : sans cette ligne, `go` est introuvable alors qu'il est installe (Go
# vit dans /usr/local/go/bin, hors du PATH par defaut de Debian). HOME est requis par le
# cache de compilation Go (« failed to initialize build cache: $HOME is not defined »).
export PATH="$GO_BIN_DIR:$PATH"
export HOME="${HOME:-/home/deploy}"
# CGO explicite : le module go-api tire DuckDB (cgo). Un CGO_ENABLED=0 herite de
# l'environnement ferait echouer le lien avec un message obscur.
export CGO_ENABLED=1

if ! command -v go >/dev/null 2>&1; then
    echo "[deploy-worker] ERREUR: toolchain Go introuvable (attendue dans $GO_BIN_DIR)."
    echo "[deploy-worker]    Le build de l'ouvrier est CGO — installer Go puis relancer."
    exit 1
fi
echo "[deploy-worker] $(go version)"

# 1. Recuperer origin/main (force, ignore les changements locaux). Ce clone ne sert QU'A
# construire l'ouvrier : il ne porte ni donnees, ni configuration locale a preserver.
# Pas de `git clean` ici, contrairement a scripts/deploy.sh : rien a nettoyer, et une
# suppression recursive sur une machine de calcul ne se justifie pas sans motif.
sha_avant="$(git rev-parse HEAD)"
echo "[deploy-worker] git fetch + reset --hard origin/main..."
git fetch origin main
git reset --hard origin/main
sha_apres="$(git rev-parse HEAD)"

if [[ "$sha_avant" == "$sha_apres" ]]; then
    echo "[deploy-worker] Depot deja a jour ($sha_apres)"
fi

# 1b. L'unite systemd versionnee a-t-elle change dans ce lot de commits ? La reponse
# decide du `daemon-reload` plus bas. Calcule ICI, tant que les deux sha sont connus.
unite_changee=0
if [[ "$sha_avant" != "$sha_apres" ]] \
    && ! git diff --quiet "$sha_avant" "$sha_apres" -- "packaging/systemd/${UNIT_NAME}.service"; then
    unite_changee=1
fi

# 2. Construire le binaire. REMPLACEMENT ATOMIQUE : on compile vers un fichier temporaire
# DU MEME REPERTOIRE (donc du meme systeme de fichiers, condition d'un rename atomique),
# puis on bascule par `mv`. Deux proprietes recherchees :
#   - un build en echec ne laisse jamais un binaire tronque en place (l'ancien tourne
#     encore, et il tournera toujours si l'on ne redemarre pas) ;
#   - le rename ne perturbe pas le processus en cours d'execution, qui garde son inode
#     ouvert jusqu'au restart explicite de l'etape 4.
mkdir -p "$BIN_DIR"
BIN_TMP="$BIN_DIR/.replay-worker.new.$$"
# shellcheck disable=SC2064  # expansion voulue MAINTENANT : le trap doit porter ce chemin.
trap "rm -f '$BIN_TMP'" EXIT

echo "[deploy-worker] go build ./cmd/replay-worker (CGO_ENABLED=1)..."
# Le module vit dans apps/go-api (go.mod) : le paquet se designe depuis la racine du
# module, jamais depuis la racine du depot.
if ! (cd "$REPO_DIR/apps/go-api" && go build -o "$BIN_TMP" ./cmd/replay-worker); then
    echo "[deploy-worker] ERREUR: build de l'ouvrier en echec — binaire en place NON touche"
    echo "[deploy-worker]    Corriger la cause (code, disque, memoire) puis relancer ce script"
    exit 1
fi

mv -f "$BIN_TMP" "$BIN_PATH"
chmod 0755 "$BIN_PATH"
echo "[deploy-worker] Binaire installe : $BIN_PATH ($(stat -c %s "$BIN_PATH") octets)"

# 3. Unite systemd. LE COMPTE `deploy` NE PEUT PAS RECOPIER L'UNITE (sudoers restreint a
# daemon-reload et restart) : l'installation nominale est donc un LIEN de
# /etc/systemd/system/levelup-worker.service vers le fichier VERSIONNE. Le git reset
# ci-dessus met alors a jour le contenu pointe, et un daemon-reload suffit a le prendre
# en compte. Les trois etats reels sont traites, aucun ne fait echouer le deploiement :
# le binaire est deja a jour a ce stade, et une unite mal installee est un probleme
# d'operateur a signaler, pas une raison de rendre le job rouge.
if [[ ! -e "$UNIT_DST" ]]; then
    echo "[deploy-worker] Unite systemd non installee ($UNIT_DST) — phase pre-activation."
    echo "[deploy-worker]    Installation (une seule fois, par le superviseur) :"
    echo "[deploy-worker]      sudo systemctl link $UNIT_SRC"
elif [[ -L "$UNIT_DST" ]]; then
    if [[ "$unite_changee" -eq 1 ]]; then
        echo "[deploy-worker] Unite versionnee modifiee — systemctl daemon-reload..."
        sudo systemctl daemon-reload
    else
        echo "[deploy-worker] Unite inchangee dans ce lot — pas de daemon-reload"
    fi
elif cmp -s "$UNIT_SRC" "$UNIT_DST"; then
    echo "[deploy-worker] Unite installee en COPIE, conforme au depot (lien recommande)"
else
    echo "[deploy-worker] AVERTISSEMENT: $UNIT_DST est une COPIE qui DIVERGE du depot."
    echo "[deploy-worker]    Ce script ne peut pas la recopier (sudoers restreint)."
    echo "[deploy-worker]    Corriger a la main, en root :"
    echo "[deploy-worker]      rm $UNIT_DST && systemctl link $UNIT_SRC && systemctl daemon-reload"
fi

# 4. Redemarrage — CONDITIONNEL, et l'etat « pas encore active » est NOMINAL.
# On compare la chaine exacte a « enabled » plutot que de se fier au code de sortie :
# `systemctl is-enabled` rend 0 pour plusieurs etats (static, indirect, alias...) et 1
# pour « linked » comme pour « disabled ». Une unite seulement LIEE (etat de la phase
# pre-activation voulue par ce lot) ne doit surtout pas etre demarree par un deploiement.
etat_unite="$(systemctl is-enabled "$UNIT_NAME" 2>/dev/null || true)"
if [[ "$etat_unite" != "enabled" ]]; then
    echo "[deploy-worker] Service desactive (etat: ${etat_unite:-absent}) : binaire mis a jour, pas de restart"
    echo "[deploy-worker]    Activation prevue a la release v7.5 — cf. docs/RUNBOOK_REPLAY_WORKER.md"
    echo "[deploy-worker] Termine : $(git log -1 --oneline)"
    exit 0
fi

echo "[deploy-worker] Service actif — systemctl restart $UNIT_NAME..."
sudo systemctl restart "$UNIT_NAME"

# Verification post-restart. Un ouvrier qui ne demarre pas doit rendre le deploiement
# ROUGE : sans ce controle, une configuration manquante (exit 2 : jeton ou racine du
# depot absents) laisserait un job vert et une file que plus personne ne vide.
if systemctl is-active --quiet "$UNIT_NAME"; then
    echo "[deploy-worker] Ouvrier actif : $(git log -1 --oneline)"
else
    echo "[deploy-worker] ERREUR: $UNIT_NAME n'est pas actif apres redemarrage"
    echo "[deploy-worker]    Journal : journalctl -u $UNIT_NAME -n 50 --no-pager"
    echo "[deploy-worker]    Exit 2 = configuration manquante (jeton /etc/levelup-worker.env"
    echo "[deploy-worker]    ou racine du depot) ; exit 3 = plafond memoire atteint sur un film."
    exit 1
fi
