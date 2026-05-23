# Catalogue des enrichissements locaux

> Référence de tous les champs calculés/dérivés qui ne viennent pas directement de l'API Halo.
> Mis à jour : 2026-05-23 (v2 — passe exhaustive post-schema.go + migrations)

Deux catégories :
- **Persistés** : écrits dans DuckDB pendant ou après le sync.
- **Calculés à la volée** : dérivés en Go lors du rendering, jamais stockés.

---

## 1. `player_match_enrichment` — player DB

Table pivot du joueur. Un row par match.

### performance_score

- **Valeur** : float 0–100 (percentile relatif au joueur)
- **Calcul** : `batchComputePerformanceScores` — percentile pondéré sur fenêtre glissante de 50 matchs **par chaîne**. Métriques : KPM, DPMDeaths (inversé), APM, KDA, Accuracy, PSPM, damage/min, kills/expected, deaths/expected, offensive_conversion, defensive_resistance, medal_exploit. Minimum : `MinMatchesPerChainForRelative` matchs dans la chaîne.
- **Fichier** : `internal/sync/performance.go`
- **Consommateurs** : match history, career top matches, home citations, compare, squad

### performance_chain

- **Valeur** : string — `ranked_slayer` | `arena_slayer` | `btb` | `tactical` | `social` | `firefight`
- **Calcul** : `GetPerformanceChain(pair_name, is_ranked, is_firefight)` — classification pour isoler les contextes de performance hétérogènes (ex : BTB ne se compare pas à ranked slayer)
- **Fichier** : `internal/sync/performance.go`
- **Consommateurs** : interne uniquement (sert à savoir si le score existant est encore valide si la classification change)

### session_id / session_label

- **Valeur** : int / string (ex : `1`, `"Session 1"`)
- **Calcul** : `RecalculatePlayerSessions` — groupement des matchs chronologiques par gap temporel + teammates_sig. Un nouveau groupe = nouvelle session.
- **Fichier** : `internal/sync/session_recalc.go`
- **Consommateurs** : match history (filtre/tri), career timeline, home citations

### is_with_friends

- **Valeur** : bool
- **Calcul** : `RecomputeIsWithFriendsCore` — `TRUE` si au moins un xuid de `settings.friend_gamertags` est dans la même team que le joueur (via shared `match_participants`). Additif : ne démote jamais un match déjà `TRUE`.
- **Fichier** : `internal/sync/friends_recompute.go`
- **Consommateurs** : squad analysis, match history filter, career stats

### had_bot_teammate

- **Valeur** : bool
- **Calcul** : `computeAndPersistHadBotTeammate` — `TRUE` si un coéquipier a un xuid commençant par `bid(` (pattern bots Halo)
- **Fichier** : `internal/sync/enrichments.go`
- **Consommateurs** : match history (badge + filtre potentiel)

### is_excluded

- **Valeur** : bool (default `FALSE`)
- **Calcul** : action utilisateur via `PATCH /players/{slug}/matches/{id}/exclusion`
- **Fichier** : `internal/platform/duckdb/match_exclusion_repo.go`
- **Consommateurs** : tous les filtres match history + performance_score (matchs exclus ne pèsent pas dans la fenêtre de calcul)

### dominance_flag

- **Valeur** : TINYINT — `0` Aucun | `1` Domination | `2` Humiliation | `3` Remontada | `4` Débâcle | `5` Contre-remontada
- **Calcul** : `BackfillDominanceFlags` — reconstruit la courbe de score depuis `highlight_events` (kill events + team_id) et détecte les patterns d'avance/retard. Déclenché aussi par la médaille Steaktacular (ID `1169390319`).
- **Fichier** : `internal/sync/comeback.go` + `internal/analysis/comeback.go`
- **Consommateurs** : match history (badge narratif), career top matches (sort priority)

### teammates_signature

- **Valeur** : string (CSV des xuids coéquipiers triés)
- **Calcul** : construit lors de `UpsertPlayerEnrichment` depuis `string_agg(xuid ORDER BY xuid)`
- **Fichier** : `internal/sync/writes.go`
- **Consommateurs** : session_recalc (input du groupement de sessions)

### known_teammates_count / friends_xuids

- **Statut** : colonnes reservées dans le schéma (`schema.go:44-45`), **jamais peuplées** par le code actuel.
- **Usage prévu** : comptage et liste des amis connus par match, probablement pour l'affichage "N amis dans ce match".
- **Action** : ne pas implémenter sans supprimer ces colonnes ou écrire le remplissage.

---

## 2. `match_skill_rank` — player DB

Un row par match. Exclusif : un match porte soit LUSR, soit CSR, jamais les deux.

### LUSR (rating_type = 'LUSR')

