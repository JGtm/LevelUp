# PLAN_FINITION_MULTI_TITLE.md — Plan d'achèvement du chantier multi-titres

> Plan rédigé le 2026-04-26 pour clore les vrais gaps du `PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` §14 et étendre le scope aux nouvelles pages title-bound (Media, Objectifs, Communauté).
>
> **État au 2026-04-28** : Phases 1–5 livrées (recalibration après audit terrain — voir §0 ci-dessous). Phase 6 (static FS title-scoping) ouverte sur branche dédiée.

---

## 0. Recalibration (audit terrain 2026-04-28)

L'audit du repo révèle que les Phases 1–5 sont **déjà livrées** :

| Phase | État | Preuve |
|---|---|---|
| **1** Backend assets/outcomes | ✅ DONE | `internal/games/adapter.go:104-111` (interface `TitleSemanticAdapter` étendue avec `Assets()`+`Outcomes()`) ; `mappings/{assets.go, outcomes.go, loader_assets.go, loader_outcomes.go}` + tests ; TOMLs HI + synthetic_title_b livrés |
| **2** Logs + Synthesis/Explorer | ✅ DONE | `api/server.go:128-152` (events `adapter_loaded`/`adapter_load_failed` émis au boot) ; `service/{career,synthesis,explorer}_service.go` ont tous `WithDataAdapter()` |
| **3** Frontend Media | ✅ DONE | `MediaToolbar.tsx:275-282` consomme `useFieldMappings` + `assets.mode.{value}.label` avec fallback `text.toolbar.modeCategories` (pattern défensif intentionnel cf. `fallback.i18n.ts`) |
| **4** Frontend Objectifs/Communauté | ✅ DONE | 4 composants Prestige (`ChallengeCard`, `LeaderboardPP`, `MomentCard`, `StatsGlobales`) consomment `useAssetLabel('challenge_tier', tier)` avec fallback `TIER_LABELS_FR` (même pattern défensif) |
| **5** Docs + verif | ✅ DONE | `docs/ARCHITECTURE_V6.md` + `docs/FR/ARCHITECTURE_V6.md` documentent assets/outcomes/useAssetLabel ; `go build ./...` OK |

**Écart vs plan original** : les Phases 3+4 ont adopté un pattern fallback (TOML → dict React legacy si MULTI_TITLE_API_ENABLED=false) au lieu d'une suppression pure des dicts. C'est plus défensif et aligné sur `fallback.i18n.ts`. Les dicts legacy (`modeCategories`, `TIER_LABELS_FR`) ne sont **pas** du dead code — ils sont la backstop de dégradation gracieuse.

**Reste réellement à faire** : Phase 6 — static FS title-scoping (cf. §6 de ce plan). Toutes les autres phases sont en attente de retrait de l'historique de ce plan une fois Phase 6 livrée.

---

## TL;DR

1. **Phases 1–5 ✅ livrées** — interfaces Go étendues + 2 TOML × 2 titres + bascule Synthesis/Explorer + frontière i18n React/TOML + docs.
2. **Phase 6 ouverte** — 3e adapter `TitleAssetURLAdapter` + migration FS title-scoped + UPDATE DB (cf. audit `.ai/BACKLOG.md` 2026-04-26).
3. Effort restant : **2j** sur 1 PR (`feat/multi-title-static-fs-rescope`).
4. Branche Git active : `feat/multi-title-finition-toml` (recalibration plan + thought_log) ; Phase 6 attaquée ensuite sur `feat/multi-title-static-fs-rescope`.

---

## 1. Inventaire des gaps

### 1.1. Gaps vs critères §14 du plan d'origine

