---
name: levelup-plan
description: Complément LevelUp des skills plan-review et plan-execution (emplacement des plans et du journal, grille d'architecture Go et web, multi-titre, tests par couche, disque et caches de compilation). À lire avec plan-review avant de valider un plan, et avec plan-execution avant d'exécuter un plan dans LevelUp.
---

# LevelUp : complément des plans

Ce skill ne remplace pas les skills généraux `plan-review` et `plan-execution`. Il ajoute ce qui est propre à LevelUp.

## 1. Où vivent les choses

| Quoi | Où |
|---|---|
| Plan actif | `.ai/PLAN_*.md` à la racine de `.ai/` (les plans clos partent sous `.ai/archive/`) |
| Journal du projet | `.ai/thought_log.md` : une entrée datée `[AAAA-MM-JJ]` par clôture d'étape (format dans `CLAUDE.md`) |
| Backlog | `.ai/BACKLOG.md` |
| Reports à la clôture d'un plan | `.ai/REGISTRE_REPORTS.md` |
| Commit | Le fichier du plan mis à jour est inclus dans le commit de l'étape ; message préfixé par la référence de l'étape |

Règles git du projet (`CLAUDE.md`, section « Stratégie de branches ») : jamais `main` (une poussée sur `main` déploie la prod), une tâche = une branche, intégration sur `release/vX.Y.Z`.

## 2. Grille LevelUp pour `plan-review`

### Architecture Go

- [ ] Algorithmes purs dans `internal/analysis/` (vérifier l'existant d'abord : skill `go-features`).
- [ ] Types de résultat dans `internal/domain/` ou `internal/games/canonical/`.
- [ ] Orchestration dans `internal/service/`, interfaces dans `internal/port/`.
- [ ] Handlers dans `internal/api/handlers/`, sans logique métier ni SQL.
- [ ] Lectures SQL sous `platform/duckdb/` (ou `importstore`, `cronstore`), derrière un port.
- [ ] Gardes de couches de `internal/archlint/` respectées (table dans `CLAUDE.md`).

### Multi-titre

- [ ] Chemins par `PathResolver`, jamais `filepath.Join(..., "data", ...)`.
- [ ] Branchement sur `HasCapability()` / `CapabilityMap`, jamais sur le slug.
- [ ] Nouveau champ : `config/titles/<slug>/mappings/fields.toml` ; nouvel outcome ou asset : `outcomes.toml` / `assets.toml`.
- [ ] Dégradation par `ErrCapabilityNotSupported` prévue et testée.
- [ ] Données lues par `TitleDataAdapter.Load*()`, types de retour canoniques.

### Écritures DuckDB

- [ ] Écriture per-match par `persist.BatchBuilder.Submit()` ; tables append-only lues par `<table>_latest`.
- [ ] Changement d'une formule de LUSR ou de note de performance : constante de révision prévue (ADR 0047).

### Tests par couche

- [ ] `internal/analysis/` : tests unitaires purs.
- [ ] `internal/service/` : tests avec ports simulés.
- [ ] Handlers : tests `httptest`.
- [ ] Nouveau repo `platform/duckdb/` : tests sur DuckDB `:memory:`.
- [ ] Persist, sync ou migration touchés : `-tags=integration` prévu au gate.
- [ ] Web : typecheck, tests des hooks si la logique est complexe.

### Web

- [ ] Routes en fichiers sous `apps/web/src/routes/` (jamais `routeTree.gen.ts`).
- [ ] Query keys dans `lib/query/keys.ts`.
- [ ] Textes dans chaque langue de `config/locales.toml` ; libellés de stats par `useFieldLabel()` / `useOutcomeLabel()`.
- [ ] Couleurs par tokens (skill `color-tokens`) ; graphes sous `components/charts/` (skill `foundations-usage`).

### Logging

- [ ] `slog.InfoContext` / `slog.ErrorContext(ctx, "...", "err", err)` ; jamais `fmt.Println` ni `log.Printf`.

## 3. Précisions LevelUp pour `plan-execution`

### Gates web

- Les agents lancent des tests ciblés : `npx vitest run <dossier> --maxWorkers=2`. Vérifier qu'aucun processus vitest ne survit à la fin d'un agent.
- La suite complète `npx vitest run` tourne **une fois**, par le pilote, sur la branche intégrée, avant toute poussée (`--maxWorkers=4` si autre chose tourne). Les cliquets de `apps/web/src/lib/` comptent sur tout le front : un refactor qui raccourcit du code peut les faire tomber.
- Après chaque poussée sur `release/vX.Y.Z` : `gh run list --branch release/vX.Y.Z --workflow CI --limit 1`.

### Disque : caches de compilation dédiés

Go ne retire de son cache que les entrées inutilisées depuis 5 jours, et seulement quand une commande réutilise **ce** cache. Un `GOCACHE` dédié abandonné ne rétrécit donc jamais.

- **Nommage.** Un lot qui compile en parallèle d'autres sessions pose `GOCACHE=%LOCALAPPDATA%\go-build-<lot>` et, s'il lance golangci-lint, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-<lot>`. Jamais un cache dédié ailleurs : seuls ces noms sont nettoyés.
- **Clôture du lot**, une fois le gate passé et le commit fait : `powershell -File scripts/disk-hygiene.ps1 -Lot <lot> -Apply`. Le script supprime `go-build-<lot>` et `golangci-<lot>`. Il refuse un cache écrit dans les 10 dernières minutes ou qui contient une jonction.
- **Racines de gate** (`replay-corpus-gate --keep-work`, copies du parc) : les nommer `gate_work*` ou `parc*` dans le scratchpad, et supprimer celles qu'un nouveau gate remplace.
- **Filet.** La tâche planifiée `LevelUp-disk-hygiene` tourne chaque jour à 4 h. Elle supprime les caches dédiés inactifs depuis 48 h et les dossiers de session inactifs depuis 72 h. Dans une session active, elle supprime les `gate_work*`, `gocache*`, `golangci*` et `parc*` inactifs depuis 24 h. Elle ne fait rien si un processus `go` ou de test tourne. Sans `-Apply`, le script simule seulement.
- **Attention :** `C:\t` (worktrees et clones compris) est vidé après 72 h. Ne jamais y laisser de travail non poussé.
