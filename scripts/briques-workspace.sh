#!/usr/bin/env bash
# scripts/briques-workspace.sh — engendre levelup-briques.work, l'espace de travail Go LOCAL qui relie
# apps/go-api à des clones de travail des briques publiées (ADR 0043).
#
# Les briques sont des dépendances de apps/go-api/go.mod, à une version publiée. Pour changer une
# brique et voir l'effet dans LevelUp avant de publier sa version suivante, on clone son dépôt, puis :
#
#   bash scripts/briques-workspace.sh ../halo-infinite-film-decoder [../halo-api ...]
#   export GOWORK="$PWD/levelup-briques.work"     # PowerShell : $env:GOWORK = "$PWD\levelup-briques.work"
#
# Le fichier n'est jamais versionné (.gitignore ; garde-rail archlint/no_go_work_versionne_test.go)
# et seul GOWORK le désigne : sans la variable, tout outil Go juge les versions publiées. Les filets
# (make gate-push, scripts/gate-push.ps1) posent GOWORK=off. Retour aux versions publiées :
# `unset GOWORK` (ou supprimer le fichier).
#
# Chaque argument est le dossier d'un clone : il doit porter un go.mod dont le module est une brique
# requise par apps/go-api/go.mod. La ligne `go` est celle de apps/go-api/go.mod.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOMOD="$REPO_ROOT/apps/go-api/go.mod"
SORTIE="$REPO_ROOT/levelup-briques.work"

if [[ $# -eq 0 ]]; then
  echo "usage: $0 <clone de brique> [<clone de brique> ...]" >&2
  exit 2
fi

version="$(awk '$1 == "go" { print $2; exit }' "$GOMOD" | tr -d '\r')"
if [[ -z "$version" ]]; then
  echo "briques-workspace: ligne go introuvable dans $GOMOD" >&2
  exit 1
fi

uses=("./apps/go-api")
for clone in "$@"; do
  # pwd -W (Git Bash) rend un chemin que le Go de Windows comprend ; ailleurs, pwd.
  dir="$(cd "$clone" && { pwd -W 2>/dev/null || pwd; })"
  if [[ ! -f "$dir/go.mod" ]]; then
    echo "briques-workspace: $clone ne porte pas de go.mod" >&2
    exit 1
  fi
  module="$(awk '$1 == "module" { print $2; exit }' "$dir/go.mod" | tr -d '\r')"
  case "$module" in
    github.com/JGtm/halo-*) ;;
    *)
      echo "briques-workspace: $clone porte le module $module, qui n'est pas une brique" >&2
      exit 1
      ;;
  esac
  if ! grep -Eq "^[[:space:]]*(require[[:space:]]+)?$module[[:space:]]" "$GOMOD"; then
    echo "briques-workspace: $module n'est pas requis par apps/go-api/go.mod" >&2
    exit 1
  fi
  uses+=("$dir")
done

{
  echo "// Engendré par scripts/briques-workspace.sh : espace de travail LOCAL, jamais versionné."
  echo "go $version"
  echo
  echo "use ("
  for u in "${uses[@]}"; do
    printf '\t%s\n' "$u"
  done
  echo ")"
} > "$SORTIE"

echo "écrit : $SORTIE"
echo "à poser : export GOWORK=\"$SORTIE\""
