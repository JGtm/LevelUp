# Axe 2 — Architecture multi-titres

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : apps/go-api/internal/{games,domain/title,sync,service,api,platform}/ + apps/web/src/ + config/titles/

## Synthèse (3-5 lignes max)

Migration multi-titres engagée mais **partielle, à environ 35-40 %**. Les fondations sont en place : `internal/games/canonical/` (12 fichiers, ~250 LoC d'enums + types) + `domain/title.Registry/PathResolver` + 3 interfaces adapter (`TitleDataAdapter`/`TitleSemanticAdapter`/`TitleAssetURLAdapter`) + TOML mappings versionnés Git + `synthetic_title_b` pour tests d'isolation + middleware `TitleExtractor` + sélecteur de titre frontend. **Mais** la couche canonique n'est consommée par aucun service produit majeur, le schéma DuckDB transverse (`match_registry`, `match_participants`, `medals_earned`, `xuid_aliases`) **n'a pas de colonne `title_id`** (isolation purement filesystem via `data/titles/{slug}/`), `MULTI_TITLE_API_ENABLED` est OFF par défaut, `HasCapability` n'est utilisé qu'à 1 endroit (asset metadata), et 105 fichiers Go contiennent des références littérales `"halo_infinite"` (dont une bonne moitié sont des fallbacks ou tests). Le verdict global : **squelette correct, chair manquante**.

## Matrice : adoption canonical par service (Go)

| Service | Row-type lu | Canonical ? | Hardcode Halo ? |
|---|---|---|---|
| `home_service.go` | `domain.HomeMatchRow` (40 colonnes) | non | implicite (consts `homeOutcome*`, `HINF-CSR_*`) |
| `synthesis_service.go` | `domain.SynthesisMatchRow` | non (DataAdapter logué mais pas branché) | `Outcome == 2/3` |
| `career_service.go` | `domain.CareerMatchRow` + canonical via `LoadEncounters` | partiel (Encounters seul) | `Outcome == 2 // WIN` |
| `match_view_service.go` | `domain.MatchView*` | hook `WithDataAdapter` mais pas activé | `titleSlug` lu mais flux legacy |
| `squad_service.go` v1 | `domain.SquadMatchRow` | non | `Outcome == 2 \|\| 3` |
| `squad_service_v2*.go` | `canonical.PlayerMatchRow` | **oui** (seul vrai consommateur prod) | non |
| `stats_service.go` | `domain.StatsMatchRow` | non | fallback `title.DefaultSlug` |
| `timeseries_service.go` | `domain.TimeseriesMatchRow` | non | `OUTCOME_WIN = 2` (front aussi) |
| `match_history_service.go` | `canonical.PlayerMatchRow` | oui (déjà noté axe 1) | `Outcome == 2` |
| `explorer_service.go` | `canonical.PlayerMatchRow` | oui (déjà noté axe 1) | non |
| `citations_service.go` | `domain.CitationContext` | non | `IsFirefight` field |
| `engagement_score_service.go` | DuckDB direct | non | fallback `"halo_infinite"` |
| `compare_service.go` | repo DuckDB | non | `titleSlug` paramètre |
| `media_service.go` | `domain.Media*` + `analysis.InferModeCategoryFromPairName` | non | catégories Halo hardcodées |
| `leaderboard_service.go` | repo DuckDB | non | Halo CSR seul |
| `season_pass_service.go` | repo DuckDB | non | Halo only |

**Résultat brut** : 3 services sur 16 consomment effectivement le canonique (Squad V2, Match History, Explorer). Les 13 autres lisent des row-types `domain.*MatchRow` calqués 1:1 sur les colonnes Halo de `match_registry` + `match_participants`.

## Constats

### [BLOQUANT] Schéma DuckDB transverse mono-titre — pas de colonne `title_id` sur les tables partagées

- **Fichier:ligne** : `apps/go-api/internal/migration/steps_shared.go:18-117`
- **Extrait** :
  ```sql
  CREATE TABLE IF NOT EXISTS match_registry (match_id VARCHAR PRIMARY KEY, ...);
  CREATE TABLE IF NOT EXISTS match_participants (match_id VARCHAR, xuid VARCHAR, ...);
  CREATE TABLE IF NOT EXISTS medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, ...);
  CREATE TABLE IF NOT EXISTS xuid_aliases (xuid VARCHAR PRIMARY KEY, ...);
  CREATE TABLE IF NOT EXISTS weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT, ...);
  ```
- **Problème** : aucune des 7 tables shared (`match_registry`, `match_participants`, `medals_earned`, `xuid_aliases`, `weapon_kills`, `killer_victim_pairs`, `highlight_events`) ne porte de dimension titre. L'isolation cross-titres repose **exclusivement** sur le chemin filesystem (`data/titles/{slug}/warehouse/shared_matches_v2.duckdb`). Conséquence : impossible de partager un `xuid_aliases` global ou un répertoire de match cross-jeu, impossible de traiter Halo Infinite + Halo MCC dans le même worker sans deux pools DuckDB distincts.
- **Action** : ajouter `title_slug VARCHAR NOT NULL DEFAULT 'halo_infinite'` au moins sur `xuid_aliases` (identité joueur globale par nature) et documenter la décision filesystem-only pour les 6 autres tables dans un ADR.

### [BLOQUANT] `canonical.PlayerStats`/`MatchSummary` non consommés par les services produit majeurs (Home, Career, Stats, Synthesis, Timeseries)

- **Fichier:ligne** : `apps/go-api/internal/service/synthesis_service.go:60-69`
- **Extrait** :
  ```go
  if s.dataAdapter != nil {
      caps := s.dataAdapter.Capabilities()
      if !caps.Has(games.CapMatchHistory) {
          slog.WarnContext(ctx, "capability_not_supported", ...)
      }
  }
  synthMatches, err := s.repo.LoadSynthesisMatches(ctx, playerXUID)
  ```
- **Problème** : Synthesis, Career, Match View, Stats déclarent un `WithDataAdapter` mais ne s'en servent que pour **logger** la capability. Le flux de données reste 100 % legacy via les `domain.*MatchRow`. La couche canonique est donc « décorative » sur 5 services majeurs. L'axe 1 a montré que `canonical.PlayerMatchRow` n'est consommé qu'en Squad V2 + Match History + Explorer.
- **Action** : décider et acter par ADR si `canonical.PlayerMatchRow` (déjà documenté évolutif via ADR 0005) doit remplacer les 4 row-types `Home/Stats/Synthesis/Squad`, ou figer la décision « canonique seulement pour les services nouveaux ».

### [BLOQUANT] `MULTI_TITLE_API_ENABLED` OFF par défaut → endpoint `/field-mappings` invisible en prod

- **Fichier:ligne** : `apps/go-api/internal/api/handlers/field_mappings.go:56-61`
- **Extrait** :
  ```go
  func MultiTitleAPIEnabled() bool {
      v := strings.ToLower(strings.TrimSpace(os.Getenv("MULTI_TITLE_API_ENABLED")))
      return v == "1" || v == "true" || v == "yes"
  }
  ```
- **Problème** : tout l'effort TOML mappings + `useFieldLabel` côté front est conditionné à un flag ENV qui par défaut renvoie `false`. Conséquence : en prod, `useFieldMappings()` reçoit un 404 et fallback systématique sur la key brute. Le pipeline « title-aware labels » est donc **dormant**.
- **Action** : passer le flag ON par défaut (Phase 6 du plan finition est livrée d'après thought_log et BACKLOG, donc plus de raison de le garder OFF) ou supprimer le flag.

### [BLOQUANT] Outcomes hardcodés en `int 2/3` dans 5 services + 2 fichiers analysis (canonical.Outcome ignoré)

- **Fichier:ligne** : `apps/go-api/internal/service/synthesis_service.go:182`, `service/career_service.go:470`, `service/match_history_service.go:291`, `service/squad_service.go:179-181`, `analysis/match_impact.go:74-76`, `analysis/home.go:31-40`
- **Extrait** :
  ```go
  if r.Outcome == 2 { // WIN
  ```
- **Problème** : la convention `2 = win, 3 = loss` vient du payload Halo Infinite natif. Aucun de ces sites n'utilise `domain.OutcomeWin` (déjà défini, voir `domain/outcomes.go:8`) ni `canonical.OutcomeWin`. En multi-titres, un titre dont l'API rendrait win=1, loss=0 (ou string) impose alors un mapping côté ingestion **et** le risque de passer à côté.
- **Action** : remplacer toutes les occurrences `Outcome == 2/3` par les constantes `domain.Outcome*`, ou mieux, faire produire `canonical.Outcome` dès la couche repository et supprimer le champ `int Outcome` dans les row-types.

### [BLOQUANT] `HasCapability` utilisé à un seul endroit en prod (asset metadata) — toutes les autres routes supposent toutes les capabilities Halo Infinite

- **Fichier:ligne** : `apps/go-api/internal/api/server.go:228-233`
- **Extrait** :
  ```go
  assetMetaHandler = handlers.NewAssetMetadataHandler(
      service.NewAssetService(...),
      func(slug string, cap titlePkg.Capability) bool {
          d := titleRegistry.Get(slug)
          return d != nil && d.HasCapability(cap)
      },
  )
  ```
- **Problème** : `TitleDescriptor.HasCapability()` n'est appelé qu'**une seule fois** (Asset Drawer V1). Les routes Battle Pass, Challenges, Season Pass, Career, Squad, Squad V2, Match View, Citations, Media n'interrogent jamais le registry pour vérifier qu'un titre supporte la fonctionnalité. Conséquence : en bootant un titre B sans capability `firefight` ou `media`, les routes `/pages/squad`, `/pages/media` etc. tomberont en run-time error DuckDB plutôt qu'en `503 capability_not_supported` propre.
- **Action** : ajouter un middleware ou un guard `RequireCapability(cap)` autour des sous-arbres de routes correspondants (au minimum BP, Challenges, Season Pass, Firefight stats, Media), retournant 503 + `capability_not_supported` côté handler.

### [DETTE] 4 row-types `domain.*MatchRow` parallèles à `canonical.PlayerMatchRow`, calqués 1:1 sur le SQL Halo

- **Fichier:ligne** : `internal/domain/home.go:15-64` (40 colonnes), `internal/domain/squad.go:29-53` (SquadMatchRow), `internal/domain/squad.go:259` (SynthesisMatchRow), `internal/domain/stats.go:12` (StatsMatchRow), `internal/domain/timeseries.go:110` (TimeseriesMatchRow)
- **Problème** : ces 5 types incluent des champs Halo-spécifiques (`SkillTier`, `IsFirefight`, `PerfectKills`, `TeamMMR/EnemyMMR`) au même niveau que des champs cross-titres (`Kills`, `Deaths`, `Outcome`). Le mapping est fait par le repo qui scanne directement les colonnes DuckDB. En multi-titres, soit chaque titre étend ces structs (couplage), soit chaque titre invente son propre `*MatchRow` (duplication N×titres).
- **Action** : à terme, projeter ces row-types vers `canonical.PlayerMatchRow` au niveau repo + dataAdapter, puis recomposer côté service via `canonical.MatchSummary + Self + Enrichment`. Sinon, expliciter dans un commentaire que ces structs sont « domain Halo Infinite » et non transverses.

### [DETTE] `analysis/home.go` : 5 constantes Halo hardcodées + slug littéral à 2 endroits

- **Fichier:ligne** : `internal/analysis/home.go:24,31-40,1218`
- **Extrait** :
  ```go
  const homeStaticTitleSlug = "halo_infinite"
  const ( homeOutcomeWin = 2; homeOutcomeLoss = 3; homeOutcomeTie = 1; homeOutcomeDNF = 4; ... )
  // ...
  mapImageURL := buildMapImageURL("halo_infinite", m.MapID, m.MapName, m.MapNameFR)
  ```
- **Problème** : le package `analysis` est censé être stateless et title-agnostique (« algorithmes purs » selon la convention `arch-rules`). Ici, il code en dur la sémantique outcome Halo + le slug pour composer une URL d'image. Quand un 2e titre arrivera, le calcul des highlights de la home Halo MCC hériterait de cette fonction comme si c'était Halo Infinite.
- **Action** : passer `titleSlug` en paramètre à `BuildHomeBlock()` (ou retourner les codes outcome bruts et laisser le service les mapper via `canonical.Outcome`). Idem pour `homeStaticTitleSlug`.

### [DETTE] `internal/analysis/mode_category.go` + `media_repo.go` — catégories `Assassin/Fiesta/BTB/Firefight/Other` codées en dur en Go

- **Fichier:ligne** : `internal/analysis/mode_category.go:46-55`
- **Extrait** :
  ```go
  const (
      ModeCategoryAssassin    = "Assassin"
      ModeCategoryFiesta      = "Fiesta"
      ModeCategorySuperFiesta = "Super Fiesta"
      ModeCategoryHuskyRaid   = "Husky Raid"
      ModeCategoryBTB         = "BTB"
      ModeCategoryRanked      = "Ranked"
      ModeCategoryFirefight   = "Firefight"
      ModeCategoryOther       = "Other"
  )
  ```
- **Problème** : ces catégories sont des regroupements de sous-modes Halo Infinite portés depuis Python. Elles sont consommées par `media_repo.go` (filtrage galerie) et `analysis/breakdown/by_mode.go`. Or `config/titles/halo_infinite/mappings/assets.toml [assets.mode.*]` définit déjà ces clés via TOML : il y a duplication FS↔Go sans synchronisation forcée.
- **Action** : déplacer la liste des catégories vers `assets.toml` côté `[assets.mode_category]`, charger via `mappings.AssetMappingSet` au boot et faire passer `media_repo.go` par le SemanticAdapter. Sinon documenter que c'est intentionnellement Halo-only et exclure du périmètre.

### [DETTE] `citations_custom.go` : 25 fonctions de calcul de médailles avec `strings.Contains(playlist, "btb")`/`"slayer"`/`"ctf"` codé en dur Halo

- **Fichier:ligne** : `internal/analysis/citations_custom.go:44-94` (et suivantes pour ~25 helpers)
- **Extrait** :
  ```go
  if strings.Contains(pl, "slayer") || strings.Contains(pl, "assassin") ||
      strings.Contains(gv, "slayer") || strings.Contains(gv, "assassin") {
      return 1
  }
  ```
- **Problème** : tout le moteur de citations est Halo-spécifique par construction. Pas de capability gate, pas d'isolation. Si Halo MCC a son propre moteur de citations, le code se duplique ou diverge.
- **Action** : entourer l'enregistrement du moteur citations par `HasCapability(CapCitationsEngine)` côté service (la capability est déjà déclarée dans `games/adapter.go:46`), et déplacer ce package sous `internal/games/halo_infinite/citations/`.

### [DETTE] Hardcode `"HINF-CSR_"` dans `home_repo.go` pour les badges CSR

- **Fichier:ligne** : `apps/go-api/internal/platform/duckdb/home_repo.go:413,418`
- **Extrait** :
  ```go
  if strings.EqualFold(normalizedTier, "Onyx") {
      id = "120px-HINF-CSR_Onyx"
  } else { ...
      id = fmt.Sprintf("120px-HINF-CSR_%s%d", normalizedTier, normalizedSubTier)
  }
  ```
- **Problème** : la composition d'URL CSR devrait passer par `TitleAssetURLAdapter.CSRRankImageURL(tier, subTier)` (déjà implémentée dans `internal/games/halo_infinite/adapter_asset_urls.go`). Ici elle est dupliquée, et le préfixe `HINF-CSR_` est codé en dur côté repo. Symétrique côté frontend (`apps/web/src/lib/staticAssets.ts:55-57`).
- **Action** : injecter le `TitleAssetURLAdapter` dans `HomeRepo` (ou plutôt dans le service au-dessus) et appeler `adapter.CSRRankImageURL(tier, sub)`. Supprimer le builder local.

### [DETTE] `engagement_score_service.go:345-355` — fallback hardcodé `"halo_infinite"` dans une lecture de contexte

- **Fichier:ligne** : `apps/go-api/internal/service/engagement_score_service.go:339-355`
- **Extrait** :
  ```go
  func titleSlugFromContext(ctx context.Context) string {
      type ctxKey string
      const titleSlugKey ctxKey = "title_slug"
      if v := ctx.Value(titleSlugKey); v != nil { ... }
      return "halo_infinite"
  }
  ```
- **Problème** : duplication locale d'une `ctxKey` au lieu d'utiliser `ctxkeys.TitleSlug(ctx)`. Le commentaire mentionne « éviter une dépendance circulaire potentielle » mais le service importe déjà `internal/domain` sans pb. Risque : la valeur lue ne sera jamais celle posée par le middleware si la `ctxKey` n'est pas exactement la même type-matched.
- **Action** : remplacer par `import "levelup/go-api/internal/ctxkeys"` + `ctxkeys.TitleSlug(ctx)` (la signature retourne déjà `"halo_infinite"` en fallback).

### [DETTE] Pas de middleware `RequireCapability` malgré le pattern explicite dans `adapter.go`

- **Fichier:ligne** : `internal/games/adapter.go:38-47` (`CapMatchHistory`, `CapMatchSkillSnapshot`, `CapPveFirefight`, `CapCitationsEngine` etc.)
- **Problème** : 8 capabilities sont déclarées et 5 retournent `ErrCapabilityNotSupported` côté `halo_infinite/adapter_data.go` (pour des champs non encore portés). Mais aucun middleware ne les expose en HTTP. Conséquence : un caller front qui appelle `/pages/match-history` sur un titre dont `CapMatchHistory` serait absent obtiendra 500 au lieu de 503 sémantique.
- **Action** : ajouter `middleware.RequireCapability(reg, cap)` et l'appliquer aux sous-arbres concernés. Le pattern existe déjà ad-hoc dans `assets_metadata.go` — le promouvoir.

### [AMÉLIORATION] `homeStaticTitleSlug` dupliqué dans 2 fichiers

- **Fichier:ligne** : `internal/analysis/home.go:24` et `internal/platform/duckdb/home_repo.go:20`
- **Extrait** :
  ```go
  const homeStaticTitleSlug = "halo_infinite"  // analysis/home.go
  const homeStaticTitleSlug = "halo_infinite"  // platform/duckdb/home_repo.go
  ```
- **Problème** : duplication de constante avec même nom dans deux packages. Devrait être centralisée ou supprimée au profit de `title.DefaultSlug`.
- **Action** : remplacer par `title.DefaultSlug` (déjà importé dans `home_repo.go`) ; ou mieux, propager `pdb.TitleSlug` depuis le PlayerDB.

### [AMÉLIORATION] Le frontend ne gate pas la navigation par capabilities — `availableTitles[0]` peut suggérer des sections non supportées

- **Fichier:ligne** : `apps/web/src/components/shell/AppShellHeader.tsx:39-53` + `apps/web/src/stores/appShellStore.ts:78`
- **Problème** : le `TitleSwitcher` propose tous les titres disponibles, mais `appShellStore` ne lit pas `TitleSummary.capabilities` pour griser des sections L1/L2 (ex: désactiver l'onglet "Firefight" sur un titre sans `CapFirefight`). La nav ignore donc les capabilities du titre courant.
- **Action** : enrichir `shellNavigation.ts` pour filtrer les liens L1/L2 selon `useAppShellStore(s => s.availableTitles.find(t => t.slug === s.currentTitleSlug)?.capabilities)`. Aujourd'hui Halo Infinite a tout, donc le bug ne se voit pas, mais c'est un piège pour le 2e titre.

### [AMÉLIORATION] `synthetic_title_b` n'est pas inclus dans `multiTitleSlugs` du loader au boot

- **Fichier:ligne** : `apps/go-api/internal/api/server.go:93-94`
- **Extrait** :
  ```go
  multiTitleSlugs := []string{titlePkg.DefaultSlug}
  for _, err := range fieldMappingsRegistry.LoadFromConfigDir(cfg.RepoRoot, multiTitleSlugs, slog.Default()) { ... }
  ```
- **Problème** : `synthetic_title_b` a son `fields.toml` + `assets.toml` + `outcomes.toml` versionnés Git (`config/titles/synthetic_title_b/mappings/`), mais le boot ne le charge pas. Les tests d'isolation passent en chargeant manuellement le TOML via `mustLoadFields`, mais le serveur live ignore complètement cette branche.
- **Action** : passer `[]string{titlePkg.DefaultSlug, synthetic_title_b.TitleSlug}` derrière un flag ou en mode test. Sinon, marquer explicitement que le synthétique est test-only et ne tournera jamais en prod.

## Cartographie : flux multi-titres pour 1 endpoint (ex: GET /api/v1/players/{slug}/pages/home)

```
Client React (apps/web/src/features/home/HomePage.tsx)
  ├─ useAppShellStore.currentTitleSlug         ← titre courant, persisté en session
  └─ api.get('/players/{slug}/pages/home')
        │
        │  (header X-LevelUp-Title injecté par client.ts:40)
        ▼
chi router (internal/api/server.go:425)
  └─ middleware.TitleExtractor (middleware/title.go:30-47)
        ├─ priorité : header X-LevelUp-Title → session.CurrentTitleSlug → "halo_infinite"
        └─ ctxkeys.WithTitleSlug(ctx, slug)
              │
              ▼
HomeHandler.GetHomePage (handlers/home.go)
  └─ reg.HomeCtxWithAuth (api/registry.go)
        ├─ resolve: PlayerResolver(ctx, slug) ← lit ctxkeys.TitleSlug
        ├─ config.ResolvePlayer(ctx, cfg, slug, titleSlug)
        │     └─ buildPoolConfig → PathResolver.PlayerDBPath(titleSlug, gamertag)
        │           = data/titles/halo_infinite/players/{gt}/stats.duckdb
        ▼
HomeService.GetHomePage(ctx, ...)
  └─ HomeRepo.LoadHomeMatches(ctx)        ← scan SQL direct, 0 canonical
        └─ retourne []domain.HomeMatchRow ← Halo-spécifique, 40 colonnes
              │
              ▼
analysis.BuildHomeBlock(rows, ...)        ← const homeStaticTitleSlug = "halo_infinite" hardcodé
  └─ buildMapImageURL("halo_infinite", ...) ← 2e hardcode
        │
        ▼
domain.HomePageResponse                    ← URLs `/static/.../halo_infinite/...`
```

**Verdict du flux** : titre injecté en surface (route + DB path), mais perdu sitôt qu'on entre dans `analysis/`. Aucune capability vérifiée. Aucun passage par `canonical.MatchSummary`. Le « titre courant » ne sert qu'à choisir la base DuckDB et composer 1 URL d'image.

## Suivi recommandé

1. **Décision canonical PlayerMatchRow** : trancher par ADR si les 4 row-types `Home/Stats/Synthesis/Squad` doivent migrer vers `canonical.PlayerMatchRow`, et avec quelle politique de transition. Sinon réécrire la doc pour assumer « canonical pour les services nouveaux uniquement ».
2. **Middleware capabilities** : `middleware.RequireCapability(cap)` autour des sous-arbres de routes (BP, Challenges, Firefight, Citations engine, Media) — débloque le « 503 propre » multi-titres et permet d'éteindre un titre incomplet sans 500.
3. **Schéma DuckDB** : ADR documentant la stratégie « isolation par chemin FS, pas de `title_id` colonne », ou ajout sélectif (au moins sur `xuid_aliases`).

## Constats hors-axe à reverser ailleurs

- **Axe Go layering / arch-rules** : `analysis/home.go` viole l'agnosticité du package `analysis` en hardcodant `homeStaticTitleSlug` + `buildMapImageURL("halo_infinite", ...)`. Logique title-aware dans la couche stateless.
- **Axe i18n / fields mappings** : `MULTI_TITLE_API_ENABLED` OFF prive le frontend du retour des mappings TOML — toute la couche `useFieldLabel` tourne en mode fallback en prod (déjà mentionné supra mais aussi pertinent pour l'axe i18n).
- **Axe tests** : aucun contracttest end-to-end avec `synthetic_title_b` sur les endpoints HTTP (uniquement des tests `_test.go` unitaires sur le loader TOML). À reverser sur l'axe « tests d'intégration ».
- **Axe outcomes/types canoniques** : 5 services + 2 fichiers analysis comparent `Outcome == 2` au lieu d'utiliser `domain.OutcomeWin` ou `canonical.OutcomeWin` — magic number, pertinent pour l'axe « anti-patterns/magic numbers ».

---

## Amendement post-vérification (2026-04-29)

> Ajouts issus de la passe de vérification finale (cf. [verification-finale-scaffolding.md](verification-finale-scaffolding.md)).

### [DETTE] 6 capabilities Halo déclarées mais jamais consommées via `HasCapability`

- **Fichier:ligne** : `apps/go-api/internal/domain/title/registry.go:31-37` (déclare 7 capabilities). Seule `CapAssetImages` est consommée runtime (`apps/go-api/internal/api/handlers/assets_metadata.go:34,58`).
- **Capabilities dormantes** : `CapMatchmaking`, `CapFirefight`, `CapForge`, `CapMedia`, `CapRanked`, `CapCareer` — référencées uniquement à l'init et dans `cmd_title.go` pour print, jamais en runtime.
- **Problème** : l'infrastructure de gating existe mais n'est jamais sollicitée pour 6 caps sur 7. Un titre B incomplet ne peut pas dégrader proprement (renvoie 500 au lieu de 503). Recoupe le BLOQUANT « `HasCapability` quasi inutilisé en prod » mais quantifie précisément le gap.
- **Action** : soit câbler `middleware.RequireCapability(cap)` autour des sous-arbres de routes correspondants (BP/Challenges, Firefight, Citations, Media, Career), soit supprimer les 6 caps mortes pour ne garder que `CapAssetImages`.

### [DETTE] Endpoint Go `/api/v1/titles/{slug}/preview/career` orphelin côté front

- **Fichier:ligne** : `apps/go-api/internal/api/server.go:260` (handler `previewHandler.GetCareerPreview`).
- **Problème** : aucun consommateur dans `apps/web/src/`. Endpoint derrière `MULTI_TITLE_API_ENABLED` mais même flag activé, rien ne le fetch.
- **Action** : à archiver (preview admin/debug ?) ou brancher dans une page admin.

### [DETTE] Endpoint Go `/api/v1/players/{slug}/preview/career-multi-title` orphelin côté front

- **Fichier:ligne** : `apps/go-api/internal/api/server.go:278` (handler `playerPreviewHandler.GetCareerPreview`).
- **Problème** : aucun consommateur côté front. Même statut que le précédent.
- **Action** : à archiver ou brancher.
