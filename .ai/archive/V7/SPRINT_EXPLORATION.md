# Sprint Exploration — LevelUp Go Migration

> Ce fichier documente l'état d'exploration du codebase Go pour les agents IA.

## Référence principale

Le suivi des sprints est maintenu dans [`SPRINT_ROADMAP.md`](.ai/go_migration_v2/SPRINT_ROADMAP.md).

> Note backend 2026-04-21 : `internal/platform/duckdb/db.go` utilise désormais un cache de connexions ref-counted. Les ouvertures temporaires de `metadata.duckdb` côté défis/Lab/PersistSink ne peuvent plus invalider `PlayerDB.Metadata`, ce qui stabilise le chargement parallèle home + season pass.

> Note backend 2026-04-21 bis : `internal/platform/duckdb/pool.go` n'attache plus `metadata.duckdb` sur `stats.duckdb`. Les rares lectures runtime de `citation_mappings` / `weapon_labels` ont été déplacées dans `MatchViewRepo` via `PlayerDB.Metadata`, afin d'éviter les conflits DuckDB entre connexion dédiée et ATTACH du même fichier.

> Note tooling 2026-04-21 ter : nettoyage des launchers racine. Le worktree Go n'expose plus `LevelUp.bat`, `LevelUp.sh` ni `run.sh`; le point d'entrée local canonique passe par le `Makefile` (`make dev`, `make go-api-dev`, `make web`) et le déploiement VPS est rangé dans `scripts/deploy.sh`.

> Note tooling 2026-04-21 quater : nettoyage des artefacts maps à la racine. `titles.json` n'a plus de lecteur côté Go/React, `migrate-static-maps` écrit maintenant `unmatched_maps.csv` sous `data/investigation/maps/`, et les logs manuels de validation maps doivent vivre sous `data/logs/maps/` plutôt qu'à la racine.

> Note home/match tiles 2026-04-21 : les tuiles de matchs récents ne dépendent plus implicitement d'un fetch distant pour les maps connues ; `internal/api/handlers/assets.go` sert maintenant `local_path` depuis `map_images_registry` avant `image_url`, `internal/analysis/home.go` normalise `mode_ui` en retirant les suffixes `on/sur <map>`, et `apps/web/src/components/ui/match-card.tsx` affiche désormais un titre centré `mode sur carte`, une ligne playlist, puis un panneau stats réservé sans redoubler le nom de carte.

> Note home/match tiles runtime 2026-04-21 : la dernière régression venait encore du payload home et non du handler d'assets. `internal/analysis/home.go` publie maintenant `recent_matches[].map_image_url` directement vers `/static/maps/<Map>.<ext>` pour les maps connues au lieu de l'endpoint UUID `/api/v1/assets/maps/...`, et la normalisation de `mode_ui` retire aussi les préfixes d'expérience `Arena:` / `Community:`. Validation live confirmée sur `GET /api/v1/players/JGtm/pages/home` et `GET /static/maps/Bazaar.png`.

> Note home/i18n 2026-04-21 : la home backend choisit maintenant les labels FR/EN à la source. `internal/platform/duckdb/queries_home_citations.go` s'appuie sur `shared.v_match_full` au lieu de `match_registry`, expose `playlist_name_fr`, et `internal/api/handlers/home.go` transmet la langue des settings à `HomeService.GetHomePage(...)`. `apps/web/src/components/ui/match-card.tsx` ne conserve qu'un choix de connecteur `sur/on` à partir de `appShellStore.locale`.

> Note home/record 2026-04-22 : `internal/platform/duckdb/home_repo.go` charge désormais aussi le dernier snapshot `career_progression` et l'enrichit via `metadata.career_ranks`, `internal/analysis/home.go` construit `spartan_identity` avec titre FR/EN + progression `%`, et `apps/web/src/features/home/HomePage.tsx` rend ce bloc dans `Performance globale` avec un `Spartan ID` compact et une barre composite partagée avec les panneaux Battle Pass.

> Note home/record 2026-04-22 bis : l'exploration a confirmé que le `spartan_id` live était déjà présent dans le payload Home ; la passe suivante a donc ciblé le vrai manque structurel, à savoir le visuel de rang carrière. `internal/platform/duckdb/home_repo.go` dérive maintenant `rank_image_url` depuis la metadata carrière, `internal/domain/home.go` l'expose dans `spartan_identity.career_rank`, et `apps/web/src/features/home/HomePage.tsx` l'affiche à côté du titre de rang dans `Performance globale`.

