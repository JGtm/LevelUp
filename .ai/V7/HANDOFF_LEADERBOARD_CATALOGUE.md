# HANDOFF — Leaderboard xuid + Catalogue playlists dynamique + saisons

> Point d'entrée UNIQUE pour reprendre après compaction. Détail complet :
> `.ai/V7/PLAN_PLAYLISTS_CATALOG_ET_LEADERBOARD.md`. Journal : `.ai/thought_log.md`
> (entrées 2026-07-02). Déclencheur initial : comparaison avec LeafApp_Infinite
> (iBotPeaches) — l'utilisateur voyait « peu de playlists actives » sur la page classement
> + des trous de données, et se demandait la logique des saisons (« X-Infinite »).

## 0. Où reprendre / état

- **Worktree** : `.claude/worktrees/feat+leaderboard-catalogue` (créé depuis `main` local).
  Branche `feat/leaderboard-xuid-et-catalogue-dynamique`, **8 commits poussés** sur origin
  (branche feature → AUCUN déploiement ; seul un push sur `main` déploie).
- **WIP séparé intact** : le repo principal est sur `fix/security-unauth-endpoints` (chantier
  sécurité non lié) — ne pas y toucher.
- **Reprendre par C1** (frontend), puis **C2** (saisons), puis **C3** (backfill saisons passées,
  basse priorité). Ordre : B1→A2→A3→B2 [FAITS] → **C1 → C2 → C3** [restants].
- **Contrat d'exécution** : skill `plan-execution` (ordre strict, pas de report d'action
  exécutable, statuer chaque item, gate + thought_log + commit par étape).

## 1. FAIT (backend complet — les 2 plaintes utilisateur réglées)

| Commit | Étape | Effet |
|---|---|---|
| `d58528501` | **B1** | Le scraper Waypoint parse déjà le xuid mais le persister le jetait. Colonne `xuid` ajoutée à `world_csr_leaderboard_snapshots` (migration `add_xuid_to_world_csr_leaderboard`) + persistée ; l'enrichissement pré-seed `xuidByGamertag` → PeopleHub court-circuité (fin des trous). Bonus : `isLocalXUID` réparé. |
| `dfce681f4` | **A2** | Le cron classement ne scrapait que `rankedplaylists.Active()` (4 en dur) → la page (qui dérive des snapshots) n'affichait que 4. Il découvre maintenant les playlists actives RÉELLES via `LeaderboardScraper.FetchActivePlaylists` (menu Waypoint), fallback statique. |
| `fddf71f5c` | **A3** | L'augment CSR post-sync (`career.go`) itérait 4 playlists en dur ; il lit maintenant les actives du dernier batch `world_csr_leaderboard_snapshots` via `SyncEngine.activeRankedPlaylists` (season-agnostic, fallback `Active()`). |
| `7a49ab0a0` | **B2** | Voir §3. Hardening par-match de l'enrichissement (un match illisible n'annule plus tout le joueur). |
| `b14e83941`,`e76c62cf3`,`3c2fe84b7`,`019f1d501` | docs | Plan + thought_log + suppression artefacts sonde. |

Gate systématique : `go build ./...` + `gofmt` + go-vet hook + tests unitaires/intégration verts.

## 2. RESTE À FAIRE

### C1 — Frontend : delta placement saison précédente (page player)
- **Fichier** : `apps/web/src/features/career/CareerRankingBlock.tsx` (colonne CSR).
- **Voie b (frontend-only, recommandée)** : 2e appel `useCareerCSRs(playerSlug, previousSeason)`
  où `previousSeason = availableSeasons[1]?.season_id` ; join par `playlist_id` ; afficher un
  delta ↑/↓ du tier/value courant vs précédent. Pattern flèche/tokens : voir `LeaderboardBlock.tsx`
  (`TREND_VAR`/`MetricWithTrend`).
