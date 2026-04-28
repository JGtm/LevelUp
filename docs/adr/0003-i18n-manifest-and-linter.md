# ADR 0003 — i18n via TOML manifests + custom ESLint linter

**Status** — Accepted (2026-04-27 → 2026-04-28). Implemented across Phase 2 + Phase 3 of `PLAN_META_FOUNDATIONS_GO.md`.

**Deciders** — Guillaume (GS), validated by 489+ migrated keys across 12 page-level manifests.

## Context

Before Phase 2, frontend i18n was **fragmented** :

- `apps/web/src/features/{home,media,palmares,squad}/i18n.ts` — feature-local FR/EN dictionaries (~250 lines each), with logic-bearing functions inside (`pageLabel(page, total)`, `progressTowardsRank(name)`).
- Hardcoded JSX strings in dozens of components (`"Sessions récentes"`, `"Aucun joueur sélectionné"`, etc.).
- Mode/asset/outcome labels duplicated across `assets.toml` (Go, by title) and TS dicts (React, by feature). No source of truth.
- No way to enforce "no hardcoded user-facing strings" automatically.

Symptoms :

- New strings ended up in 5 different places: feature i18n.ts, ad-hoc JSX, fallback dicts, mock fixtures, openapi.yaml.
- Adding a second locale (EN was approximate) meant hunting through 50+ files.
- Multi-title labels (`mode_categories`, `outcome_labels`) needed to come from TOML (Halo-side) but were also hardcoded React-side.

## Decision

**Adopt TOML manifests as the single source of truth for user-facing UI strings**, with a custom ESLint rule enforcing the contract.

### Architecture

```
apps/web/src/lib/i18n/
  manifests/
    home.toml          ← source of truth (FR + EN per key)
    media.toml
    palmares.toml
    explorer.toml
    citations.toml
    career.toml
    synthesis.toml
    timeseries.toml
    session.toml
    squad.toml
    match_view.toml
    common.toml
  generated/           ← built by scripts/build_i18n_manifests.mjs
    home.ts            ← export const homeManifest = { ... } as const
    home.ts            ← + export type HomeManifestKey = keyof typeof homeManifest
    ...
  format.tsx           ← formatMessage(manifest, key, locale, values?)
```

- Each manifest key has both `fr` and `en` values.
- ICU MessageFormat for plurals and parameter interpolation: `{n, plural, one {# match} other {# matches}}`.
- Generated `*.ts` files are **committed** (no runtime build dependency on TOML parser).
- Consumers call `formatMessage(homeManifest, 'home.challenges.title', locale, { count: 3 })`.

### Linter

Custom ESLint rule `@levelup/no-hardcoded-strings` (`apps/web/eslint-rules/no-hardcoded-strings.js`) detects JSX text literals and string-typed attributes (`title`, `aria-label`) that are not from the manifest.

A second project-level lint script `tools/lint-no-hardcoded-fields.mjs` detects hardcoded labels that match a canonical `FieldKey` from `config/titles/halo_infinite/mappings/fields.toml` outside an allow-list (legacy `i18n.ts` files, fallback dicts, sandbox routes).

## Consequences

### Positive

- **Single source of truth** — 489+ keys migrated across 12 manifests by 2026-04-28. Ten-fold reduction in places where strings live.
- **FR + EN locked at the source** — adding a key forces both locales. Phase 2 alone migrated 397 keys × 2 = 794 locale-string pairs.
- **Type safety** — `HomeManifestKey` is a union literal type. `formatMessage(homeManifest, "wrong.key", locale)` is a TS error.
- **Multi-title alignment** — when a label exists both in `config/titles/halo_infinite/mappings/fields.toml` (Go side) and in a manifest (React side), the React side resolves via `useFieldLabel(key)` which prefers the backend, falls back to the manifest. Consistency without duplication.
- **Linter prevents regressions** — `npm run lint` errors out on hardcoded JSX strings in `features/` and `components/`. `node tools/lint-no-hardcoded-fields.mjs` runs in pre-commit.
- **Adapter modules for legacy** — `media/i18n.ts` (251 lines), `palmares/i18n.ts` (251 lines), `home/{kpi,highlights,spartanIdentity}.i18n.ts` were rewritten as thin **adapters** preserving the existing `getXxxText(locale)` signature but resolving values via `formatMessage`. Zero churn in consumer pages while the source of truth migrated.

### Negative

- **Generated files are committed** — `apps/web/src/lib/i18n/generated/*.ts` must be regenerated when a manifest changes. `scripts/build_i18n_manifests.mjs` automates it but the dev must remember to run it (or rely on pre-commit).
- **ICU MessageFormat learning curve** — pluralization syntax (`{n, plural, one {# X} other {# Xs}}`) is unfamiliar to most. Mitigated by 12+ examples in existing manifests.
- **Lint allow-list maintenance** — `tools/lint-no-hardcoded-fields.mjs` whitelist (`/i18n.ts`, `/lab/charts/`, etc.) must be kept in sync as legacy `i18n.ts` modules are absorbed into manifests.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **i18next with JSON files** | Heavy runtime, no native ICU, no compile-time key checks, generated FR/EN out of sync risk. |
| **react-intl** | ICU-native but no compile-time key validation, JS-side key resolution at every render. |
| **Inline JS dicts (status quo)** | Fragmented, hard to lint, no FR/EN parity enforcement. |
| **Backend-side resolution via API** | Adds latency to every page render; defeats client-side i18n. |
| **gettext / .po files** | No JSON-friendly tooling, plural rules clunky for our needs. |

## References

- Manifests source: `apps/web/src/lib/i18n/manifests/*.toml` (12 files, 489+ keys).
- Generator: `apps/web/scripts/build_i18n_manifests.mjs`.
- Helper: `apps/web/src/lib/i18n/format.tsx`.
- Custom lint rule: `apps/web/eslint-rules/no-hardcoded-strings.js`.
- Project-level lint: `tools/lint-no-hardcoded-fields.mjs`.
- Migration commits: P2.A→G + P3.A→C (12 commits across 2026-04-27 / 2026-04-28).
