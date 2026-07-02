# Skill : foundations-usage — Checklist quand tu écris une nouvelle page ou un nouveau chart

## Quand activer ce skill

À l'écriture de **toute nouvelle page service Go** ou **tout nouveau composant frontend** qui consomme des données Halo. Vérifie d'abord que tu n'as pas à réinventer une fondation existante.

Ce skill est une **checklist d'application des 4 fondations** documentées dans `docs/FOUNDATIONS_GUIDE.md`.

## Les 4 fondations transverses

| # | Fondation | Source de vérité | Skill détaillé |
|---|---|---|---|
| 1 | Types canoniques | `apps/go-api/internal/games/canonical/` | `canonical-types` |
| 2 | Pattern adapter | `apps/go-api/internal/games/adapter.go` | `arch-rules` |
| 3 | Manifests TOML i18n | `apps/web/src/lib/i18n/manifests/*.toml` | (ce skill) |
| 4 | Wrappers ECharts | `apps/web/src/components/charts/*.tsx` | `color-tokens` |

## Checklist à parcourir avant de coder

### Côté Go (nouveau service de page)

- [ ] **Types de retour canoniques ?** Service retourne `*domain.MyPage` qui contient des `[]canonical.MatchSummary`, `*canonical.PlayerStats`, `*canonical.CareerSnapshot` — **jamais** des structs title-specific.
- [ ] **Adapter au lieu de SQL inline ?** `s.data.LoadMatchSummaries(ctx, ids)` — pas de `conn.Query("SELECT ... FROM match_participants ...")` dans le service.
- [ ] **Dégradation gracieuse ?** `if errors.Is(err, games.ErrCapabilityNotSupported) { return &domain.MyPage{HasData: false}, nil }` — jamais de panic, jamais de 500 sur un titre qui ne supporte pas la feature.
- [ ] **Pas de slug en dur ?** Branche sur `desc.HasCapability(title.CapXxx)` ou les clés fines `capabilities.toml`, jamais sur `slug == "halo_infinite"`.
- [ ] **Logging structuré ?** `slog.InfoContext(ctx, "...", "player_xuid", xuid)` + `slog.ErrorContext(ctx, "...", "err", err)`.
- [ ] **Aucun `filepath.Join`(repoRoot, "data", ...)`** — passe par `paths.PlayerDBPath(slug, gt)` / `paths.SharedDBPath(slug)`.
- [ ] **Tests par couche** : pure (`analysis/`), mock-port (`service/`), `httptest` (`api/handlers/`), `:memory:` (`platform/duckdb/`).

### Côté frontend (nouvelle page React)

- [ ] **Manifest TOML créé ?** `apps/web/src/lib/i18n/manifests/<page>.toml` avec **clés FR + EN par entrée**.
- [ ] **Manifest régénéré ?** `node apps/web/scripts/build_i18n_manifests.mjs` → produit `lib/i18n/generated/<page>.ts`.
- [ ] **Strings via `formatMessage` ?** Aucune string FR/EN hardcodée dans le composant. Vérifier `npm run lint` (ESLint custom rule).
- [ ] **Couleurs via tokens ?** `tokenCssVar('outcome-win')` ou `resolveToken('perf-tier-2')` — aucun hex `#RRGGBB` ni classe Tailwind couleur (`text-red-*`, `bg-green-*`).
- [ ] **Wrapper ECharts existant ?** Avant de créer un nouveau chart, vérifier `apps/web/src/components/charts/README.md` (catalogue 11 wrappers) + visiter `/lab/charts` (sandbox).
- [ ] **Route file-based ?** Nouvelle route dans `apps/web/src/routes/`, jamais éditer `routeTree.gen.ts` à la main.
- [ ] **Query keys** dans `apps/web/src/lib/query/keys.ts`.
- [ ] **Labels stats canoniques** via `useFieldLabel('kills')` plutôt que hardcodé.

### Side notes

- **Service qui appelle un autre service** = anti-pattern. Si deux services ont la même logique, extraire dans `analysis/` (pur) ou `service/shared.go` (helper d'orchestration sans state).
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
<p style={{ color: tokenCssVar('outcome-win') }}>{t('home.matches.empty')}</p>
```

```go
// ❌ Branchement sur le slug
if slug == "halo_infinite" { /* feature halo */ }

// ✅ Branchement sur capability
if desc.HasCapability(title.CapRanked) { /* feature ranked */ }
```

## Workflow type pour ajouter une page

1. **Plan rapide** (mental ou dans `.ai/PLAN_<page>.md`) : couches touchées, types canoniques requis, capabilities branchées, charts utilisés, manifest i18n.
2. **Backend** : domain → port → service (avec adapter) → handler → tests par couche → registry à jour.
3. **Frontend** : manifest TOML FR+EN → regen → query/hook → composant page consommant `formatMessage` + wrappers ECharts → tests vitest.
4. **Smoke check** : `go test ./...`, `npm run typecheck && npm run lint && npx vitest run`, `node tools/lint-no-hardcoded-fields.mjs`.
5. **Doc** : entrée `thought_log.md` (date + statut + décision + résultats + prochaine étape).

## Refs

- `docs/FOUNDATIONS_GUIDE.md` — guide consolidé avec exemple de bout en bout
- `docs/adr/0001-charts-stack-echarts.md` — décision ECharts
- `docs/adr/0002-canonical-player-match-row.md` — décision types canoniques
- `docs/adr/0003-i18n-manifest-and-linter.md` — décision i18n manifest + lint
- `docs/adr/0004-narrative-engine.md` — décision narrative engine
- Skills connexes : `arch-rules`, `canonical-types`, `color-tokens`, `delivery-checklist`
