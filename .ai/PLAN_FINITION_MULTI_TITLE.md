# PLAN_FINITION_MULTI_TITLE.md — Plan d'achèvement du chantier multi-titres

> Plan rédigé le 2026-04-26 pour clore les vrais gaps du `PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` §14 et étendre le scope aux nouvelles pages title-bound (Media, Objectifs, Communauté).
>
> Contexte : la branche `feat/multi-title-adapters-and-mappings` a livré Phases A→F mais les critères §14.2 (synthesis + explorer non câblés) et §6.3-§6.4 (assets.toml + outcomes.toml absents, méthodes `Assets()`/`Outcomes()` absentes de l'interface) restent ouverts. Plus, depuis la rédaction du plan d'origine, 3 nouvelles surfaces produit title-bound ont été ajoutées : Media (refondue), Objectifs (Prestige), Communauté (Leaderboard PP).

---

## TL;DR

1. **6 gaps réels** vs critères §14 du plan d'origine + **3 nouvelles surfaces title-bound** non couvertes.
2. Périmètre : interfaces Go étendues + 2 TOML supplémentaires + bascule services Synthesis/Explorer + logs `adapter_loaded`/`adapter_load_failed` + frontière i18n React/TOML pour les nouvelles pages.
3. Effort estimé : **3–4j** (toutes corrections incluses), découpé en 5 phases.

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
5. Mise à jour `.ai/BACKLOG.md` : retirer l'entrée `[Multi-titre/Phase D-bis]` qui est désormais terminée. Conserver `weapon_family` + autres entrées multi-titre.

---

## 3. Critères d'acceptation finaux

Le chantier est livré quand **les 11 critères du plan d'origine §14 sont tous ✅** + 3 nouveaux critères :

| # | Critère | Source |
|---|---|---|
| 1–11 | Critères §14 plan d'origine | PLAN_MULTI_TITLE §14 |
| **12** | `assets.toml` + `outcomes.toml` créés pour HI et synthetic_title_b | Phase 1 ce plan |
| **13** | `Assets()` + `Outcomes()` méthodes implémentées dans `TitleSemanticAdapter` | Phase 1 ce plan |
| **14** | Pages Media + Objectifs + Communauté consomment `useAssetLabel`/`useOutcomeLabel` | Phases 3+4 ce plan |

---

## 4. Effort consolidé

| Phase | Effort | Détail |
|---|:---:|---|
| Phase 1 — Backend assets/outcomes | 1.5j | Interface + 2 TOML × 2 titres + loaders + tests |
| Phase 2 — Logs + Synthesis/Explorer | 0.5j | adapter_loaded + 2 services câblés + golden parity étendu |
| Phase 3 — Frontend Media | 0.5j | hooks + migration MediaToolbar |
| Phase 4 — Frontend Objectifs/Communauté | 0.5j | migration TIER/Cadence/Status + lint étendu |
| Phase 5 — Docs + verif | 0.5j | 3 docs + golden + ckecks |
| **Total** | **3.5j** | — |

---

## 5. Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|---|:---:|:---:|---|
| Bascule Synthesis casse un endpoint | Moyen | Haut | Golden parity diff = 0 + flag |
| Migration Objectifs casse les `TIER_COLORS` (couleurs UI) | Faible | Moyen | `color_token` côté TOML + résolution thème côté React |
| Endpoint `/field-mappings` payload trop gros (43 fields + 30 assets + 4 outcomes) | Faible | Faible | Compression gzip déjà active + ETag |
| Drift entre `Tier` enum Go et TOML | Moyen | Moyen | Test cross-titre §12.4 (déjà en place pour FieldKey, à étendre) |

---

## 6. Documents liés

1. `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` — plan parent.
2. `.ai/AUDIT_I18N_REACT_2026-04-25.md` — base de la frontière TOML/i18n React.
3. `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` — chantier weapon_family (toujours bloqué par 2nd titre, hors scope ce plan).
4. `.ai/BACKLOG.md` — entrées Phase D-bis (retirée à la livraison) + weapon_family (conservée).
