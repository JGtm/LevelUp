# =============================================================================
# LevelUp — Makefile (macOS / Linux)
#
# Usage:
#   make install       # Premier lancement : crée .venv + installe les dépendances
#   make run           # Lance le dashboard
#   make sync          # Synchronise tous les joueurs
#   make add-player    # Ajoute un nouveau joueur (guidé)
#   make doctor        # Vérifie l'environnement
#   make update        # Met à jour les dépendances
#   make check         # Lint + format + taille (avant chaque commit)
#   make test          # Lance la suite de tests rapide
#   make test-all      # Lance tous les tests (incl. intégration)
#   make clean         # Supprime le venv
# =============================================================================

PYTHON   := $(shell command -v python3 2>/dev/null || command -v python 2>/dev/null)
VENV_PY  := .venv/bin/python
LAUNCHER := $(VENV_PY) launcher.py

.PHONY: install run sync add-player doctor update check test test-all clean help

## Installe le venv et les dépendances (idempotent)
install:
	@if [ -f "$(VENV_PY)" ]; then \
		echo "✓ .venv déjà présent — utilisez 'make update' pour mettre à jour"; \
	else \
		$(PYTHON) launcher.py setup; \
	fi

## Lance le dashboard (mode interactif si aucun joueur)
run: _check_venv
	$(LAUNCHER)

## Synchronise tous les joueurs configurés
sync: _check_venv
	$(LAUNCHER) sync

## Ajoute / synchronise un joueur (guidé si --gamertag absent)
## Usage : make add-player  ou  make add-player GAMERTAG=JGtm
add-player: _check_venv
ifdef GAMERTAG
	$(LAUNCHER) add-player --gamertag $(GAMERTAG)
else
	$(LAUNCHER) add-player
endif

## Vérifie l'environnement (packages, données, configuration)
doctor: _check_venv
	$(LAUNCHER) doctor

## Met à jour les dépendances du venv
update: _check_venv
	$(LAUNCHER) setup --update

## Lint + format + vérification taille (identique au pre-commit, à lancer avant git commit)
check: _check_venv
	$(VENV_PY) -m ruff check src/ scripts/ --fix
	$(VENV_PY) -m ruff format src/ scripts/ tests/
	$(VENV_PY) scripts/check_code_size.py

## Lance la suite de tests rapide (hors intégration)
test: _check_venv
	$(VENV_PY) -m pytest --ignore=tests/integration -q

## Lance tous les tests (intégration incluse)
test-all: _check_venv
	$(VENV_PY) -m pytest -q

## Supprime le venv (les données ne sont PAS affectées)
clean:
	@echo "Suppression de .venv…"
	rm -rf .venv
	@echo "✓ Terminé. Relanez 'make install' pour recréer l'environnement."

## Affiche cette aide
help:
	@grep -E '^##' Makefile | sed 's/^## //'

# ── Cible interne ─────────────────────────────────────────────────────────────
_check_venv:
	@if [ ! -f "$(VENV_PY)" ]; then \
		echo "❌ .venv introuvable — lancez 'make install' d'abord."; \
		exit 1; \
	fi