> Note home/record 2026-04-22 ter : l'étape suivante a supprimé une autre fragilité côté BDD. La Home ne lit plus `spartan_id` uniquement sur la dernière ligne de `career_progression` : `Q26cHomeSpartanIdentity` retombe désormais sur la dernière valeur non vide pour `spartan_id`, `emblem_image_url` et `backdrop_image_url`. En parallèle, `internal/sync/halo_client.go` résout ces deux visuels depuis la customisation publique Halo + GameCMS, `internal/sync/career.go` les persiste dans `career_progression`, et [apps/web/src/features/home/HomePage.tsx](apps/web/src/features/home/HomePage.tsx) rend maintenant un bandeau identitaire visuel inspiré de SpartanRecord.

> Note home/assets 2026-04-22 : la passe suivante a réaligné l'identité Spartan sur la couche `internal/assets/`. `internal/assets/kinds.go` expose maintenant `spartan-emblem`, `spartan-backdrop` et `career-rank-image`, `internal/api/handlers/assets.go` ajoute `/api/v1/assets/spartan/{image_type}/{title_id}/*`, et `internal/platform/duckdb/home_repo.go` publie ces URLs internes dans la payload Home au lieu d'URLs GameCMS directes. Le browser déclenche donc le même cycle local-first → fetch distant → persistance locale que pour les autres visuels du produit.

> Note home/assets 2026-04-22 bis : l'exploration du legacy restant dans le repo (`static/styles.css`, `app_settings.json`, `.env.local.example`) a confirmé que `banner/nameplate` et `backdrop` sont deux assets distincts. La passe suivante a donc ajouté `banner_image_url` à `career_progression`, un kind `spartan-banner` dans `internal/assets`, un parsing Halo plus permissif côté `internal/sync/halo_client.go`, puis un rendu React où la bannière centrale et le fond du bloc identitaire sont enfin séparés.

> Note home/assets 2026-04-22 ter : l'archéologie plus précise de `v7/cockpit` a montré que la bannière Python n'était pas lue depuis un champ `BannerImagePath` direct, mais reconstruite depuis `player_title_path` ou, si absent, depuis `emblem_path + configuration_id` vers un PNG Waypoint `nameplates/<stem>_<cfg>.png`. `internal/sync/halo_client.go` porte maintenant ce fallback, un test ciblé `TestGetCareerRank_DerivesBannerFromEmblemWhenNameplateMissing` le couvre, et une resync live a confirmé la présence de `banner_image_url` sur `GET /api/v1/players/JGtm/pages/home` et `GET /api/v1/players/Chocoboflor/pages/home`.

> Note home/record 2026-04-22 quater : le bandeau Spartan de la Home lit désormais aussi les pics compétitifs historiques depuis `match_skill_rank`. `internal/platform/duckdb/home_repo.go` charge le meilleur `CSR` et le meilleur `LUSR` par `rating_type`, dérive pour chacun un badge `/static/ranks/120px-HINF-CSR_<Tier><SubTier>.png`, `internal/analysis/home.go` les publie sous `spartan_identity.highest_csr` / `highest_lusr`, et [apps/web/src/features/home/HomePage.tsx](apps/web/src/features/home/HomePage.tsx) les rend en deux cartes compactes à droite du `Spartan ID`.

> Note home/record 2026-04-22 quinquies : l'audit statique du backend Go a ensuite confirmé une faille de vérité métier. Le post-sync actif ne branche pas encore de pipeline CSR dédié, tandis que Home et Match View lisaient directement `match_skill_rank.rating_type`, ce qui laissait des matchs classés remonter en `LUSR` pour des profils comme JGtm. Le correctif est volontairement local au dépôt DuckDB : `Q22MatchSkillRank`, `Q26eHomeSkillPeakByType` et `Q26fHomeLastSkillRank` dérivent maintenant le type effectif depuis `shared.match_registry.is_ranked` (`CSR` si classé, sinon `LUSR`), avec fallback sur la valeur stockée seulement quand la ligne shared manque. Les tests d'intégration repo couvrent explicitement le cas "match classé stocké en LUSR".

