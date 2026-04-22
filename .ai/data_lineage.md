# Data Lineage - Traçabilité des Données Halo

> Ce fichier trace l'origine, les transformations et la destination de chaque flux de données.
> Mis à jour : 2026-04-18

> Note bugfix 2026-04-21 : aucun flux de données modifié. Le correctif porte uniquement sur le cycle de vie des connexions DuckDB partagées (`metadata.duckdb`) afin d'éviter qu'une lecture temporaire des défis ferme la connexion réutilisée ensuite par le season pass/home.

> Note bugfix 2026-04-21 bis : aucun flux de données modifié non plus sur cette passe. Le backend Go ne joint simplement plus `meta.*` depuis `stats.duckdb` ; les labels médailles/armes de la vue match sont lus directement via la connexion dédiée `PlayerDB.Metadata`, ce qui ne change ni les tables sources ni les destinations persistées.

> Note bugfix 2026-04-21 ter : pas de changement de schéma ni de destination persistée sur cette passe non plus. La home React lit maintenant les défis depuis `SeasonPassPageResponse.challenges` au lieu d'appeler aussi `/challenges`, et le provider Halo déduplique les fetchs live `/decks` concurrents ; les snapshots `challenge_snapshots` et le cache metadata restent inchangés.

> Note bugfix 2026-04-21 quater : aucun nouveau flux métier ni nouvelle persistance sur cette passe. Le handler Go des assets maps sert désormais d'abord `map_images_registry.local_path` pour les maps déjà indexées localement, et la home transporte `playlist_ui` ainsi qu'un `mode_ui` déjà normalisé sur les matchs récents pour piloter la présentation des tuiles React sans doublonner le nom de carte.

> Note bugfix 2026-04-21 quinquies : aucun flux persistant supplémentaire sur cette passe non plus. Le changement porte sur la valeur sérialisée de `recent_matches[].map_image_url` : pour les maps connues, la home publie maintenant directement `/static/maps/<Map>.<ext>` à partir du nom de map, avec fallback vers l'endpoint cache-aside UUID seulement si aucun asset local ne peut être déduit. `mode_ui` retire aussi les préfixes d'expérience (`Arena:`, `Community:`) avant rendu. Aucune table ni colonne n'est modifiée.

> Note bugfix 2026-04-21 sexies : aucun flux persistant ni schéma supplémentaire sur cette passe non plus. La source des labels home passe de `shared.match_registry` à `shared.v_match_full` pour les matchs récents, afin de consommer les variantes localisées déjà calculées (`map_name_fr`, `pair_name_fr`, `playlist_name_fr`). Le payload `recent_matches[]` choisit maintenant FR ou EN selon la langue active de l'application, mais aucune donnée nouvelle n'est écrite en base.

> Note home/record 2026-04-22 : aucun nouveau flux persistant non plus sur cette passe. La home lit désormais le dernier snapshot `career_progression` de la player DB (`spartan_id`, `rank`, `current_xp`, `xp_for_next_rank`) puis l'enrichit avec `metadata.career_ranks.title_en/title_fr` pour publier `spartan_identity` dans `/pages/home`. Le frontend React consomme cette payload pour rendre un bloc `Spartan ID` compact et une progression de rang carrière localisée, sans nouvelle écriture en base.

> Note home/record 2026-04-22 bis : toujours aucun nouveau flux persistant ni changement de schéma. L'enrichissement Home lit maintenant aussi `metadata.career_ranks.large_icon_path` / `adornment_icon_path` / `icon_path` pour publier `spartan_identity.career_rank.rank_image_url`. La destination reste uniquement la payload HTTP `/pages/home`; aucune donnée supplémentaire n'est écrite dans DuckDB.

> Note home/record 2026-04-22 ter : le correctif suivant change cette fois la BDD joueur de manière ciblée. `GetCareerRank()` lit la customisation publique du joueur (`ServiceTag`, `BackdropImagePath`, `EmblemPath`, `ConfigurationId`), résout les URLs d'images GameCMS, puis persiste `emblem_image_url` et `backdrop_image_url` dans `career_progression` en plus de `spartan_id`. Côté lecture, la Home ne dépend plus strictement du dernier snapshot : `Q26cHomeSpartanIdentity` retombe sur la dernière valeur non vide de `spartan_id` et des assets identitaires pour construire `/pages/home`.

