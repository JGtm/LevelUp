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
- **Reprendre par C3** (backfill saisons passées, basse priorité).
  Ordre : B1→A2→A3→B2→**C1→C2** [FAITS] → **C3** [restant].
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

### C1 — LIVRÉ (2026-07-03, voie b frontend-only)
Delta placement saison précédente sur la page player (`CareerRankingBlock.tsx`, colonne CSR).
- 2e appel `useCareerCSRs(playerSlug, previousSeasonId, enabled)` — param `enabled` AJOUTÉ à
  `useCareerCSRs` (sinon collision de query key quand pas de saison antérieure). `previousSeasonId
  = availableSeasons[selectedIdx+1]` (tri desc backend confirmé, `sortCSRSeasonsDesc`).
- Join par `playlist_id`, helper pur `csrSeasonTrend` (compare `current.value`, null si non
  classé de part et d'autre) ; flèche ▲▼= via composant PARTAGÉ `components/ui/metric-trend.tsx`
  (extrait de LeaderboardBlock → 0 nouvelle copie) + garde-rail `metric-trend.guard.test.ts`.
- i18n : clé `career.ranking.vs_prev_season` (FR/EN) + regen `generated/career.ts`. Couleurs =
  tokens `--narrative-trend-*` (aucun hex).
- **Reporté [!]** : tri `is_active`-d'abord → exige un flag backend sur `CareerCSRRank`
  (full-stack). Liste triée par `alltime_value DESC` en attendant.
- Gate : `tsc -b` + `eslint .` (0 err) + `vitest run` HORS sandbox (237 fichiers / 2070 tests PASS).

### C2 — LIVRÉ (2026-07-03) : persister + surfacer les saisons
- **C2a (persistance)** : table `season_catalog` dans la SHARED DB (migration `create_season_catalog`,
  PK-only, ART-safe). Scraper parse `translations` + `FetchSeasons()`→`[]WorldSeasonRef` (FR résolu).
  `world_leaderboard_cron` upsert via `ops.RefreshSeasonCatalog` dans la même fenêtre writer que le
  snapshot CSR. Choix shared (pas metadata) : seul writer sanctionné du cron = writer shared ;
  metadata violerait le mono-process (cf. gotcha §3).
- **C2b (surfaçage)** : helper canonique `duckdb.SeasonSelectorLabel` + `LoadSeasonCatalogNames`
  réutilisé par le classement (`GetWorldLeaderboardCatalog`) et la page player (`AvailableCSRSeasons`).
  Libellé « Saison N · Nom » localisé (décision user). Front `LeaderboardBlock` : précédence inversée
  (backend autoritatif ; `KNOWN_SEASON_LABEL`/seasons.i18n.ts = secours offline). Match season_id
  casse-insensible (API "CsrSeason13-2" vs Waypoint "csrseason13-2"). Dégradation gracieuse
  (table absente → fallback dérivé).

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

Non bloquant (les gens regardent surtout la saison courante) ; à faire APRÈS C1/C2.

**VALIDÉ SUR SAMPLE (2026-07-03)** — pipeline testé de bout en bout sur csrseason12-1 (Shadows) :
snapshot 6 playlists (4→6, +Tactique +Duel 1v1) → 300 lignes avec xuid (B1) ; enrich 3 joueurs
via **pool de 7 tokens round-robin** (`-all-tokens`), xuid pré-seedé du snapshot (PeopleHub
court-circuité), filtre fenêtre-date saison actif → 6 lignes persistées, 0 erreur.

**Fix livré (les CLI étaient incomplets)** : `snapshot-world-leaderboard` et
`backfill-world-player-stats` n'appelaient PAS `migration.SetTitleStepsProvider` → `RunForDB`
ratait les migrations title-owned (add_xuid, create_season_catalog) sur une DB non pré-migrée
par le serveur → `InsertWorldCSRSnapshot`/`WorldSeasonPlayers` échouaient (colonne xuid absente).
Ajouté aux 2 CLI (auto-migration prouvée sur DB fraîche). Alternative manuelle si besoin :
`go run ./cmd/apply_shared_migrations -shared <db>`.

**Commandes du backfill COMPLET** (serveur ARRÊTÉ ; depuis `apps/go-api` ; `LEVELUP_REPO_ROOT`
= racine repo principal ; `DB` = chemin absolu shared_matches_v2.duckdb) :

```
# jeu complet des 16 playlists ranked (actives + historiques) :
PL="edfef3ac-9cbe-4fa2-b949-8f29deafd483,dcb2e24e-05fb-4390-8076-32a0cdb4326e,fa5aa2a3-2428-4912-a023-e1eeea7b877c,c94cb508-2fbd-450a-81db-bb74f7741d45,6233381c-fc96-40b9-b1ff-f6a4de72dd7a,57e417dd-7366-4dda-9bdd-2802151d5e81,71734db4-4b8e-4682-9206-62b6eff92582,28bfa5f4-89b0-47dc-86e8-1a7cc5b593fc,a4a4453c-7a91-4b27-b952-2456c5ce3205,6dc5f699-d6d9-41c4-bdf8-7ae11dec2d1b,f3738fae-bd09-4fd1-9dea-e32f546bbbfd,0b42053a-32c5-4c2d-b8b8-5f07274a0117,7c60fb3e-656c-4ada-a085-293562642e50,a883e7e1-9aca-4296-9009-3733a0ca8081,f7eb8c71-fedb-4696-8c0f-96025e285ffd,f7f30787-f607-436b-bdec-44c65bc2ecef"

# ÉTAPE 1 — re-scraper TOUTES les saisons × 16 playlists (public, sans token ; ~quelques min) :
# top 100 par playlist ; les playlists non classées d'une saison → 404 = skip nominal
# (affiché "— (non classée cette saison)", plus une ERROR depuis le fix 23f3c3c58).
go run ./cmd/snapshot-world-leaderboard -season all -shared-db "$DB" -playlists "$PL" -limit 100 -polite-ms 800

# ÉTAPE 2 — enrichir TOUTES les saisons via le POOL de tokens (LONG, off-peak ; checkpoint reprend) :
go run ./cmd/backfill-world-player-stats -token-gamertag JGtm -season all -all-tokens -deep -max-pages 120 -quiet
```

Notes : `-season all` = toutes les saisons (csrseason3-1 → active) — idempotent (append-only,
_latest gagne) ; la saison COURANTE est aussi re-couverte (inoffensif, le cron la gère aussi.
Étape 2 : `-all-tokens` OBLIGATOIRE pour le pool (sinon single-token) ; ~3 comptes ont un RT
périmé (XxDaemonGamerxX/Chocoboflor/Madina97294 — `invalid_grant`, NE PAS re-capturer, ADR 0023),
le pool tourne sur 7 comptes valides ; Ctrl-C = arrêt propre + reprise via checkpoint
`data/world_backfill_checkpoint.json` (relancer la MÊME commande). Durée étape 2 = plusieurs
heures (deep fetch × ~276 joueurs × ~14 saisons) — lancer en heures creuses. `season_catalog`
(C2) N'est PAS peuplé par ce backfill : c'est le cron serveur (world_leaderboard_cron) qui le
remplit — sans ça les libellés de saison retombent en fallback « Saison N » (dégradation OK).

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
7. **Worktree SANS node_modules** : `apps/web/node_modules` n'existe pas dans le worktree →
   i18n build / tsc / eslint / vitest échouent (`ERR_MODULE_NOT_FOUND`). Remède utilisé en C1 :
   jonction vers le repo principal (`cmd //c "mklink /J node_modules C:\...\apps\web\node_modules"`).
   RETIRER la jonction AVANT tout `git worktree remove` (cf. memory
   `reference_worktree_remove_follows_junctions`). Attention : lancer les commandes web depuis
   le chemin worktree, PAS `C:\...\LevelUp-go-migration\apps\web` (= repo principal).

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