> Note home/backend 2026-04-21 quinquies : la home React ne consomme plus `/players/{slug}/challenges` en parallèle du payload season pass. Les défis affichés viennent désormais de `SeasonPassPageResponse.challenges`, et `internal/platform/halo/provider.go` déduplique les fetchs live concurrents des challenges via `singleflight` pour éviter plusieurs appels Waypoint `/decks` sur le même `xuid`.

## État actuel (Phase 11 — Sprint 49)

### Packages Go compilables localement (sans CGO/DuckDB)

| Package | Description |
|---------|-------------|
| `internal/domain/...` | Types métier purs (0 import externe) |
| `internal/analysis/...` | Algorithmes d'analyse (sessions, performance) |
| `internal/domain/title/...` | Registre multi-titres |

### Packages nécessitant CGO (DuckDB)

| Package | Description |
|---------|-------------|
| `internal/platform/duckdb/...` | Pool de connexions DuckDB |
| `internal/service/...` | Services métier |
| `internal/api/handlers/...` | Handlers HTTP (transitif via config) |
| `internal/api/...` | Router chi, middleware, server |

### Architecture contractuelle

- **OpenAPI** : `api/openapi.yaml` — source de vérité
- **Test contrat** : `internal/api/contract_test.go` — vérifie alignement OpenAPI ↔ chi
- **Exemptions** : 0 (vidées au Sprint 49)
- **Routes match exclusion** : `PATCH /players/{player_slug}/matches/{match_id}/exclusion` + `GET /players/{player_slug}/match-exclusions` documentées dans OpenAPI et branchées dans le router chi
- **Routes média** : `POST /players/{player_slug}/pages/media` documente maintenant une vraie réponse paginée média et `PATCH /players/{player_slug}/media/likes` persiste l'état liked dans la player DB
- **Auth Halo live** : Battle Pass / Challenges lisent désormais `HaloTokens` + `XUID` depuis `ctxkeys`, injectés par le middleware de session

### Points d'entrée clés

| Fichier | Rôle |
|---------|------|
| `cmd/server/main.go` | Point d'entrée du serveur |
| `Makefile` | Workflow dev racine (`dev`, `go-api-dev`, `web`) avec `API_PORT` configurable et réutilisation d'une API déjà active |
| `scripts/deploy.sh` | Script de déploiement VPS hors racine, appelé par GitHub Actions |
| `internal/api/server.go` | Assembly du router chi |
| `internal/api/middleware/session.go` | Injection session HTTP + auth Halo dans le contexte |
| `internal/service/bootstrap_service.go` | Bootstrap du shell React |
| `internal/api/handlers/session_context.go` | Contexte session (titre, joueur, locale) |
| `internal/api/handlers/match_exclusion.go` | Exclusion manuelle de matchs au niveau joueur |
| `internal/platform/halo/provider.go` | Provider Halo live pour Battle Pass / Challenges |
| `internal/ctxkeys/ctxkeys.go` | Clés de contexte partagées titre + auth Halo |
| `internal/domain/title/registry.go` | Registre des titres supportés |
| `internal/platform/duckdb/persist_sink.go` + `internal/platform/duckdb/home_repo.go` | Persistance fire-and-forget et cache joueur local des snapshots battle pass / challenges |

### Frontend web — shell joueur (2026-04-18)

