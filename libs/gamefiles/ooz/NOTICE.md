# NOTICE — provenance & licence du package `ooz`

Ce package vendorise un **décodeur Oodle clean-room** pour décompresser les blocs
Kraken/Mermaid/Selkie/Leviathan des archives `.module` Halo Infinite.

## Origine

- `kraken.cpp`, `bitknit.cpp`, `lzna.cpp` proviennent de **ooz** par *Powzix*
  (https://github.com/powzix/ooz), **licence GPLv3**.
- Modifications locales :
  - `#include "stdafx.h"` (MSVC/Windows.h) remplacé par `ooz_compat.h` (shim mingw-w64).
  - Troncature de `kraken.cpp` après `Kraken_Decompress` (suppression du `main()` et du
    loader DLL Oodle propriétaire).
- `ooz_compat.h`, `wrap.cpp`, `ooz.go` sont des fichiers de liaison écrits pour ce projet.

## ⚠️ Implication licence (GPLv3)

Le code dérivé d'ooz est **GPLv3**. Pour éviter toute contagion de licence sur l'app
distribuée, ce package **ne doit servir que dans l'outillage OFFLINE** (extraction de
géométrie de maps en amont, génération d'assets data) — l'application serveur/web livrée
**ne doit jamais linker ce package**. La géométrie extraite (données : contours, meshes)
n'est pas du code et peut être bakée comme asset sans contrainte.

Si un décodage Kraken à chaud (runtime serveur) devient nécessaire, remplacer ooz par une
implémentation sous licence compatible (ou un `oo2core_*.dll` sous licence Oodle).

## Build

CGO requis (`CGO_ENABLED=1`) + toolchain C++ mingw-w64 (ucrt64 g++). Voir `ooz.go`
(`#cgo CXXFLAGS -msse4.1 -std=c++14`, `-static-libstdc++`).
