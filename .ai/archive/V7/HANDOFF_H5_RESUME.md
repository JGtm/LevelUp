# Handoff — Halo 5 (lecture + LUSR + KDA + PeopleHub + assets)

Doc de reprise (post-compaction). Branche **`feat/multititre-peripherie`** (worktree
`c:/Users/Guillaume/Downloads/Scripts/levelup-multititre`), **locale, ahead ~20, PAS de PR**.
Données runtime (db_profiles, tokens, data/titles/halo_5) = clone **main**
`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration` (split 2-clones : code+config
worktree, data+tokens main). Toolchain : `PATH=/c/msys64/ucrt64/bin`, `CGO_ENABLED=1`.

## LIVRÉ + committé (tout testé, `go build ./...` vert)
1. **Lecture h5** title-aware (match history/compare/stats/home/career déjà branchés) — validé live (10 matchs JGtm) via `cmd/h5-read-smoke`.
2. **LUSR v2 title-generic** branché pour h5 (`feat(lusr) 4cd2f3f84`) : classifier title-aware (`SetLUSRChainClassifierForTitle`/`GetLUSRChainForTitle`, lu via `ctxkeys.TitleSlug`), classifier h5 `halo_5/lusr_chain.go` (chaîne unique `h5_arena`), fix filtre `start_time→COALESCE(start_time_utc,start_time)`, cap registry → `title.DefaultRegistry()`. Test `TestRunLUSRV2Shadow_Halo5_TitleAwareChain` vert. v2 marche sur données BASIQUES (k/d/a/outcome/time, **sans MMR**).
3. **KDA = calculé à l'INGESTION** (`f0b8d6758`) — RÈGLE : h5 est le SEUL titre où on calcule le KDA, **à l'ingestion**, jamais en lecture. Forme = **FDA NET `(k+a/3)−d`** (peut être négatif). `mapping_carnage` (par match) + `mapping_servicerecord` (carrière `/games`) le stockent ; reads (`compare_repo` AVG(mp.kda), `BuildSampleStats` FDA NET moyen) lisent le stocké. Infinite garde son quotient API.
4. **Exclusion Warzone** (`f0b8d6758`) : `capture.isExcludedH5GameMode` (GameMode 4) — produit + anti-storm PeopleHub.
5. **PeopleHub optimisé** (`3ff2afe55`) : (a) ingest écrit `shared.xuid_aliases` (`AddXUIDAliases`) = mapping xuid↔gamertag au même endroit ; (b) resolver amorcé depuis xuid_aliases (`loadXUIDAliasesSeed`) = pas de re-résolution ; (c) **multi-compte round-robin** (`BuildMultiResolver` sur tous les comptes) = ~N× le quota. (Warzone exclu = 8 vs 24 joueurs.)

## FAIT CONFIRMÉ (Chrome DevTools sur Halowaypoint, match Arena JGtm `5d16ff8d`)
Le carnage `spartanstats.svc/h5/arena/matches/{id}?view=full` (MÊME endpoint+token v4 que nous) renvoie **`"Xuid":null` pour TOUS les joueurs** → **aucun xuid dispo dans la donnée match h5**. Halowaypoint lui-même affiche les gamertags (seul xuid de la page = le self via `profile.svc/users/me`). ⇒ PeopleHub est l'UNIQUE voie ; l'optim ci-dessus est la bonne.

## RESTE
- **Valider LUSR live** : re-sync Arena (`LEVELUP_REPO_ROOT=…/LevelUp-go-migration go run ./cmd/h5-sync JGtm 30` — devrait ne plus storm grâce au multi-compte+Warzone-exclu ; sinon le quota PeopleHub était encore en cooldown, réessayer) PUIS `go run ./cmd/h5-lusr-backfill JGtm` (provisionne player DB h5 + écrit `match_skill_rank` canonical lu par l'UI). Vérifier μ/σ + nb lignes. Outils : `cmd/h5-lusr-smoke` (shadow état), `cmd/h5-lusr-backfill` (canonical).
- **Assets h5** (minimum prod user : médailles, cartes, images rang CSR, images rang XP, armes) — **pas récupérés** (pas de `metadata.duckdb` h5). Plan dans workflow scoping (cf. thought_log) : **ROOT FIX d'abord** = enregistrer un `TitleMigrationSet` metadata h5 (`internal/games/halo_5/migrations` + `migration.RegisterMigrationSet`) SINON provisionner la metadata h5 injecte les rangs/playlists/CSR/prestige **d'Infinite** (pollution). Puis quick wins (playlists-from-registry, mode_name_tr) + **sous-projet** : fetchers CMS Halo 5 (ContentHacs `content-hacs.svc` médailles/icônes ; UGC `ugc.svc` maps ; référentiel tiers CSR ≠ ladder 272 rangs Infinite). Catalogues d'IDs présents dans les réponses Halowaypoint (`halo5api.svc/fr-fr/medals|weapons|spartan-ranks|csr-designations|maps|game-variants`) — utiles pour mapper.
- **KDA nullable compare/explorer** (cosmétique : h5 affiche `0` au lieu de `—` quand pas de données ; `domain` `float64`→`*float64` + front + SDK) — basse priorité, la règle no-fabrication est déjà respectée.

## Garde-fous
- Ne jamais re-fabriquer un KDA façon Infinite pour h5 (forme = FDA NET, calculée à l'ingestion).
- Sélection runner/title : registry-driven (`livesync.HandlesTitle`/`ProvidesNativeKDA`), JAMAIS `slug==` literal (archlint `no_slug_comparison` ; comparer à `halo5.TitleSlug` const OK).
- Agents de workflow : les ÉPINGLER au worktree (bug cwd rencontré : un workflow a lu le clone main = mauvaise branche → verdict « fictif » faux).
