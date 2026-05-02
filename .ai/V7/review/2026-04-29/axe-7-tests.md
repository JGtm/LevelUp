# Axe 7 — Testabilite & couverture

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Perimetre : apps/go-api/ (tests Go) + apps/web/ (tests Vitest) + contracttest/ + e2e/

## Synthese

Couverture Go affichee a 84.5% mais profondement trompeuse : le ratchet exclut handlers, middleware, sync, migration, platform/duckdb et platform/halo (la majorite des LOC critiques). Le contrat OpenAPI (45 paths) couvre moins de la moitie des 102 routes chi enregistrees, et omet completement les endpoints recents (engagement, squad/v2, multi-title, asset-drawer). Quatre bugs critiques fix dans le sync engagement (B1-B4 thought_log) sont sans test de regression. Cote front, 88 fichiers .test.* pour 305 sources (~29% nominal), pas de seuil enforced. Verdict : moyen — bonne hygiene unit test handlers + service, mais surface mesuree faussement haute, multi-titres pas verifie cross-titre, regressions recentes invisibles.

## Compteurs

- Tests Go par couche : **368 fichiers test** pour **451 sources** dans `internal/` (ratio 0.82)
  - api/handlers : 52 tests / 48 sources (incl. extra/internal helpers)
  - service : 56 tests / 46 sources
  - analysis (recursif) : 47 tests / 53 sources
  - platform/duckdb : 31 tests / 53 sources (ratio 0.58)
  - sync : 43 tests / 29 sources, dont 19 sont `//go:build integration` (CGO requis)
  - games (recursif) : 17 tests / 26 sources
  - api/middleware : 12 tests / 12 sources
  - platform/halo : 4 tests / 10 sources (ratio 0.40)
- Tests Vitest front : **88 fichiers .test.\*** pour **305 sources** (.ts/.tsx hors test) — ratio nominal **~29%**
  - 50 .test.tsx (composants) + 36 .test.ts (logique pure)
  - 19 specs Playwright e2e
