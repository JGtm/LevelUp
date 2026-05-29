#!/usr/bin/env bash
# Vérifie le formatage gofmt des fichiers Go passés en arguments ({staged_files}).
set -euo pipefail

if [ "$#" -eq 0 ]; then
  exit 0
fi

bad=$(gofmt -l "$@")
if [ -n "$bad" ]; then
  echo "gofmt -w requis sur ces fichiers :"
  echo "$bad"
  exit 1
fi
exit 0
