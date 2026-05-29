# Audit knip — Dead code TypeScript (apps/web)

> Généré le 2026-05-28 · outil : knip 6.14.2 · branche : feat/lusr-v2-phase0-metrics

---

## Résumé exécutif

| Catégorie | Count | Sévérité |
|---|---|---|
| Dépendance utilisée mais absente de package.json | 1 (`zod`) | 🔴 Critique |
| Fichiers entièrement orphelins | 40 | 🟠 Élevée |
| Exports non consommés | 90 | 🟡 Moyenne |
| Doublons de symboles | 2 fichiers | 🟡 Moyenne |
| Dépendances npm non utilisées | 2 | 🟡 Moyenne |
| Faux positifs documentés | 3 | ⚪ À ignorer |

---

## 🔴 P0 — Corriger immédiatement

### `zod` manquant de package.json

`zod` est importé dans 5 fichiers de routes mais absent de `package.json`. Ça fonctionne
uniquement parce que c'est une dépendance transitive — ça peut silencieusement casser
à la prochaine mise à jour de TanStack Router ou d'une autre lib.

```
src/routes/players/$playerSlug/compare.tsx:2
src/routes/players/$playerSlug/explorer/index.tsx:5
src/routes/players/$playerSlug/stats/timeseries.tsx:5
src/routes/players/$playerSlug/stats/sessions.tsx:5
src/routes/players/$playerSlug/matches/$matchId.tsx:8
```

**Fix :**
```bash
cd apps/web && npm install zod
```

---

## 🟠 P1 — Features entières orphelines (non câblées au routing)

Ces features existent dans le code mais ne sont importées par aucune route active.
**Décision requise : câbler ou supprimer.**

### `session-compare/` — 13 fichiers (feature complète)

La page et tous ses sous-composants sont déconnectés du routing TanStack.

```
src/features/session-compare/queries.ts
src/features/session-compare/SessionComparePage.tsx          ← page racine
src/features/session-compare/SessionCompareBarMetrics.tsx
src/features/session-compare/SessionCompareCumulative.tsx
src/features/session-compare/SessionCompareEngagement.tsx
src/features/session-compare/SessionCompareKDProgression.tsx
src/features/session-compare/SessionCompareKillsDonut.tsx
src/features/session-compare/SessionCompareMatchHistory.tsx
src/features/session-compare/SessionCompareMMR.tsx
src/features/session-compare/SessionCompareOCDR.tsx
src/features/session-compare/SessionCompareOutcomeTape.tsx
src/features/session-compare/SessionCompareParticipation.tsx
src/features/session-compare/SessionComparePerfProgression.tsx
src/features/session-compare/SessionCompareRadar.tsx
src/features/session-compare/SessionCompareSkillHeader.tsx
src/features/session-compare/SessionCompareSkillProgression.tsx
src/features/session-compare/SessionOutcomesDonut.tsx
```

Options :
- Si la feature est planifiée → créer la route dans `src/routes/` pour la câbler
- Si abandonnée → supprimer le dossier entier

### `squad/v2/` — 5 fichiers (refactor en cours)

```
src/features/squad/v2/queries.ts
src/features/squad/v2/SquadV2Page.tsx
src/features/squad/v2/SquadV2RouteHost.tsx
src/features/squad/v2/components/SquadCombatProfileRow.tsx
src/features/squad/v2/types.ts              ← types, non listé comme fichier orphelin
                                               mais utilisé seulement par les fichiers ci-dessus
```

Même situation : le refactor v2 existe mais n'est pas activé en routing.
`src/features/squad/WinRateVsHistoryChart.tsx` est aussi orphelin (v1 ou v2 ?).

---

## 🟠 P2 — Composants isolés (orphelins hors features complètes)

Fichiers unitaires sans importeur. La plupart sont probablement des composants retirés
d'une page sans avoir été supprimés.