> Note home/record 2026-04-22 quater : aucun nouveau schéma ni nouveau flux persistant sur cette passe. Le changement porte sur la destination HTTP publiée par la Home : `emblem_image_url`, `backdrop_image_url` et `rank_image_url` ne pointent plus directement vers GameCMS / Waypoint mais vers `/api/v1/assets/spartan/{image_type}/{title_id}/*`. La récupération distante et la persistance locale de ces binaires sont désormais mutualisées par `internal/assets` comme pour les maps, défis et battle pass.

> Note home/record 2026-04-22 quinquies : la passe suivante rétablit un flux distinct pour la bannière `nameplate`. `GetCareerRank()` lit la customisation publique Halo, essaie plusieurs clés candidates (`BannerImagePath`, `NameplateImagePath`, sous-objets `Banner`/`Nameplate`), puis persiste `banner_image_url` dans `career_progression` en plus de `emblem_image_url` et `backdrop_image_url`. La Home lit ensuite séparément la dernière valeur non vide de `banner_image_url` et la convertit en URL interne `/api/v1/assets/spartan/banner/{title_id}/*`, tandis que `backdrop_image_url` reste le fond du bloc identitaire.

> Note home/record 2026-04-22 sexies : la source réelle de la bannière legacy a ensuite été précisée via l'archéologie de `v7/cockpit`. Quand `player_title_path` est nul, le legacy Python reconstruisait une `nameplate` Waypoint depuis `emblem_path + configuration_id`. Le flux Go fait maintenant la même chose dans `GetCareerRank()` : `PlayerTitlePath` devient une clé candidate pour `banner_image_url`, et, en dernier recours, `banner_image_url` est dérivée sous la forme `hi/Waypoint/file/images/nameplates/<emblem_stem>_<configuration_id>.png`. Aucune nouvelle colonne n'est ajoutée ; seule la source d'alimentation de `career_progression.banner_image_url` est enrichie, ce qui permet à `/pages/home` de republier enfin une bannière réelle pour les profils resynchronisés.

> Note home/record 2026-04-22 septies : toujours aucun changement de schéma ni nouvelle persistance sur cette passe. La Home lit maintenant aussi `match_skill_rank` dans la player DB pour en extraire le meilleur enregistrement `CSR` et le meilleur enregistrement `LUSR` (`rating_value`, `tier_label`, `tier`, `sub_tier`), puis publie ces pics sous `spartan_identity.highest_csr` et `spartan_identity.highest_lusr`. Le badge associé est dérivé vers une URL statique `/static/ranks/120px-HINF-CSR_<Tier><SubTier>.png`; la destination finale reste uniquement la payload HTTP `/pages/home`.

> Note home/record 2026-04-22 octies : aucun nouveau schéma ni nouvelle persistance non plus sur ce correctif. La lecture produit ne prend plus `match_skill_rank.rating_type` comme vérité lorsqu'une ligne `shared.match_registry` existe : le type effectif (`CSR` ou `LUSR`) est désormais dérivé de `is_ranked` dans `Q22MatchSkillRank`, `Q26eHomeSkillPeakByType` et `Q26fHomeLastSkillRank`, avec fallback sur `rating_type` seulement si la métadonnée shared manque. La destination reste la payload HTTP `/pages/home` et la Match View ; aucun write path supplémentaire n'est ajouté.

> Note tooling dev 2026-04-20 : aucun flux de données modifié dans cette passe. Les changements portent uniquement sur la chaîne de démarrage locale Go/React : port API configurable via `API_PORT`, réutilisation d'une API déjà active, et proxy Vite dev paramétrable via `VITE_API_PROXY_TARGET`.

> Note tooling repo 2026-04-21 : aucun flux de données modifié non plus. Le nettoyage porte uniquement sur les points d'entrée et l'organisation du repo Go : suppression des wrappers `LevelUp.bat` / `LevelUp.sh` / `run.sh`, documentation réalignée sur `make dev`, et déplacement du script de déploiement vers `scripts/deploy.sh`.

> Note hygiene repo 2026-04-21 : aucun flux métier modifié. Seule la sortie de diagnostic de `migrate-static-maps` change d'emplacement, de la racine vers `data/investigation/maps/unmatched_maps.csv`, et les logs ponctuels maps sont considérés comme artefacts locaux.

