# Project Map - LevelUp

> **[HISTORIQUE — GELÉ, NE FAIT PLUS FOI]** (dernière révision de fond ~2026-04-28).
> Ce fichier n'est plus tenu à jour. Le **code fait foi** ; pour l'état courant, voir
> `CLAUDE.md`, `docs/ARCHITECTURE_V6.md`, `.ai/thought_log.md` et les ADRs `docs/adr/`.
> Affirmations connues **PÉRIMÉES** : monde Python (supprimé — migration Go terminée),
> chemins `data/warehouse/` plats (réels = `data/titles/{slug}/warehouse/`, ADR 0008),
> « Film Chunks NON EXPLOITABLES » (démenti — décodage kill-feed résolu),
> routes front `/players/{slug}/…` (depuis 2026-07 : `/{-lang}/t/{titleSlug}/players/{slug}/…`,
> titre et langue en segments d'URL — plan `.ai/PLAN_TITLE_SLUG_URL_2026-07.md`, les
> anciennes URLs redirigent via un splat). Ne pas s'appuyer
> sur les sections ci-dessous sans re-vérifier dans le code.

> 📋 **Tâches et TODO centralisés** : voir `.ai/BACKLOG.md` et `.ai/PUNCHLIST.md` (handover GS↔OP, sources de vérité courtes).

> 🧭 **Chantier Go — corpus restructuré** : point d'entrée dans `.ai/go_migration_v2/README.md` ; le corpus historique détaillé reste dans `.ai/go_migration/`.

> 📘 **Onboarding nouveau dev** : `docs/FOUNDATIONS_GUIDE.md` (EN) + `docs/FR/FOUNDATIONS_GUIDE.md` — guide consolidé sur les 4 fondations transverses (canonical types + adapters + i18n manifests + ECharts wrappers). 4 ADRs dans `docs/adr/000{1,2,3,4}.md`.

## ⚠️ Limitations Connues

**IMPORTANT** : Consulter `.ai/API_LIMITATIONS.md` avant d'implémenter des fonctionnalités liées aux armes.

- **Weapon Stats par arme** : NON DISPONIBLE dans l'API (vérifié 2026-02-02)
- **Film Chunks** : NON EXPLOITABLES pour l'identification d'armes
- **SQLite** : PROSCRIT - Tout le code doit utiliser DuckDB uniquement. Aucun fallback SQLite (0 `import sqlite3` dans src/)
- **Pandas** : PROSCRIT - Utiliser **Polars** uniquement pour DataFrames/séries. Audit : `.ai/PANDAS_TO_POLARS_AUDIT.md`, `.ai/CONSOLIDATED_AUDITS_AND_ROADMAP.md`

## Architecture Multi-Joueurs (v5.1)

