#!/usr/bin/env bash
# Détection de secrets sur les fichiers STAGÉS (lefthook pre-commit).
#
# Pourquoi en pre-commit et pas seulement en CI : un secret commité en local puis jamais
# poussé reste dans l'historique local pour toujours, et AUCUN scan serveur (push
# protection GitHub, job CI) ne le verra jamais. Le seul moment où l'on peut encore
# empêcher un secret d'entrer dans un historique git, c'est ici. Coût : ~0,2 s (le scan
# porte sur le diff stagé, pas sur l'arbre).
#
# DÉGRADATION SI LE BINAIRE MANQUE : avertissement bruyant puis exit 0 — délibéré,
# même doctrine que shared-social-gate.sh. Un hook qui refuse tout commit sur toute
# machine où l'outil n'est pas installé se fait désinstaller (ou contourner par
# LEFTHOOK=0) en une semaine, et on perd la protection sur les machines qui l'avaient.
# L'AUTORITÉ du verdict reste le job CI .github/workflows/gitleaks.yml, doublé par la
# push protection GitHub côté serveur : rien ne peut atteindre le dépôt distant sans
# passer par eux. Ce hook est un raccourci de boucle de feedback, pas le gate.
#
# Config + doctrine d'allowlist (que faire d'un faux positif) : .gitleaks.toml.

set -euo pipefail

# PIN — même version que .github/workflows/gitleaks.yml et que l'en-tête de
# .gitleaks.toml. Sert ici à avertir en cas d'écart : une version locale différente peut
# rendre un verdict différent de la CI (nouvelles règles = nouveaux faux positifs).
GITLEAKS_VERSION_ATTENDUE="8.30.1"

if ! command -v gitleaks >/dev/null 2>&1; then
  echo "==============================================================================" >&2
  echo "  ATTENTION — gitleaks absent du PATH : AUCUN scan de secrets sur ce commit." >&2
  echo "==============================================================================" >&2
  echo "" >&2
  echo "  Ce dépôt est PUBLIC et manipule des refresh tokens OAuth, un secret client" >&2
  echo "  Azure et un accès SSH prod. Le scan local est la seule barrière AVANT que le" >&2
  echo "  secret n'entre dans l'historique git." >&2
  echo "" >&2
  echo "  Installer gitleaks ${GITLEAKS_VERSION_ATTENDUE} (choisir une méthode) :" >&2
  echo "    Windows (scoop)  : scoop install gitleaks" >&2
  echo "    Windows (manuel) : https://github.com/gitleaks/gitleaks/releases/tag/v${GITLEAKS_VERSION_ATTENDUE}" >&2
  echo "                       -> gitleaks_${GITLEAKS_VERSION_ATTENDUE}_windows_x64.zip, dézipper dans un dossier du PATH" >&2
  echo "    macOS (brew)     : brew install gitleaks" >&2
  echo "    Go (toute plate-forme, version exacte) :" >&2
  echo "                       go install github.com/zricethezav/gitleaks/v8@v${GITLEAKS_VERSION_ATTENDUE}" >&2
  echo "" >&2
  echo "  Commit AUTORISÉ malgré tout (ce hook ne bloque pas) : le job CI" >&2
  echo "  .github/workflows/gitleaks.yml et la push protection GitHub couvrent derrière," >&2
  echo "  mais ils n'attrapent qu'au push — un secret commité et jamais poussé leur" >&2
  echo "  échappe définitivement." >&2
  echo "" >&2
  exit 0
fi

# Le hook peut être invoqué depuis un sous-dossier : on ancre sur la racine du dépôt
# pour que --config trouve .gitleaks.toml et que les chemins des allowlists matchent.
racine="$(git rev-parse --show-toplevel)"
cd "$racine"

# `|| true` : sous `set -euo pipefail`, un binaire présent mais cassé (mauvaise
# architecture, téléchargement tronqué) ferait échouer l'affectation et donc le hook,
# avec un message incompréhensible. On veut un avertissement, jamais un commit bloqué
# par un diagnostic de version.
version_locale="$(gitleaks version 2>/dev/null | tr -d '[:space:]' || true)"
if [ "$version_locale" != "$GITLEAKS_VERSION_ATTENDUE" ]; then
  echo "[gitleaks] version locale ${version_locale:-inconnue} != ${GITLEAKS_VERSION_ATTENDUE} attendue (pin CI)." >&2
  echo "[gitleaks] Verdict possiblement différent de celui de la CI. Scan joué quand même." >&2
fi

# `git --staged` = scan du diff stagé (successeur de l'ancien `protect --staged`).
# --redact : ne jamais afficher un secret en clair, y compris en local (le terminal
# finit dans des transcripts d'agents, des captures et des rapports de bug).
# --verbose : sans lui, seul le nombre de fuites est affiché — inexploitable.
set +e
gitleaks git --staged --config .gitleaks.toml --redact --verbose --no-banner
code=$?
set -e

# Codes de sortie gitleaks : 0 = rien trouvé, 1 = fuite(s) détectée(s), autre = l'outil
# lui-même a échoué. On ne bloque QUE sur 1 : un binaire cassé ou une config illisible
# n'est pas un verdict de fuite, et bloquer là-dessus ferait désinstaller le hook (même
# raisonnement que la branche « binaire absent » plus haut). La CI, elle, échoue sur
# n'importe quel code non nul — c'est elle le gate.
case "$code" in
  0)
    exit 0
    ;;
  1)
    echo "" >&2
    echo "[gitleaks] COMMIT BLOQUÉ — secret potentiel dans les fichiers stagés (valeurs masquées)." >&2
    echo "" >&2
    echo "  VRAIE FUITE : rotation D'ABORD côté émetteur (Azure, Discord, Xbox, VPS), puis" >&2
    echo "  retirer la valeur du code. Ne jamais la mettre en allowlist. Si elle est déjà" >&2
    echo "  commitée, prévenir le mainteneur (purge d'historique)." >&2
    echo "" >&2
    echo "  FAUX POSITIF : exception CIBLÉE dans .gitleaks.toml (targetRules + chemin exact" >&2
    echo "  + regex sur la valeur + commentaire justifiant). Doctrine complète en tête du" >&2
    echo "  fichier. Jamais d'exclusion large d'un dossier." >&2
    echo "" >&2
    echo "  Contournement d'urgence (à justifier dans le message de commit) :" >&2
    echo "    LEFTHOOK=0 git commit ...   — la CI et la push protection GitHub bloqueront quand même." >&2
    echo "" >&2
    exit 1
    ;;
  *)
    echo "[gitleaks] l'outil a échoué (code de sortie $code) — ce n'est PAS un verdict de fuite." >&2
    echo "[gitleaks] Commit autorisé ; le job CI .github/workflows/gitleaks.yml tranchera." >&2
    exit 0
    ;;
esac