> Note média 2026-04-18 : les likes de la galerie React sont désormais persistés dans `players/{gamertag}/stats.duckdb`, table `media_files`, colonnes `liked` et `liked_at`, via `PATCH /players/{player_slug}/media/likes`. `POST /players/{player_slug}/pages/media` renvoie aussi `liked` + `like_count` pour la home et la galerie.

> Note validation 2026-04-18 : aucune évolution de flux de données sur cette passe. Les correctifs portaient sur la stabilité des suites Go/React et sur l'alignement code/tests autour de `player_match_enrichment.is_excluded`, puis sur le rendu frontend des payloads nulles ou sections vides via des empty states explicites côté React. Un cadrage UX Carrière / Synthèse a aussi été formalisé côté go-migration, puis détaillé via un blueprint du hub Carrière, un blueprint contrat/UI de Synthèse et un cadrage d'ajouts inspirés de Spartan Record pour la home record. Ces documents décrivent des cibles de routes, de payloads et de surfaces UI, mais n'introduisent encore aucun changement de schéma, de flux ou de contrat API effectif.

> Note analyse 2026-04-18 : `.ai/go_migration_v2/DAMAGE_EFFICIENCY_INTEGRATION.md` formalise l'adoption potentielle d'une famille de metriques derivees des degats (`conversion offensive`, `resistance defensive`). A ce stade, il s'agit d'un cadrage analytique et produit uniquement : aucun nouveau flux, aucune nouvelle table et aucun contrat API supplementaire ne sont encore introduits.

## Architecture v5.1 - Shared Matches + Player Enrichments

```
┌─────────────────┐      ┌─────────────────────────────────────────────┐
│   API SPNKr     │      │              DuckDB Engine                  │
│  (Halo Infinite)│      │                                             │
└────────┬────────┘      │  ┌─────────────────────────────────────┐   │
         │               │  │  metadata.duckdb (global)           │   │
         ▼               │  │  - playlists, maps, game_modes      │   │
┌─────────────────┐      │  │  - medal_definitions, career_ranks  │   │
│  Pydantic v2    │      │  └─────────────────────────────────────┘   │
│  Validation     │      │                                             │
└────────┬────────┘      │  ┌─────────────────────────────────────┐   │
         │               │  │  shared_matches.duckdb (centralisée)│   │
         ▼               │  │  - match_registry (1 ligne/match)   │   │
┌─────────────────┐      │  │  - match_participants (31 col, MMR) │   │
│ DuckDBSyncEngine│──────│  │  - highlight_events                 │   │
│  Transformers   │      │  │  - medals_earned                    │   │
└────────┬────────┘      │  │  - killer_victim_pairs              │   │
         │               │  │  - xuid_aliases                     │   │
         │               │  └─────────────────────────────────────┘   │
         │               │                                             │
         │               │  ┌─────────────────────────────────────┐   │
         ├───────────────│  │  shared_pve.duckdb (Firefight) v5.2 │   │
         │               │  │  - pve_match_stats (waves, boss,    │   │
         │               │  │    kills par type d'ennemi)         │   │
         │               │  └─────────────────────────────────────┘   │
         │               │                                             │
         └───────────────│  ┌─────────────────────────────────────┐   │
                         │  │  players/{gt}/stats.duckdb          │   │
                         │  │  - player_match_enrichment (SEULE)  │   │
                         │  │  - personal_score_awards             │   │
                         │  │  - match_citations                   │   │
                         │  │  - career_progression, sessions      │   │
                         │  │  - media_files, media_match_assoc    │   │
                         │  │  - match_skill_rank (LUSR/CSR) v5.3 │   │
                         │  │  - mv_* (vues matérialisées)        │   │
                         │  └─────────────────────────────────────┘   │
                         └─────────────────────────────────────────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │   Streamlit UI  │
                                   │  (Polars DFs)   │
                                   └─────────────────┘
```

## Flux de Données Principaux

### 1. API Halo → DuckDB (Synchronisation v5.1)

```
Source: API Halo Infinite (via SPNKr)
     ↓
Client: SPNKrAPIClient (src/data/sync/api_client.py)
     ↓
Transformers: transform_match_stats(), extract_participants(), etc.
     ↓          (package `src/data/sync/transformers/` : 7 sous-modules)
Engine: DuckDBSyncEngine (src/data/sync/engine.py)
     ├─→ Match connu → enrichissement personnel uniquement (player_match_enrichment)
     └─→ Match nouveau → shared (registry + participants + events + medals)
     ↓
Destinations:
  - shared_matches.duckdb : matchs, participants, events, médailles, xuid_aliases
  - players/{gamertag}/stats.duckdb : enrichissements, awards uniquement
```