En v5.1, les stats coéquipiers sont chargées depuis `shared.match_participants` (plus besoin d'accéder aux DBs individuelles).

Le sync écrit dans les player DBs : `player_match_enrichment` + `personal_score_awards` uniquement.

## Validation transversale (2026-04-18)

- Backend Go : `CGO_ENABLED=1 go test -tags=integration ./... -timeout 120s -count=1` passe intégralement.
- Frontend React : `npm run typecheck`, `npm run lint`, `npm run build`, `npm run test:run` et `npm run test:e2e` passent sur `apps/web`.
- Correctifs clés de cette passe : ordre de scan `MatchHistoryRepo` réaligné avec `is_excluded`, tests Go dupliqués renommés, `HomePage`/`SynthesisPage` tolérants aux fixtures partielles, specs Vitest réalignées avec l'UI actuelle.
- Home match tiles (2026-04-21) : `AssetHandler.GetMapImage` sert maintenant d'abord `map_images_registry.local_path` pour les maps connues, le payload home expose aussi `playlist_ui` et normalise `mode_ui` (suppression des suffixes `on/sur <map>`), et `MatchCard` affiche désormais `mode sur carte` centré avec une ligne playlist puis un panneau stats réservé, avec garde locale contre les doublons de nom de carte.
- Home match tiles runtime (2026-04-21) : `internal/analysis/home.go` ne sérialise plus les images récentes via l'endpoint UUID quand la map est connue localement ; `map_image_url` pointe maintenant directement vers `/static/maps/<Map>.<ext>` et `mode_ui` retire aussi les préfixes d'expérience `Arena:` / `Community:`. Validation live sur `GET /api/v1/players/JGtm/pages/home` : `Bazaar -> /static/maps/Bazaar.png`, `Team Slayer`, `Quick Play`.
- Home match tiles i18n (2026-04-21) : `Q26HomeMatches` relit désormais les labels via `shared.v_match_full`, la home transporte `playlist_name_fr` en plus de la variante EN, et `internal/analysis/home.go` choisit FR/EN selon la langue issue des settings du shell React. `MatchCard` ne traduit plus les labels lui-même ; il ne fait qu'adapter le connecteur `sur/on` à `appShellStore.locale`.
- Médias React : `PATCH /players/{player_slug}/media/likes` persiste désormais `liked` / `liked_at` dans `media_files`, et `POST /pages/media` expose l'état liked utilisé par la home et la galerie.
- Workflow dev Go/React (2026-04-20) : `Makefile` racine accepte `API_PORT`, réutilise une API déjà saine sur ce port, et `apps/web/vite.config.ts` lit `VITE_API_PROXY_TARGET` pour suivre automatiquement le backend choisi en dev.
- Workflow dev Go/React (2026-04-21) : le `Makefile` racine ne pointe plus par défaut vers le sibling `../LevelUp` pour `LEVELUP_REPO_ROOT`; en dev, l'API Go travaille maintenant sur le repo courant `LevelUp-go-migration` sauf surcharge explicite, ce qui remet aussi le cache badges local sous `data/cache/challenge_badges` du workspace actif.
- Battle Pass metadata concurrency (2026-04-22) : `apps/go-api/internal/platform/halo/battlepass_details.go` coalesce désormais les fetchs concurrents d'une même reward track via `singleflight` sur `metadataPath|trackPath`, ce qui supprime les rafales parallèles `cache miss -> GameCMS -> upsert` observées autour de `battlepass_track_definitions` dans un même processus.
- Home record block (2026-04-22) : `GET /api/v1/players/{slug}/pages/home` expose maintenant `spartan_identity` à partir de `career_progression` (player DB) + `career_ranks` (metadata), et `apps/web/src/features/home/HomePage.tsx` affiche dans `Performance globale` un `Spartan ID` compact ainsi que le rang carrière courant avec titre localisé FR/EN et barre composite `current_xp -> xp_for_next_rank`.
- Home record block (2026-04-22 bis) : `spartan_identity.career_rank` transporte désormais aussi `rank_image_url`, dérivée des colonnes `large_icon_path` / `adornment_icon_path` / `icon_path` de `metadata.career_ranks`, et la Home React rend ce visuel à côté du titre de rang pour expliciter le bloc identitaire sans dépendre d'un asset local CSR.
- Home record block (2026-04-22 ter) : `career_progression` transporte maintenant aussi `emblem_image_url` et `backdrop_image_url`, `Q26cHomeSpartanIdentity` retombe sur la dernière valeur non vide pour `spartan_id` et les assets identitaires, et [apps/web/src/features/home/HomePage.tsx](apps/web/src/features/home/HomePage.tsx) rend désormais un vrai bandeau d'identité joueur inspiré de SpartanRecord au lieu de deux cartes minimales.
- Home record block (2026-04-22 quater) : la Home ne publie plus de liens GameCMS directs pour l'identité Spartan. [apps/go-api/internal/platform/duckdb/home_repo.go](apps/go-api/internal/platform/duckdb/home_repo.go) convertit maintenant emblème, backdrop et image de rang vers des URLs internes `/api/v1/assets/spartan/...`, et [apps/go-api/internal/assets](apps/go-api/internal/assets) les sert via le même resolver cache-aside local-first que les autres assets du produit.
- Home record block (2026-04-22 quinquies) : la distinction legacy `banner/nameplate` vs `backdrop` est maintenant rétablie de bout en bout. `career_progression` persiste aussi `banner_image_url`, `internal/assets` ajoute `spartan-banner`, la Home backend publie `/api/v1/assets/spartan/banner/...` séparément de `/backdrop/...`, et [apps/web/src/features/home/HomePage.tsx](apps/web/src/features/home/HomePage.tsx) rend enfin la bannière centrale et le fond comme deux couches différentes.
- Home record block (2026-04-22 sexies) : l'archéologie de `v7/cockpit` a confirmé que la vraie bannière legacy était reconstruite depuis `player_title_path`, ou à défaut depuis `emblem_path + configuration_id`. `apps/go-api/internal/sync/halo_client.go` porte maintenant ce fallback `nameplate`, et une resync live fait ressortir `banner_image_url` dans `GET /api/v1/players/JGtm/pages/home` et `GET /api/v1/players/Chocoboflor/pages/home` avec des URLs internes `/api/v1/assets/spartan/banner/halo_infinite/hi/Waypoint/file/images/nameplates/...`.
- Home record block (2026-04-22 septies) : `spartan_identity` transporte maintenant aussi `highest_csr` et `highest_lusr`, lus directement dans `match_skill_rank` par `rating_type` depuis la player DB. Le backend dérive leurs badges depuis les assets statiques existants `/static/ranks/120px-HINF-CSR_<Tier><SubTier>.png`, et [apps/web/src/features/home/HomePage.tsx](apps/web/src/features/home/HomePage.tsx) rend ces deux pics compétitifs dans le bandeau Spartan, juste à droite du `Spartan ID`.
- Home skill type fix (2026-04-22) : `match_skill_rank.rating_type` n'est plus traité comme source d'autorité pour les surfaces produit quand `shared.match_registry` est disponible. [apps/go-api/internal/platform/duckdb/queries_match.go](apps/go-api/internal/platform/duckdb/queries_match.go) et [apps/go-api/internal/platform/duckdb/queries_home_citations.go](apps/go-api/internal/platform/duckdb/queries_home_citations.go) dérivent maintenant le type effectif via `is_ranked` (`CSR` si classé, sinon `LUSR`), avec fallback sur la valeur stockée seulement si le match n'existe plus dans `shared`. Cela corrige les profils comme JGtm où des matchs classés remontaient encore en `LUSR` sur la Home et la Match View.
- Home rank states (2026-04-22) : `HomePageResponse` expose désormais `has_ranked_history` et `has_unranked_history`, calculés depuis les matchs Home hors Firefight. [apps/web/src/features/home/HomePage.tsx](apps/web/src/features/home/HomePage.tsx) distingue maintenant quatre états dans le panneau CSR/LUSR : valeur connue, `En placement` pour un CSR sans rang final, `Sans classement` pour un historique non classé sans LUSR, et `Aucune partie classée/non classée` quand le joueur n'a jamais joué ce type. [apps/web/src/features/home/HomeRecentPlaylistsCard.tsx](apps/web/src/features/home/HomeRecentPlaylistsCard.tsx) ne réutilise plus `Unranked.png` pour les playlists non classées sans rang.
- Runtime schema fixes (2026-04-22) : plusieurs surfaces Go lisaient encore des colonnes qui n'existent pas dans les schémas live. [apps/go-api/internal/platform/duckdb/queries_career.go](apps/go-api/internal/platform/duckdb/queries_career.go) dérive maintenant `offensive_conversion` et `defensive_resistance` depuis les dégâts/kills au lieu de lire deux colonnes absentes sur `shared.match_participants`. [apps/go-api/internal/platform/duckdb/queries_home_citations.go](apps/go-api/internal/platform/duckdb/queries_home_citations.go) n'utilise plus `mf.indexed_at` pour le tri media shared_social et retombe sur `updated_at`/`created_at`. [apps/go-api/internal/platform/duckdb/leaderboard_repo.go](apps/go-api/internal/platform/duckdb/leaderboard_repo.go) a abandonné la lecture legacy `shared.match_participants.csr_after` et publie désormais le CSR courant du joueur depuis `match_skill_rank`, avec sérialisation alignée sur l'UI.
- Palmares relations runtime (2026-04-22) : la page Relations React n'appelle plus le faux endpoint absent `/pages/palmares/relations`. [apps/web/src/features/palmares/queries.ts](apps/web/src/features/palmares/queries.ts) consomme désormais l'endpoint existant `/pages/career/encounters` puis mappe la payload carrière vers le shape `RelationsPageResponse` attendu par [apps/web/src/features/palmares/PalmaresRelationsPage.tsx](apps/web/src/features/palmares/PalmaresRelationsPage.tsx).
- Nettoyage racine (2026-04-21) : les wrappers Python `LevelUp.bat`, `LevelUp.sh` et `run.sh` sont supprimés du worktree Go ; les points d'entrée locaux documentés sont désormais `make dev`, `make go-api-dev` et `make web`, et le déploiement VPS vit sous `scripts/deploy.sh`.
- Hygiène repo maps (2026-04-21) : `titles.json` n'est plus gardé à la racine faute de consommateur runtime, `migrate-static-maps` écrit désormais son CSV de non-correspondances sous `data/investigation/maps/`, et les logs ponctuels `populate-maps.log` / `migrate-static-maps-dry.log` ne vivent plus en racine.
- Défis home weekly (2026-04-21) : `apps/go-api/internal/platform/halo/challenges_details.go` extrait désormais la vraie famille sous `WeeklyChallenges/<family>/...` au lieu de limiter la résolution de badges à `action|gametype|weapon`, ce qui permet de servir les images hebdo propres comme `weekly-vehicle-*.png`.
- Défis home seasonal (2026-04-21) : les chemins `S5WinterChallenges` et assimilés essaient maintenant d'abord le schéma simple `weekly-<difficulty>.png` (`weekly-normal`, `weekly-heroic`, `weekly-legendary`) avant les fallbacks plus spécifiques, ce qui rétablit les images live de défis comme `Bravoure dans la victoire`, `Oiseau de fête` et `Score Moar !`.
- Cache badges défis (2026-04-21) : après correction du `Makefile` racine et du helper API, les badges weekly récupérés live sont bien persistés dans `data/cache/challenge_badges` du repo courant, au lieu du sibling `LevelUp`.
- Régression home Go/React (2026-04-21) : `Q26HomeMatches` ne dépend plus de `shared.v_match_full`, `buildPoolConfig()` ne prend les chemins title-aware que si les trois DBs critiques existent vraiment, la migration/player schema réintroduit `player_match_enrichment.session_label`, et la home React espace désormais correctement les états chargement / erreur sous la L1.

## État Actuel (2026-03-13) — v5.7 Stable

### Historique des versions

- **v5.1** : Architecture Shared DB, éradication SQLite/Pandas, cleanup tables legacy ✅
- **v5.2** : Filtres intent-based, Stats PvE Firefight (`shared_pve.duckdb`), Scoreboard, palette Okabe-Ito ✅
- **v5.3** : LUSR/CSR TrueSkill 2 per-groupe, Notifications Discord, 20 tests corrigés ✅
- **v5.4** : i18n split, logging centralisé, SyncScope cleanup, refactoring modules >500L (Phases 0-6, 72 sous-modules) ✅
- **v5.5** : Setup Wizard, Xbox OAuth → Device Code Flow, comparaison sessions, compatibilité macOS/Linux ✅
- **v5.6** : Extraction armes depuis films SPNKr (`weapon_kills`), Friends Impact Matrix ✅
- **v5.7** : Top 10 meilleurs/pires matchs (Carrière), détection Domination/Humiliation, CSS map hover, Pandas→Polars, launchers bilingues ✅

### Architecture v5.3

```
data/
├── players/                    # Enrichissements uniquement (~4 MB/joueur)
│   └── {gamertag}/
│       ├── stats.duckdb       # player_match_enrichment, awards, citations,
│       │                      #   match_skill_rank (LUSR/CSR par match)
│       └── archive/           # Archives temporelles
├── warehouse/
│   ├── metadata.duckdb        # Référentiels (playlists, maps, medals, ranks)
│   ├── shared_matches.duckdb  # Matchs centralisés (registry, participants, events, medals)
│   └── shared_pve.duckdb      # Stats PvE Firefight (pve_match_stats) — v5.2
└── backups/                   # Backups Parquet
```

## Go API — Couverture par package (baseline 35.0%)

| Package | Tests existants | Notes |
|---------|----------------|-------|
| `internal/sync` | `writes_test.go` (8 fonctions), `transforms_test.go`, `backfill_flags_test.go` | `//go:build integration` (CGO) |
| `internal/api/handlers` | `testhelpers_test.go`, `sessions_test.go`, `health_test.go`, `game_cms_test.go` | HTTP handlers + middleware |
| `internal/api/middleware` | `session_test.go`, `request_id_test.go`, `rate_limit_test.go`, `shadow_test.go`, `cors_test.go` | session + auth context + garde-fous HTTP |
| `internal/config` | `config_test.go`, `feature_flags_test.go` | Unit tests purs |
| `internal/domain/title` | `multititle_test.go`, `registry_test.go` | Unit tests purs |
| `internal/platform/halo` | `provider_test.go` | Battle Pass + Challenges live, retry HTTP, auth context |
| `internal/ctxkeys` | `ctxkeys_test.go` | clés de contexte titre + auth Halo |
| `internal/api/contract` | `contract_test.go` | `//go:build cgo` |

> Baseline global : **35.0%** (mesuré avec `coverage_baseline.txt`). Cible Phase 10 : 70%.

## Go API — Points chauds récupérés le 2026-04-18

- `apps/go-api/internal/platform/duckdb/db.go` + `internal/platform/duckdb/persist_sink.go` + `internal/platform/lab/provider.go` : le cache global des connexions DuckDB est désormais compté par références ; une ouverture temporaire de `metadata.duckdb` ne peut plus fermer `PlayerDB.Metadata` et casser la home / le season pass quand les défis et le battle pass chargent en parallèle.
- `apps/go-api/internal/platform/duckdb/pool.go` + `queries_match.go` + `match_view_repo.go` : `metadata.duckdb` n'est plus attachée à `stats.duckdb` dans le pool joueur ; les labels médailles/armes de la vue match sont désormais enrichis via `PlayerDB.Metadata`, ce qui élimine les conflits DuckDB de type `same database file with a different configuration` / `Unique file handle conflict`.
- `apps/web/src/features/home/HomePage.tsx` + `queries.ts` + `apps/go-api/internal/platform/halo/provider.go` : la home ne déclenche plus un endpoint `/challenges` en plus du payload season pass ; les défis affichés viennent de `SeasonPassPageResponse.challenges`, et le provider Halo protège désormais les fetchs live `/decks` concurrents avec un `singleflight` par `xuid`.
- `apps/go-api/internal/api/handlers/match_exclusion.go` : endpoints `PATCH /matches/{match_id}/exclusion` + `GET /match-exclusions` pour ignorer/réactiver des matchs au niveau joueur.
- `apps/go-api/internal/platform/duckdb/match_exclusion_repo.go` : persistance `player_match_enrichment.is_excluded` avec UPSERT côté player DB.
- `apps/go-api/internal/service/match_history_service.go` : filtrage des matchs exclus avant pagination, export CSV et agrégats de win rate.
- `apps/go-api/internal/api/middleware/session.go` + `apps/go-api/internal/ctxkeys/ctxkeys.go` : injection des `HaloTokens` et du `XUID` depuis la session HTTP dans le contexte Go.
- `apps/go-api/internal/platform/halo/provider.go` : implémentation live des appels Battle Pass / Challenges à partir du contexte auth, au lieu du stub `auth_required` permanent.
- `apps/go-api/internal/api/handlers/media.go`, `internal/service/media_service.go`, `internal/platform/duckdb/media_repo.go` : likes média backend persistés dans `media_files` et nouvelle route `PATCH /media/likes` documentée dans OpenAPI.
- `apps/web/src/components/shell/AppShell.tsx`, `AppShellHeader.tsx`, `PlayerScopeNav.tsx` : nouveau shell React sans sidebar, avec header global, navigation joueur compacte en deux niveaux et changement de joueur qui préserve la section courante quand c'est possible.
- `apps/web/src/components/shell/NavL1.tsx` + `apps/web/src/features/settings/SettingsPage.tsx` : le Lab interne est de nouveau exposé dans l'UI courante quand `capabilities.can_manage_instance` est actif, avec entrée visible dans la barre globale et carte d'accès dédiée dans Paramètres.
- `apps/web/src/components/shell/shellNavigation.ts` : source de vérité du mapping navigation primaire / secondaire et helper `buildPlayerDestination()` pour recalculer la route lors d'un changement de joueur.
- `Makefile` (racine) + `apps/go-api/.air.toml` + `apps/web/vite.config.ts` : démarrage dev backend/frontend harmonisé sous Windows avec cleanup `server.exe` côté Makefile, `API_PORT` override, réutilisation d'une API déjà up et proxy Vite configurable.

## Multi-titres — couche canonical + adapters + TOML mappings (Phase A–F, branche `feat/multi-title-adapters-and-mappings`)

- `apps/go-api/internal/games/canonical/` : schéma canonique services (43 FieldKey, enums Outcome/MatchType/RatingType, scopes StatsScope/TimeseriesQuery/CareerOptions, MatchSummary/Detail, CareerSnapshot, MetricSeries) — tous les services produit consomment ce schéma au lieu d'accéder aux colonnes DuckDB directement.
- `apps/go-api/internal/games/mappings/` : loader TOML strict (`pelletier/go-toml/v2`), validation locales+formats+collisions+conversions d'unités, `FieldMappingSet`, `Registry` title-aware, formatters (integer/percent/kdr/duration_hms/etc.) + conversions ratio↔percent et ms↔seconds.
- `apps/go-api/internal/games/halo_infinite/` : `DataAdapter` (wrap `CareerSource`, projette vers `CareerSnapshot`) + `SemanticAdapter` (wrap `FieldMappingSet`).
- `apps/go-api/internal/games/synthetic_title_b/` : corpus synthétique de tests d'isolation cross-titres (jamais référencé en prod).
- `apps/go-api/internal/games/{adapter,resolver}.go` : interfaces `TitleDataAdapter` + `TitleSemanticAdapter` (SRP), `StaticResolver` injecté au boot.
- `apps/go-api/internal/api/handlers/field_mappings.go` : `GET /api/v1/titles/{slug}/field-mappings?locale=fr` (ETag + Cache-Control), exposé seulement si `MULTI_TITLE_API_ENABLED=true`.
- `apps/go-api/internal/api/handlers/multi_title_preview.go` : `GET /api/v1/titles/{slug}/preview/career?xuid=...` — proof-of-concept end-to-end du pipeline canonique avec libellés FR/EN.
- `config/titles/halo_infinite/mappings/fields.toml` + `config/titles/synthetic_title_b/mappings/fields.toml` : TOML versionnés Git, source de vérité des libellés/format/group.
- `apps/web/src/lib/i18n/fieldMappings.ts` : hook React `useFieldLabel(key)` + `useFieldMappings()` consommant l'endpoint backend via TanStack Query (staleTime infini), fallback gracieux sur la key.
- `tools/mappings/CHANGELOG.md` : historique des bumps de `schema_version` des TOML.
- Plan + audits : `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`, `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md`, `.ai/AUDIT_I18N_REACT_2026-04-25.md`, `.ai/AUDIT_WEAPONS_2026-04-25.md`.
- Tests : 32 tests Phase A (canonical 5 + mappings 22 + handlers 5), 17 tests Phase B (5 resolver + 8 data + 4 semantic), 4 tests Phase C (preview), 5 tests Vitest Phase D, 5 tests Phase E (isolation cross-titres). Total 63+ tests sur la couche.

## Modules Clés

### Frontend web (Go migration)
- `apps/web/src/components/shell/AppShell.tsx` : shell top-level sans sidebar, fond atmosphérique et conteneur principal centré.
- `apps/web/src/components/shell/AppShellHeader.tsx` : header global avec identité produit, titre courant, session Halo, liens utilitaires et sélecteur de joueur.
- `apps/web/src/components/shell/NavL1.tsx` + `ThemeToggle.tsx` + `src/app/providers/theme-provider.tsx` : la barre globale expose désormais un switch dark/light local, persisté dans `levelup-ui-prefs` et appliqué au document via `data-theme`.
- `apps/web/src/components/shell/PlayerScopeNav.tsx` : navigation compacte du scope joueur, séparée entre parcours principal et vues secondaires, exposée en `nav` sémantique.
- `apps/web/src/components/shell/KPIBar.tsx` : bande de KPIs repensée en cartes lisibles au lieu d'une simple ligne tabulaire.
- `apps/web/src/components/shell/PageHeader.tsx` : entête de page plus premium, avec hiérarchie visuelle renforcée.
- `apps/web/src/components/shell/shellNavigation.ts` : constantes de navigation et logique de destination lors d'un switch joueur.
- `apps/web/src/components/shell/shellNavigation.test.ts` : test unitaire Vitest du helper de navigation joueur.
- `apps/web/src/components/ui/empty-state.tsx` : pattern partagé `EmptyStateCard` / `EmptyStateNotice` pour les payloads nulles et sections analytiques vides.
- `apps/web/src/features/media/queries.ts`, `MediaViewer.tsx`, `MediaPage.tsx`, `MediaToolbar.tsx`, `i18n.ts`, `home/RecentMediaRail.tsx`, `apps/go-api/internal/platform/duckdb/media_repo.go`, `queries_home_citations.go`, `internal/service/media_service.go` : likes média désormais lus depuis l'API Go ; la galerie React garde une toolbar compacte `Filtrer :` / `Trier :`, reconstruit cartes/modes depuis les items si `available_filters` est vide, et le backend Go choisit désormais la bonne requête média selon le schéma réellement utilisé (`shared_social` avec `media_file_id` ou fallback legacy player DB), avec modes normalisés avant tri/filtrage.
- `.ai/go_migration_v2/UX_CAREER_SYNTHESIS_BOUNDARY.md` : cadrage UX go-only pour la frontière Carrière / Synthèse ; `Profil` disparaît de la cible produit, `Carrière` devient le hub `Progression + Citations`, et `Synthèse` absorbe l'overview filtrée, les performances marquantes et les rivalités.
- `.ai/go_migration_v2/UX_CAREER_HUB_BLUEPRINT.md` : blueprint détaillé du hub `Carrière`, avec route canonique unique, tabs deep-linkables `Progression` / `Citations`, retrait des blocs analytiques et stratégie de transition depuis `CareerPage` + `CitationsPage`.
- `.ai/go_migration_v2/SYNTHESIS_TARGET_CONTRACT_AND_UI.md` : composition cible de `Synthèse` côté UI et contrat Go/React ; extraction recommandée hors `SquadHandler`, ajout d'une vraie `overview`, de previews lazy et migration des anciens `top-matches` / `encounters` de Carrière.
- `.ai/go_migration_v2/UX_HOME_RECORD_SPARTAN_ADDITIONS.md` : cadrage d'ajouts inspirés de Spartan Record pour la home/record existante ; conserve la page actuelle, rejette le toggle global `Overall / Per Match`, ajoute `Spartan ID`, `Data Set`, tuiles de match en complément, hiérarchie médailles et stratégie d'images de maps dynamiques.
- `.ai/go_migration_v2/DAMAGE_EFFICIENCY_INTEGRATION.md` : cadrage analytique et produit du `rendement combat` ; fixe les gardes-fous data, la taxonomie recommandée (`conversion offensive`, `resistance defensive`), les surfaces d'intégration Go/React, les impacts potentiels sur `Performance` / `LUSR` et la stratégie de tests.
- `apps/web/src/features/home/HomePage.tsx` + `HomeBattlePassPanel.tsx` + `queries.ts` : home joueur avec quick actions en routes typées, unité de précision alignée sur le backend Go (`avg_accuracy` déjà en %), section battle pass enrichie via l'endpoint season pass (image principale, rail horizontal des paliers, centrage du palier courant, progression composite du palier actif désormais rendue sur une ligne `valeur courante - barre composite - valeur cible`), et cartes de défis actifs regroupées en sections `Quotidien` / `Hebdo` avec en-tête texte simple et trait blanc pleine largeur, sans badge de cadence par carte, chaque carte affichant aussi sa progression sur une seule ligne `valeur - barre - pourcentage`, tandis que la carte `Défis actifs` elle-même est `self-start` avec min-height modérée pour éviter les grands vides quand peu ou aucun défi est présent.
- `apps/go-api/internal/domain/home.go`, `internal/platform/duckdb/home_repo.go`, `internal/analysis/home.go`, `internal/service/home_service.go`, `apps/go-api/internal/sync/halo_client.go`, `apps/go-api/internal/sync/career.go`, `apps/web/src/features/home/HomePage.tsx`, `apps/web/src/components/ui/composite-progress-bar.tsx` : la home transporte et rend désormais un vrai bloc `spartan_identity` (Spartan ID + rang carrière courant), avec titre de rang localisé selon la langue active, `rank_image_url` dérivée de la metadata carrière, `emblem_image_url` + `backdrop_image_url` issus de la customisation joueur, fallback sur les dernières valeurs non vides en BDD, et bandeau visuel identitaire inspiré de SpartanRecord.
- `apps/go-api/internal/sync/halo_client.go`, `apps/go-api/internal/sync/halo_client_extra_test.go` : le fallback de bannière Home est maintenant aligné sur le legacy Python `v7/cockpit`. Si `PlayerTitlePath` est absent, le sync reconstruit une `nameplate` Waypoint depuis `EmblemPath + ConfigurationId`, ce qui repeuple `career_progression.banner_image_url` pour les profils où la customisation publique ne livre pas explicitement de bannière.
- `apps/go-api/internal/assets/kinds.go`, `internal/assets/fetcher_gamecms.go`, `internal/api/handlers/assets.go`, `internal/api/server.go`, `internal/platform/duckdb/home_repo.go` : le pipeline d'assets unifié gère maintenant aussi les visuels du bloc Spartan (`spartan-emblem`, `spartan-banner`, `spartan-backdrop`, `career-rank-image`) via `/api/v1/assets/spartan/{image_type}/{title_id}/*`, avec fetch distant GameCMS / Waypoint et persistance locale automatique.
- `apps/web/src/features/home/HomePage.tsx`, `career/CareerPage.tsx`, `timeseries/TimeseriesPage.tsx`, `squad/SquadPage.tsx`, `citations/CitationsPage.tsx`, `synthesis/SynthesisPage.tsx`, `session-compare/SessionComparePage.tsx`, `explorer/ExplorerPage.tsx` : plus de `return null` silencieux sur ce périmètre, avec placeholders explicites quand une section ne peut pas s'afficher.
- `apps/web/src/features/palmares/SeasonPassPage.tsx` : la carte de progression du palier actif reprend le même layout composite que la home, avec valeur courante à gauche, barre au centre et valeur cible à droite.
- `apps/web/package.json` : dépendance explicite `plotly.js`, requise au build par `react-plotly.js`.
- `apps/web/src/lib/accessibility/` : système d'accessibilité Okabe-Ito (Phases 1-7, 2026-04-25) :
  - `palettes.ts` : `defaultPalette` + `okabePalette` (Okabe-Ito 2008). `applyPalette(palette, key)` écrit les 40 variables CSS `--ac-*` sur `:root`.
  - `semantic-tokens.ts` : union type `SemanticToken` (40 tokens : `perf-tier-1..5`, `outcome-win/loss/draw/dnf`, `divergent-pos/neg/neutral`, `narrative-dominant/secondary`, etc.).
  - `resolver.ts` : `resolveToken(token)` — lit `getComputedStyle()` synchronement (usage Plotly). `tokenCssVar(token)` — retourne `var(--ac-token)` pour JSX réactif.
  - `scales.ts` : `makeOrdinalScale`, `makeDivergentScale`, `makeCategoricalScale`. 8 instances exportées : `perfScale`, `accuracyScale`, `kdScale`, `progressScale`, `mmrDeltaScale`, `skillDeltaScale`, `outcomeScale`, `narrativeScale`.
  - `plotlyColorscale.ts` : `buildOrdinalColorscale(tokens[])`, `buildDivergentColorscale(neg, neutral, pos)`, `getSeriesColors(n, tokens[])`.
  - `useColorPaletteVersion.ts` : hook React — `MutationObserver` sur `style` de `:root` → version incrémentale → force re-render des `useMemo` Plotly lors de changement de palette.
  - `index.ts` : barrel export de tous les symboles publics du module.

### Accès aux Données
- `src/data/repositories/duckdb_repo.py` : Repository principal DuckDB (splitté: `_awards_repo`, `_diagnostic_repo`, `_legacy_compat`, `_match_queries_helpers`, `_match_queries_polars`, `_metadata_resolution`, `_schema_introspection`, `_archives_repo`, `_events_repo`, `_medals_repo`, `_gamertag_resolver`)
- `src/data/repositories/factory.py` : Factory pattern
- `src/data/challenges.py` : Façade publique des défis Halo ; délègue le catalogue metadata à `src/data/_challenge_catalog.py` et les snapshots joueur à `src/data/_challenge_snapshots.py`
- `src/data/battlepass.py` : Façade publique du catalogue metadata battle pass ; délègue les reward tracks et items partagés à `src/data/_battlepass_catalog.py` dans `metadata.duckdb`
- `src/data/sync/engine.py` : Moteur de synchronisation (8 mixins MRO : `_shared_writes`, `_performance`, `_skill_rating`, `_career`, `_aggregates`, `_match_processing`, `_engine_connections`, `_engine_schema` + `_protocol.py`)
- `src/data/sync/_engine_weapon_kills.py` : Mixin extraction armes depuis films (`WeaponKillsEngineMixin`) — v5.6
- `src/data/services/weapon_extraction_service.py` : Service hexagonal extraction armes (`WeaponExtractionService`) — v5.6
- `src/data/media_indexer.py` : Indexation médias (splitté: `media_helpers`, `media_loaders`, `media_thumbnails`) ; le scan ignore désormais `thumbs/` pour éviter la récursion thumbnails d'images

### Analyse
- `src/analysis/killer_victim.py` : Calcul antagonistes (splitté: `_killer_victim_polars`, `_kv_types`)
- `src/analysis/antagonists.py` : Agrégation rivalités
- `src/analysis/sessions.py` : Détection sessions
- `src/analysis/performance_score.py` : Score de performance (splitté: `_performance_relative`, `_performance_session`)
- `src/analysis/objective_participation.py` : Participation objectifs (splitté: `_objective_helpers`, `_objective_profile`, `_objective_summary`)
- `src/analysis/weapon_parser.py` : Parser pur d'armes depuis films SPNKr (0 IO, architecture hexagonale) — v5.6
- `src/data/sync/transformers/` : Package (7 sous-modules: `_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`)

### Visualisation & UI (splits phases 4-6)
- `src/visualization/antagonist_charts.py` : Charts antagonistes (splitté: `_antagonist_kv`, `_antagonist_duels`)
- `src/ai/rag.py` : RAG IA (splitté: `_rag_models`, `_rag_github`, `_rag_chunker`)
- `src/data/repositories/refdata.py` : Référentiels (splitté: `_refdata_personal_scores`)
- `src/app/cache_filters.py` : Cache & filtres (splitté: `_cache_loading`, `_cache_sessions`)
- `src/app/filters_render.py` : Rendu filtres (splitté: `_filters_apply`, `_filters_period`, `_filters_session`, `_filters_cascade`)
- `src/visualization/session_compare_charts.py` : Comparaison sessions (splitté: `_session_compare_history`)

### Infrastructure transversale (v5.4)
- `src/data/sync/_protocol.py` : `_SyncProtocol` — contrat Protocol pour les 8 mixins engine
- `src/app/_page_context.py` : `PageContext` + `MatchViewParams` — types réels pour pages
- `src/app/session_keys.py` : `SessionKeys` / `SK` — clés session_state centralisées
- `src/data/query/_sql_fragments.py` : `WIN_RATE_EXPR`, `IS_WIN`, `IS_LOSS` centralisés
- `src/analysis/playlist_groups.py` : 6 groupes Halo Infinite — v5.3
- `src/analysis/skill_rating.py` / `skill_rating_config.py` / `skill_rating_calibration.py` : LUSR/CSR TrueSkill 2 — v5.3

### UI
- `src/ui/pages/` : Pages du dashboard
- `src/ui/pages/media_v2.py` + `media_v2_grid.py` : Page Médias V2 ; lightbox Streamlit partagée, miniatures désormais rendues nativement via `st.image` pour éviter les iframes par carte
- `src/ui/components/media_thumbnail.py` : Composant thumbnail HTML legacy (survol GIF + lightbox optionnelle) ; expose aussi `load_native_thumbnail_source()` pour le rendu léger de Media V2
- `src/ui/pages/home_mission_control.py` : Rendu Streamlit de l'accueil Mission Control V7 (briefing, CTA, timeline, sections)
- `src/ui/pages/home_mission_control_cards.py` : Builders HTML de la home V7 (hero, highlights, actions, cartes session, timeline récente, bloc médias)
- `src/ui/pages/home_mission_control_logic.py` : Logique pure du Mission Control V7 (dataclasses, navigation contextuelle, highlights, résumés, sélection des matchs/médias récents)
- `src/ui/pages/home_mission_control_challenges.py` : Helpers défis live pour la home V7 (résumé `/decks`, fallback metadata, dérivation + cache des badges Waypoint)
- `src/ui/pages/home_mission_control_battlepass.py` + `home_mission_control_battlepass_render.py` + `home_mission_control_battlepass_assets.py` : pass actif joueur de la home V7, navigateur unique de paliers sur tout le track (fenêtre précédente/courante/suivante extensible), barre XP composite, cache metadata partagé reward track ou item dans `metadata.duckdb`, cache lazy d'assets et fallback repo statique pour `xpboost` / `rerollcurrency`
- `src/ui/pages/explorer_results.py` + `match_table_html.py` : résultats Explorer avec pagination légère des gros tableaux HTML (filtres / alliés / adversaires) pour réduire le DOM injecté
- `src/ui/pages/match_view.py` : Vue match détaillée ; le badge Match ID utilise désormais un popover Streamlit natif, le bloc carte/rang s'appuie sur des colonnes Streamlit + `st.image`, et la rangée KPI utilise une structure native `st.columns` au lieu d'un wrapper HTML unique
- `src/ui/pages/v7_sections.py` : Couche de composition temporaire du cockpit V7 (regroupement des pages legacy par section) ; enveloppe aussi désormais Stats/Escouade/Explorer/Médias/Profil dans une vraie surface de workspace
- `src/ui/layout/` : Shell V7 (header L1/L2, KPI bar, chips de filtres) ; la L2 pilote désormais le contexte Stats/Escouade avec filtre visible, scope de session et navigation précédente / dernière session
- `src/ui/theme/` : Thème V7 (chargement CSS + feuille dédiée) ; surcharge aussi désormais les panneaux d'onglets, cartes bordées, expanders, métriques, tags de multiselect, popovers, checkboxes et sliders des pages legacy réutilisées dans le cockpit
- `src/ui/pages/career_top_matches_data.py` + `career_top_matches_render.py` : Top 10 meilleures/pires performances (Carrière) — v5.7
- `src/ui/pages/match_view_weapon_kills.py` : Section armes dans vue match — v5.6
- `src/ui/pages/match_view_scoreboard_detail.py` : Détails inline du scoreboard match (POC CSS-only, ligne dépliable) — v5.7
- `src/ui/pages/teammates_weapons.py` : Onglet armes coéquipiers — v5.6
- `src/ui/pages/setup_wizard.py` + `setup_wizard_logic.py` + `setup_wizard_xbox.py` : Assistant configuration initiale — v5.5
- `src/ui/xbox_oauth_ui.py` + `src/utils/msal_device_flow.py` : Device Code Flow Xbox OAuth — v5.5
- `src/app/player_provisioning.py` : Provisionnement automatique joueur — v5.5
- `src/utils/auth.py` : `AuthStatus` + gestion credentials — v5.5
- `src/ui/pages/teammates_views.py` : Vues coéquipiers (splitté: `_teammates_trio.py`)
- `src/ui/components/radar_chart.py` : Radar charts (splitté: `_radar_participation`, `_radar_teammates`)
- `src/ui/cache_loaders.py` : Cache Streamlit (splitté: `_cache_core`, `_cache_queries`)
- `src/ui/sync.py` : UI sync (splitté: `_sync_utils`, `_sync_indicator`, `_sync_duckdb_ops`)
- `src/ui/streamlit_modern.py` : Wrappers Streamlit moderne
- `src/ui/filter_state.py` : Filtres intent-based v5.2
- `src/utils/discord_notifier.py` : Notifications Discord (splitté: `_discord_embed`, `_discord_queries`) — v5.3
- `src/utils/safe_types.py` / `async_compat.py` / `env.py` : Utilitaires partagés — v5.4
- `src/visualization/` : Graphiques Plotly
- `src/visualization/timeseries_combat.py` : Séries temporelles (splitté: `_timeseries_helpers`, `_timeseries_progression`)
- `src/visualization/friends_impact_heatmap.py` : Friends Impact Matrix (séparateurs verticaux, renommé depuis Heatmap) — v5.6
- `src/ui/i18n/ranks.py` : Traductions FR des rangs Halo (17 rangs + 6 tiers CSR) — v5.7
- `static/battlepass-assets/` : visuels repo-tracked des monnaies battle pass non exposées par GameCMS (`xpboost.png`, `rerollcurrency.png`)

## Tables DuckDB

### shared_matches.duckdb (centralisée)

| Table | Description |
|-------|-------------|
| `match_registry` | Registre central (1 ligne par match unique) |
| `match_participants` | Stats de tous les joueurs (31 colonnes, incl. MMR) |
| `highlight_events` | Événements filmés de tous les matchs |
| `medals_earned` | Médailles de tous les joueurs |
| `killer_victim_pairs` | Paires killer→victim |
| `xuid_aliases` | Mapping global XUID→Gamertag |
| `weapon_kills` | Kills par arme par joueur par match (weapon_id UBIGINT, PK=match_id+xuid+weapon_id) — **v5.6** |

### Base Joueur stats.duckdb (v5.3 — enrichissements uniquement)

> 8 tables supprimées (v5.1) : match_stats, match_participants, highlight_events,
> medals_earned, killer_victim_pairs, player_match_stats, xuid_aliases, teammates_aggregate

| Table | Description |
|-------|-------------|
| `player_match_enrichment` | performance_score, session_id, is_with_friends (**SEULE table match**) |
| `personal_score_awards` | Awards objectifs (PersonalScores API) |
| `match_citations` | Citations calculées par match |
| `career_progression` | Historique rangs |
| `media_files` | Fichiers médias indexés (status, thumbnail_path, capture_end_utc) |
| `media_match_associations` | Média ↔ match ↔ xuid (map_name, match_id) |
| `sessions` | Sessions groupées |
| `sync_meta` | Métadonnées sync |
| `match_skill_rank` | Rating LUSR/CSR par match (PK=match_id — exclusif LUSR ou CSR) — **v5.3** |
| `challenge_snapshots` | Historique append-only dédupliqué des défis joueur (active/completed/upcoming, progression, XP, expiry) |
| `mv_*` | Vues matérialisées (mv_player_matches, mv_map_stats, etc.) |

### Base Métadonnées (metadata.duckdb)

| Table | Description |
|-------|-------------|
| `playlists` | Définitions playlists |
| `game_modes` | Modes de jeu (FR/EN) |
| `medal_definitions` | Référentiel médailles |
| `challenge_definitions` | Définitions versionnées des défis Halo (category, difficulty, seuil, XP, hash de contenu) |
| `challenge_translations` | Titres + descriptions multi-langues des défis (BCP-47, fallback EN) |
| `career_ranks` | Rangs de carrière |

## Scripts Utilitaires

| Script | Description |
|--------|-------------|
| `scripts/sync.py` | Synchronisation SPNKr |
| `scripts/backup_player.py` | Export Parquet Zstd |
| `scripts/restore_player.py` | Import depuis backup |
| `scripts/archive_season.py` | Archivage temporel |
| `scripts/migrate_*.py` | Scripts de migration |

## Dépendances Critiques

| Package | Version | Usage |
|---------|---------|-------|
| `duckdb` | >=1.4.0 | Moteur unique |
| `polars` | >=1.38.0 | DataFrames |
| `pydantic` | >=2.5.0 | Validation |
| `streamlit` | >=1.37.0 | Interface (@st.fragment, st.navigation) |

## Points d'Entrée

- `streamlit_app.py` : Application principale
- `streamlit_app_v7.py` : Entrée dédiée du cockpit V7, basée sur le bootstrap legacy
- `launcher.py` : Lanceur CLI

## Documentation

> Convention :
> - `docs/` = documentation EN (publique)
> - `docs/FR/` = sources FR
> - `docs/archive/` = docs conservées mais non traduites

| Document | Contenu |
|----------|---------|
| `docs/INSTALL.md` | Installation |
| `docs/CONFIGURATION.md` | Configuration |
| `docs/COMMANDS.md` | Commandes usuelles |
| `docs/ARCHITECTURE_V6.md` | Architecture DuckDB v6 |
| `docs/SYNC_GUIDE.md` | Guide synchronisation |
| `docs/BACKUP_RESTORE.md` | Backup/Restore |
| `docs/TESTING_V5.md` | Tests (v5) |
| `docs/FAQ.md` | Questions fréquentes |
| `docs/COMMENDATIONS.md` | Commendations (ex "citations") |
| `docs/COMMENDATIONS_REFERENCE.md` | Référentiel complet des commendations |

### Documentation IA (.ai/)

| Document | Contenu |
|----------|---------|
| `.ai/DATA_KILLER_VICTIM.md` | Guide killer/victim et antagonistes |
| `.ai/DATA_MATCH_RANK.md` | Rang d'un joueur lors d'un match (API vs recalcul, tie-breaker) |
| `.ai/MIGRATION_MASTER.md` | Point d'entrée unique du chantier FastAPI/React, avec état courant, priorités MVP et navigation vers les sous-docs de migration |
| `.ai/go_migration_v2/HALO_CANONICAL_MODEL.md` | Contrat canonique Halo entre provider de titre, produit LevelUp et analytics métier |
| `.ai/go_migration_v2/HALO_INFINITE_CAPABILITY_MAP.md` | Capability map initiale mono-titre pour `halo_infinite`, avec projection bootstrap minimale |
| `.ai/go_migration_v2/HALO_BOOTSTRAP_CONTRACT.md` | Contrat produit du bloc `halo` dans le bootstrap : titre, provider, capabilities et limitations utiles au consommateur |
| `.ai/go_migration_v2/HALO_GO_TYPE_BLUEPRINT.md` | Projection documentaire des structs, enums et interfaces Go canoniques avant implémentation |
| `.ai/go_migration_v2/HALO_INFINITE_CANONICAL_MAPPING.md` | Discipline de projection des payloads Halo Infinite vers le modèle canonique, sans mélanger analytics ni contrats HTTP |
| `.ai/go_migration_v2/HALO_PRODUCT_CONTRACT_ADAPTERS.md` | Cadrage de la projection du canonique Halo vers les read models produit et les DTO OpenAPI |
| `.ai/go_migration_v2/HALO_PROVIDER_ERROR_TAXONOMY.md` | Taxonomie des erreurs et limitations entre provider Halo et API produit, avec projection HTTP normalisée |
| `.ai/go_migration_v2/OPENAPI_MVP_P0_P1.md` | Gel des contrats HTTP MVP P0/P1 à préserver avant le démarrage du backend Go |
| `.ai/go_migration_v2/SPRINT_44_WORKPACKAGES.md` | Découpage technique par couches du Sprint 44 multi-titres : design, config, migration, validation, observabilité |
| `.ai/go_migration_v2/ADR_S44_MULTI_TITLE_NAMESPACE.md` | ADR actant le namespace par titre et l'introduction explicite de `title_slug` dans le runtime Go |
| `.ai/go_migration_v2/AUDIT_PLANS_VS_REALITE_2026-04-17.md` | Audit transverse plans vs réalité : Go migration, no-streamlit, écarts documentaires, vrais restants et priorités actionnables |
| `.ai/migration/` | Corpus de migration FastAPI/React découpé par sujet : décisions, invariants, parité, slices, contrats API, audit de codebase |
| `.ai/PLAN_MIGRATION_FASTAPI_REACT.md` | Audit exhaustif + plan de migration Streamlit vers FastAPI/React, avec perimetre fige, matrice de parite, contrats API MVP, extraction du state model, structure cible du repo, delivery par slices, cohabitation front, auth/session, tests de parite et pilotage par metriques |
| `.ai/go_migration/` | Corpus isole du chantier Python -> Go : plan maitre, checklist, matrice, compat ops et strategie zero Python |
| `.ai/go_migration/GO_MIGRATION_CHECKLIST.md` | Suivi vivant du chantier Python -> Go : ordre des lots, statuts d'avancement, preuves attendues, blocages et prochaine action |
| `.ai/go_migration/MATRIX.md` | Matrice de couverture Python -> Go : packages, scripts, surfaces hors scope, bitmask et priorites de portage |
| `.ai/go_migration/OPS_COMPAT_CHECKLIST.md` | Checklist runtime/exploitation : auth, refresh tokens, jobs persistants, mode de test, packaging, migration utilisateur |
| `.ai/go_migration/ZERO_PYTHON_STRATEGY.md` | Cible terminale zero Python : destin de chaque module, perimetre d'extinction et contraintes de livraison |
| `.ai/go_migration/PLAN_MIGRATION_PYTHON_TO_GO.md` | Plan de migration complete du runtime Python vers Go : perimetre, architecture cible, phasage, gates Go/No-Go, conditions de succes et d'echec ; isole avec son corpus `go_migration/` |
| `.ai/sprints/SPRINT_GAMERTAG_ROSTER_FIX.md` | Sprint correction gamertags et roster |
| `.ai/API_LIMITATIONS.md` | Limitations connues de l'API |

## Problèmes Connus

Aucun problème bloquant connu.

## État technique (v5.7)

- **4479 tests** passent, 0 échecs
- **Architecture DuckDB v5.3** : shared_matches + shared_pve + player enrichments
- **Polars** comme moteur DataFrame (0 Pandas dans code métier)
- **0 SQLite** dans le code runtime
- **Streamlit ≥1.37** avec @st.fragment, st.navigation, column_config
- **Taille player DB** : ~4 MB (vs ~30 MB en v5.0)
- **Refactoring v5.4** : 72 nouveaux sous-modules (phases 0-6)
- **weapon_kills** : extraction armes via films SPNKr (~87.5% couverture POV) — v5.6
- **Setup Wizard + Device Code Flow** : configuration guidée sans redirect URI — v5.5
- **CSS map thumbnails** : hover pur CSS, sans JS sandboxé — v5.7

## Exploration Complète du Projet

Une exploration détaillée de tout le projet (modules, scripts, tests, docs) a été refaite le **2026-02-05** :

📄 **`.ai/explore/PROJECT_EXPLORE_2026-02-05.md`**

Contenu :
- Vue d’ensemble (stack, points d’entrée, règles critiques)
- Arborescence `src/` complète (rôle de chaque module : app, data, ui, analysis, visualization, db, ai, utils)
- Scripts catégorisés (~100) : sync, backup, migration, backfill, diagnostic, analyse/recherche, API, tests
- Tests listés par thème
- Documentation `docs/` et `.ai/`
- Structure données et config
- Flux d’entrée et dépendances
- Référence aux audits (SQLite, Pandas→Polars, problèmes connus)

Consulter ce fichier pour une cartographie exhaustive ; le présent `project_map.md` reste la cartographie vivante (état, problèmes, sprints).

## Dernière Mise à Jour

**2026-03-13** : **v5.7.0** — Top 10 meilleurs/pires matchs, CSS map hover, Pandas→Polars, launchers bilingues, 4479 tests
**2026-03-10** : **v5.6.0** — weapon_kills (extraction armes films), Device Code Flow Xbox, Friends Impact Matrix
**2026-03-07** : **v5.5.0** — Setup Wizard, Xbox OAuth, comparaison sessions, macOS/Linux, packaging portable
**2026-03-05** : **v5.4** — Refactoring modules >500L, 72 sous-modules, SyncScope, logging centralisé
**2026-02-25** : **v5.3.0** — LUSR/CSR TrueSkill 2 per-groupe, Notifications Discord
**2026-02-20** : **v5.2.0** — Filtres intent-based, Stats PvE shared_pve.duckdb, Scoreboard, Okabe-Ito
**2026-02-17** : **v5.1.0 Release** — Documentation finale, archivage, release tag
