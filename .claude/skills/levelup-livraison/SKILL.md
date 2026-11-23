---
name: levelup-livraison
description: Complément LevelUp du skill delivery-checklist (quel gate à quel moment : gate-push avant main seulement ; commandes exactes de gate Go et web, tag integration, pièges -p 1 et tsbuildinfo, gate-push, CI, multi-titre, révision des notes, thought_log). À lire avec delivery-checklist avant tout commit de clôture, PR ou fusion dans LevelUp.
---

# LevelUp : gates de livraison

Ce skill ne remplace pas `delivery-checklist`. Il donne les commandes exactes de LevelUp et ses pièges propres.

## 0. Quel gate à quel moment (règle de l'utilisateur, 2026-11-22)

Les contrôles ne doivent pas coûter autant que le développement. Trois niveaux, jamais plus :

| Moment | Ce qui tourne | Ce qui NE tourne PAS |
|---|---|---|
| Fin d'un lot d'agent | tests des paquets et dossiers touchés (`go test ./internal/<touchés>/...`, `npx vitest run <dossiers> --maxWorkers=2`), typecheck, lint | aucune suite complète |
| Fusion d'un chantier dans `release/vX.Y.Z` | `go test ./...` ; `-tags=integration -p 1` sur les paquets TOUCHÉS seulement, et seulement si le diff touche `persist/`, `sync/`, `migration/` ou `platform/duckdb/` ; `npx vitest run` complet UNE fois ; typecheck purgé ; lint ; puis poussée, et la CI fait foi | **pas de `make gate-push`**, pas de `-tags=integration ./...` complet |
| Avant fusion vers `main` (geste de l'utilisateur) | `make gate-push` UNE fois, sur la release | |

Une suite qui a tourné sur un état ne se relance pas sur le même état. Les contrôles d'un chantier tournent en parallèle (Go et web ensemble), jamais à la file.

## 1. Go

Depuis `apps/go-api/` :

```bash
go test ./...                                        # suite complète (fusion dans la release)
go test -tags=integration -p 1 ./internal/<touchés>/... # si le diff touche persist/, sync/, migration/ ou platform/duckdb/
go vet ./...
```

- **Le tag `integration` est obligatoire** pour persist, sync et migration. Les tests anti-ART sont derrière ce tag : sans lui, on obtient un faux vert.
- **`-p 1` n'est pas négociable.** Le pilote DuckDB est mono-process. En parallèle, les paquets `platform/duckdb` et `sync` se disputent la base, s'interrompent avec des durées fantômes (environ 28 000 s) et masquent les vrais échecs. Le 2026-07-03, un gate a ainsi paru vert alors que 20 tests étaient rouges.
- `-race` est incompatible avec le pilote DuckDB tel quel : ajouter `-gcflags=all=-d=checkptr=0`.
- Tests qui lisent l'installation du jeu : tag `gamefiles` (`make go-api-test-gamefiles`, environ 6 min).
- Lister les échecs avec `grep '^--- FAIL:'` (ou `Select-String "^--- FAIL:"`), et vérifier `$LASTEXITCODE`. Un `FAIL` seul attrape « Failure while replaying WAL ».

## 2. Web

Depuis `apps/web/` :

```bash
Remove-Item -Recurse -Force node_modules\.tmp   # purge le .tsbuildinfo AVANT le typecheck de clôture
npm run typecheck     # tsc -b
npm run lint          # eslint
npx vitest run        # suite complète, une fois, par le pilote
node ../../tools/lint-cross-feature-imports.mjs   # script à la racine du dépôt
```

- `tsc -b` est incrémental : un cache chaud peut rendre vert en sautant des erreurs. Purger `node_modules\.tmp` (ou `tsc -b --force`).
- Vitest ne vérifie pas les types.
- Les cliquets `*.guard.test.ts` de `src/lib/` comptent sur tout le front : un test limité au dossier touché ne les voit pas.

## 3. Gate local avant `main` seulement

`make gate-push` (environ 40 min) : cliquet du lint Go, harnais d'équivalence du rejeu (20 films du parc), typecheck et lint web, baseline de tests en voies briques et DuckDB (environ 300 paquets en `-p 1`), avec `GOWORK=off`. Il se lance UNE fois, avant la fusion de la release vers `main`, jamais avant une fusion dans la release (section 0). La CI reste le gate d'autorité. Depuis un worktree, il lit seul le parc et la liste des originaux du dépôt principal (cible `replay-equiv` du `Makefile`).

Après chaque poussée : `gh run list --branch <branche> --workflow CI --limit 1`.

## 4. Contrôles LevelUp

- [ ] Chemins par `PathResolver` ; branchement par capability, jamais par slug ; `ErrCapabilityNotSupported` géré.
- [ ] Nouveau champ ou outcome : TOML de `config/titles/<slug>/mappings/` à jour.
- [ ] Écritures per-match par `persist.BatchBuilder` ; lectures append-only par `_latest`.
- [ ] Formule ou paramètre de LUSR ou de note de performance changé (`internal/sync/skill/`, `skill_config.go`, `internal/analysis/skill_v2`, `performance.go`) : dans le même commit, la constante de révision (`PerformanceScoreRev` ou `LUSRRev`, ADR 0047) change, l'empreinte de `sync/scoring_revision_test.go` est ajoutée, et une ligne entre dans les notes de version. Un changement de logique sans changement de nombre n'est pas détecté : changer la révision à la main.
- [ ] Logging `slog.*Context` structuré, clés standards : `"err"`, `"match_id"`, `"player"`, `"titleSlug"`, `"duration"`.
- [ ] Interrupteur conservé : modèle `scheduler/auto_sync_convergence.go` (date de bascule, date de retrait, critère).
- [ ] Textes dans chaque langue de `config/locales.toml` ; couleurs par tokens ; query keys dans `lib/query/keys.ts`.
- [ ] Entrée datée dans `.ai/thought_log.md` (format dans `CLAUDE.md`). Sans entrée, la tâche n'est pas terminée.

## 5. Recherches rapides

```bash
grep -rn "fmt\.Println\|log\.Printf\|log\.Println" apps/go-api/internal/
grep -rn 'filepath\.Join.*"data"' apps/go-api/internal/
```
