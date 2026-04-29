# ADR 0012 — Halo-only adapters extraction

**Date** : 2026-04-29
**Statut** : Accepted
**Phase** : P5.4 (revue 2026-04-29 axes #8 + #9)

## Contexte

Le projet est conçu pour être multi-titres (Halo Infinite, Halo MCC futur, Halo
Wars, etc.) mais le code historique mélange logique cross-titre et logique
Halo-specific dans le même package `internal/analysis/`. Deux fichiers en
particulier :

1. `analysis/mode_category.go` : préfixes de `pair_name` Halo Infinite
   (Assassin/Fiesta/BTB/Ranked/Firefight/Other) — ces préfixes sont des
   conventions de l'API Waypoint, propres à Halo Infinite.
2. `analysis/citations_custom.go` : 12 fonctions custom citations
   (`computeBulldozer`, `computeWinsCTF`, etc.) — règles métier Halo-only
   reposant sur des conventions Halo (playlist names, modes, awards).

De plus, `platform/duckdb/home_repo.go` contenait un format de chemin badge CSR
hardcodé `"120px-HINF-CSR_*"` qui dupliquait celui déjà exposé par
`halo_infinite.AssetURLAdapter.CSRRankImageURL` (gap #9).

## Décision

**Extraire toute la logique Halo-only de `internal/analysis/` vers
`internal/games/halo_infinite/`** et matérialiser la frontière inter-titres :

- `analysis/mode_category.go` → `games/halo_infinite/mode_category.go`
- `analysis/citations_custom.go` → `games/halo_infinite/citations_custom.go`
- `platform/duckdb/home_repo.go::buildHomeSkillPeakBadgeURL` délègue désormais
  à `halo_infinite.AssetURLAdapter.CSRRankImageURL` (DRY).

**Hook citations** : pour éviter le cycle d'import `analysis → halo_infinite →
platform/duckdb → analysis`, le moteur citations dans `analysis/` expose
`RegisterCustomDispatcher(fn)`. Le package `halo_infinite` enregistre sa
fonction via `init()` au chargement.

**Migration `ranks_loader.go`** : la fonction `LoadRankCatalog` qui dépendait
de `platform/duckdb` a été déplacée de `games/halo_infinite/` vers
`platform/duckdb/halo_ranks_loader.go` pour casser le cycle
`duckdb → halo_infinite → duckdb`.

## Conséquences

### Positives

- Frontière inter-titres claire : un futur titre (Halo MCC, Halo Wars) crée
  son propre dossier `internal/games/<titre>/` avec ses propres
  `mode_category.go` / `citations_custom.go`.
- Le package `analysis/` ne contient plus que des algorithmes purs cross-titre.
- DRY sur le format CSR badge (gap #9 résolu).

### Coûts

- Les consommateurs externes (`platform/duckdb/media_repo.go`,
  `platform/duckdb/queries_home_citations.go`) doivent désormais importer
  `internal/games/halo_infinite` au lieu de `internal/analysis`.
- Pattern `init()` + dispatcher pour citations : moins explicite qu'un appel
  direct, mais nécessaire pour briser le cycle.

### Plan d'application multi-titres

Quand un nouveau titre arrive :

1. Créer `internal/games/<slug>/mode_category.go` avec ses préfixes.
2. Créer `internal/games/<slug>/citations_custom.go` avec ses règles.
3. Implémenter `TitleSemanticAdapter` et `TitleAssetURLAdapter` dans
   `internal/games/<slug>/`.
4. Enregistrer via `analysis.RegisterCustomDispatcher` dans `init()`.
5. Le `Resolver` injecté en DI route les services produit vers le bon adapter
   selon le `titleSlug` du contexte.

## Alternatives considérées

- **Garder dans `analysis/` avec un préfixe `halo`** : refusé, brouille la
  frontière et n'aide pas à matérialiser l'extension vers d'autres titres.
- **Tout dans une interface unique** : les fonctions sont pures (entrée →
  sortie), le coût d'une interface (boilerplate, DI) n'apporte rien quand
  un seul titre les implémente.