### 2. JSON → DuckDB (Référentiels)

```
Source: Fichiers JSON locaux
     ↓
Script: scripts/ingest_halo_data.py
     ↓
Destination: data/warehouse/metadata.duckdb
```

### 3. DuckDB → Parquet (Archive)

```
Source: DuckDB (match_stats)
     ↓
Script: scripts/archive_season.py
     ↓
Destination: data/players/{gamertag}/archive/matches_*.parquet
```

### 4. Parquet → DuckDB (Restore)

```
Source: Backup Parquet
     ↓
Script: scripts/restore_player.py
     ↓
Destination: data/players/{gamertag}/stats.duckdb
```

### 5. LUSR — Backfill local (shared_matches → stats.duckdb) — v5.3

```
Source: shared_matches.duckdb (match_participants + match_registry)
     ↓
scripts/backfill_data.py --lusr [--force-lusr]
     → scripts/backfill/strategies.compute_lusr_for_player()
     ↓
src/analysis/skill_rating.compute_skill_ratings_batch()
  - Séquentiel (match i dépend de match i-1)
  - PlayerState par playlist_group (ranked/arena/btb/tactical/social/fun)
  - composite_score [0,1] : kills 31% + deaths 28% + damage 23% + accuracy 13% + win 5%
  - delta_mu = K_ELO × (composite − 0.5) × weight_factor  (Elo-style)
  - sigma : TrueSkill réduction à t=0
     ↓
Destination: players/{gamertag}/stats.duckdb → match_skill_rank
  (PK=match_id — exclusif : un match = LUSR OU CSR, jamais les deux)
```

### 6. CSR — Sync API (RankRecap → stats.duckdb) — v5.3

```
Source: API Halo Infinite → Result.RankRecap.PreMatchCsr / PostMatchCsr
     ↓
src/data/sync/transformers.transform_all_skill_stats()
  → SkillParticipantUpdate.post_match_csr, .pre_match_csr, .csr_tier, .csr_sub_tier
     ↓
src/data/sync/engine._upsert_csr_rating()
  → Si is_ranked=True et post_match_csr non-null
     ↓
Destination: players/{gamertag}/stats.duckdb → match_skill_rank (rating_type='CSR')
```

### 7. PvE Stats Firefight — Sync/Backfill → shared_pve.duckdb — v5.2

```
Source: API Halo Infinite (match JSON) ou shared_matches.duckdb (backfill)
     ↓
_is_firefight_match() → True (GameVariantCategory 41/42, PublicName firefight/baptême)
     ↓
extract_pve_stats(match_json) → list[PveMatchStatsRow]
  - waves_completed, boss_kills, kills par type d'ennemi (Banished + Forerunner)
  - pve_bits : bitmask granulaire PveBits(IntFlag)
     ↓
batch_insert_pve_stats(pve_conn, rows)
     ↓
Destination: data/warehouse/shared_pve.duckdb → pve_match_stats
  + bit guard MatchBits.PVE_STATS (1 << 20) dans match_registry.backfill_completed
```

### 8. Films SPNKr → weapon_kills (Extraction Armes) — v5.6

```
Source: API Halo Infinite (film chunks REPLICATION_DATA)
     ↓
src/analysis/weapon_parser.py (domaine pur, 0 IO)
  - Corrélation kill → dernier événement fire dans fenêtre 2000 ms
  - Melee/grenade/véhicule via médailles (MELEE_API_ID=1, GRENADE_API_ID=0)
  - POV coverage : ~87,5% des kills
     ↓
src/data/services/weapon_extraction_service.py (WeaponExtractionService)
  - Orchestration : téléchargement chunks, appel parser, agrégation
  - Cache local dans data/investigation/chunks/<match_id>/
     ↓
src/data/sync/_engine_weapon_kills.py (WeaponKillsEngineMixin)
  - Contrôlé par SyncOptions.with_weapons
  - Bit MatchBits.WEAPON_KILLS (1 << 21) dans match_registry.backfill_completed
     ↓
Destination: data/warehouse/shared_matches.duckdb → weapon_kills
  (PK = match_id, xuid, weapon_id UBIGINT ; index idx_wk_match_xuid)
```