- **Valeur** : `rating_value` float, `rating_deviation` float (sigma TrueSkill), `tier_label` string, `playlist_group` string, `rating_delta` float
- **Calcul** : `batchComputeLUSR` — TrueSkill Elo-style séquentiel par `playlist_group`. `composite_score = kills×0.31 + deaths×0.28 + damage×0.23 + accuracy×0.13 + win×0.05`. `delta_mu = K_ELO(32) × (composite − 0.5) × weight_factor`. Seulement les matchs non-ranked (`is_ranked = FALSE`).
- **Fichier** : `internal/sync/halo_skill.go`
- **Consommateurs** : match history (rating badge), career skill progression, compare, leaderboard

### CSR (rating_type = 'CSR')

- **Valeur** : `rating_value` int, `tier` string EN, `tier_fr` string FR, `sub_tier` int, `tier_label` string formaté, `rating_delta` float, `playlist_group` string
- **Calcul** : `ExtractCSRRowIfRanked` — lit `RankRecap.PostMatchCsr` depuis le payload skill Halo (`/hi/matches/{id}/skill`). Seulement les matchs ranked (`is_ranked = TRUE`). Gère le cas placement (`MeasurementMatchesRemaining > 0`).
- **Fichier** : `internal/sync/csr_writes.go`
- **Consommateurs** : match history (badge CSR), career, home (pic CSR/LUSR dans spartan identity)

---

## 3. `player_csr_snapshots` — player DB

Snapshot officiel Microsoft du CSR du joueur par playlist et par saison.

- **Valeur** : PK `(playlist_id, season_id)`. Colonnes : `current_value/tier/sub_tier/measurement_remaining`, `season_value/tier/sub_tier`, `alltime_value/tier/sub_tier`, `playlist_name`, `queue`, `input`, `fetched_at`
- **Calcul** : `saveCSRSnapshots()` — lit `syncPlayerCSRs()` → endpoint Halo skill rankings → liste de `PlayerPlaylistCSR`. Persisté à chaque sync post-sync (best-effort, skippé si `csrSeasonID` vide).
- **Fichier** : `internal/sync/career.go`, `internal/sync/engine_postsync.go`
- **Consommateurs** :
  - Home : `loadCSRAlltimePeak` → `alltime_value` comme source de vérité du pic CSR officiel Waypoint (priorité sur `match_skill_rank`)
  - Career : `GetCSRSnapshots` → historique par playlist
  - Diagnostic : `CSRSnapshotsCoverage` (couverture des playlists trackées)

---

## 4. `match_citations` — player DB

Cumul des "réalisations" par match (médailles-objectif agrégées).

### match_citations

- **Valeur** : `citation_name_norm` string, `value` int (delta), cumul calculé au runtime
- **Calcul** : `BackfillMatchCitations` — pipeline multi-source :
  1. Règles depuis `metadata.citation_mappings`
  2. Médailles depuis `shared.medals_earned`
  3. Stats depuis `shared.match_participants` + weapon_kills
  4. Awards depuis `player.personal_score_awards`
  5. Events depuis `shared.highlight_events`
  → `ComputeFullMatchCitations` (analysis) → deltas écrits dans `match_citations`
- **Fichier** : `internal/sync/citations.go`
- **Consommateurs** : page Citations (totaux agrégés `Q35`), match view citations `Q38`, home citations `Q26i/Q26j`

---

## 4. `personal_score_awards` — player DB

Awards d'objectifs depuis l'API PersonalScores Halo.

### personal_score_awards

- **Valeur** : `award_name`, `award_category`, `award_count`, `award_score`
- **Calcul** : `transforms_personal_scores.go` — extrait les `PersonalScores` du JSON match et les insère. Remplace atomiquement (DELETE + INSERT) pour idempotence.
- **Fichier** : `internal/sync/transforms_personal_scores.go`, `internal/sync/writes.go`
- **Consommateurs** : match view (score objectif par catégorie), citations (input du pipeline BackfillMatchCitations)

---

## 5. `career_progression` — player DB

Historique des rangs de carrière (snapshots).

### career_progression

- **Valeur** : `rank`, `current_xp`, `xp_for_next_rank`, `spartan_id`, `emblem_image_url`, `backdrop_image_url`, `banner_image_url`
- **Calcul** : `GetCareerRank()` — appel API Halo + résolution assets GameCMS. Persisté à chaque sync.
- **Consommateurs** : home (spartan identity, progression de rang), career live data, compare (profil joueur)

---

## 6. `media_files` / `media_match_associations` — player DB

Index des captures et vidéos.

- **Calcul** : `MediaIndexer.scan_and_index()` — scan delta récursif, ffprobe/EXIF, génération de thumbnails
- **Consommateurs** : galerie médias (liste, likes, associations match)

---

## 7. Tables shared DB — enrichissements calculés

### medals_earned — `shared_matches_v2.duckdb`

