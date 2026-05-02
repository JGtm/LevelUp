# Axe 9 — Code mort & dette

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : tout le repo (sauf node_modules/, .git/, dist/, .venv/)

## Synthèse (3-5 lignes max)

Repo fortement pollué par des artifacts de build/coverage non nettoyés. **6 fichiers binaires/coverage tracked dans git** (87 MB de `apps/tmp/server.exe` + 1.3 MB de coverage HTML/raw + `nohup.out`) en plus de **9 .exe Go non-tracked mais présents** (~700 MB) à la racine `apps/go-api/`. **3 scripts Python tracked** alors que la migration Go est terminée. **Module entier mort** (`internal/observability/`, 6 fonctions exportées non appelées) + **3 commands `//go:build ignore`** (mortes by design) + **`useFieldLabel` 0 usage**. Documentation et plans en partie obsolètes (références à recharts/plotly retirés, fichiers `.ai/PLAN_*` superseded en place sans archivage). Pattern systémique cohérent avec axes 4, 5, 8 : code introduit derrière flag/scaffolding puis jamais retiré.

## Compteurs

- **Binaires/artifacts tracked dans git** : **6** (1 .exe 87 MB + 4 coverage 1.3 MB + 1 .log)
- **Binaires/artifacts non-tracked mais présents au workspace** : **15** (9 .exe Go ~700 MB + 4 coverage Go 9 MB + `.coverage` 1.5 MB + `nul` Windows reserved fake)
- **Fonctions Go exportées mortes (échantillon)** : **6** dans `internal/observability/` + ~15 dans 3 cmds `//go:build ignore`
- **Composants/hooks React non importés** : **2** (`useFieldLabel` documenté + `apps/web/src/app/routes/__root.tsx` duplicate dead)
- **Routes front orphelines (non-link)** : **1** (`/lab/charts` — sandbox dev, intentionnel mais à documenter)
- **Endpoints API orphelins** : non-quantifié exhaustivement (>100 routes chi) — `error_tracker` middleware reste désactivé (cf. axe 8)
- **Fichiers Python résiduels tracked** : **3** (`apps/go-api/tests/create_*_fixture.py` + `scripts/analyze_prestige_tuning.py`)
- **TODO/FIXME/HACK total runtime (hors tests)** : **6** (4 Go runtime, 2 TS runtime — tous datés/nominatifs ; faible dette)
- **Migrations one-shot non archivées** : **5** (`migrate-to-shared-social`, `migrate-static-maps`, `migrate-static-paths`, `apply_tz_migration`, `backfill_bp_items`)
- **Plans `.ai/PLAN_*` partiellement supersedés non archivés** : **5+** (`PLAN_TIMESERIES_*`, `PLAN_CAREER_*`, `PLAN_MATCH_VIEW_*`, `PLAN_SYNTHESIS_*`, `PLAN_TEAMMATES_*`)
- **Cmds Go `//go:build ignore`** : **3** (`check_playlists`, `inspect_bp`, `seed-medal`)
- **Dépendances npm probablement non utilisées** : **1** (`recharts` ^3.8.1)

## Top dette par catégorie

| Catégorie | Élément | Fichier:ligne | Sévérité |
|---|---|---|---|
| Binaire tracked git | `apps/tmp/server.exe` (87 MB, ajouté commit a453da8c) | `apps/tmp/server.exe` | BLOQUANT |
| Coverage tracked git | `coverage.html`, `cover_total.out`, `cover_sync.out`, `session_cover.out`, `nohup.out` | `apps/go-api/cover_*.out` + `.html` + `nohup.out` | BLOQUANT |
| Module Go entier mort | `internal/observability/` — 6 funcs exportées, 0 caller hors tests | `apps/go-api/internal/observability/expvar_metrics.go:41-180` | DETTE |
| Cmds build-ignore | 3 cmds Go avec `//go:build ignore` | `apps/go-api/cmd/{check_playlists,inspect_bp,seed-medal}/main.go:1` | DETTE |
| Path Go dupliqué | `apps/go-api/apps/go-api/cmd/test-gamecms/main.go` (chemin déformé) | `apps/go-api/apps/go-api/cmd/test-gamecms/main.go` | DETTE |
| Hook React 0 usage | `useFieldLabel` (documenté ECharts/canonical mais aucun call site) | `apps/web/src/lib/i18n/fieldMappings.ts:133` | DETTE |
| Route React duplicate dead | `apps/web/src/app/routes/__root.tsx` (vrai root est `apps/web/src/routes/__root.tsx`) | `apps/web/src/app/routes/__root.tsx` | DETTE |
| Dep npm non utilisée | `recharts` ^3.8.1 — 0 import dans `src/` | `apps/web/package.json:33` | AMÉLIORATION |
| Python résiduel | 3 .py tracked, projet Go-only | `apps/go-api/tests/create_*_fixture.py`, `scripts/analyze_prestige_tuning.py` | AMÉLIORATION |
| Migrations one-shot | 5 cmds one-shot finalisés non archivés | `apps/go-api/cmd/{migrate-to-shared-social,apply_tz_migration,backfill_bp_items,migrate-static-maps,migrate-static-paths}/main.go` | AMÉLIORATION |
| Plans superseded | `.ai/PLAN_TIMESERIES_GO_PORTAGE.md` etc. avec note "supersedé par méta-plan" | `.ai/PLAN_*_GO_PORTAGE.md` (5+) | AMÉLIORATION |
| Sandbox /lab/charts non documenté | route createFileRoute existe, 0 link UI (dev-only intentionnel mais non flagué) | `apps/web/src/routes/lab/charts.tsx:5` | AMÉLIORATION |