| # | Critère | Gap réel |
|---|---|---|
| §14.2 | Tous endpoints clés basculés | `synthesis_service.go` et `explorer_service.go` ne reçoivent pas `WithDataAdapter` |
| §14.6 | 4 logs structurés émis | `adapter_loaded` et `adapter_load_failed` (§8.1) jamais émis |
| §5.2 | `TitleSemanticAdapter` interface complète | Méthodes `Assets()` et `Outcomes()` absentes de l'interface |
| §6.3 | `assets.toml` par titre | Fichier non créé, seul `fields.toml` existe |
| §6.4 | `outcomes.toml` par titre | Fichier non créé |
| §10.5 | Schema versioning N+1 toléré | Pas de mécanisme de dispatcher dans le loader, pas de migration documentée |

### 1.2. Nouvelles surfaces title-bound (apparues post-plan)

| Surface | Fichier | Contenu sémantique title-bound |
|---|---|---|
| **Media** | `apps/web/src/features/media/i18n.ts:50-58` | `modeCategories` : `Assassin/Slayer`, `Fiesta`, `Super Fiesta`, `Husky Raid`, `BTB`, `Ranked`, `Firefight`, `Other` |
| **Objectifs** (Prestige) | `apps/web/src/lib/prestige.ts:59-64` + `ObjectifsPage.tsx` | `Tier` labels (`Normal/Heroic/Legendary/Mythic`), `Cadence` labels (`daily/weekly/monthly/free`), `ChallengeStatus` labels, métriques (réutilisent `kills`, `kda`, `accuracy` etc.) |
| **Communauté** (Leaderboard PP) | `apps/web/src/features/prestige/LeaderboardPPPage.tsx` | `brut/bonus/total` labels — peu de FieldKey métier, surtout structurel |

Ces 3 surfaces sont **title-bound par construction** (les modes, les médailles, les métriques de défi sont propres au titre), donc elles ont besoin du même traitement TOML que les pages migrées Phase D.

---

## 2. Plan en 5 phases

### Phase 1 — Backend : interface + TOML assets/outcomes (1.5j)

#### 1.1. Étendre l'interface `TitleSemanticAdapter`

```go
// internal/games/adapter.go
type TitleSemanticAdapter interface {
    TitleSlug() string
    SchemaVersion() int
    Fields() *mappings.FieldMappingSet
    Ranks() *mappings.RankCatalog
    Assets() *mappings.AssetMappingSet      // NOUVEAU
    Outcomes() *mappings.OutcomeMappingSet  // NOUVEAU
}
```

#### 1.2. Créer `internal/games/mappings/assets.go` + `outcomes.go`

- `AssetMappingSet` indexé par `kind` (`mode`, `playlist`, `medal_tier`, `challenge_tier`, `cadence`, `challenge_status`) puis par `id`.
- `OutcomeMappingSet` mappe `Outcome` enum (`win`, `loss`, `tie`, `dnf`) → `{labels, color_token}`.
- Loader strict avec validation locales `en+fr`, `display_order` cohérent par groupe, `color_token` non vide.

#### 1.3. Créer les TOML

```
config/titles/halo_infinite/mappings/
  ├── fields.toml      (existant, 43 entrées)
  ├── assets.toml      (NOUVEAU — modes + medal_tiers + challenge_tiers + cadences + challenge_statuses)
  └── outcomes.toml    (NOUVEAU — win/loss/tie/dnf)
config/titles/synthetic_title_b/mappings/
  ├── fields.toml      (existant)
  ├── assets.toml      (NOUVEAU — labels divergents)
  └── outcomes.toml    (NOUVEAU — labels divergents)
```

Contenu HI initial pour `assets.toml` (~30 entrées pour couvrir les modes Media + tiers Prestige + cadences) :

