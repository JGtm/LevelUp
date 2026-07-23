# Tests E2E Playwright — LevelUp

Suite end-to-end (Playwright, projet `chromium`) contre l'app réelle : API Go sur
`:8000` + Vite sur `:5173`. Config : [`../playwright.config.ts`](../playwright.config.ts)
(`workers: 1`, `fullyParallel: false` — l'API DuckDB est mono-fichier).

## Lancer en local

```bash
make dev                       # démarre API Go (:8000) + Vite (:5173)
cd apps/web && npx playwright test --project=chromium
# première fois : npx playwright install chromium
```

## URLs de route (titre dans l'URL) et redirections legacy

Depuis le chantier « titre dans l'URL » (`feat/title-slug-in-url`,
[`PLAN_TITLE_SLUG_URL_2026-07`](../../../.ai/PLAN_TITLE_SLUG_URL_2026-07.md)), les routes
FRONT joueur sont title-préfixées : **`/t/{titleSlug}/players/{playerSlug}/…`** (le titre
actif est un segment d'URL, plus un état de session implicite). Les specs tournent contre
la session serveur dont le titre par défaut est `halo_infinite`.

- **Ne jamais recopier le préfixe** dans un `page.goto` / une assertion d'URL. Passer par
  [`_helpers/routes.ts`](./_helpers/routes.ts) — `playerPath(slug, suffix)` (titre par
  défaut), `titlePath(titleSlug, slug, suffix)`, `playerUrlPattern(...)` /
  `titleSegmentPattern(...)` (RegExp pour `waitForURL`/`toMatch`). Un seul endroit à
  corriger si le schéma d'URL évolue (règle CLAUDE.md « ≤ 2 copies »).
- **Les URLs d'API restent inchangées** : `/api/v1/players/{slug}/…` n'est PAS
  title-préfixée (le titre transite par le header `X-LevelUp-Title` / la session, pas par
  le chemin). Ne pas les construire via `routes.ts`.
- **`legacy-redirect.spec.ts`** (spec INFRA, pas de skip) exerce en réel la matrice de
  redirection de l'ancien format `/players/…` → `/t/{slug}/players/…` : bookmark,
  préservation `?f=` + `#hash`, remaps internes (objectifs→ascension, palmares→community),
  et honneur d'une session H5 committée. Le joueur y est résolu à l'exécution via
  `GET /api/v1/players` (robuste au jeu de données du serveur). La logique pure est
  couverte en table-driven : `src/lib/title-routing/buildLegacyRedirect.test.ts`.

## Données démo et politique de skip (« vert ou signal, jamais structurellement rouge »)

En CI comme en local, le backend tourne en `LEVELUP_DEMO_MODE=true` et lit les fixtures
démo dans `LEVELUP_DEMO_FIXTURES_DIR` (défaut `<repo>/data/demo`). **Ce dossier est
gitignoré.**

Deux générateurs de fixtures démo :

- `levelup seed-demo` (défaut) — extrait des données RÉELLES du joueur de prod, anonymisées.
  Ni déterministe ni auto-suffisant (CGO + DBs prod) → **non exécutable en CI** ; ne tourne
  que sur l'hôte de prod (job `deploy-demo`).
- **`levelup seed-demo --synthetic`** — construit `data/demo/` DE ZÉRO : DuckDB vierges
  migrées (mêmes migrations que la prod → mêmes vues `_latest`) + INSERT SYNTHÉTIQUES
  déterministes (60 matchs / 5 sessions / 3 joueurs, aucune donnée réelle, aucune DB de
  prod, aucun fichier externe). **C'est la voie CI** : le job `e2e-react` le lance avant de
  démarrer le backend → la sonde résout `demo-player` → les ~60 specs data-dépendantes
  s'exécutent. Détail : `apps/go-api/internal/ops/seed_demo_synthetic.go`.

La démo force par défaut la locale UI **anglaise** (vitrine internationale) ; les specs E2E
vérifiant l'UI française, la CI pinne `LEVELUP_DEMO_LOCALE=fr` au démarrage du backend.

Politique de skip conservée : quand les fixtures sont absentes (ex. run local sans seed),
les specs data-dépendantes appellent le garde [`_helpers/demoData.ts`](./_helpers/demoData.ts)
et sont **SKIP (visibles, motivées)** au lieu de FAIL.

- **Specs infra** (montage du shell, absence d'erreur 500, i18n, redirections de routes
  — `legacy-redirect.spec.ts`, onboarding) : n'appellent PAS le garde → toujours exécutées.
- **Specs / tests data-dépendants** : `await skipIfNoDemoData()` en tête (ou
  `test.beforeEach` pour une spec entièrement data-dépendante). Sonde
  `GET /api/v1/healthz/home?player=demo-player` :
  - `404` (joueur démo non résolvable → fixture absente) ⇒ **skip** ;
  - `200` / `503` (joueur résolu, home complète ou partielle) ⇒ **exécuté**.

Résultat attendu sans fixture : `42 passed, 65 skipped, 0 failed`. Avec la fixture
SYNTHÉTIQUE (CI + local `seed-demo --synthetic`, locale `fr`) : `76 passed, ~31 skipped,
0 failed` — les specs data-dépendantes s'exécutent vraiment contre les pages.

### Skips résiduels documentés (fixture synthétique)

Certaines specs restent skippées avec un motif explicite (helpers de
[`_helpers/demoData.ts`](./_helpers/demoData.ts)) :

- **`skipObsoleteSpec(reason)`** — spec devenue OBSOLÈTE : la fixture l'a rendue exécutable
  et a révélé qu'elle vise une route / un endpoint / une structure UI **qui a changé** depuis
  son écriture (elle skippait toujours faute de démo, la dérive n'avait jamais été détectée).
  À RÉÉCRIRE pour l'UI courante. Concernées : `slice-3-match-history` (route `/stats/history`
  supprimée), `slice-3c-session-compare` (endpoint fusionné dans `/pages/timeseries`),
  `match-view-combat` (onglet « Combat » → « Général »/« Détails »), `period-session-rail`
  (route `/stats/history`), 1 test `p7-dto-rename` (sélecteur canvas du graphique bipolaire).
  L'ancienne spec `ascension-2tabs` a été SUPPRIMÉE (2026-07-22, F6) : la page Ascension est
  passée à 4 onglets ; sa réécriture e2e est un chantier dédié.
- **`skipRequiresRealPlayer(reason)`** — exige les données d'un JOUEUR RÉEL spécifique
  (gamertags nommés + synergies, chart LUSR multi-groupes) non reproductibles en synthétique ;
  skip UNIQUEMENT quand `E2E_SYNTHETIC_DEMO=1` (posé par la CI), s'exécute contre une démo
  réelle. Concernées : `career-lusr-legend`, `squad-charts-render`, `theme-switch-charts`.
- **`engagement`** — skip via `E2E_DEMO_MODE=1` (backfill engagement absent en démo).

### Ajouter une spec

- Si elle lit des données joueur (pages/*, career, timeseries, médias, escouade…) :
  ajouter `await skipIfNoDemoData()` en première ligne du test (importer depuis
  `./_helpers/demoData`), ou `test.beforeEach(async () => { await skipIfNoDemoData() })`
  si toute la spec en dépend.
- Si elle ne teste que l'infra (montage, contrat HTTP sans données, i18n) : ne rien
  ajouter — elle doit rester exécutée même sans fixture.
- Toute URL de route FRONT (`page.goto`, assertions `page.url()`/`waitForURL`) passe par
  `_helpers/routes.ts` (`playerPath`/`titlePath`/…) — jamais de préfixe `/t/{slug}/players/`
  recopié en dur. Les URLs d'API (`/api/v1/players/…`) restent hors de ce helper.

## CI

Le job `e2e-react` (`.github/workflows/ci.yml`) ne tourne **que sur pull_request** (coût +
flakiness ; les tests auth exigent des identifiants absents en CI). Le job **seede la
fixture synthétique** (`levelup seed-demo --synthetic`) AVANT de démarrer le backend
(`LEVELUP_DEMO_MODE=true LEVELUP_DEMO_LOCALE=fr`) → les specs data-dépendantes s'exécutent
réellement. Le front dev utilise le proxy Vite `/api/v1` (NE PAS définir `VITE_API_BASE_URL`,
qui casserait le préfixe `/api/v1`).