| Fichier | Contexte probable |
|---|---|
| `src/App.tsx` | Reste d'une architecture pre-router — à vérifier, `main.tsx` monte directement le routeur TanStack |
| `src/components/shell/AppShellHeader.tsx` | Shell alternatif remplacé |
| `src/components/shell/KPIBar.tsx` | Barre KPI retirée du layout |
| `src/components/shell/PlayerScopeNav.tsx` | Navigation retirée |
| `src/components/ui/star-button.tsx` | Composant UI jamais branché |
| `src/features/ascension/StreakBadge.tsx` | Badge Ascension non utilisé |
| `src/features/career/CareerCitationsTab.tsx` | Onglet Citations retiré de Career |
| `src/features/career/CareerLusrCards.tsx` | Cards LUSR retirées de Career |
| `src/features/home/HomeOutcomeBar.tsx` | Barre Outcomes retirée de Home |
| `src/features/prestige/components/ChallengesCarousel.tsx` | Carousel Prestige retiré |
| `src/features/synthesis/SynthesisCombatProfileSection.tsx` | Section retirée de Synthesis |
| `src/features/timeseries/TimeseriesCorrelationScatter.tsx` | Chart retiré |
| `src/features/timeseries/TimeseriesOutcomesOverTime.tsx` | Chart retiré |
| `src/lib/query/prefetch.ts` | Utilitaire prefetch non branché |
| `src/stores/careerPageStore.ts` | Store Zustand orphelin |
| `src/lib/feature-flags.ts` | Système de feature flags non consommé |

### Fichiers i18n générés sans consommateur

```
src/lib/i18n/generated/coaching_tips.ts
src/lib/i18n/generated/engagement.ts
```

Ces fichiers semblent générés (même pattern que les autres `generated/`) mais aucun
composant n'importe leurs clés. Soit la feature est retirée, soit les manifests
sont générés mais pas encore branchés.

---

## 🟡 P3 — Exports non consommés (90)

### `src/lib/api/types.ts` — ~75 types inutilisés

La majorité des 90 exports non consommés vient de ce fichier. Il contient des centaines
d'interfaces générées depuis le schéma OpenAPI, dont une grande partie ne sont pas encore
utilisées côté React.

Exemples représentatifs :
```
FeatureFlags, SettingsExcerpt, PlayersListResponse,
SessionContextRequest/Response, HealthResponse,
MediaResetRequest, PaginationRequest/Meta/Response,
MatchHistoryQuerySummary, ExportHint, FileTokenResponse,
ExplorerEncounterRow, HomeHeroCard, RecentMatchMedal,
ChallengesResponse, RadarAxes, SquadMatchSeriesPoint,
SynthesisKPIs, CombatStyle*, SessionCompare*, ...
```

**Recommandation :** ne pas traiter individuellement. Ajouter ce fichier en
`ignoreExportsUsedInFile` dans `knip.config.ts` ou marquer comme `ignore` dans
un second temps. Les types OpenAPI sont par nature partiellement consommés.

À ajouter dans `knip.config.ts` :
```ts
ignoreExportsUsedInFile: true,  // ou :
ignore: ['src/lib/api/types.ts'],
```

### Exports hors `types.ts` — à traiter en priorité

Ces exports existent dans des fichiers applicatifs actifs mais ne sont importés nulle part :

**`src/components/shell/FilterOmnibar.tsx`** — 7 exports orphelins
```
SessionPill, CheckboxGroup, PERIOD_PRESETS, detectActivePreset,
isoDate, presetPeriod, useDismissable, PresetId
```
→ exports rendus publics lors d'un refactor mais jamais consommés depuis l'extérieur

**`src/features/prestige/hooks.ts`** — fichier legacy avec 10 re-exports
```
challengeKeys, useUpdateChallenge, arcKeys, useCreateArc,
prestigeKeys, useJoinSquadChallenge, profileKeys,
useActiveCampaign, useCampaignMutations, usePlayerProfile
```
→ Ce fichier semble être un barrel legacy qui a été remplacé par les fichiers
`src/features/prestige/hooks/useChallenges.ts`, `useArcs.ts`, `usePrestige.ts`
(eux-mêmes aussi partiellement orphelins). Double refactor à finir.