```toml
[meta]
title_slug     = "halo_infinite"
schema_version = 1

# ── Modes (catégories Media + canoniques Halo) ───────────────────────────────
[assets.mode.assassin]
labels = { en = "Slayer", fr = "Slayer" }
display_order = 10

[assets.mode.fiesta]
labels = { en = "Fiesta", fr = "Fiesta" }
display_order = 20

[assets.mode.super_fiesta]
labels = { en = "Super Fiesta", fr = "Super Fiesta" }
display_order = 25

[assets.mode.husky_raid]
labels = { en = "Husky Raid", fr = "Husky Raid" }
display_order = 30

[assets.mode.btb]
labels = { en = "Big Team Battle", fr = "Grande bataille en équipe" }
display_order = 40

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50

[assets.mode.firefight]
labels = { en = "Firefight", fr = "Baptême du feu" }
display_order = 60

[assets.mode.other]
labels = { en = "Other", fr = "Autre" }
display_order = 99

# ── Tiers de défi Prestige ──────────────────────────────────────────────────
[assets.challenge_tier.normal]
labels = { en = "Normal", fr = "Normal" }
color_token = "challenge.normal"
display_order = 10

[assets.challenge_tier.heroic]
labels = { en = "Heroic", fr = "Héroïque" }
color_token = "challenge.heroic"
display_order = 20

[assets.challenge_tier.legendary]
labels = { en = "Legendary", fr = "Légendaire" }
color_token = "challenge.legendary"
display_order = 30

[assets.challenge_tier.mythic]
labels = { en = "Mythic", fr = "Mythique" }
color_token = "challenge.mythic"
display_order = 40

# ── Cadences de défi ────────────────────────────────────────────────────────
[assets.cadence.daily]
labels = { en = "Daily", fr = "Quotidien" }
display_order = 10

[assets.cadence.weekly]
labels = { en = "Weekly", fr = "Hebdomadaire" }
display_order = 20

[assets.cadence.monthly]
labels = { en = "Monthly", fr = "Mensuel" }
display_order = 30

[assets.cadence.free]
labels = { en = "Free", fr = "Libre" }
display_order = 40

# ── Statuts de défi ─────────────────────────────────────────────────────────
[assets.challenge_status.draft]
labels = { en = "Draft", fr = "Brouillon" }
display_order = 10

[assets.challenge_status.active]
labels = { en = "Active", fr = "Actif" }
display_order = 20

[assets.challenge_status.completed]
labels = { en = "Completed", fr = "Terminé" }
display_order = 30

[assets.challenge_status.expired]
labels = { en = "Expired", fr = "Expiré" }
display_order = 40

[assets.challenge_status.abandoned]
labels = { en = "Abandoned", fr = "Abandonné" }
display_order = 50

# ── Tiers de médaille (regroupements UX) ────────────────────────────────────
[assets.medal_tier.bronze]
labels = { en = "Bronze", fr = "Bronze" }
color_token = "medal.bronze"
display_order = 10

[assets.medal_tier.silver]
labels = { en = "Silver", fr = "Argent" }
color_token = "medal.silver"
display_order = 20

[assets.medal_tier.gold]
labels = { en = "Gold", fr = "Or" }
color_token = "medal.gold"
display_order = 30

[assets.medal_tier.mythic]
labels = { en = "Mythic", fr = "Mythique" }
color_token = "medal.mythic"
display_order = 40
```

Contenu HI initial pour `outcomes.toml` :

```toml
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Loss", fr = "Défaite" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"

[outcomes.dnf]
labels = { en = "DNF", fr = "Abandon" }
color_token = "outcome.neutral"
```

#### 1.4. Tests

- `loader_assets_test.go` : valide TOML + 3 fixtures invalides (locale manquante, color_token manquant, asset_id collision).
- `loader_outcomes_test.go` : valide TOML + 2 fixtures invalides (outcome inconnu, locale manquante).
- `recorder_test.go` étendu : `asset_lookup_missing` rate-limité, mêmes seuils que `field_lookup_missing`.

---

### Phase 2 — Backend : émettre `adapter_loaded` + bascule Synthesis/Explorer (0.5j)

#### 2.1. Logs `adapter_loaded` au boot