### 9. Dossiers médias → DuckDB (Onglet Médias)

```
Source: Dossiers configurés (Paramètres → media_screens_dir, media_videos_dir)
     ↓
MediaIndexer.scan_and_index() — scan delta récursif, ffprobe/EXIF
     ↓
media_files (status=active/deleted), media_match_associations (après associate_with_matches)
     ↓
Thumbnails: generate_thumbnails_for_new() → thumbs/ (GIF vidéo, miniatures images)
     ↓
UI: media_tab.py (load_media_for_ui → sections Mes captures / Captures de XXX / Sans correspondance)
```

Note 2026-04-11 : le scan, le watcher et le fallback UI ignorent explicitement `thumbs/`. Les miniatures générées ne doivent jamais réentrer dans `media_files`, sinon elles se transforment en nouvelles sources et provoquent une récursion infinie de miniatures d'images.

Lancement : thread en arrière-plan au démarrage de l’app (`_background_media_indexing` dans streamlit_app.py).

### 10. Home Challenges live → metadata.duckdb + stats.duckdb — v7

```
Source: HaloStats /hi/players/xuid(...)/decks + CMS ChallengeContent/ClientChallengeDefinitions/*.json
     ↓
src/ui/pages/home_mission_control_api.py
     ↓
src/ui/pages/home_mission_control_challenges.py
  - résumé deck actif
  - badge Waypoint
  - progression x/y
     ↓
src/data/challenges.py
  - persist_challenge_catalog()
  - persist_challenge_snapshots()
  - load_challenge_metadata_map() (fallback local)
     ↓
Destinations:
  - data/warehouse/metadata.duckdb
      • challenge_definitions (versionnées via content_hash)
      • challenge_translations (toutes langues disponibles, BCP-47)
  - data/players/{gamertag}/stats.duckdb
      • challenge_snapshots (active/completed/upcoming, progression, expiry, XP)
```

Note 2026-04-12 : la persistance des défis est best-effort. Si `metadata.duckdb` est verrouillée
par un autre process, la home continue de fonctionner en live et le stockage est simplement ignoré.

### 11. Home Battle Pass live → Mission Control V7 — v7

```
Source: Economy /hi/players/xuid(...)/rewardtracks/operations
     ↓
ActiveOperationRewardTrackPath (source de vérité joueur)
     ↓
metadata.duckdb (`battlepass_track_definitions` / `battlepass_track_translations`)
     ↓ cache miss seulement
GameCMS /hi/Progression/file/{RewardTrackPath}
     ↓
src/ui/pages/home_mission_control_battlepass.py
  - nom localisé du pass actif
  - progression courante / premium ownership
  - liste complète des paliers du track (y compris paliers vides pour navigation 0 → max)
  - fenêtrage précédent / courant / suivants selon le volume de rewards affichables
  - résolution des items inventaire via metadata.duckdb (`battlepass_item_definitions` / `battlepass_item_translations`) puis GameCMS en cache miss
  - fallback repo statique pour les monnaies sans image CMS (`static/battlepass-assets/xpboost.png`, `rerollcurrency.png`)
     ↓
src/ui/pages/home_mission_control_api.py
     ↓
src/ui/pages/home_mission_control.py
     ↓
Destination: carte Home Mission Control (pass réel du joueur, plus basée sur le calendrier saisonnier, avec navigateur unique `Paliers` et barre XP du palier)
```

Note 2026-04-12 : la home n'utilise plus le calendrier GameCMS comme source primaire pour le pass affiché. Le track courant du calendrier peut diverger du pass réellement actif côté joueur.

Note 2026-04-12 : les rewards `xpboost` et `rerollcurrency` utilisent désormais des PNG embarqués dans le repo quand GameCMS ne fournit pas de `DisplayPath`; ces assets sont référencés via `battlepass_asset_refs` avec origine `repo-static`.

Note 2026-04-12 : les définitions GameCMS de reward tracks et d'items battle pass sont désormais persistées dans `metadata.duckdb` pour mutualiser le cache entre joueurs partageant le même season pass ; seuls l'appel Economy joueur et la progression personnelle restent spécifiques au joueur.