**`src/lib/accessibility/index.ts`** — 7 re-exports non consommés depuis l'index
```
ALL_TOKENS, useColor, useScaleColor, useColorPaletteVersion,
relLuminance, contrastRatio, wcagGrade, WcagGrade
```
→ Les consommateurs importent directement depuis les sous-modules
(`useColor.ts`, `useColorPaletteVersion.ts`) plutôt que depuis l'index

**`src/features/match-view/MatchStatCards.tsx`** — 5 exports non utilisés
```
StatExpectedCard, ExpectedCardsSection, MatchRankBadge, KdIndicatorCard, MatchVsStatCard
```
→ composants exportés mais apparemment non montés dans la vue match actuelle

**`src/lib/i18n/fieldMappings.ts`** — 6 exports orphelins
```
fetchFieldMappings, useFieldMapping, useAssetMapping, useOutcomeMapping,
AssetMappingDTO, OutcomeMappingDTO
```
→ API i18n préparée mais pas encore consommée depuis les features

**Autres exports isolés (1 par fichier) :**
```
src/features/career/lusrSeries.ts          isCanonicalGroup, lusrGroupLabel, LusrSeriesMeta
src/features/match-view/MatchHeader.tsx    DominanceBadgeInline, MatchNavigation
src/features/match-view/MatchScoreboard.logic.ts  formatRank, formatScore
src/features/settings/WatcherCard.tsx      WatcherTokenStatus, WatcherSectionBody
src/lib/formatters/index.ts               formatPercentValue, Locale
src/test/handlers.ts                       emptyKPIs
src/features/ascension/queries.ts          useProfile
src/features/auth/queries.ts               useLogout
src/features/setup/queries.ts              useStartSmokeTest, useStartDeltaSync
src/features/home/queries.ts               useBattlePass
src/features/engagement/queries.ts         useEngagementProfile
src/features/compare/queries.ts            useComparePrefetch
src/features/timeseries/queries.ts         useCombatYieldHistory
src/lib/staticAssets.ts                    DEFAULT_TITLE_SLUG
src/lib/perf-color.ts                      getPerfColorLevel, PerfColorLevel
src/lib/outcome-color.ts                   OUTCOME_COLORS
src/features/coach/i18n.ts                 coachStringsFR, coachStringsEN
src/features/friends/AddFriendFlow.tsx     useAddFriend
src/features/media/queries.ts              useInvalidateOnFeedVersion
src/features/notifications/queries.ts     maxIdOf
src/features/match-view/queries.ts         useMatchNeighbors
src/features/media/i18n.ts                 normalizeMediaLocale, MediaLocale
src/features/home/highlights.i18n.ts       normalizeHighlightLocale, HighlightLocale
src/features/home/kpi.i18n.ts              normalizeKPILocale, KPILocale
src/features/home/spartanIdentity.i18n.ts  normalizeSpartanIdentityLocale, SpartanIdentityLocale
src/components/ui/card.tsx                 CardFooter
src/components/feedback/BadgeIcon.tsx      BADGE_SVG
src/features/asset-drawer/index.ts         useAssetDrawerStore
src/features/feedback-drawer/index.ts      useFeedbackDrawerStore
src/features/settings/watcher-queries.ts   watcherKeys
src/features/ascension/profile/queries.ts  profileKeys
src/lib/query/prefetch.ts                  (fichier entier orphelin)
src/features/lab/_labShared.tsx            getStatusVariant, formatDecimal
src/features/match-view/MatchHeader.card.tsx  RankedIcon, DominanceBadgeInline
src/features/timeseries/matchLabels.ts     truncateMap
src/stores/createFilterStore.ts            FilterStoreState
src/components/shell/_filter_pills/SessionPill.tsx  SessionPill, SessionPillProps
src/lib/accessibility/scales/index.ts     makeOrdinalScale, makeDivergentScale, makeCategoricalScale
src/lib/accessibility/useColor.ts          useColor, useScaleColor
```

