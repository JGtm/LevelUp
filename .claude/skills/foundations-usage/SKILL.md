---
name: foundations-usage
description: Checklist des 4 fondations transverses (types canoniques, adapters, manifests i18n, wrappers ECharts) à parcourir avant d'écrire un nouveau service de page Go ou une nouvelle page React. À invoquer au début de toute page ou de tout graphe neuf, pour ne pas réinventer une fondation.
---

# Skill : foundations-usage — Checklist quand tu écris une nouvelle page ou un nouveau chart

## Quand activer ce skill

À l'écriture de **toute nouvelle page service Go** ou **tout nouveau composant frontend** qui consomme des données Halo. Vérifie d'abord que tu n'as pas à réinventer une fondation existante.

Ce skill est une **checklist d'application des 4 fondations** documentées dans `docs/FOUNDATIONS_GUIDE.md`.

## Les 4 fondations transverses

| # | Fondation | Source de vérité | Skill détaillé |
|---|---|---|---|
| 1 | Types canoniques | `apps/go-api/internal/games/canonical/` | `canonical-types` |
| 2 | Pattern adapter (`TitleDataAdapter` pour les données ; `TitleSemanticAdapter` pour les libellés, implémentation unique `GenericSemanticAdapter` dans `games/semantic_adapter.go` : un nouveau titre n'en écrit pas, il ajoute ses TOML) | `apps/go-api/internal/games/adapter.go` | `arch-rules` |
| 3 | Manifests TOML i18n | `apps/web/src/lib/i18n/manifests/*.toml` | (ce skill) |
| 4 | Wrappers ECharts | `apps/web/src/components/charts/*.tsx` | `color-tokens` |

## Checklist à parcourir avant de coder

### Côté Go (nouveau service de page)

- [ ] **Types de retour canoniques ?** Service retourne `*domain.MyPage` qui contient des `[]canonical.MatchSummary`, `*canonical.PlayerStats`, `*canonical.CareerSnapshot` — **jamais** des structs title-specific.
- [ ] **Adapter au lieu de SQL inline ?** Le service reçoit un `games.TitleDataAdapter` injecté (modèle : `CareerService.WithDataAdapter`, `service/career_service.go`) et appelle par exemple `s.dataAdapter.LoadEncounters(ctx, xuid)` — pas de `conn.Query("SELECT ... FROM match_participants ...")` dans le service. Aucun SQL dans `service/` : garde `TestMetierSansSQL` (`internal/archlint/frontieres_sql_test.go`).
- [ ] **Dégradation gracieuse ?** `if errors.Is(err, games.ErrCapabilityNotSupported) { return &domain.MyPage{HasData: false}, nil }` — jamais de panic, jamais de 500 sur un titre qui ne supporte pas la feature.
- [ ] **Pas de slug en dur ?** Branche sur `desc.HasCapability(title.CapXxx)` ou les clés fines `capabilities.toml`, jamais sur `slug == "halo_infinite"`.
- [ ] **Logging structuré ?** `slog.InfoContext(ctx, "...", "player_xuid", xuid)` + `slog.ErrorContext(ctx, "...", "err", err)`.
- [ ] **Aucun `filepath.Join`(repoRoot, "data", ...)`** — passe par `paths.PlayerDBPath(slug, gt)` / `paths.SharedDBPath(slug)`.
- [ ] **Tests par couche** : pure (`analysis/`), mock-port (`service/`), `httptest` (`api/handlers/`), `:memory:` (`platform/duckdb/`).

### Côté frontend (nouvelle page React)

- [ ] **Manifest TOML créé ?** `apps/web/src/lib/i18n/manifests/<page>.toml` avec **une clé par langue de `config/locales.toml` (fr + en obligatoires) par entrée**.
- [ ] **Manifest régénéré ?** `node apps/web/scripts/build_i18n_manifests.mjs` → produit `lib/i18n/generated/<page>.ts`. Le script refuse toute entrée qui n'a pas toutes les langues de `config/locales.toml` (ADR 0040).
- [ ] **Dictionnaire de feature ?** Les textes propres à une feature vivent dans `features/X/i18n/<langue>.ts`, typés `Record<Locale, T>`.
- [ ] **Strings via `formatMessage` ?** Aucune string FR/EN hardcodée dans le composant. Vérifier `npm run lint` (ESLint custom rule).
- [ ] **Couleurs via tokens ?** `tokenCssVar('outcome-win')` ou `resolveToken('perf-tier-2')` — aucun hex `#RRGGBB` ni classe Tailwind couleur (`text-red-*`, `bg-green-*`).
- [ ] **Wrapper ECharts existant ?** Avant de créer un nouveau chart, vérifier `apps/web/src/components/charts/README.md` (catalogue des wrappers, 12 à ce jour) + visiter `/lab/charts` (sandbox).
- [ ] **Pas de slug de titre en dur ?** Règle ESLint `no-title-slug-literal` (`apps/web/eslint-rules/`) : brancher sur les capacités du bootstrap.
- [ ] **Route file-based ?** Nouvelle route dans `apps/web/src/routes/`, jamais éditer `routeTree.gen.ts` à la main.
- [ ] **Query keys** dans `apps/web/src/lib/query/keys.ts`.
- [ ] **Labels stats canoniques** via `useFieldLabel('kills')` plutôt que hardcodé.

### Side notes

- **Service qui appelle un autre service** = anti-pattern. Si deux services ont la même logique, extraire dans `analysis/` (pur) ou dans un helper sans état du paquet `service/` (fichier `*_helpers.go`).
- **Si nouveau FieldKey** : ajouter dans `canonical/fields.go` ET dans le fields.toml de CHAQUE titre actif (`config/titles/{halo_infinite,halo_5}/mappings/fields.toml`) — un titre sans l'entrée ne supporte pas la surface.
- **Si nouvel asset/outcome** : ajouter dans `assets.toml` / `outcomes.toml` du titre.
- **Logique métier dans un handler** = anti-pattern. Le handler doit juste decoder/encoder.

## 3 antipatterns fréquents à attraper en review

```go
// ❌ Service qui requête DuckDB directement
func (s *MyService) Get(ctx) error {
    rows, _ := s.db.Query("SELECT kills FROM match_participants ...")
}

// ✅ Via adapter
func (s *MyService) Get(ctx) error {
    summaries, err := s.data.LoadMatchSummaries(ctx, ids)
    if errors.Is(err, games.ErrCapabilityNotSupported) { /* dégrader */ }
}
```

```tsx
// ❌ String hardcodée + couleur Tailwind
<p className="text-green-500">Aucun match disponible</p>

// ✅ formatMessage + token
<p style={{ color: tokenCssVar('outcome-win') }}>{t('home.empty.no_data_title')}</p>
```

```go
// ❌ Branchement sur le slug
if slug == "halo_infinite" { /* feature halo */ }

// ✅ Branchement sur capability
if desc.HasCapability(title.CapRanked) { /* feature ranked */ }
```

## Workflow type pour ajouter une page

1. **Plan rapide** (mental ou dans `.ai/PLAN_<page>.md`) : couches touchées, types canoniques requis, capabilities branchées, charts utilisés, manifest i18n.
2. **Backend** : domain → port (interface) → service (avec adapter) → handler (qui ne voit que l'interface du port) → câblage dans `api/wire/registry*.go` → tests par couche.
3. **Frontend** : manifest TOML FR+EN → regen → query/hook → composant page consommant `formatMessage` + wrappers ECharts → tests vitest.
4. **Smoke check** : `go test ./...` (avec `-tags=integration` si tu touches sync ou persist), `npm run typecheck && npm run lint && npx vitest run`, `npm run lint:fields`, `npm run lint:colors`, `node tools/lint-cross-feature-imports.mjs` (depuis la racine pour ce dernier).
5. **Doc** : entrée `thought_log.md` (date + statut + décision + résultats + prochaine étape).

## Refs

- `docs/FOUNDATIONS_GUIDE.md` — guide consolidé avec exemple de bout en bout
- `docs/adr/0001-charts-stack-echarts.md` — décision ECharts
- `docs/adr/0002-canonical-player-match-row.md` — décision types canoniques
- `docs/adr/0003-i18n-manifest-and-linter.md` — décision i18n manifest + lint
- `docs/adr/0040-i18n-n-locales.md` — i18n à N langues (amende 0003)
- `docs/adr/0004-narrative-engine.md` — décision narrative engine
- Skills connexes : `arch-rules`, `canonical-types`, `color-tokens`, `delivery-checklist`