## Constats

### [BLOQUANT] 1. Binaire 87 MB committé dans git — `apps/tmp/server.exe`

**Fichier** : `apps/tmp/server.exe` — 87 MB, ajouté lors du commit `a453da8c` (`feat(s54): season calendars...`).

`.gitignore` couvre `apps/go-api/*.exe` et `bin/` mais PAS `apps/tmp/`. Le binaire pollue chaque clone du repo. À supprimer du tracking immédiatement (`git rm --cached apps/tmp/server.exe` + ajout `apps/tmp/` au gitignore + suppression locale).

### [BLOQUANT] 2. Fichiers de coverage Go committés (1.3 MB)

**Fichiers tracked** :
- `apps/go-api/coverage.html` (344 KB)
- `apps/go-api/cover_total.out` (412 KB)
- `apps/go-api/cover_sync.out` (76 KB)
- `apps/go-api/session_cover.out` (496 KB)
- `apps/go-api/nohup.out` (8 KB, log d'exécution)

`.gitignore` ligne 53-67 exclut `coverage.out`, `cover_*.out`, `coverage*.raw` etc. — **la règle existe mais ces 4 fichiers ont été committés avant la règle ou explicitement ajoutés** (`git add`) puis jamais retirés du tracking. À nettoyer (`git rm --cached`). Date dernier commit : 2026-04-22 (commit `f55b8dc4`).

### [BLOQUANT] 3. 9 binaires .exe Go (~700 MB) à la racine `apps/go-api/`

**Fichiers présents (non-tracked mais polluant le worktree)** :
```
admin.exe (3.9 MB), engagement-validate.exe (72 MB), levelup.exe (82 MB),
levelup-api.exe (88 MB), migrate-static-maps.exe (72 MB), migrate-static-paths.exe (73 MB),
refresh-career-ranks.exe (80 MB), refresh-metadata.exe (78 MB), warm_bp_assets.exe (81 MB)
```

Plus `apps/go-api/bin/server.exe` (94 MB), `bin/server` (91 MB), `bin/levelup.exe` (82 MB), `bin/levelup-api.exe` (91 MB), `bin/test-server.exe` (91 MB), `bin/migrate-static-maps` (75 MB), `bin/populate-assets` (81 MB).

`.gitignore` les couvre, donc pas un blocker git. Mais polluent le worktree dev (~1.4 GB cumulés). Recommandation : `make clean` ou `rm apps/go-api/*.exe apps/go-api/bin/*` après build, ou builder vers `tmp/` ignoré.

### [DETTE] 4. Module Go `internal/observability/` complètement mort

**Fichier** : `apps/go-api/internal/observability/expvar_metrics.go` (180 lignes).

6 fonctions exportées : `IncCounter`, `AddInt`, `LoadCounter`, `RecordDurationMS`, `LoadDurationStats`, `Reset`. Le seul fichier qui les importe est son test associé `expvar_metrics_test.go`. Aucun handler/middleware/service ne les invoque.

Cohérent avec axe 8 (`error_tracker` middleware désactivé en dur) : LevelUp a un scaffold d'observabilité mais aucun call site. Soit câbler dans les middlewares HTTP (`stats.go`, `match_history.go`, etc.), soit supprimer le package.

### [DETTE] 5. 3 commandes Go marquées `//go:build ignore`

**Fichiers** :
- `apps/go-api/cmd/check_playlists/main.go:1` — `//go:build ignore`
- `apps/go-api/cmd/inspect_bp/main.go:1` — `//go:build ignore`
- `apps/go-api/cmd/seed-medal/main.go:1` — `//go:build ignore`

Ces 3 cmds ne compilent jamais (build tag stdout `ignore`). Ce sont des outils de débogage one-shot (seed-medal pose un dominance_flag, inspect_bp lit la BD). Recommandation : déplacer sous `apps/go-api/scripts/_devtools/` ou supprimer.

### [DETTE] 6. Path Go bizarre dupliqué — `apps/go-api/apps/go-api/cmd/test-gamecms/`

**Fichier** : `apps/go-api/apps/go-api/cmd/test-gamecms/main.go` — chemin clairement résultat d'une mauvaise mv. Le fichier est valide Go (compile en module `main`) mais son emplacement est un accident. À déplacer vers `apps/go-api/cmd/test-gamecms/` ou supprimer si one-shot.

### [DETTE] 7. `useFieldLabel` 0 usage runtime — symptôme dead-by-flag (cf. axe 4)

**Fichier** : `apps/web/src/lib/i18n/fieldMappings.ts:133` (export).

Hook documenté dans `squad/i18n.ts:6`, `squad/metrics.ts:7,13,28`, mais **aucun call site `useFieldLabel(...)` dans le code de production**. Le test `fieldMappings.test.ts` valide juste sa logique pure. Pattern axe 4 confirmé : foundation introduite, jamais branchée, dette accumulée.

### [DETTE] 8. Duplicate dead `apps/web/src/app/routes/__root.tsx`

**Fichier** : `apps/web/src/app/routes/__root.tsx` — pas référencé ailleurs (le vrai root utilisé par TanStack Router est `apps/web/src/routes/__root.tsx`, généré dans `routeTree.gen.ts`).

`apps/web/eslint.config.js:40` mentionne encore `'src/app/routes/**/*'` dans les patterns lint mais aucun consommateur. Dossier `apps/web/src/app/routes/` à supprimer (legacy de l'organisation initiale avant TanStack file-based routing).

### [DETTE] 9. Plans `.ai/PLAN_*_GO_PORTAGE.md` superseded en place

**Fichiers** :
- `.ai/PLAN_TIMESERIES_GO_PORTAGE.md:9-15` — note "**partiellement supersedé** par PLAN_META_FOUNDATIONS_GO.md, **stack chart ECharts (Plotly.js retiré)**"
- `.ai/PLAN_CAREER_GO_PORTAGE.md:9-15` — note "**recharts retiré**"
- `.ai/PLAN_MATCH_VIEW_GO_PORTAGE.md`, `PLAN_SYNTHESIS_GO_PORTAGE.md`, `PLAN_TEAMMATES_GO_PORTAGE.md`, `PLAN_SQUAD_GO_PORTAGE.md` — pattern similaire.

Ces plans pointent eux-mêmes vers leur successeur mais restent dans `.ai/` actif. Ils devraient être déplacés sous `.ai/archive/v6.x/portage_plans/` selon la convention déjà en place (`.ai/archive/v6.0/`).

### [AMÉLIORATION] 10. Dépendance npm `recharts` non utilisée

**Fichier** : `apps/web/package.json:33` — `"recharts": "^3.8.1"`.

0 import dans `apps/web/src/`. La stack charts est ECharts (cf. PLAN_META_FOUNDATIONS_GO + ADR `0001-charts-stack-echarts.md`). Cohérent avec la note des plans portage qui mentionnent "recharts retiré" — la dépendance n'a juste pas été dégagée du `package.json`. À supprimer (`npm uninstall recharts`).

`plotly.js` est déjà retiré du package.json malgré la note du `project_map.md:154` qui mentionne encore "dépendance explicite plotly.js, requise au build par react-plotly.js" — incohérence à nettoyer aussi côté doc.

### [AMÉLIORATION] 11. 3 fichiers Python résiduels — projet officiellement Go-only

**Fichiers** :
- `apps/go-api/tests/create_test_fixture.py` (création fixtures DuckDB pour CI)
- `apps/go-api/tests/create_multititle_fixture.py` (idem multi-titres)
- `scripts/analyze_prestige_tuning.py` (rapport tuning prestige stdout)

Aucun n'est invoqué par les workflows CI (`.github/workflows/*.yml` n'utilise Python que pour parser YAML inline, pas ces fichiers). `requirements.txt` / `pyproject.toml` / `setup.py` absents — donc pas de dette package mais ces 3 .py orphelins. Soit les porter en Go (`tools/seed-fixtures/main.go`), soit documenter dans un README pourquoi on garde Python pour ces outils dev.

### [AMÉLIORATION] 12. Migrations one-shot non archivées (5 cmds)

**Fichiers** : `apps/go-api/cmd/{migrate-to-shared-social,apply_tz_migration,backfill_bp_items,migrate-static-maps,migrate-static-paths}/main.go`.

Tous ont des docstrings explicites ("À exécuter UNE SEULE FOIS", "migration one-shot des données médias", "migration tz", etc.). Ils sont déjà tagués `//go:build cgo` pour limiter, mais polluent encore `cmd/` au même niveau que les commands actives (`server`, `levelup`, `admin`, `refresh-*`).

Recommandation : créer `apps/go-api/cmd/_archive/` ou `apps/go-api/cmd/migrations/` et y déplacer ces 5 cmds avec un README de garde "ne supprimez pas, déjà exécutées en prod, conserver pour audit/replay".

## Constats hors-axe (relèvent d'autres axes)

- **Axe 7 (tests)** : `apps/go-api/tests/golden/golden_test.go` charge des fixtures JSON statiques figées, pas de génération automatique. Vérifier que le générateur Python `create_test_fixture.py` est encore le seul moyen — sinon le supprimer.
- **Axe 8 (logs)** : `apps/go-api/nohup.out` (8 KB) tracked confirme un déploiement manuel ad hoc qui a fui dans le repo.
- **Axe 5 (color tokens)** : la couleur `okabe-ito` dans `palettes/okabe-ito.ts` matche les tokens "heatmap-divergent" déjà flaggés morts à l'axe 5. Pas de nouveau finding.

## Suivi recommandé

1. **Hygiène git immédiate (1h)** :
   - `git rm --cached apps/tmp/server.exe apps/go-api/cover_*.out apps/go-api/coverage.html apps/go-api/nohup.out apps/go-api/session_cover.out`
   - Ajouter `apps/tmp/` à `.gitignore` racine
   - Cleanup local `make clean` ou `rm -f apps/go-api/*.exe apps/go-api/bin/*`

2. **Suppression du module mort + cmds ignore (2h)** :
   - Décider : câbler `internal/observability` dans middlewares HTTP OU supprimer le package + 6 fonctions exportées + tests
   - Déplacer les 3 cmds `//go:build ignore` (`check_playlists`, `inspect_bp`, `seed-medal`) sous `apps/go-api/cmd/_devtools/` ou les supprimer
   - Réparer le path corrompu `apps/go-api/apps/go-api/cmd/test-gamecms/`

3. **Archivage plans superseded + déps non utilisées (1h)** :
   - `mv .ai/PLAN_{TIMESERIES,CAREER,MATCH_VIEW,SYNTHESIS,TEAMMATES,SQUAD}_GO_PORTAGE.md .ai/archive/v6.x/portage_plans/`
   - `npm uninstall recharts` dans `apps/web/`
   - Supprimer `apps/web/src/app/routes/__root.tsx` + l'entrée correspondante dans `eslint.config.js:40`
   - Décision sur `useFieldLabel` : câbler dans les charts squad (cf. axe 4) ou supprimer

---

## Amendement post-vérification (2026-04-29)

> Ajouts issus de la passe de vérification finale (cf. [verification-finale-scaffolding.md](verification-finale-scaffolding.md)).

### [DETTE] Endpoints Go montés mais non consommés côté front

- `GET /match-exclusions` — `apps/go-api/internal/api/server.go:485` (handler `excl.ListExclusions`). Le PATCH `/matches/{id}/exclusion` est consommé (`features/match-history/queries.ts:46`), mais le **GET de listing** n'a aucun consommateur. Vue admin probablement prévue puis jamais implémentée.
- `POST /media/reassociate` — `apps/go-api/internal/api/server.go:460` (handler `media.PostReassociateMedia`). Le front utilise `/media/associate` (`features/media/queries.ts:359`). Doublon non utilisé, probable refactor `reassociate` → `associate` qui n'a pas supprimé l'ancien handler.
- `GET /api/v1/titles/{slug}/preview/career` et `GET /api/v1/players/{slug}/preview/career-multi-title` — déjà ajoutés en amendement axe 2.

**Action** : pour chacun, soit brancher (s'il y a un usage admin prévu), soit supprimer le handler avec son test.

### [DETTE] 3 composants Prestige exportés sans importateur (déjà détaillés en amendement axe 4)

- `MomentCard.tsx:25`, `ArcSummary.tsx:19`, `StatsGlobales.tsx:29` dans `apps/web/src/features/prestige/components/`.
- **Action** : à archiver ou brancher (cf. axe 11 sur le module Prestige entier dormant).

### [DETTE] 6 capabilities Halo déclarées mais jamais consommées (déjà détaillées en amendement axe 2)

- `apps/go-api/internal/domain/title/registry.go:31-37` — `CapMatchmaking`, `CapFirefight`, `CapForge`, `CapMedia`, `CapRanked`, `CapCareer` (seule `CapAssetImages` est utilisée runtime).
- **Action** : câbler `RequireCapability` ou supprimer les 6 caps mortes.

### [DETTE] `apps/web/src/app/routes/__root.tsx` (~70 L) orphelin — duplicata après migration

- Constat déjà présent en suivi recommandé §3, **promu en constat explicite** : c'est du code mort à supprimer, pas juste une recommandation d'archivage. Détail en amendement axe 4.