- **Valeur** : `medal_name_id` (int64), `count` par (match, xuid)
- **Calcul** : `ExtractMedals` — extrait `CoreStats.Medals[]` du JSON match pour tous les joueurs
- **Fichier** : `internal/sync/transforms.go`
- **Consommateurs** : match view (liste médailles), home (médailles récentes `Q26h`), citations (input pipeline)

### killer_victim_pairs — `shared_matches_v2.duckdb`

- **Valeur** : `killer_xuid`, `victim_xuid`, `time_ms` — 1 row par kill event
- **Calcul** : `InsertKillerVictimPairsFromEvents` — corrélation kill/death dans `highlight_events` avec tolérance ±5ms (`ComputeKillerVictimPairs`)
- **Fichier** : `internal/sync/writes.go`
- **Consommateurs** : compare (head-to-head kills), match scoreboard

### weapon_kills — `shared_matches_v2.duckdb`

- **Valeur** : `weapon_id` (UBIGINT), `time_ms`, `confidence`, `delta_ms`, `attribution_path` — 1 row par kill
- **Calcul** : film REPLICATION_DATA — corrélation kill → dernier événement `fire` dans fenêtre 2000ms. POV coverage ~87,5%. Backfill uniquement (`--weapons`).
- **Fichier** : `internal/sync/` (weaponkills pipeline)
- **Consommateurs** : match view (arme favorite), compare (weapon stats), scoreboard, citations (input)

### xuid_aliases — `shared_matches_v2.duckdb`

- **Valeur** : `xuid → gamertag` + `last_seen`
- **Calcul** : `UpsertXUIDAlias` — mis à jour à chaque sync depuis le gamertag extrait du JSON. Bots normalisés (`bid(3.0)` → `"343 Bot 3"`).
- **Fichier** : `internal/sync/writes.go`
- **Consommateurs** : résolution gamertag partout (media, squad, compare, leaderboard)

### mode_category — `match_registry` (shared)

- **Valeur** : string — `ranked_slayer` | `btb` | `tactical` | `social` | `firefight` | `other`
- **Calcul** : `determineModeCategory(pair_name)` — classification à partir du nom de pair au moment de l'insertion registry
- **Fichier** : `internal/sync/transforms.go`
- **Consommateurs** : engagement scoring (coefficients team/lobby share)

### match_csrs — `shared_matches_v2.duckdb`

Table ajoutée par migration (`steps_shared_match_csrs.go`). Stocke le CSR de **tous** les participants d'un match ranked (pas seulement le joueur synchronisé).

- **Valeur** : PK `(match_id, xuid)`. Colonnes : `rating_value`, `tier`, `sub_tier`, `tier_label`, `rating_delta`, `measurement_matches_remaining`, `season_id`
- **Calcul** : `UpsertSharedCSRs` — lors du sync d'un match ranked, extrait le CSR de chaque participant depuis le payload skill Halo et batch-insère dans cette table. Backfill disponible via `--shared-csr`.
- **Fichier** : `internal/sync/csr_shared_writes.go`
- **Consommateurs** : données CSR adversaires/coéquipiers dans match view et compare (contexte lobby)

### pve_match_stats — `shared_pve.duckdb`

Stats Firefight (PvE) par joueur par match.

- **Valeur** : `waves_completed`, `boss_kills`, kills par type d'ennemi : `grunt_kills`, `elite_kills`, `jackal_kills`, `brute_kills`, `hunter_kills`, `skimmer_kills`, `crawler_kills`, `soldier_kills`, `knight_kills`, `warden_kills`, `sentinel_kills`, `marine_kills`, `total_enemy_kills`, `pve_bits` (bitmask de complétion par colonne)
- **Calcul** : `extract_pve_stats()` — depuis le JSON match quand `_is_firefight_match()` retourne `TRUE` (GameVariantCategory 41/42). Backfill via `--pve`.
- **Consommateurs** : pages stats Firefight, breakdown PvE, match view Firefight

---

## 9. `player_assists_model` — player DB / `assists_model_coefs` — metadata DB

Coefficients OLS pour le calcul de `expected_assists` par mode de jeu.

### player_assists_model (player DB)

- **Valeur** : PK `game_variant_name`. Colonnes : `intercept`, `coef_kills`, `coef_deaths`, `coef_damage_dealt`, `coef_damage_taken`, `coef_mmr_delta`, `r2`, `n_samples`
- **Calcul** : `RunBackfillAssistsModel` — régression linéaire OLS 6-features résolue par élimination gaussienne. Entraînée sur l'historique du joueur dans `shared.match_participants`. Seuil minimum : 15 matchs par mode.
- **Fichier** : `internal/sync/assists_model.go`
- **Fallback** : si moins de 15 matchs dans un mode → utilise `assists_model_coefs` (modèle populationnel dans metadata DB)
- **Consommateurs** : `computeExpectedAssists()` dans `match_view_builders_summary.go` → exposé dans match view comme `expected_assists`