- `apps/web/src/components/shell/AppShell.tsx` : shell désormais sans sidebar, avec header global sticky et zone de rendu centrée.
- `apps/web/src/components/shell/AppShellHeader.tsx` : identité produit, titre courant, liens utilitaires et sélecteur de joueur branché au router.
- `apps/web/src/components/shell/PlayerScopeNav.tsx` : navigation du scope joueur découpée entre parcours principaux et vues secondaires, désormais rendue en `nav` sémantique pour l'accessibilité et les tests E2E.
- `apps/web/src/components/shell/shellNavigation.ts` : définition des items de navigation et helper `buildPlayerDestination()` pour préserver la section active lors d'un switch joueur.
- `apps/web/vite.config.ts` : le proxy dev `/api` cible maintenant `VITE_API_PROXY_TARGET` (défaut `http://127.0.0.1:8000`), ce qui permet de lancer `go-api-dev` sur un port alternatif sans reconfig manuelle du front.
- `apps/web/src/routes/players/$playerSlug.tsx` : montage du nouveau scope joueur (`PlayerScopeNav` + `KPIBar` + contenu).
- `apps/web/src/features/home/HomePage.tsx` + `apps/web/src/components/shell/KPIBar.tsx` : correction du contrat KPI côté frontend (`win_rate` = ratio, `avg_accuracy` = pourcentage déjà normalisé), et passage des liens player-scoped en routes typées TanStack Router.
- `apps/web/src/features/media/queries.ts`, `MediaPage.tsx`, `MediaViewer.tsx`, `home/RecentMediaRail.tsx` : likes média branchés sur l'API Go avec mutation optimiste, et normalisation frontend pour tolérer l'ancienne payload média plate si besoin.
- `apps/web/src/components/ui/empty-state.tsx` + pages player-scoped (`home`, `career`, `timeseries`, `squad`, `citations`, `synthesis`, `sessions`, `explorer`) : harmonisation des états vides, avec messages explicites quand une payload manque ou qu'une section analytique ne peut pas être rendue.
- `.ai/go_migration_v2/UX_CAREER_SYNTHESIS_BOUNDARY.md` : mini spec produit de référence pour le prochain reslicing UX ; cible retenue = hub `Carrière` avec onglets `Progression` et `Citations`, et `Synthèse` recentrée sur overview, performances marquantes et rivalités.
- `.ai/go_migration_v2/UX_CAREER_HUB_BLUEPRINT.md` : détaille le redesign concret de `/players/$playerSlug/career` en hub à tabs `Progression` / `Citations`, la stratégie de route canonique et la sortie des blocs `top matches` / `encounters`.
- `.ai/go_migration_v2/SYNTHESIS_TARGET_CONTRACT_AND_UI.md` : décrit la cible de `Synthèse` côté UI et contrat API, avec extraction hors `SquadHandler`, `scope` explicite, `overview` en tête, previews lazy et migration des anciens endpoints analytiques de Carrière.
- `.ai/go_migration_v2/UX_HOME_RECORD_SPARTAN_ADDITIONS.md` : cadre les enrichissements inspirés de Spartan Record pour la home/record go-migration sans remplacer la page existante ; focus sur `Spartan ID`, `Data Set`, médailles, tuiles de matchs et stratégie dynamique pour les images de maps.
- `.ai/go_migration_v2/DAMAGE_EFFICIENCY_INTEGRATION.md` : formalise l'adoption potentielle d'une nouvelle famille de metriques de `rendement combat`, en separant metriques exactes, proxies et integrations candidates sur `Escouade`, `Synthese`, `Forme`, `Performance`, `LUSR` et les surfaces match.
- Validation locale finale : `npm run -s typecheck` OK, `npm run -s build` OK, `vitest` OK sur `shellNavigation.test.ts`, `playwright` OK sur `e2e/slice-0a-shell.spec.ts` (5/5).

### Workflow dev Windows (2026-04-20)

- `.air.toml` n'exécute plus de `pre_cmd` PowerShell-incompatible pour tuer `server.exe`.
- Le cleanup Windows est géré côté `Makefile` via `cmd /C taskkill ...`, ce qui évite le parser error `||` dans PowerShell.
- `make go-api-run API_PORT=8011` a été validé localement ; `/health` répond 200.
- Une relance sur le même port réutilise l'instance existante au lieu d'échouer immédiatement sur `port already occupied`.

### Validation locale Go + React (2026-04-18)

- `CGO_ENABLED=1 go test -tags=integration ./... -timeout 120s -count=1` : OK.
- `apps/web` : `npm run typecheck`, `npm run lint`, `npm run build`, `npm run test:run`, `npm run test:e2e` : OK.
- `apps/web` : revalidation ciblée empty states via `vitest` (`HomePage`, `CareerPage`, `SquadPage`, `SynthesisPage`, `ExplorerPage`) + `npm run build` : OK.
- Les corrections déterminantes sur cette passe ont porté sur les tests Go dupliqués, l'alignement `MatchHistoryRepo`/`Q5MatchHistory`, le hook conditionnel de `MediaPage`, et la mise à jour des specs Vitest obsolètes pour `SetupPage`, `SquadPage`, `SynthesisPage`, `MediaPage` et `HomePage`.