- **Item 1 (tri actives-d'abord)** : nécessite un flag `is_active` par playlist dans la réponse
  CSR carrière — ABSENT de `CareerCSRRank` (champs : badge_image_url, measurement_matches_remaining,
  placement_total, sub_tier, tier, value). → full-stack (backend `career_service` + type). À part.
- **Friction** : nouvelle clé i18n (career.toml n'a PAS de clé trend/delta réutilisable) → éditer
  `apps/web/src/lib/i18n/manifests/career.toml` + REGÉNÉRER `generated/career.ts` (build i18n) ;
  gate `make check-types` (sandbox OK) + `make test-web` (**vitest HORS sandbox** :
  `dangerouslyDisableSandbox=true`, cf. memory `reference_vitest_outside_sandbox`).

### C2 — Saisons : persister + surfacer
- **Enquête DÉJÀ LIVRÉE** (fixture `leaderboard_sample.html`) : `seasonId`=identifiant CSR,
  `displayName`=nom d'Operation. `csrseason13-2`→"Infinite", `csrseason12-1`→"Shadows"(FR "Ombres"),
  `csrseason11-1`→"Last Stand"(FR "Dernier bastion"). `translations` par locale DANS le payload
  Waypoint. « X-Infinite » = `csrseasonX-Y` (n° CSR) + Operation.
- **Reste** : persister la liste (table `season_catalog` ou colonnes ; upsert ART-safe SELECT-then-write,
  cf. `internal/ops/catalog_refresh.go`). Le scraper renvoie déjà `seasons` via `FetchCatalog`
  (jeté chez l'appelant `snapshot-world-leaderboard:147`). Surfaçage DÉJÀ partiel :
  `useLeaderboardCatalog` (leaderboard) + `availableSeasons` (career) exposent `display_name`.
  C2 n'ajoute que la persistance autoritative noms+traductions. Multi-titre : gate capability, pas slug.

### C3 — Backfill des saisons PASSÉES (à faire ; basse priorité, fenêtre dédiée)
Les corrections (A2 : 3 playlists en plus ; B1 : xuid ; B2 : hardening) s'auto-appliquent à la
saison COURANTE via le cron, mais les saisons PASSÉES sont figées → backfill manuel pour combler
les mêmes trous rétroactivement. **2 étapes, serveur ARRÊTÉ + tokens frais** (les CLI ouvrent la
shared DB en RW direct) :

1. **Re-scraper les leaderboards** des (saison passée × playlists nouvellement actives) via
   `cmd/snapshot-world-leaderboard` — scrape HTML public, **pas de token**. Uniquement pour les
   playlists qui avaient un classement classé à cette saison (sinon rien à scraper).
2. **Re-lancer l'enrichissement** par saison via `cmd/backfill-world-player-stats` — **utilise
   DÉJÀ le pool de tokens partagés** (`worldenrich.BuildMultiHaloSource` + `BuildMultiResolver`,
   PolicyAnyPublic round-robin) et **embarque déjà** les corrections : lit les joueurs via
   `WorldSeasonPlayers` (xuid B1 pré-seedé → PeopleHub court-circuité) et construit l'agrégateur
   durci (B2, `NewWorldStatsAggregator` l.313). → « **faut juste l'adapter** » = surtout s'assurer
   que l'étape 1 a peuplé les snapshots des nouvelles playlists ; l'enrichissement couvre ensuite
   toutes les playlists jouées automatiquement (agrégation par-match). Vérifier les flags saison
   (`--season`, checkpoint) et, si besoin, le format saison CMS validé (`Csr/Seasons/CsrSeasonX-Y.json`).

Non bloquant (les gens regardent surtout la saison courante) ; à faire APRÈS C1/C2. Préparer les
commandes exactes (chemins CMS de saison validés) au moment de lancer.

## 3. B2 — ce qui a été prouvé (NE PAS refaire l'erreur)

- **Sonde live (autorisée)** a établi : l'endpoint service-record `/hi/players/xuid(N)/Matchmade/servicerecord`
  accepte `seasonId` **au format chemin CMS** `Csr/Seasons/CsrSeason13-2.json` (PAS le format
  Waypoint `csrseason13-2` qui 404). JGtm = 367 matchs saison, CoreStats complets.
- **`playlistAssetId` NON SUPPORTÉ** : AUCUNE des 16 playlists ne renvoie de données malgré 367
  matchs. Le service-record ne donne QUE l'agrégat par SAISON → impossible de peupler
  `world_player_season_stats` (clé saison×playlist). **Design "1 SR/(joueur,playlist)" MORT.**
- **Pivot livré** : hardening de `collectPlayerMatches` (`world_player_stats_aggregator.go`) — match
  illisible (403/404/timeout après retries) → `continue` (ignoré) au lieu d'annuler le joueur ;
  erreur historique après collecte partielle → garder le partiel ; compteur `world_enrich.match_skipped`.

## 4. GOTCHAS / invariants à respecter

1. **Page classement = source snapshots, pas catalogue** : le sélecteur playlists dérive de
   `SELECT DISTINCT playlist_id FROM world_csr_leaderboard_latest` (`leaderboard_world_repo.go:267`),
   PAS de `playlists_catalog`. Donc « rendre les playlists visibles » = les faire scraper par le cron (A2).
2. **`playlists_catalog` SANS index secondaire** (ratchet `playlists_catalog_no_index_test.go`) :
   UPDATE OK sans index ; ne JAMAIS recréer `idx_playlists_catalog_active` (corruption ART).
3. **metadata.duckdb = writer mono-process** (ADR 0013/0016) : NE PAS écrire `is_active` depuis le
   cron (contention avec la sync). A3 lit donc les actives depuis les snapshots shared, pas le catalogue.
4. **Manifest de build discovery = PlaylistLinks VIDE** (OpenSpartan wiki) : ne PAS repartir sur
   l'idée d'énumérer les playlists via le manifest. Source = `FetchCatalog` (menu Waypoint).
5. **Tokens (ADR 0023)** : toute sonde/CLI qui rote un RT DOIT persister le RT roté
   (`store.Upsert`) sinon churn/invalid_grant (incident vécu, JGtm). Vérifier bannière reauth JGtm ;
   diagnostiquer avant re-capture. Tokens dans `data/auth/watcher_tokens/` (repo principal, partagé
   par le worktree). Harnais auth de référence : `cmd/probe-world-stats/main.go` (lignes 78-126).
6. **CGO requis** (driver DuckDB, gcc msys64). Tests : jamais `-race` (incompatible driver).
   `go test -race` → utiliser `-gcflags=all=-d=checkptr=0` si besoin.

## 5. Reprise concrète

```bash
# Depuis le worktree :
cd .claude/worktrees/feat+leaderboard-catalogue
git log --oneline -8            # confirmer les 8 commits
# C1 : éditer CareerRankingBlock.tsx + career.toml + regen i18n
make check-types && make generate-types   # (vérifier la commande de regen i18n dans le Makefile)
# gate front : make check-types (sandbox) + make test-web (HORS sandbox)
# Backend inchangé : cd apps/go-api && go build ./... && go test ./internal/... -run 'World|CSR|Career'
```

- **Ne pas re-décider** ce qui est tranché : B2 par-playlist est MORT (prouvé) ; A3 lit les
  snapshots (pas le catalogue metadata) ; A2 = cron discovery, pas manifest.
- Chaque étape close : gate + statuer items du plan + entrée thought_log + commit préfixé (C1…/C2…)
  + point à l'utilisateur. Pas de push `main`.