---

## 🟡 P4 — Doublons de symboles

### `src/features/match-view/MatchHeader.tsx`
```
MatchNavigationBar (ligne 105)  ─┐ même responsabilité,
MatchNavigation    (ligne 215)  ─┘ deux implémentations
```
→ Probablement un refactor à moitié fait. `MatchNavigation` semble être le nouveau nom.

### `src/lib/formatters/number.ts`
```
formatRatio (ligne 68)  ─┐ logique similaire,
formatKDA   (ligne 77)  ─┘ à vérifier si vraiment dupliquée
```

---

## 🟡 P5 — Dépendances npm

### À retirer (vraiment inutilisées)

| Package | Type | Raison |
|---|---|---|
| `@tailwindcss/typography` | dep | Aucun usage trouvé dans `src/`. Vérifier si utilisé en CSS pur (`@import`) — si oui, c'est un faux positif knip |
| `@tanstack/react-query-devtools` | dep | Probablement import conditionnel `if (import.meta.env.DEV)`. Vérifier — si c'est le cas, ajouter en `ignoreDependencies` |

### Faux positifs confirmés (ne pas retirer)

| Package | Raison |
|---|---|
| `tailwindcss` (devDep) | Utilisé via le plugin Vite `@tailwindcss/vite`, pas importé en JS |
| `@iarna/toml` (devDep) | Utilisé dans `tools/lint-no-hardcoded-fields.mjs` (racine repo) — pas dans `apps/web/src/`. Normal. |

---

## Plan d'action priorisé

### Immédiat (< 30 min)

- [ ] `npm install zod` dans `apps/web/` — dépendance fantôme P0
- [ ] Ajouter `src/lib/api/types.ts` en `ignore` dans `knip.config.ts` pour réduire le bruit des 75 types OpenAPI

### Court terme (1-2h, une PR)

- [ ] Décider du sort de `session-compare/` : câbler la route ou `git rm -rf`
- [ ] Décider du sort de `squad/v2/` : activer ou `git rm -rf`
- [ ] Supprimer `src/App.tsx` si confirmé obsolète (main.tsx monte directement le routeur)
- [ ] Supprimer les 14 composants isolés de P2 (ou les documenter comme WIP)
- [ ] Finir le refactor `prestige/hooks.ts` → supprimer le barrel legacy

### Moyen terme (itératif)

- [ ] Traiter les exports orphelins fichier par fichier (lancer `npm run knip -- --include exports`)
- [ ] Clarifier `@tailwindcss/typography` et `@tanstack/react-query-devtools`
- [ ] Résoudre les doublons `MatchHeader.tsx` et `formatters/number.ts`
- [ ] Activer knip en mode bloquant dans le hook `pre-push` (retirer `--no-exit-code`) une fois la baseline nettoyée

---

## Configuration knip recommandée (mise à jour)

Après la première passe de nettoyage, mettre à jour `apps/web/knip.config.ts` :

```ts
import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  entry: [
    'src/main.tsx',
    'vite.config.ts',
    'playwright.config.ts',
    'eslint.config.js',
  ],
  project: ['src/**/*.{ts,tsx}'],
  ignore: [
    'src/lib/api/generated.ts',   // généré par openapi-typescript
    'src/lib/api/types.ts',        // types OpenAPI partiellement consommés — bruit acceptable
    'src/routeTree.gen.ts',        // généré par TanStack Router
  ],
  ignoreDependencies: [
    'jsdom',                        // vitest env config
    'globals',                      // eslint config
    'tailwindcss',                  // via plugin Vite, pas d'import JS direct
    '@iarna/toml',                  // tools/ racine repo, pas src/
  ],
}

export default config
```
