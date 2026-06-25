# Contribuer à LevelUp

Version anglaise : [../CONTRIBUTING.md](../CONTRIBUTING.md)

Merci de votre intérêt pour contribuer à LevelUp ! Ce document explique comment participer au projet.

## Table des Matières

- [Code de Conduite](#code-de-conduite)
- [Structure du Dépôt](#structure-du-dépôt)
- [Configuration de l'Environnement](#configuration-de-lenvironnement)
- [Comment Contribuer](#comment-contribuer)
- [Stratégie de Branches](#stratégie-de-branches)
- [Standards Backend Go](#standards-backend-go)
- [Standards Frontend](#standards-frontend)
- [Workflow Agent (thought_log + skills)](#workflow-agent-thought_log--skills)
- [Processus de Pull Request](#processus-de-pull-request)
- [Signaler un Bug](#signaler-un-bug)
- [Proposer une Fonctionnalité](#proposer-une-fonctionnalité)
- [Crédits Open Source](#crédits-open-source)

---

## Code de Conduite

Ce projet suit un code de conduite respectueux et inclusif. Soyez bienveillant envers les autres contributeurs.

---

## Structure du Dépôt

LevelUp est un monorepo backend Go + frontend React. Les deux applications sont sous `apps/` :

| Chemin | Stack | Rôle |
|--------|-------|------|
| `apps/go-api/` | Go (CGO + DuckDB) | API HTTP, moteur de sync, analyses, outillage CLI |
| `apps/web/` | React 19 / Vite / TypeScript | Frontend du dashboard |
| `docs/` | Markdown | Documentation anglaise (`docs/FR/` = miroir français) |
| `data/` | DuckDB / Parquet / JSON | Entrepôts, DB par joueur, store de tokens, config |
| `.ai/` | Markdown | Mémoire de travail agent (carte projet, journal, plans) |
| `.claude/skills/` | Markdown | Skills agent (règles d'architecture, conventions) |

Zones clés dans `apps/go-api/` :

| Chemin | Rôle |
|--------|------|
| `cmd/server/` | Point d'entrée du serveur HTTP |
| `cmd/levelup/` | CLI d'exploitation (backup, restore, sync, seed, migrate, ...) |
| `cmd/*` | Outils ponctuels de diagnostic / backfill / migration |
| `internal/api/` | Handlers HTTP, middleware, routeur |
| `internal/analysis/` | Algorithmes purs stateless (temporal, breakdown, narrative) |
| `internal/service/` | Orchestration (accès repo + analyses) |
| `internal/sync/` | Moteur de sync (delta / full, pipeline persist) |
| `internal/platform/duckdb/` | Accès DuckDB, leases, écritures shared_social |
| `internal/games/canonical/` | Types canoniques inter-titres |
| `internal/migration/` | Étapes de migration de schéma |

Références d'architecture : `docs/ARCHITECTURE_V6.md`, `docs/FOUNDATIONS_GUIDE.md`, et les ADR dans `docs/adr/`.

---

## Configuration de l'Environnement

### Prérequis

- **Go** à la version indiquée dans `apps/go-api/go.mod`. DuckDB exige **CGO**, donc une toolchain C est obligatoire. Sur Windows, utiliser `gcc` MSYS2/MinGW avec `CGO_ENABLED=1`.
- **Node.js** (LTS) + npm, pour `apps/web/`.
- **air** pour le hot-reload Go (`go install github.com/air-verse/air@latest`) — utilisé par `make dev`.
- Git.

### Dev en une commande

Depuis la racine du dépôt :

```bash
make dev
```

Lance l'API Go (via `air`, port 8000 par défaut) et le frontend Vite (port 5173). Ouvrir `http://localhost:5173`. `Ctrl+C` arrête les deux. `make stop` force l'arrêt des serveurs dev par port ; `make restart` fait `stop` puis `dev`.

Installer les dépendances frontend la première fois :

```bash
make install-web      # = cd apps/web && npm install
```

### Tokens d'authentification

Les tokens d'auth ont une source unique : `data/auth/watcher_tokens/{xuid}.json`, gérée par `MultiUserTokenStore` (voir `docs/adr/0023-auth-tokens-single-source.md`). Le joueur doit d'abord être déclaré dans `db_profiles.json` (avec `xuid`). Options d'onboarding :

```bash
# Onboarding avancé (capture par device-code, écrit direct dans le store)
go run ./cmd/token-capture/ <Gamertag>

# Import d'un refresh token depuis stdin
go run ./cmd/token-import/ <Gamertag>
```

Ne jamais utiliser `.env.local` ou `sync_meta` comme source de credentials (fallbacks legacy uniquement).

---

## Comment Contribuer

1. Fork et clone du dépôt.
2. Créer une branche de travail (voir [Stratégie de Branches](#stratégie-de-branches)) — **ne jamais committer sur `main`**.
3. Implémenter le changement en suivant les standards ci-dessous.
4. Lancer le lint + les tests concernés (Go et/ou frontend).
5. Ajouter une entrée `.ai/thought_log.md` (voir [Workflow Agent](#workflow-agent-thought_log--skills)).
6. Ouvrir une Pull Request en suivant Conventional Commits.

---

## Stratégie de Branches

Règle (issue de `CLAUDE.md`) : **1 tâche = 1 branche, N commits**. Les phases séquentielles d'une même tâche sont des commits sur une seule branche, pas des branches séparées.

```bash
# Correct — phases d'une tâche = commits sur une branche
git checkout -b refactor/cleanup-all
git commit -m "refactor(phase1): dead code cleanup"
git commit -m "refactor(phase2): DRY violations"
```

Règles d'application :

- **Ne jamais travailler sur `main`** — sans exception. Si vous êtes sur `main`, créer une branche de travail d'abord.
- Vérifier la branche courante avant de committer : `git branch --show-current`.
- Créer une nouvelle branche pour chaque feature/fix depuis la branche courante (`git checkout -b <type>/<nom>`).
- Ne pas changer de branche si un travail différent est déjà en cours sur la branche courante.
- Pousser `main` déclenche un déploiement de production automatique — merger vers `main` délibérément.

Format des messages de commit (Conventional Commits) :

```
<type>(<scope>): <description>
```

Types : `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. Exemples :

```
feat(api): ajouter l'endpoint CSR par playlist
fix(sync): corriger le parsing des modes Firefight
docs: mettre à jour le guide de contribution
```

---

## Standards Backend Go

Les équivalents d'outillage (formatage / lint / typage) sont assurés par `gofmt`, `go vet` et `golangci-lint` (config : `apps/go-api/.golangci.yml`).

### Format et vet

```bash
cd apps/go-api
gofmt -l .            # liste les fichiers non formatés (doit être vide)
go vet ./...
```

La config golangci-lint active `revive`, `gocyclo`, `funlen`, `lll`, `goconst`, `unconvert`, `unparam`, `bodyclose`, `noctx`, `prealloc`, plus le jeu standard et `staticcheck`. Seuils : complexité cyclomatique 15, longueur de fonction 100 lignes / 80 statements, longueur de ligne 220, limite d'arguments 7. `gofmt` + `goimports` sont les formatters.

```bash
cd apps/go-api && golangci-lint run
# Raccourci Makefile (vet sur domain + analysis) :
make go-api-lint
```

### Tests

DuckDB exige CGO. Deux niveaux de tests (détail complet dans `docs/testing.md`) :

```bash
# Niveau rapide — sans DuckDB (CGO off) : domain + analysis + contract
cd apps/go-api
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -timeout 60s -count=1
# Raccourci Makefile :
make go-api-test

# Niveau complet — avec DuckDB (CGO on)
cd apps/go-api
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1
```

Sur Windows, s'assurer que `gcc` MinGW est sur le PATH (`CC=gcc`, `CGO_ENABLED=1`). Note : `go test -race` est incompatible avec le driver DuckDB sauf en passant `-gcflags=all=-d=checkptr=0`.

La couverture est un ratchet non-régressif (baseline dans `apps/go-api/coverage_baseline.txt`) :

```bash
make go-api-coverage              # résumé func rapide
make go-api-test-coverage-ratchet # vérifie le ratchet vs baseline
```

Voir `docs/testing.md` pour les patterns de tests par couche (handlers mock service, DuckDB in-memory, gates de validation).

### Règles d'architecture

- `internal/analysis/` = algorithmes purs stateless (zéro accès DB). `internal/service/` = orchestration (repo + analyses).
- Toute nouvelle écriture dans une DB partagée sur un chemin per-match passe par `internal/persist/BatchBuilder.Submit()` — pas d'UPSERT/UPDATE concurrent sur les tables critiques (ART-safe, voir ADR 0019, 0026).
- Les tables d'état sont en append-only (lecture via les vues `<table>_latest`). Test garde-fou : `internal/sync/no_art_patterns_test.go`.
- Tout accès DuckDB via context manager / lease — pas de fuite `db.Close()` nue.
- Lire les skills agent `arch-rules`, `db-schema`, `canonical-types` et `go-features` avant de modifier la structure backend.

---

## Standards Frontend

Depuis `apps/web/` (scripts définis dans `apps/web/package.json`) :

```bash
npm run typecheck     # tsc -b — aucune erreur de type
npm run lint          # eslint .
npm run lint:fields   # garde contre les noms de champs API en dur
npm run test:run      # vitest run (sans watch)
npm run test:e2e      # playwright (nécessite `make dev` lancé)
```

Raccourcis Makefile : `make check-types`, `make test-web`, `make test-e2e`.

Note Vitest : lancer les tests hors de tout sandbox qui bloque les workers ; typecheck et eslint passent en sandbox.

### Tokens de couleur (obligatoire)

Aucun hex brut (`#RRGGBB`) ni classe de couleur Tailwind (`text-red-*`, `bg-green-*`, ...) dans `apps/web/src/features/` ou `apps/web/src/components/`, sauf exceptions documentées. Toute couleur sémantique doit passer par `tokenCssVar(token)` (JSX), `resolveToken(token)` (Plotly/SVG) ou `getSeriesColors(n, tokens[])` (séries). Les palettes brutes sont centralisées dans `apps/web/src/lib/accessibility/palettes/`. Voir le skill `color-tokens`.

### i18n

Les chaînes destinées à l'utilisateur doivent être fournies en **FR et EN** via les manifests i18n (manifests TOML + linter custom, voir ADR 0003). Ne pas coder en dur les chaînes d'affichage dans les composants.

### Charts et pages

Utiliser les wrappers ECharts canoniques (`apps/web/src/components/charts/README.md`) et les fondations (types canoniques + adapters + i18n + wrappers chart) décrites dans `docs/FOUNDATIONS_GUIDE.md`. Voir les skills `frontend-patterns` et `foundations-usage`.

---

## Workflow Agent (thought_log + skills)

Ce dépôt est maintenu en partie par des agents IA. Deux conventions s'appliquent à tout contributeur :

1. **thought_log (obligatoire)** — avant chaque commit (ou au minimum avant de rendre la main), ajouter une entrée dans `.ai/thought_log.md` avec : la date `[YYYY-MM-DD]`, le titre de la tâche, le statut (En cours / Complété), la décision technique principale, les résultats observés, et la conclusion / prochaine étape. Absence d'entrée = tâche non terminée.
2. **Skills agent** — consulter le skill pertinent dans `.claude/skills/{arch-rules, canonical-types, color-tokens, foundations-usage, delivery-checklist, plan-review, halo-modes, db-schema, frontend-patterns, go-features}/SKILL.md` avant tout changement structurel.

Avant de commencer, relire aussi la mémoire de travail `.ai/` : `project_map.md`, `thought_log.md`, `data_lineage.md`.

---

## Processus de Pull Request

### Checklist

Avant de soumettre une PR, vérifier :

- [ ] Sur une branche de travail, pas `main`
- [ ] Go : `gofmt -l .` propre, `go vet ./...` propre, `golangci-lint run` passe
- [ ] Tests Go passent (niveau rapide toujours ; niveau complet CGO pour les changements DB/sync)
- [ ] Le ratchet de couverture ne régresse pas
- [ ] Frontend : `npm run typecheck`, `npm run lint`, `npm run test:run` passent
- [ ] Aucun hex brut / classe couleur Tailwind dans `features/` ou `components/`
- [ ] Chaînes i18n FR + EN fournies pour tout nouveau texte UI
- [ ] Miroir `docs/FR/` mis à jour si un fichier `docs/` change
- [ ] Entrée `.ai/thought_log.md` ajoutée
- [ ] Messages de commit au format Conventional Commits

### Review

Un mainteneur reviendra vers vous pour des questions de clarification, des suggestions d'amélioration, puis la validation et le merge.

---

## Signaler un Bug

### Avant de Signaler

1. Vérifier que le bug n'est pas déjà signalé.
2. Reproduire sur le dernier `main`.

### Créer une Issue

Inclure :

- **Description** : comportement observé vs attendu.
- **Reproduction** : étapes pour reproduire.
- **Environnement** : OS, version Go (`go version`), version Node, navigateur.
- **Logs** : messages d'erreur complets. Les logs Go sont écrits par catégorie sous `logs/*.log` (ex. `logs/handlers.log`, `logs/general.log`), pas seulement sur stdout — les grep tous.

```markdown
## Bug

### Description
Le dashboard ne charge pas les matchs pour le joueur X.

### Reproduction
1. Ouvrir le dashboard
2. Sélectionner le joueur X
3. Observer l'erreur

### Environnement
- OS: Windows 11
- Go: go1.x
- Node: 20.x

### Logs
(message d'erreur de logs/handlers.log)
```

---

## Proposer une Fonctionnalité

### Avant de Proposer

1. Vérifier que la feature n'est pas déjà proposée ou en cours (plans `.ai/`).
2. Réfléchir à l'implémentation face à l'architecture (ADR, fondations).

### Créer une Issue

Inclure :

- **Description** : qu'est-ce que la feature fait ?
- **Motivation** : pourquoi est-ce utile ?
- **Implémentation** (optionnel) : quelle couche (analysis/service/handler, ou feature frontend) et quels types canoniques sont touchés.

---

## Crédits Open Source

Ce projet dépend de plusieurs briques communautaires. Les crédits sont centralisés dans [ACKNOWLEDGMENTS.md](../ACKNOWLEDGMENTS.md). Avant d'ajouter une dépendance externe majeure, documentez-la là pour garder une attribution claire.

---

## Questions ?

Si vous avez des questions, ouvrez une issue avec le tag `question`.

---

**Merci de contribuer à LevelUp !**