### assists_model_coefs (metadata DB)

- **Valeur** : coefs populationnels par `mode_category` — mêmes colonnes que `player_assists_model`
- **Calcul** : seedé manuellement via `cmd/seed-assists-model`
- **Consommateurs** : fallback de `player_assists_model` quand le joueur n'a pas assez d'historique dans un mode

### expected_assists (à la volée)

- **Statut** : non stocké — calculé au runtime dans le service layer
- **Formule** : `β0 + β1·kills + β2·deaths + β3·damage_dealt + β4·damage_taken + β5·mmr_delta`
- **Note** : `assists_expected` / `assists_stddev` ont été droppés de `match_participants` par migration (`drop_assists_expected_halo_infinite`) — l'API Halo Infinite ne fournit jamais ces valeurs (contrairement à `kills_expected` / `deaths_expected` qui viennent du skill endpoint Waypoint).
- **Consommateurs** : match view (comparaison assists réels vs attendus), scoreboard équipe

---

## 10. Métriques calculées à la volée (non stockées)

### offensive_conversion / defensive_resistance

- **Formule** :
  - `offensive_conversion = 225 × (kills + assists/3) / damage_dealt`
  - `defensive_resistance = damage_taken / (225 × max(deaths, 1))`
- **Fichier** : `internal/analysis/combat_yield.go`
- **Consommateurs** : home KPIs, match view, performance_score (input du calcul percentile)

---

## Bitmasks de tracking

### `match_registry.backfill_completed`

Deux couches : bits 0-15 (legacy `BackfillFlags`) + bits ≥16 (`MatchBits`).

| Valeur | Constante | Signification |
|--------|-----------|---------------|
| 1 (`1<<0`) | `medals` | medals_earned peuplées |
| 2 (`1<<1`) | `events` | highlight_events chargés |
| 4 (`1<<2`) | `skill` / `backfillFlagSkill` | Données skill (MMR) chargées |
| 8 (`1<<3`) | `personal_scores` | personal_score_awards peuplées |
| 32 (`1<<5`) | `accuracy` | accuracy backfillée |
| 64 (`1<<6`) | `shots` | shots_fired/hit backfillés |
| 128 (`1<<7`) | `enemy_mmr` | enemy_mmr backfillé |
| 512 (`1<<9`) | `participants` / `backfillFlagParticipants` | Participants re-synced |
| 65536 (`1<<16`) | `MBitEvents` | highlight_events chargés (nouveau bit) |
| 524288 (`1<<19`) | `MBitKillerVictim` | killer_victim_pairs calculés |
| 1048576 (`1<<20`) | `MBitPVEStats` | pve_match_stats tentées |
| 2097152 (`1<<21`) | `MBitWeaponKills` | weapon_kills chargés (film OK) |
| 4194304 (`1<<22`) | `MBitWeaponKillsNoFilm` | film 404/expiré, 0 chunk dispo |

### `match_participants.backfill_bits` (ParticipantBits)

Granularité par joueur × match. Indique quelles colonnes sont fiables.

| Valeur | Constante | Colonne(s) |
|--------|-----------|------------|
| 1 | `PBitTeamMMR` | `team_mmr` |
| 2 | `PBitEnemyMMR` | `enemy_mmr` |
| 4 | `PBitKillsExp` | `kills_expected`, `kills_stddev` |
| 8 | `PBitDeathsExp` | `deaths_expected`, `deaths_stddev` |
| 32 | `PBitAccuracy` | `accuracy` |
| 64 | `PBitShots` | `shots_fired`, `shots_hit` |
| 128 | `PBitDamage` | `damage_dealt`, `damage_taken` |
| 256 | `PBitAvgLife` | `avg_life_seconds` |
| 512 | `PBitMedals` | présence dans `medals_earned` |
| 65536 | `PBitKDA` | `kda` (calculé) |
| 131072 | `PBitTimePlayed` | `time_played_seconds` |
| 262144 | `PBitKillerVictim` | présence dans `killer_victim_pairs` |

### `pve_match_stats.pve_bits` (PveBits)

Granularité par joueur × match Firefight. Indique quelles colonnes de kills sont renseignées.

| Valeur | Constante | Colonne |
|--------|-----------|---------|
| 1 | `PveBitTotalKills` | `total_enemy_kills` |
| 2 | `PveBitBossKills` | `boss_kills` |
| 4–128 | `PveBitGrunt…Skimmer` | `grunt/elite/jackal/brute/hunter/skimmer_kills` |
| 256–8192 | `PveBitCrawler…Marine` | `crawler/soldier/knight/warden/sentinel/marine_kills` (Forerunner/alliés) |
