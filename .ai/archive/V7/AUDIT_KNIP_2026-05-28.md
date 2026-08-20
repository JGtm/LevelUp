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

---

## ✅ RÉSOLUTION — 2026-05-29 (branche `chore/knip-cleanup-p0-p3`)

> Vérification systématique du routing/nav **avant** toute suppression (consigne : vérifier les pages, ne pas nettoyer aveuglément). `tsc -b` ✅ + `vite build` ✅ après cleanup.

### Fait

- **P0** ✅ `zod` ajouté (`^4.4.3`) — résout les 5 imports *unlisted*. Version 4.x alignée sur le runtime déjà en place.
- **P3** ✅ `knip.config.ts` : `src/lib/api/types.ts` ignoré (types OpenAPI partiellement consommés) + faux positifs deps neutralisés : `tailwindcss` (plugin Vite), `@tailwindcss/typography` (via `@plugin` dans `globals.css`), `@iarna/toml` (tools/ racine).
- **P1/P2** ✅ 5 fichiers **confirmés morts** supprimés — remplaçant actif identifié pour chacun :

| Supprimé | Remplaçant actif (vérifié) |
|---|---|
| `features/career/CareerCitationsTab.tsx` | `features/citations/CitationsPage` (route `/citations` ; `CareerHubPage` documente « les citations ont leur propre page dédiée ») |
| `components/shell/AppShellHeader.tsx` | `NavL1` (AppShell n'importe que NavL1) |
| `components/shell/KPIBar.tsx` | `NavL2` (layout `$playerSlug.tsx`) |
| `components/shell/PlayerScopeNav.tsx` | sélecteur joueur intégré à NavL1 |
| `stores/careerPageStore.ts` | store orphelin, zéro référence |

### Gardé délibérément (WIP / intention explicite — NE PAS supprimer sans accord)

| Élément | Raison |
|---|---|
| `features/session-compare/*` (17 fichiers) | **Travail en cours utilisateur** (confirmé 2026-05-29, source d'inspiration). Fonctionnellement superseded par `session-detail/SessionDetailPage` (route active `stats/sessions`) mais **conservé sur demande**. |
| `features/squad/v2/{SquadV2Page,SquadV2RouteHost,queries.ts,components/SquadCombatProfileRow}` | v2 non câblée (routes actives = v1 `SquadSynergiesPage`/`SquadContributionsPage`). ⚠️ `squad/v2/types.ts` + `SquadEngagementView` RESTENT vivants (importés par `SessionBriefing` + `engagement`) → ne jamais `rm -rf` le dossier entier. |
| `features/synthesis/SynthesisCombatProfileSection.tsx` | WIP `PLAN_COMBAT_PROFILE_WIRING` Phase 1 |
| `features/prestige/components/ChallengesCarousel.tsx` | feature prestige référencée par commentaire dans `HomePrestigeSection` |
| `lib/query/prefetch.ts` (`useNavPrefetch`) | feature prefetch-au-survol construite mais jamais câblée dans NavL1 |
| `lib/feature-flags.ts` (`REJEU_2D_ENABLED`) | flag pour projet replay 2D externe planifié (stub `replay.tsx`) |
| `src/App.tsx` | annotation explicite « conserve pour reference » (boilerplate Vite) |
| `lib/i18n/generated/{coaching_tips,engagement}.ts` | artefacts générés depuis TOML → traiter via le générateur, pas à la main |
| Charts isolés : `StreakBadge`, `CareerLusrCards`, `HomeOutcomeBar`, `WinRateVsHistoryChart`, `TimeseriesCorrelationScatter`, `TimeseriesOutcomesOverTime` | zéro réf mais **pas de remplaçant identifié** → possibles leftovers de redesign OU composants à recâbler. À confirmer au cas par cas. |

### Reste (hors scope P0-P3 → backlog)

- **90 unused exports** (hooks queries, consts i18n, re-exports de barrels) dans des fichiers vivants → nettoyage itératif fichier-par-fichier.
- **`@tanstack/react-query-devtools`** : vraie dépendance inutilisée (zéro import) → suppression = **P5 (dépendances)**, hors scope.
- **Doublons P4** (`MatchHeader.tsx`, `formatters/number.ts`) → non traités.

### Nouveau baseline knip

| Catégorie | Avant | Après |
|---|---|---|
| Unlisted deps (zod) | 5 | **0** |
| Unused devDeps | 2 | **0** |
| Unused deps | 2 | **1** (react-query-devtools, P5) |
| Unused files | 40 | **35** (= set « gardé » ci-dessus) |

---

## ✅ RÉSOLUTION phase 2 — 2026-05-29 (charts isolés + quick wins + dette lint)

> Traitement des items « reste hors scope + charts isolés ». Même méthode : vérif page parente avant suppression. `tsc -b` ✓ + `vite build` ✓ + 3 linters custom ✓.

### Charts isolés — 6 supprimés (leftovers confirmés, remplaçant actif vérifié)

| Chart supprimé | Remplaçant actif |
|---|---|
| `ascension/StreakBadge` | `StreakDashboard` (AscensionRealisationsTab) |
| `career/CareerLusrCards` | `CareerRankingBlock` + `CareerChartsSection` |
| `home/HomeOutcomeBar` | `OutcomeSequenceTape` (HomePage) |
| `squad/WinRateVsHistoryChart` | `WinRateVsHistoryBulletChart` (SquadSynergiesPage) |
| `timeseries/TimeseriesCorrelationScatter` | retiré au rework timeseries (zéro import) |
| `timeseries/TimeseriesOutcomesOverTime` | idem |

`git log` confirme : tous issus d'anciens commits de feature, orphelinisés par refontes — aucun n'est du WIP récent.

### Quick wins

- **P4 doublons** : `MatchNavigation` (alias rétrocompat, migration vers `MatchNavigationBar` terminée) supprimé. `formatKDA` **gardé** — alias domaine de `formatRatio` réellement utilisé (`PalmaresRelationsPage` + tests) ; le flag knip « duplicate » est informatif.
- **react-query-devtools** : **câblé** en dev-only dans `app/providers` plutôt que supprimé (dep « inutilisée » → outil dev utile, tree-shaké en prod). Résout le dernier *unused dependency*.

### Dette lint pré-existante — résolue

- **cross-feature 11 → 0** : 8 paires durables (pattern agrégateur, cohérent avec le précédent de l'allowlist) ajoutées à `ALLOWED_CROSS_IMPORTS` (`auth/onboarding=>setup`, `home=>ascension`, `explorer=>home`, `squad=>timeseries`, `synthesis=>filters/explorer`, `timeseries=>career`).
- **faux positif CSR/LUSR** : `lint-no-hardcoded-fields.mjs` étendu pour skip les opérations de collection (`.has`/`.includes`/`.get('X')`), comme il skippait déjà `=== 'X'`. Fix général.

### knip config

Minimisée : entries auto-détectées par les plugins knip (Vite/Playwright/ESLint), ignores redondants (`routeTree.gen.ts`, `jsdom`, `globals`) retirés. Reste : `ignore` [generated.ts, types.ts] + `ignoreDependencies` [tailwindcss, @tailwindcss/typography, @iarna/toml].

### Délibérément NON traité (backlog itératif)

- **89 unused exports + 83 unused exported types** : dispersés dans des fichiers **vivants** (hooks `queries.ts`, normalizers i18n, sous-composants `MatchStatCards`…). Suppression non automatisable sans risque : (1) édite des fichiers vivants, (2) beaucoup peuvent être du WIP construit en avance (hooks queries). À traiter par **lots vérifiés** (ex. « hooks queries orphelins » en un lot, en confirmant chacun). knip reste advisory (hors gate pre-push).
- Orphelins gardés (cf. phase 1 + WIP utilisateur) : squad/v2 pages, SynthesisCombatProfileSection, ChallengesCarousel, prefetch, feature-flags, App.tsx, i18n générés, session-compare.

### Baseline knip final

| Catégorie | Après phase 2 |
|---|---|
| Unlisted deps | **0** |
| Unused deps | **0** (react-query-devtools câblé) |
| Unused files | **29** (orphelins gardés volontairement) |
| Unused exports / types | 89 / 83 (backlog itératif) |

---

## ✅ RÉSOLUTION phase 2bis — 2026-05-29 (ratchet + investigation hooks)

### Ratchet anti-régression

`tools/knip-ratchet.mjs` (lancé en pre-push via lefthook) fige le code mort à un plafond et échoue si dépassé — même mécanisme que `lint-cross-feature-imports` (plafond 10) et `lint-no-hardcoded-colors`. Plafonds figés : **files=29, exports=87, types=83**. Vérifié : passe à l'état courant, bloque une régression simulée (exit 1). Pour abaisser au fil du nettoyage : éditer `THRESHOLDS`.

### Investigation 1-par-1 des 14 hooks `queries.ts` orphelins

Objectif : distinguer le vrai code mort du WIP. Verdict :

| Hook | Verdict | Raison |
|---|---|---|
| `useBattlePass` (home) | 🗑️ **SUPPRIMÉ** | superseded par `useSeasonPassPreview` (le panel home reçoit déjà la data season-pass) |
| `useMatchNeighbors` (match-view) | 🗑️ **SUPPRIMÉ** | superseded par `useMatchNeighborsResolved` (+ mock obsolète nettoyé) |
| `useProfile` (ascension) | ✅ gardé | PlayerProfile partiel — ADR 0015 (7 commits reportés) |
| `useCreateArc` / `useUpdateChallenge` / `useJoinSquadChallenge` (prestige) | ✅ gardé | Prestige à activation phasée — ADR 0005 |
| `useStartSmokeTest` / `useStartDeltaSync` (setup) | ✅ gardé | feature setup/onboarding **en cours de réécriture** |
| `useLogout` (auth) | ✅ gardé | endpoint `/auth/logout` réel, feature probablement non câblée (pas mort) |
| `useComparePrefetch` (compare) | ✅ gardé | prefetch au survol non câblé (cohérent avec `lib/query/prefetch.ts`) |
| `useInvalidateOnFeedVersion` (media) | ✅ gardé | helper d'invalidation cache « à utiliser avec useEffect » — non câblé |
| `useEngagementProfile` (engagement) | ✅ gardé | feature engagement active, métrique probablement à afficher |
| `useCombatYieldHistory` (timeseries) | ✅ gardé | alimente `CombatYieldTimeseries` (S56), consommateur nommé |
| `maxIdOf` (notifications) | ✅ gardé | petit helper pur, commenté « exposé pour le toastBridge » |

**Conclusion rassurante** : sur 14, **0 random dead code** — chacun mappe à une vraie feature/endpoint. 12 sont du WIP/phasé/intentionnel (gardés), 2 étaient de vrais doublons superseded (retirés avec preuve + `tsc`/test/build verts).

### Baseline knip après phase 2bis

| Catégorie | Valeur |
|---|---|
| Unlisted / unused deps | **0 / 0** |
| Unused files | **29** (gardés volontairement) |
| Unused exports / types | **87 / 83** (gelés sous ratchet) |

---

## ✅ RÉSOLUTION phase 2ter — 2026-05-29 (investigation des ~73 fonctions/valeurs non-queries)

Passage au crible 1-par-1 des exports restants (fonctions/valeurs hors hooks `queries.ts` déjà traités). Méthode : occurrence intra-fichier (occ) + commit d'origine + lecture ciblée.

**Verdict : 0 suppression sûre à haute valeur.** Contrairement aux hooks (2 superseded retirés), ce lot est entièrement de la surface intentionnelle ou du WIP — aucun « random dead code ».

| Catégorie | Exemples | Décision |
|---|---|---|
| **Sur-exports** (occ ≥ 2, utilisés dans leur propre fichier) | `MatchVsStatCard` (via `MatchSummaryCardsSection`), `formatRank`, `useAddFriend`, `watcherKeys`, normalizers i18n | garder (retirer `export` = churn zéro-valeur, comme les types) |
| **Barrels vivants** | `prestige/hooks.ts` (Ascension importe `useChallenges`/`useArcs`/`useMyPrestige`), `FilterOmnibar` | garder (supprimer casserait des imports) |
| **API publique / barrels** | `lib/accessibility/index.ts`, `scales/index.ts`, `formatters/index.ts`, drawers | garder |
| **Testé** | `formatPercentValue` (couvert par `percent.test.ts`) | garder |
| **Primitive UI** | `CardFooter` (set Card/Header/Content/Footer) | garder |
| **WIP partiellement câblé** | `MatchStatCards` C3/C4/C5 (C7 `MatchSummaryCardsSection` vivant), `fieldMappings` hooks (multi-titre Phase D), hooks `accessibility` | garder |
| **Compat / helper marginal** | `OUTCOME_COLORS`, `getPerfColorLevel`, `formatScore` | garder (valeur infime, fichiers vivants) |

**Near-miss notable** : `prestige/hooks.ts` ressemblait au « barrel legacy supprimable » — mais `grep` a montré que `AscensionRealisationsTab`/`AscensionProfileTab` en importent `useChallenges`/`useArcs`/`useMyPrestige`. knip ne flaggait que le **sous-ensemble** non importé de ses re-exports. Le supprimer aurait cassé Ascension → illustration concrète de pourquoi `grep` avant suppression est obligatoire.

**Conséquence** : le ratchet (87 / 83) est la bonne réponse — il fige cette surface intentionnelle et bloque toute *nouvelle* régression, sans churn risqué sur du code vivant/WIP.
