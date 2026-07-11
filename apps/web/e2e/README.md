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

## Données démo et politique de skip (« vert ou signal, jamais structurellement rouge »)

En CI comme en local, le backend tourne en `LEVELUP_DEMO_MODE=true` et lit les fixtures
démo dans `LEVELUP_DEMO_FIXTURES_DIR` (défaut `<repo>/data/demo`). **Ce dossier est
gitignoré** : il n'existe donc pas en CI. Le seul générateur (`levelup seed-demo`) extrait
des données RÉELLES du joueur de prod et ne tourne que sur l'hôte de prod (job
`deploy-demo`) — il n'est ni déterministe ni auto-suffisant, donc **non exécutable en CI**.

Sans fixture, toutes les routes `/players/<slug>/pages/*` échouent : ~60 specs
data-dépendantes cassaient structurellement (rouge permanent qui masquait les vrais
signaux). Correctif : ces specs appellent le garde
[`_helpers/demoData.ts`](./_helpers/demoData.ts) et sont **SKIP (visibles, motivées)**
quand les fixtures sont absentes, au lieu de FAIL.

- **Specs infra** (montage du shell, absence d'erreur 500, i18n, redirections,
  onboarding) : n'appellent PAS le garde → toujours exécutées (~42 tests verts).
- **Specs / tests data-dépendants** : `await skipIfNoDemoData()` en tête (ou
  `test.beforeEach` pour une spec entièrement data-dépendante). Sonde
  `GET /api/v1/healthz/home?player=demo-player` :
  - `404` (joueur démo non résolvable → fixture absente) ⇒ **skip** ;
  - `200` / `503` (joueur résolu, home complète ou partielle) ⇒ **exécuté**.

Résultat attendu sans fixture : `42 passed, 65 skipped, 0 failed`. Avec un démo seedé
(local/prod), la sonde renvoie 200/503 et les specs data-dépendantes s'exécutent
normalement.

### Ajouter une spec

- Si elle lit des données joueur (pages/*, career, timeseries, médias, escouade…) :
  ajouter `await skipIfNoDemoData()` en première ligne du test (importer depuis
  `./_helpers/demoData`), ou `test.beforeEach(async () => { await skipIfNoDemoData() })`
  si toute la spec en dépend.
- Si elle ne teste que l'infra (montage, contrat HTTP sans données, i18n) : ne rien
  ajouter — elle doit rester exécutée même sans fixture.

## CI

Le job `e2e-react` (`.github/workflows/ci.yml`) ne tourne **que sur pull_request** (coût +
flakiness ; les tests auth exigent des identifiants absents en CI). Il n'y a pas de seed
démo en CI : les specs data-dépendantes y apparaissent en `skipped`, les specs infra
doivent rester vertes.