```go
// internal/games/resolver.go (StaticResolver)
func (r *StaticResolver) RegisterData(slug string, adapter TitleDataAdapter) {
    // ...existing...
    r.logger.Info("adapter_loaded",
        "title_slug", slug,
        "kind", "data",
        "capabilities_count", len(adapter.Capabilities()),
    )
}
```

Pareil pour `adapter_load_failed` quand un titre échoue à charger (try/recover lors du boot d'un titre).

#### 2.2. `WithDataAdapter` sur Synthesis et Explorer

- `synthesis_service.go` : ajouter `dataAdapter games.TitleDataAdapter` + setter + tentative de bascule sur `LoadPlayerStats` (synthesis lit déjà des stats agrégées). Capability gating : si non supportée, fallback repo direct.
- `explorer_service.go` : idem — Explorer charge des matchs filtrés, naturellement compatible avec `LoadMatchSummaries`.

Test golden parity diff = 0 sur ces 2 services (étendre `multi_title_parity_test.go`).

---

### Phase 3 — Frontend : hooks + migration page Media (0.5j)

#### 3.1. Étendre le hook frontend

```ts
// apps/web/src/lib/i18n/fieldMappings.ts (existant)
// Ajouter :
export function useAssetLabel(kind: string, id: string): string { ... }
export function useOutcomeLabel(outcome: string): string { ... }
```

L'endpoint backend `/field-mappings` retourne déjà `outcomes` et `assets` dans la réponse (cf. plan §7.4 schéma JSON), il faut juste les exposer côté React.

#### 3.2. Migrer `apps/web/src/features/media/i18n.ts`

- Supprimer `modeCategories` (8 entrées) du dict React.
- Remplacer `text.toolbar.modeCategories[cat.value]` par `useAssetLabel('mode', cat.value)` dans `MediaToolbar.tsx`.
- Adapter `MediaPage.tsx` si la liste de catégories est dérivée du dict React.
- Mise à jour test Vitest `MediaToolbar.test.tsx`.

---

### Phase 4 — Frontend : migration pages Objectifs/Communauté (0.5j)

#### 4.1. Objectifs

- Migrer `TIER_LABELS_FR` vers `useAssetLabel('challenge_tier', tier)` dans `ObjectifsPage.tsx`, `ChallengeCard.tsx`, `CreateChallengeForm.tsx`.
- Idem pour Cadence (`useAssetLabel('cadence', cadence)`) et `ChallengeStatus` (`useAssetLabel('challenge_status', status)`).
- Métriques de défi (`kills`, `kda`, etc.) : déjà couvertes par `useFieldLabel` du plan d'origine — vérifier que les usages dans `ChallengeCard` et `CreateChallengeForm` passent bien par le hook.

#### 4.2. Communauté

- `LeaderboardPPPage.tsx` : peu de labels métier (brut/bonus/total sont structurels UI). Vérifier qu'aucun label de tier ou statut n'est hardcodé.
- Si détection : appliquer `useAssetLabel`.

#### 4.3. Lint anti-hardcode

Étendre `tools/lint-no-hardcoded-fields.mjs` pour scanner aussi les labels d'`assets.toml` (modes, tiers, cadences, statuses) et d'`outcomes.toml`. Aujourd'hui le script ne scanne que `fields.toml`.

---

### Phase 5 — Documentation + verif finale (0.5j)

1. Mise à jour `docs/ARCHITECTURE_V6.md` + `docs/FR/ARCHITECTURE_V6.md` avec la couche Assets/Outcomes.
2. Mise à jour `.ai/project_map.md`.
3. Entrée `.ai/thought_log.md` documentant les 5 phases.
4. Vérif finale :
   - `golangci-lint run ./...` clean.
   - `go test -count=1 ./...` vert (yc multi-title 100%).
   - `npm run typecheck && npm run lint && npm run build && npm run test:run` clean.
   - `npm run lint:fields` 0 violation (étendu aux assets/outcomes).
   - Vérif manuelle Media/Objectifs/Communauté en FR + EN.
5. Mise à jour `.ai/BACKLOG.md` : retirer l'entrée `[Multi-titre/Phase D-bis]` qui est désormais terminée. Conserver `weapon_family` + l'entrée `Migration static/ vers une arborescence title-scoped` (déplacée vers Phase 6 ci-dessous, à retirer du BACKLOG en fin de Phase 6).

---

### Phase 6 — Static FS title-scoping (1.5–2j, PR séparée)

> Périmètre repris in extenso de l'audit `.ai/BACKLOG.md` §`[Multi-titre] Migration static/`. Ne pas redécrire l'audit ici — ce plan définit la **séquence de livraison** et les **critères de done**. La conception en 4 couches SRP, le contenu des fichiers, et les sites de hardcodage (A1–A5, B1–B2, C1, D1–D4, E1–E2, F, G) sont la source de vérité dans le BACKLOG.
>
> **Branche** : `feat/multi-title-static-fs-rescope` (créée depuis `main` après merge de la PR Phases 1–5).
>
> **Stratégie** : strangler-fig + feature flag + commit big bang atomique (cf. BACKLOG §`Méthodologie de bascule sans casse`). Aucun moment de la séquence ne doit produire un état où URLs émises ≠ structure FS.

#### 6.1. Couche 2 — package `internal/assets/static/` (0.25j)

- Créer `internal/assets/static/{kinds.go, layout.go, urls.go, fs.go}` (~80 LoC) — pur, zéro dépendance titre.
- Tests `static_test.go` table-driven : 15 cas (Kind valide/invalide, slug vide, id vide, ext optionnelle, encodage path-safe).
- Critère done : `go test ./internal/assets/static/...` vert. Aucun caller modifié à ce stade.

#### 6.2. Couche 3 — `TitleAssetURLAdapter` interface + impl HI (0.5j)

- Ajouter `TitleAssetURLAdapter` dans `internal/games/adapter.go` (méthodes `MapImageURL`, `MedalImageURL`, `CSRRankImageURL`, `CSRRankImageURLOnyx`, `WeaponImageURL`, `CommendationImageURL`).
- Étendre `Resolver` (`internal/games/resolver.go`) : ajouter `AssetURL(slug string) (TitleAssetURLAdapter, error)` + setter `RegisterAssetURL`. Émettre `adapter_loaded` (kind=`asset_url`) au boot — réutilise le pattern des Phases 1–2.
- Créer `internal/games/halo_infinite/adapter_asset_urls.go` (impl HI) avec **flag `titleScoped` lu depuis ENV `STATIC_PATHS_TITLE_SCOPED`** (default OFF). Quand OFF : retourne URLs sans `titleSlug` (compat existant). Quand ON : insère `titleSlug` dans le path.
- Déplacer `mapPNGNames` de `home.go` vers l'adapter (la connaissance des extensions `.png` vs `.jpg` par map est title-spécifique).
- Câbler l'adapter dans le `Resolver` au boot (`internal/api/server.go`) — synthetic_title_b reçoit un stub minimal (toutes méthodes retournent "" — il n'a pas d'assets statiques).
- Tests :
  - `adapter_asset_urls_test.go` : encodage map names (espaces, accents, UUID rejection), CSR rank format, flag ON/OFF parité.
  - Test multi-titre : `synthetic_title_b` retourne "" sans paniquer.
- Critère done : 0 caller migré, flag OFF, comportement identique au runtime.

#### 6.3. Couche 4 — bascule des callers Go (0.25j)

- A1 : `internal/analysis/home.go::buildMapImageURL` → injecter `games.TitleAssetURLAdapter` via paramètre. Retirer `mapStaticImagePath` local.
- A2/A3/A4 : `internal/platform/duckdb/home_repo.go` → injecter `assetURL` dans le constructeur du repo, remplacer 3 `fmt.Sprintf("/static/...")`.
- A5 : `cmd/migrate-static-maps/main.go` → utiliser `static.AbsKindRoot` + `assetURL.MapImageURL`.
- B1 : `internal/api/server.go:199` → retirer `StaticMapDir` du config (la title-scoping passe par l'adapter, plus par un override) ; B2 inchangé.
- C1 : retirer `WithRootOverride(KindMapImage, ...)` cassé dans `internal/assets/wire.go` (le bug est documenté dans le BACKLOG §C1).
- F : rafraîchir les 4 commentaires (media.go:239, map_cache_repo.go:20, citation_snippets.go:96, kinds.go:14).
- Tests Go affectés : `home_test.go`, `home_repo_test.go`, `commendation_handler_test.go`, `store_localfs_test.go`, `store_duckdb_test.go` — fixtures inchangées tant que flag OFF (URLs identiques).
- Critère done : `go test ./... && go vet ./...` vert avec flag OFF, comportement runtime identique.

#### 6.4. Frontend — bascule vers `useAssetURL` (0.25j)

- Créer `apps/web/src/lib/staticAssets/{kinds.ts, layout.ts, halo_infinite.ts, useAssetURL.ts}` — symétrique au backend, **flag lu depuis env `VITE_STATIC_PATHS_TITLE_SCOPED`**.
- Migrer D1 : `HomePage.tsx:412` → `useAssetURL().csrRankImageURL('Unranked', 0)`.
- Migrer D2 : `HomeRecentPlaylistsCard.tsx:77` → idem.
- Tests Vitest D3 : `HomePage.test.tsx` fixtures inchangées tant que flag OFF.
- Critère done : `npm run typecheck && npm run lint && npx vitest run` vert avec flag OFF.

#### 6.5. Big bang atomique — flip flag + git mv + UPDATE DB (0.5j)

> 1 seul commit, 1 seule PR, atomique. Ordre opératoire (cf. BACKLOG `strangler-fig + flag + commit big bang atomique`) :

1. **Pre-flight check local** :
   - Snapshot DB : `cp data/warehouse/metadata.duckdb data/warehouse/metadata.duckdb.bak.<date>`
   - Inventaire FS : `find static -type f | sort > /tmp/static-before.txt`
2. **Migration FS** :
   - `git mv static/maps/* static/maps/halo_infinite/` (créer dossier d'abord)
   - Idem pour `static/medals/icons/`, `static/ranks/`, `static/weapons-assets/`
   - `git mv static/commendations/h5g/ static/commendations/halo_5_guardians/` (G — décision verrouillée)
   - `git mv static/commendations/hi/ static/commendations/halo_infinite/`
3. **UPDATE DB** (E1 + G) — script `cmd/migrate-static-paths/main.go` (jetable, supprimé après run prod) :
   ```sql
   UPDATE map_images_registry
   SET local_path = REPLACE(local_path, '/static/maps/', '/static/maps/halo_infinite/')
   WHERE title_id = 'halo_infinite' AND local_path LIKE '/static/maps/%';

   UPDATE citation_mappings
   SET image_path = REPLACE(image_path, 'static/commendations/h5g/', 'static/commendations/halo_5_guardians/')
   WHERE image_path LIKE 'static/commendations/h5g/%';

   UPDATE citation_mappings
   SET image_path = REPLACE(image_path, 'static/commendations/hi/', 'static/commendations/halo_infinite/')
   WHERE image_path LIKE 'static/commendations/hi/%';
   ```
   Wrappé par script Go avec `slog.InfoContext(..., "rows_updated", n)` + dry-run mode.
4. **Flip flags** :
   - `cmd/api/main.go` (ou config init) : default `STATIC_PATHS_TITLE_SCOPED=true` (ENV reste overridable).
   - `apps/web/.env` (ou Vite config) : `VITE_STATIC_PATHS_TITLE_SCOPED=true`.
5. **Mise à jour fixtures** :
   - D3 : `HomePage.test.tsx` fixtures `'/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png'`.
   - D4 : `home_test.go`, `player_repos_test.go`, `commendation_handler_test.go`, `store_localfs_test.go`, `store_duckdb_test.go` — fixtures cohérentes.
6. **Vérif post-merge local** :
   - `find static -type f | sort > /tmp/static-after.txt && diff /tmp/static-before.txt /tmp/static-after.txt` → seul changement = paths title-scoped.
   - Démarrer api + web, parcourir HomePage / Media / Objectifs en FR + EN, **inspecter network 0 × 404 sur `/static/...`**.
   - `duckdb data/warehouse/metadata.duckdb "SELECT COUNT(*) FROM map_images_registry WHERE local_path NOT LIKE '/static/maps/halo_infinite/%' AND title_id = 'halo_infinite'"` → 0.

**Rollback** : `git revert <commit>` + `cp metadata.duckdb.bak.<date> metadata.duckdb`. Atomicité garantit qu'on n'a jamais d'état partiel.

#### 6.6. Cleanup + documentation (0.25j)

- Supprimer `cmd/migrate-static-paths/main.go` (script jetable post-migration prod).
- Supprimer la branche flag dans `adapter_asset_urls.go` (Go) et `halo_infinite.ts` (TS) : la title-scoping devient le seul comportement. Garder le flag ENV serait du code mort.
- Mise à jour `docs/ARCHITECTURE_V6.md` + FR : section "Static assets — title-scoped via 4 couches SRP".
- Mise à jour `.claude/skills/arch-rules/SKILL.md` : ajouter règle "Aucun `/static/...` hardcodé hors `internal/assets/static/`".
- Étendre lint : `tools/lint-no-hardcoded-static.mjs` ou règle ESLint custom qui interdit `'/static/'` dans `apps/web/src/features|components/`.
- Mise à jour `.ai/project_map.md` + entrée `.ai/thought_log.md` Phase 6.
- **Retirer l'entrée `[Multi-titre] Migration static/`** de `.ai/BACKLOG.md`.
- Critère done : `golangci-lint run ./...` clean, `go test -count=1 ./...` vert, `npm run typecheck && lint && build && test:run` vert, lint-no-hardcoded-static 0 violation.

---

## 3. Critères d'acceptation finaux

Le chantier est livré quand **les 11 critères du plan d'origine §14 sont tous ✅** + 6 nouveaux critères :

| # | Critère | Source |
|---|---|---|
| 1–11 | Critères §14 plan d'origine | PLAN_MULTI_TITLE §14 |
| **12** | `assets.toml` + `outcomes.toml` créés pour HI et synthetic_title_b | Phase 1 ce plan |
| **13** | `Assets()` + `Outcomes()` méthodes implémentées dans `TitleSemanticAdapter` | Phase 1 ce plan |
| **14** | Pages Media + Objectifs + Communauté consomment `useAssetLabel`/`useOutcomeLabel` | Phases 3+4 ce plan |
| **15** | `TitleAssetURLAdapter` 3e interface ajoutée au `Resolver`, impl HI complète | Phase 6 ce plan |
| **16** | `static/{maps,medals/icons,ranks,weapons-assets,commendations}/halo_infinite/` arborescence active, 0 fichier flat restant | Phase 6 ce plan |
| **17** | 0 hardcodage `/static/...` hors `internal/assets/static/` (Go) et hors `apps/web/src/lib/staticAssets/` (TS), enforced par lint | Phase 6 ce plan |

---

## 4. Effort consolidé

| Phase | Effort | PR | Détail |
|---|:---:|:---:|---|
| Phase 1 — Backend assets/outcomes | 1.5j | 1 | Interface + 2 TOML × 2 titres + loaders + tests |
| Phase 2 — Logs + Synthesis/Explorer | 0.5j | 1 | adapter_loaded + 2 services câblés + golden parity étendu |
| Phase 3 — Frontend Media | 0.5j | 1 | hooks + migration MediaToolbar |
| Phase 4 — Frontend Objectifs/Communauté | 0.5j | 1 | migration TIER/Cadence/Status + lint étendu |
| Phase 5 — Docs + verif | 0.5j | 1 | 3 docs + golden + checks |
| **Sous-total PR `feat/multi-title-finition-toml`** | **3.5j** | | |
| Phase 6.1 — Couche 2 `internal/assets/static/` | 0.25j | 2 | Package pur + tests table-driven |
| Phase 6.2 — Couche 3 `TitleAssetURLAdapter` HI | 0.5j | 2 | Interface + impl HI + flag + Resolver |
| Phase 6.3 — Bascule callers Go (A1–A5, C1, F) | 0.25j | 2 | 5 sites + suppression override cassé |
| Phase 6.4 — Frontend `useAssetURL` (D1–D2) | 0.25j | 2 | Hook + 2 sites HomePage |
| Phase 6.5 — Big bang atomique (FS + DB + flip + D3–D4) | 0.5j | 2 | git mv + UPDATE + fixtures + smoke |
| Phase 6.6 — Cleanup + docs + lint | 0.25j | 2 | Suppression flag + ARCHITECTURE_V6 + lint-no-hardcoded-static |
| **Sous-total PR `feat/multi-title-static-fs-rescope`** | **2j** | | |
| **Total chantier** | **5.5j** | | |

---

## 5. Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|---|:---:|:---:|---|
| Bascule Synthesis casse un endpoint | Moyen | Haut | Golden parity diff = 0 + flag |
| Migration Objectifs casse les `TIER_COLORS` (couleurs UI) | Faible | Moyen | `color_token` côté TOML + résolution thème côté React |
| Endpoint `/field-mappings` payload trop gros (43 fields + 30 assets + 4 outcomes) | Faible | Faible | Compression gzip déjà active + ETag |
| Drift entre `Tier` enum Go et TOML | Moyen | Moyen | Test cross-titre §12.4 (déjà en place pour FieldKey, à étendre) |
| **Phase 6 : URL émise ≠ FS pendant la transition** | Moyen | Haut | Strangler-fig + flag OFF par défaut sur 6.1–6.4 ; commit atomique 6.5 ; rollback `git revert + restore .duckdb.bak` |
| **Phase 6 : UPDATE DB partiel (rows_updated < expected)** | Faible | Haut | Script Go avec dry-run + `slog.Info("rows_updated", n)` + assertion post-UPDATE (`COUNT WHERE NOT LIKE 'halo_infinite/'` = 0) |
| **Phase 6 : 404 sur asset oublié post-flip** | Moyen | Moyen | Smoke check manuel network tab Home/Media/Objectifs FR+EN avant push ; lint-no-hardcoded-static détecte les reliquats ; Phase 6.4 fixtures pré-flippées |
| **Phase 6 : régression d'un caller hors audit BACKLOG** | Faible | Moyen | `grep -rn '"/static/' apps/go-api apps/web/src` post-Phase 6.3+6.4 doit retourner 0 hors `internal/assets/static/` et `apps/web/src/lib/staticAssets/` |

---

## 6. Documents liés

1. `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` — plan parent.
2. `.ai/AUDIT_I18N_REACT_2026-04-25.md` — base de la frontière TOML/i18n React.
3. `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` — chantier weapon_family (toujours bloqué par 2nd titre, hors scope ce plan).
4. `.ai/BACKLOG.md` — entrées Phase D-bis (retirée à la livraison Phase 5) + `Migration static/` (audit + 4-couches SRP, source de vérité conception Phase 6, retirée à la livraison Phase 6) + weapon_family (conservée).
5. `.ai/PLAN_ASSETS_ABSTRACTION.md` — chantier resolver cache-aside (`TitleAssetURLAdapter` est compatible, convergence future).