- Couverture Go affichee : **84.5%** (`coverage_baseline.txt`) — exclut 8 packages dans `coverage_filter.sh` (sync, migration, platform/duckdb, platform/halo, api/handlers, api/middleware, api/registry, gen/, port/, cmd/)
- Couverture front : **non mesuree** en CI (script existe `npm run test:coverage` mais aucun job dans `ci.yml` ne l'execute)
- Tests skip/todo : **17 occurrences `t.Skip`** (9 fichiers Go) ; **2 `test.skip()` Playwright** dans media-like-bug.spec.ts ; 0 `it.todo`/`xit` cote Vitest
- E2E maintenus : oui, 19 specs Playwright lancees en CI (job `e2e-react`), serial (`workers: 1`)

## Constats

### [BLOQUANT] OpenAPI/contracttest perime sur les endpoints recents

`apps/go-api/api/openapi.yaml` declare **45 paths** (`grep -cE "^  /"`). Le router chi enregistre **102 routes** (`internal/api/server.go:r.(Get|Post|...)`). Les endpoints actifs suivants ne sont pas dans `openapi.yaml` :

- `/players/{slug}/engagement/timeseries`, `/matches/{id}/engagement`, `/engagement_profile`, `/pages/squad/v2/engagement` (handler : `internal/api/handlers/engagement.go`)
- `/pages/squad/v2/*` (handler : `internal/api/handlers/squad_v2.go:1-end`)
- `/titles/{slug}/preview/career`, `/titles/{slug}/field-mappings` (handlers `multi_title_player_preview.go` + `field_mappings.go`)
- `/assets/{title_id}/maps`, `/assets/{title_id}/weapons` (asset-drawer Phase 0+1, handlers `assets_metadata.go`)

`contracttest/contract_yaml_test.go:158-187` valide une whitelist de 10 paths obligatoires, tous legacy. Aucun garde-fou snake_case/types/nullable cote contracttest : il valide uniquement l'existence de l'OpenAPI et la coherence YAML→JSON. Les tests YAML ne previennent donc rien sur les nouveaux endpoints.

### [BLOQUANT] Couverture Go 84.5% mensongere — 8 packages exclus du ratchet

`apps/go-api/scripts/coverage_filter.sh:27-43` exclut explicitement de la mesure :

```
internal/sync/             (orchestration sync, 29 sources)
internal/migration/
internal/platform/duckdb/  (53 sources — repository principal)
internal/platform/halo/    (10 sources — provider live)
internal/api/handlers/     (48 sources — toute la couche HTTP)
internal/api/middleware/   (12 sources)
internal/api/registry      (DI)
internal/port/             (interfaces)
```

Justifie comme "teste en integration" mais le ratchet `coverage_check.sh` lit le `total:` sur le profil **filtre**, pas sur ces packages. Resultat : la baseline 84.5% mesure essentiellement `internal/analysis/`, `internal/service/`, `internal/games/`, `internal/config/`. La couche HTTP + persistence + halo provider — qui represente la majorite du code metier reel — n'est sous **aucun** ratchet de couverture.

`ci.yml:175-185` ajoute une mesure isolee `internal/sync/` mais sans seuil enforced (`go tool cover -func=cov_sync.out | tail -5` est juste un `Logf`).

### [BLOQUANT] 4 bugs critiques engagement sans test de regression

Thought_log [2026-04-29] documente 4 bugs critiques fix dans `internal/sync/engagement.go` (434L) :

- B1 : `match_registry.is_pve` n'existe pas, doit etre `is_firefight` (`engagement.go:191,195`)
- B2 : `match_participants.is_bot` inexistant, convention projet `xuid LIKE 'bid(%'`
- B3 : confusion `MatchStartMS` epoch UTC vs `time_ms` relatif (le bug "le plus subtil") — `engagement_player_service.go buildInputForMatch`
- B4 : tags JSON manquants sur `domain.EngagementScoreResult` → PascalCase au lieu de snake_case

`internal/sync/engagement.go` n'a **aucun fichier test** (`find internal/sync -name "engagement*"` → seulement engagement.go). La regression peut reapparaitre silencieusement : les 4 specs Playwright `engagement.spec.ts` ne couvrent que JGtm sur localhost:8000 (cf [BLOQUANT] suivant), donc CI ne les execute pas. `internal/analysis/temporal/engagement_score_test.go` teste l'algorithme avec des `MatchStartMS: 0` deja "corriges" — il aurait passe meme avec le bug B3 en place.

### [BLOQUANT] engagement.spec.ts incompatible avec la CI demo-mode

`apps/web/e2e/engagement.spec.ts:19-20` :

```ts
const API = 'http://localhost:8000/api/v1'
const PLAYER_SLUG = 'JGtm'
```

Hardcode l'URL absolue (les autres slices utilisent `baseURL` Playwright via `page.goto`) et le slug `JGtm` (les autres slices utilisent `demo-player`). Le job `e2e-react` (`ci.yml:236-313`) demarre l'API avec `LEVELUP_DEMO_MODE=true` : il n'existe pas de joueur JGtm avec engagement scores backfilles. Les 6 tests "PASS local" cites dans thought_log ne sont que la verite sur la machine du dev. En CI, ces tests vont 500 ou retourner array vide.

### [BLOQUANT] platform/halo 60% des sources sans test

`internal/platform/halo/` : 10 sources, 4 tests. Sans test :

- `challenges_details.go`, `compare_provider.go`, `discovery_client.go`, `discovery_types.go`, `medal_provider.go`, `season_provider.go`, `player_token_cache.go`

Couche live API Halo (auth, fetchs) — aucune protection contre une regression provider. Note : `provider_test.go` couvre Battle Pass + Challenges live + retry HTTP, donc ce n'est pas zero, mais la moitie des fetchs reels (medals, season, decks par xuid via singleflight) ne sont jamais exercees. Le filtrage couverture exclut le package du ratchet.

### [BLOQUANT] Vitest coverage non mesuree en CI

`apps/web/vite.config.ts:42-53` configure le provider v8 avec rapport HTML/lcov, mais `ci.yml:41-42` lance uniquement `npm run test --if-present -- --run` (sans `--coverage`). Le script `npm run test:coverage` existe (`package.json:15`) mais n'est cable nulle part dans la pipeline, et aucun seuil n'est defini (pas de `coverage.thresholds` dans vite.config.ts). Front sans garde-fou.

### [DETTE] platform/duckdb couverture 58% sources/tests

`internal/platform/duckdb/` : 53 sources, 31 tests (ratio 0.58). Inclut le repository principal du produit, exclu du ratchet. Plusieurs `t.Skip` documentent des bugs SQL connus laisses en place :

- `repos_coverage_test.go:191,217` : `t.Skip("pre-existing SQL bug: ambiguous gamertag reference in GROUP BY")`
- `media_repo_filters_test.go:610` : `t.Skip("semantique ModeFilter ambigue : sous-mode seul vs categorie/sous_mode")`

Trois bugs reconnus skipes en silence — ils ne remontent jamais en rouge.

### [DETTE] Tests "coverage_boost" / "extra" — qualite signalee suspecte

27 fichiers nommes `*_extra_test.go` ou `coverage_boost*` (`internal/analysis/coverage_boost_test.go`, `notify/version_extra_test.go`, etc.). Lecture rapide : les tests sont legitimes (decodage frames, parsing JSON degrade). Mais le pattern de naming "extra/boost" est un signal connu d'inflation defensive de couverture — ces tests existent pour atteindre les chiffres du ratchet, pas pour documenter le contrat. Recommander un audit cible : combien de chemins d'erreur y sont-ils testes vs combien de path nominaux ?

### [DETTE] CLI levelup backfill (cmd_backfill.go) sans test

`cmd/levelup/cmd_backfill.go` (nouveau, thought_log [2026-04-29]) execute le backfill engagement-scores en CLI. Aucun fichier `*_test.go` dans `cmd/levelup/`. Le filtrage exclut `cmd/` du ratchet, donc absence non visible. Bug typique CLI : flag parsing, exit codes, aggregation report — tous a la merci d'une regression silencieuse.

### [DETTE] Front features critiques sans aucun test Vitest

Au-dela des 4 deja signales en axe 4 (Notifications, SessionCompare, AssetDrawer, AdminPage) :

- `features/auth/` : 0 test
- `features/changelog/` : 0 test
- `features/engagement/` : 0 test (alors que c'est l'enjeu principal de la branche en cours)
- `features/filters/` : 0 test (logique cross-page partagee)
- `features/match-view/` : 1 seul test `MatchNarrativeSection.test.tsx` (la page principale Match View n'a aucun test composant)
- `features/squad/v2/` : 4 tests sur composants (FloatingLegend, HistoryTable, MedalsGallery, WeaponsTable) mais pas de test sur SquadV2Page elle-meme
- `features/prestige/` : 0 test malgre la presence de hooks et components/

Sur 88 fichiers de tests pour 305 sources, le ratio est plombe par les routes/components UI sans test.

### [DETTE] Tests lents : workers=1 force par DuckDB mono-fichier

`apps/web/playwright.config.ts:14-17` :

```
fullyParallel: false, // API DuckDB mono-fichier → sériel pour éviter les locks
workers: 1,
retries: process.env.CI ? 2 : 0,
```

Avec 19 specs serielles + retry 2x en CI, le job e2e-react va prendre du temps a chaque PR. Les tests Go integration utilisent `:memory:` (`internal/sync/testutil/fixture.go:17`) donc ils sont parallelisables, mais `t.Parallel()` n'apparait que dans les tests purs (`448 occurrences sur 59 fichiers`). Pas de probleme bloquant, mais la CI e2e va lentement etrangler la velocity.

### [DETTE] Pas de test croise canonical.PlayerMatchRow cross-titre

`internal/games/synthetic_title_b/isolation_test.go` existe mais teste l'isolation des mappings TOML (champ "Phase E" du plan multi-titre). Le pipeline canonique `MatchHistory/MatchView/Home/Timeseries WithDataAdapter()` est valide uniquement par compile-time check (cite litteralement dans `internal/service/multi_title_parity_test.go:21-23` : "Les autres services...sont valides indirectement par leur wiring dans api/registry.go (compile-time check)"). Si `synthetic_title_b` regresse silencieusement, aucun test ne le verra. Constat lie a l'axe 2, mais ici l'angle est : la "preuve par compile" remplace le test d'integration multi-titre.

### [AMELIORATION] Pas de testify, stdlib testing-only

`go.mod` n'inclut pas `stretchr/testify` (3 occurrences dans `go.sum` = transitif). Tests stdlib-only avec `t.Errorf` partout. Pas un probleme, mais empeche les patterns `mock.AssertCalled` standardises et les assertions structurees. Le projet utilise des "fake structs" (cf `fakeSquadLoader` dans `service/squad_service_v2_test.go:17-69`) — pattern propre. Constat AMELIORATION : pas d'urgence a changer.

### [AMELIORATION] Fixtures golden centralisees mais limitees

`apps/go-api/tests/fixtures/golden_values/` : 13 JSONs (health, bootstrap, players_list, career_*, match_view_slayer, filters_*, gamertag_search_*). Tests `golden_test.go` valident la **structure** des fixtures (cle existe, type string, etc.) mais ne comparent pas une vraie reponse handler avec le JSON attendu — c'est juste un linter de fixtures. Bon depart, mais sous-exploite : pas de golden test sur les payloads recents (squad/v2, engagement, multi-title).

## Cartographie : qualite par couche

| Couche | Tests presents | Qualite | Gap principal |
|---|---|---|---|
| api/handlers/ | 52 fichiers (httptest dominant, mocks `port.Repo`) | bonne | exclu du ratchet, recents (squad_v2/engagement) ne sont pas dans openapi.yaml |
| api/middleware/ | 12 / 12 | tres bonne | sous le ratchet exclu, mais 1:1 |
| service/ | 56 / 46 | bonne (fakes structs, table-driven sporadique) | parite multi-titre par compile-time check uniquement |
| analysis/ (recursif) | 47 / 53 | bonne (purs, table-driven, t.Parallel partout) | tests "coverage_boost" suspects mais legitimes |
| analysis/temporal/engagement_score | 1 / 1 | bonne | ne couvre pas le bug B3 (ms relatif vs absolu) |
| platform/duckdb/ | 31 / 53 (0.58) | moyenne | 4 t.Skip silencieux sur bugs SQL connus |
| platform/halo/ | 4 / 10 (0.40) | mauvaise | 60% sources sans test, exclu du ratchet |
| sync/ | 43 / 29 (parmi lesquels 19 integration only) | bonne en surface | engagement.go (434L) sans **aucun** test, 4 bugs recents non couverts |
| migration/ | tests presents avec t.Skip | moyenne | 4 t.Skip "aucune migration enregistree" rendent les tests no-op |
| games/ (recursif) | 17 / 26 | bonne | isolation cross-titre limitee a synthetic_title_b |
| cmd/levelup/ | 0 fichiers | absente | nouveaux sous-commandes (backfill) sans test |
| web features/ | 88 / 305 (0.29) | moyenne | aucun seuil enforced, pas de coverage CI, features critiques nues |
| web e2e (Playwright) | 19 specs | mixte | engagement.spec.ts incompatible CI demo-mode |
| contracttest/ (YAML) | 4 tests | mauvaise | 45 paths declares, 102 routes vives — recents jamais documentes |

## Suivi recommande

1. **Restaurer la verite du ratchet de couverture Go** : produire un second profil non-filtre avec un seuil minimum par-package (ex `api/handlers >= 70%`, `platform/halo >= 50%`, `sync >= 60%`), et bloquer les regressions sur ce profil. Le 84.5% actuel est marketing.
2. **Cloturer la dette OpenAPI + contracttest** : completer `openapi.yaml` avec les ~57 routes manquantes (engagement, squad/v2, multi-title, asset-drawer), enrichir `contract_yaml_test.go` pour valider snake_case + types + nullable sur les payloads recents (snapshot d'un golden contre la spec).
3. **Tests de regression sur les 4 bugs engagement** : creer `internal/sync/engagement_test.go` avec un cas table-driven qui aurait detecte B1+B2+B3+B4 (schemas avec `is_firefight`, fixtures avec `xuid LIKE 'bid(%'`, `time_ms` relatif depasse `start_time` epoch, deserialisation snake_case attendue cote DTO). En parallele, fixer `engagement.spec.ts` pour utiliser le slug `demo-player` et le `baseURL` Playwright afin qu'il tourne en CI.