Note 2026-04-20 : côté Go, la progression battle pass réellement visible par le joueur est persistée localement en append-only dans `stats.duckdb` (`battlepass_snapshots`) via `internal/platform/duckdb/persist_sink.go`, sur le même modèle que `challenge_snapshots`. Le catalogue metadata reste mutualisé dans `metadata.duckdb`, tandis que Home et Season Pass relisent désormais la progression depuis ces snapshots joueur plutôt que depuis un payload partagé global.

### 12. Contrat KPIs Home Go → Frontend web (2026-04-18)

```
Source: apps/go-api/internal/analysis/home.go
  - ComputeKPIs()
    • WinRate = wins / total                  -> ratio [0..1]
    • AvgAccuracy = moyenne déjà en pourcent -> ex: 42.0
     ↓
Service: apps/go-api/internal/service/home_service.go
     ↓
DTO: HomePageResponse.hero.kpis
     ↓
Destination frontend:
  - apps/web/src/components/shell/KPIBar.tsx
  - apps/web/src/features/home/HomePage.tsx
```

Règle de transformation validée localement :

- `win_rate` doit être multiplié par 100 pour l'affichage UI.
- `avg_accuracy` ne doit **pas** être remultiplié ; la valeur backend est déjà un pourcentage prêt à afficher.

## Tables PvE (shared_pve.duckdb)

| Table | Cardinalité | Description |
|-------|-------------|-------------|
| `pve_match_stats` | N:1 par match | Stats par joueur par match Firefight (waves, boss, kills Grunt/Elite/Jackal/Brute/Hunter/Skimmer/Crawler/Soldier/Knight/Warden, `pve_bits`) |

## Tables et Cardinalité

### Métadonnées (metadata.duckdb)

| Table | Lignes | Description |
|-------|--------|-------------|
| `playlists` | ~14 | Définitions playlists |
| `game_modes` | ~313 | Modes de jeu (FR/EN) |
| `categories` | ~16 | Catégories de modes |
| `medal_definitions` | ~153 | Définitions médailles |
| `career_ranks` | 273 | Rangs (0-272) |
| `players` | Variable | Joueurs connus |

### shared_matches.duckdb (centralisée)

| Table | Cardinalité | Description |
|-------|-------------|-------------|
| `match_registry` | 1:1 par match | Registre central (données communes du match) |
| `match_participants` | N:1 par match | Stats de tous les joueurs (31 col, incl. MMR) |
| `highlight_events` | N:1 par match | Événements filmés |
| `medals_earned` | M:N | Médailles de tous les joueurs |
| `killer_victim_pairs` | N:1 par match | Paires killer→victim |
| `xuid_aliases` | 1:1 | Mapping global XUID→Gamertag |
| `weapon_kills` | N:1 par match | Kills par arme par joueur (weapon_id UBIGINT, ~87,5% POV) — **v5.6** |

### Données Joueur stats.duckdb (v5.3 — enrichissements uniquement)

> 8 tables supprimées (v5.1) : match_stats, match_participants, highlight_events,
> medals_earned, killer_victim_pairs, player_match_stats, xuid_aliases, teammates_aggregate

| Table | Cardinalité | Description |
|-------|-------------|-------------|
| `player_match_enrichment` | 1:N par joueur | performance_score, session_id, is_with_friends, `is_excluded` (**SEULE table match**) |
| `personal_score_awards` | M:N | Awards objectifs (PersonalScores API) |
| `match_citations` | 1:N | Citations calculées par match |
| `career_progression` | 1:N | Historique rangs |
| `sessions` | 1:N | Sessions groupées |
| `sync_meta` | 1:1 | Métadonnées sync |
| `media_files` | 1:N | Fichiers médias indexés (captures/vidéos), status active/deleted |
| `media_match_associations` | M:N | Association média ↔ match ↔ xuid |
| `match_skill_rank` | 1:1 par match | Rating LUSR ou CSR (exclusif), tier, delta, playlist_group — **v5.3** |

### Vues Matérialisées

| Vue | Description | Rafraîchissement |
|-----|-------------|------------------|
| `mv_map_stats` | Stats par carte | Post-sync |
| `mv_mode_category_stats` | Stats par mode | Post-sync |
| `mv_session_stats` | Stats par session | Post-sync |
| `mv_global_stats` | Stats globales | Post-sync |

## Transformations Clés

| Donnée | Source | Formule |
|--------|--------|---------|
| `kda` | match_stats | `(kills + assists/3) / max(deaths, 1)` |
| `rating_value` (LUSR) | match_participants | TrueSkill 2 Elo-style : `mu += K_ELO × (composite − 0.5) × wf` (K=32, wf par groupe) |
| `composite_score` | match_participants | `kills_vs_exp×0.31 + deaths_vs_exp×0.28 + dmg_eff×0.23 + acc_delta×0.13 + win×0.05` |
| `rating_delta` | match_skill_rank | `rating_value[i] − rating_value[i-1]` pour le même `playlist_group` |
| `shots_fired` / `shots_hit` | API → match_stats, match_participants | `Players[].PlayerTeamStats[].Stats.CoreStats.ShotsFired` / `ShotsHit` ; joueur propriétaire dans match_stats, tous les joueurs dans match_participants. |
| `accuracy` | match_stats | `shots_hit / shots_fired * 100` (ou API si fourni) |
| `win_rate` | mv_global_stats | `wins / total_matches * 100` |
| `headshot_rate` | weapon_stats | `headshot_kills / total_kills * 100` |

### Rang dans le match

Le **rang d'un joueur lors d'un match** (position 1, 2, 3…) a deux origines :
- **Sync** : `match_stats.rank` ← API (`Players[].Rank`) via `transformers._extract_player_rank()`.
- **Vue match** : `MatchPlayerStats.rank` ← recalcul par `loaders.load_match_players_stats()` (tri par score, puis attribution 1, 2, 3…). Utilisé notamment pour le tie-breaker dans l’analyse killer/victim.

Détail : `.ai/DATA_MATCH_RANK.md`.

## Validations

### Pydantic v2

- [x] MatchStatsRow : Validation des champs matchs
- [x] PlayerMatchStatsRow : Validation MMR
- [x] HighlightEventRow : Validation événements
- [x] XuidAliasRow : Validation XUID (16 chiffres)
- [x] CareerRankData : Validation progression

### Contraintes DuckDB

- Clés primaires sur toutes les tables
- Index sur colonnes fréquemment filtrées (`start_time`, `playlist_id`)
- Colonnes GENERATED pour les calculs (`net_kills`, `accuracy`)

## Architecture Multi-Joueurs (v5.1)

En v5.1, les stats coéquipiers sont chargées depuis `shared.match_participants` :

```
1. Identifier match_id communs via shared.match_participants
      ↓
2. Charger stats coéquipier depuis shared.match_participants (xuid)
      ↓
3. Pas besoin d'accéder aux DBs individuelles des coéquipiers
```

Le sync écrit dans les player DBs : `player_match_enrichment` + `personal_score_awards` uniquement.

## Flux complémentaires Go Migration

### 12. Exclusion manuelle d'un match — React/Go API → player_match_enrichment

```
Source: action utilisateur (Match History ou Match View React)
     ↓
Endpoint Go: PATCH /api/v1/players/{player_slug}/matches/{match_id}/exclusion
     ↓
api/handlers/match_exclusion.go
     ↓
service/match_exclusion_service.go
     ↓
platform/duckdb/match_exclusion_repo.go
     ↓
Destination: players/{gamertag}/stats.duckdb → player_match_enrichment.is_excluded
     ↓
Consommation:
  - match_history_service.go filtre les matchs exclus avant pagination/export
  - match_view_service.go expose is_excluded dans le header JSON
```

### 13. Session HTTP → provider Halo live (Battle Pass / Challenges)

```
Source: session HTTP LevelUp avec HaloTokens + LinkedHaloIdentity
     ↓
api/middleware/session.go
     ↓
ctxkeys.WithHaloAuth(ctx, tokens, xuid)
     ↓
platform/halo/provider.go
  - GetBattlePass() → economy.svc.halowaypoint.com
  - GetChallenges() → halostats.svc.halowaypoint.com
     ↓
Destination: réponses JSON Home / bootstrap via services Go
```

## Problèmes Connus

- Aucun problème majeur identifié

## Références

- `docs/SQL_SCHEMA.md` : Schémas complets
- `docs/SYNC_GUIDE.md` : Guide de synchronisation
- `.ai/ARCHITECTURE_ROADMAP.md` : Roadmap des phases
- `.ai/DATA_MATCH_RANK.md` : Rang d'un joueur lors d'un match (API vs recalcul)
