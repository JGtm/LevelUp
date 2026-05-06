# Plan — Cohérence light/dark : éliminer les couleurs hardcodées dans `apps/web/`

> **Branche cible** : `fix/theme-consistency-tokens` (à créer **après** finalisation du WIP `fix/synergy-radar-calibration`)
> **Date** : 2026-05-05
> **Statut** : v4 — audit codebase post-v3 : ajout du pivot `components/charts/_utils.ts` (4 constantes utilisées par 11 wrappers ECharts), de 6 chart wrappers partagés, de 2 squad charts manqués + `PlayerChips.tsx`. La Phase 5 change de stratégie : refactor du pivot d'abord, propagation dérivée.

## 1. Contexte

L'utilisateur a constaté qu'en switchant `light ↔ dark`, **certaines zones restent figées en sombre** (notamment Home banner, Battle Pass panel, lightbox récompense, Media viewer/lightbox).

**Cause racine** : des classes Tailwind couleur hardcodées (`bg-slate-950/X`, `text-cyan-100/X`, `text-white/X`, `bg-black/X`, `border-white/X`) ont été apposées avec l'annotation `// color-allow: thématique Spartan UI` — supposant que ces zones devaient rester sombres en permanence (skin Halo). Cette hypothèse est invalide : l'utilisateur préfère que **tout suive le thème**.

**Décision** : Option 1 — passer **toutes** ces zones aux tokens sémantiques shadcn (`bg-card`, `text-foreground`, `bg-muted`, `border-border`, etc.) qui basculent automatiquement via `[data-theme='light' | 'dark']` (cf. [globals.css:7-134](apps/web/src/styles/globals.css#L7-L134)).

**Décisions complémentaires (validées avec l'utilisateur, 2026-05-04)** :
- ✅ **CoverFlow modal** : thématisée comme le reste (pas de mode cinema spécial). Chrome + backdrop suivent le thème.
- ✅ **Banner identity hero** : thématisé. `text-foreground` + on adapte/supprime l'overlay si besoin pour garantir la lisibilité dans les 2 thèmes.
- ✅ **Charts ECharts** : approche pilote — refactorer **1 chart** (radar synergy retenu — visible et instrumenté), valider via un **test E2E Playwright** qui toggle le thème et vérifie la couleur des axes/labels, **puis** propager au pivot `_utils.ts` + 6 wrappers partagés + 8 charts spécifiques + `PlayerChips` (cf. v4 — détail § 2.4–2.5 et Phase 5).

**Hors scope** :
- Couleurs de rareté Halo ([palmares/rarity.ts](apps/web/src/features/palmares/rarity.ts)) — exception § 20 légitime.
- Couleurs `liked / rose` (heart icons) — exception § 20 légitime.
- Couleurs structurelles SVG ([components/ui/rank-progress-gauge.tsx](apps/web/src/components/ui/rank-progress-gauge.tsx)) — exception § 20 légitime.
- Hex `color-allow:` justifiés (heatmap squad intensity, fallback citations API) — exception § 20.

---

## 2. État des lieux — inventaire complet (v4 — audit codebase complet)

> ⚠️ **Découvertes v4 (passe code)** :
> - **`components/charts/_utils.ts`** : 4 hardcodes (`GRID_COLOR`, `TEXT_COLOR`, `ZERO_LINE`, `tooltipBase.textStyle.color`) — utilisés par **11 wrappers** ([BarGroupedChart](apps/web/src/components/charts/BarGroupedChart.tsx), [BarStackedChart](apps/web/src/components/charts/BarStackedChart.tsx), [Heatmap2DChart](apps/web/src/components/charts/Heatmap2DChart.tsx), [EngagementCurve](apps/web/src/components/charts/EngagementCurve.tsx), [DonutChart](apps/web/src/components/charts/DonutChart.tsx), [TimeseriesLineChart](apps/web/src/components/charts/TimeseriesLineChart.tsx), [RadarChart](apps/web/src/components/charts/RadarChart.tsx), [ScatterChart](apps/web/src/components/charts/ScatterChart.tsx), [OutcomeSequenceTape](apps/web/src/components/charts/OutcomeSequenceTape.tsx), [HistogramChart](apps/web/src/components/charts/HistogramChart.tsx), via `axisBase` / `tooltipBase` / `legendBase`). **C'est LE pivot ECharts** — tout fix ici se propage automatiquement.
> - **6 chart wrappers** ont en plus des `rgba(255,255,255,X)` inline pour des éléments spécifiques (radar `axisName`/`splitArea`/`splitLine`/`axisLine`, histogram `nameTextStyle`, scatter `nameTextStyle`, donut label, engagement-curve `nameTextStyle`, heatmap `visualMap.textStyle`).
> - **2 squad charts manqués en v3** : [squadFirstEventsChart.ts](apps/web/src/features/squad/charts/squadFirstEventsChart.ts) (2 occ) et [squadPerformanceLineCharts.ts](apps/web/src/features/squad/charts/squadPerformanceLineCharts.ts) (2 occ).
> - **[components/PlayerChips.tsx](apps/web/src/components/PlayerChips.tsx)** : 3 hardcodes ECharts manqués.
>
> **Découvertes v2 (déjà intégrées)** : `components/ui/match-card.tsx` (17 hardcodes), axes/labels charts ECharts, dispersion `text-white` Home/Palmarès, divider `bg-white/90` HomeChallengesList.

### Mécanisme du switch
Le thème est piloté par `data-theme` sur `<html>` (cf. [theme-provider.tsx:7](apps/web/src/app/providers/theme-provider.tsx#L7)). Les CSS vars définies dans `:root[data-theme='dark']` et `:root[data-theme='light']` ([globals.css:7-134](apps/web/src/styles/globals.css#L7-L134)) basculent automatiquement. Toute couleur **non liée** à ces vars (hex, rgba, classes Tailwind nommées sans token) reste figée.

### 2.1. Tailwind couleurs nommées (`bg-slate-950/X`, `text-cyan-100/X`, etc.)

| Fichier | Occ. | Zone visible |
|---|---|---|
| [features/home/HomeSpartanIdentityBanner.tsx](apps/web/src/features/home/HomeSpartanIdentityBanner.tsx) | 8 | Banner hero identity Home |
| [features/home/HomeSkillPeakCard.tsx](apps/web/src/features/home/HomeSkillPeakCard.tsx) | 6 | Cartes CSR / LUSR Home |
| [features/home/HomeBattlePassPanel.tsx](apps/web/src/features/home/HomeBattlePassPanel.tsx) | 2 | Frame image Battle Pass Home |
| [features/media/MediaViewer.tsx](apps/web/src/features/media/MediaViewer.tsx) | 3 | Like button / likers line |
| [features/media/MediaToolbar.tsx](apps/web/src/features/media/MediaToolbar.tsx) | 1 | Bouton filtre liked-only |
| [features/palmares/BattlePassRewardLightbox.tsx](apps/web/src/features/palmares/BattlePassRewardLightbox.tsx) | 3 | Lightbox récompense Palmarès |

**Total** : 23 occurrences sur 6 fichiers à refactorer (les 15 occurrences de `palmares/rarity.ts` restent — exception légitime).

### 2.2. Tailwind `white/black` avec opacité (`bg-black/X`, `text-white/X`, `border-white/X`)

Ces classes ne déclenchent **pas** le linter custom mais figent l'apparence indépendamment du thème :

| Fichier | Occ. | Zone visible |
|---|---|---|
| [features/media/CoverFlowModal.tsx](apps/web/src/features/media/CoverFlowModal.tsx) | 10 | Modale lightbox plein écran |
| [features/palmares/BattlePassRewardLightbox.tsx](apps/web/src/features/palmares/BattlePassRewardLightbox.tsx) | 9 | Header / boutons lightbox récompense |
| [features/home/HomeSkillPeakCard.tsx](apps/web/src/features/home/HomeSkillPeakCard.tsx) | 4 | Bordures cartes peaks |
| [features/media/MediaViewer.tsx](apps/web/src/features/media/MediaViewer.tsx) | 3 | Like button (non liked state) |
| [features/palmares/SeasonPassPage.tsx](apps/web/src/features/palmares/SeasonPassPage.tsx) | 2 | Stats line Palmarès |
| [features/timeseries/TimeseriesCorrelationScatter.tsx](apps/web/src/features/timeseries/TimeseriesCorrelationScatter.tsx) | 1 | Toggle palette |
| [features/home/HomeChallengesList.tsx](apps/web/src/features/home/HomeChallengesList.tsx) | 1 | Border carte challenges |
| [features/home/HomeBattlePassPanel.tsx](apps/web/src/features/home/HomeBattlePassPanel.tsx) | 1 | Border frame image |
| [features/home/HomeSpartanIdentityBanner.tsx](apps/web/src/features/home/HomeSpartanIdentityBanner.tsx) | 1 | Empty state border |
| [features/compare/CompareDrawer.tsx](apps/web/src/features/compare/CompareDrawer.tsx) | 1 | Backdrop drawer |
| [features/media/MediaViewer.tsx](apps/web/src/features/media/MediaViewer.tsx) (player_chips) | 1 | Border preview joueur |
| [features/media/MediaMatchPicker.tsx](apps/web/src/features/media/MediaMatchPicker.tsx) | 1 | Backdrop modal |
| [features/palmares/BattlePassRewardCarousel.tsx](apps/web/src/features/palmares/BattlePassRewardCarousel.tsx) | ~5 | Badges + glow + gradient |
| [features/settings/AccessibilityTab.tsx](apps/web/src/features/settings/AccessibilityTab.tsx) | 1 | Preview palette swatch |

**Total estimé** : ~36 occurrences réparties sur ~13 fichiers.

### 2.3. Composants partagés (`components/ui/`) — manqués en v1

| Fichier | Occ. | Problème |
|---|---|---|
| [components/ui/match-card.tsx](apps/web/src/components/ui/match-card.tsx) | 17 | **Critique** — `bg-white/5` panneau stats invisible en light, `text-white` sur 13 valeurs (kills/assists/MMR) illisible en light, `border-white/[0.06]`, `border-white/10`, `bg-white/[0.02]`, `bg-white/20` séparateur |
| [components/ui/citation-progress-ring.tsx](apps/web/src/components/ui/citation-progress-ring.tsx) | 1 + 1 rgba | `rgba(255,255,255,0.12)` track + `bg-white/10` placeholder |
| [components/ui/star-button.tsx](apps/web/src/components/ui/star-button.tsx) | 1 | Variante compact : `bg-black/45 text-white/60` (overlay sur image map = OK, à valider) |

**Impact** : `MatchCard` est utilisé dans [HomeRecentMatches](apps/web/src/features/home/HomeRecentMatches.tsx), [match-history/MatchHistoryTable](apps/web/src/features/match-history/MatchHistoryTable.tsx), partout où des matchs s'affichent. **C'est probablement la zone n°1 que l'utilisateur a vu cassée.**

### 2.4. Pivot partagé `components/charts/_utils.ts` — point d'effet de levier (NOUVEAU v4)

[components/charts/_utils.ts](apps/web/src/components/charts/_utils.ts#L17-L44) exporte 4 hardcodes consommés par **11 wrappers** :

| Constante / objet | Ligne | Valeur figée | Consommateurs |
|---|---|---|---|
| `GRID_COLOR` | L18 | `rgba(255,255,255,0.06)` | `axisBase` (axisLine, splitLine), `tooltipBase` (borderColor) |
| `TEXT_COLOR` | L19 | `rgba(255,255,255,0.45)` | `axisBase.axisLabel.color`, `legendBase.textStyle.color` |
| `ZERO_LINE` | L20 | `rgba(255,255,255,0.15)` | (export, peu utilisé directement) |
| `tooltipBase.textStyle.color` | L34 | `rgba(255,255,255,0.85)` | `tooltipBase` |
| `tooltipBase.backgroundColor` | L32 | `rgba(20,24,30,0.92)` | `tooltipBase` (sombre figé) |

**Stratégie** : transformer ces constantes en **getters runtime** (`getGridColor()`, `getTextColor()`...) ou en factories de bases (`getAxisBase()`, `getTooltipBase()`, `getLegendBase()`). Tous les wrappers qui spread `...axisBase` ou `...tooltipBase` bénéficient automatiquement.

**Conséquence** : ces 11 wrappers (BarGrouped, BarStacked, Heatmap2D, EngagementCurve, Donut, TimeseriesLine, Radar, Scatter, OutcomeSequenceTape, Histogram) **n'ont qu'à recevoir le themeVersion en dépendance de leur `useMemo`** et se rendent automatiquement à la couleur du thème. Pas de patch ligne par ligne sur chacun.

### 2.5. Charts ECharts — `rgba(255,255,255,X)` figés (inline, après pivot _utils)

Au-delà du pivot, certains wrappers et charts spécifiques ont des hardcodes inline qui doivent être migrés explicitement (sites où les valeurs sont écrites directement, pas via les bases du pivot) :

| Fichier | Occ. | Élément |
|---|---|---|
| [components/charts/RadarChart.tsx](apps/web/src/components/charts/RadarChart.tsx) | 4 | `axisName`, `splitArea` (2 valeurs), `splitLine`, `axisLine` (radar-spécifique, hors `axisBase`) |
| [components/charts/HistogramChart.tsx](apps/web/src/components/charts/HistogramChart.tsx) | 2 | `nameTextStyle` (axes X et Y) |
| [components/charts/ScatterChart.tsx](apps/web/src/components/charts/ScatterChart.tsx) | 2 | `nameTextStyle` (axes X et Y) |
| [components/charts/Heatmap2DChart.tsx](apps/web/src/components/charts/Heatmap2DChart.tsx) | 1 | `visualMap.textStyle` |
| [components/charts/DonutChart.tsx](apps/web/src/components/charts/DonutChart.tsx) | 1 | `label.color` |
| [components/charts/EngagementCurve.tsx](apps/web/src/components/charts/EngagementCurve.tsx) | 1 | `nameTextStyle` |
| [components/PlayerChips.tsx](apps/web/src/components/PlayerChips.tsx) | 3 | (à inventorier — chip color refs) |
| [features/squad/charts/squadSynergyRadarChart.ts](apps/web/src/features/squad/charts/squadSynergyRadarChart.ts) | 4 | axisName / splitLine / axisLine / splitArea — **chart pilote** |
| [features/squad/charts/squadSessionTimelineChart.ts](apps/web/src/features/squad/charts/squadSessionTimelineChart.ts) | 2 | nameTextStyle |
| [features/squad/charts/squadIntensityHeatmapChart.ts](apps/web/src/features/squad/charts/squadIntensityHeatmapChart.ts) | 2 | visualMap.textStyle |
| [features/squad/charts/squadMapHeatmapChart.ts](apps/web/src/features/squad/charts/squadMapHeatmapChart.ts) | 2 | visualMap.textStyle |
| [features/squad/charts/squadPerMinuteChart.ts](apps/web/src/features/squad/charts/squadPerMinuteChart.ts) | 2 | label sur barres |
| [features/squad/charts/squadWeaponKillsChart.ts](apps/web/src/features/squad/charts/squadWeaponKillsChart.ts) | 2 | label |
| [features/squad/charts/squadFirstEventsChart.ts](apps/web/src/features/squad/charts/squadFirstEventsChart.ts) | 2 | (à inventorier — labels/axes) **NOUVEAU v4** |
| [features/squad/charts/squadPerformanceLineCharts.ts](apps/web/src/features/squad/charts/squadPerformanceLineCharts.ts) | 2 | (à inventorier — labels/axes) **NOUVEAU v4**, hors `'#aaa'` self-ghost |
| [features/timeseries/TimeseriesKdaBars.tsx](apps/web/src/features/timeseries/TimeseriesKdaBars.tsx) | 1 | nameTextStyle |
| [features/squad/v2/SquadEngagementView.tsx](apps/web/src/features/squad/v2/SquadEngagementView.tsx) | 1 | nameTextStyle |
| [components/ui/citation-progress-ring.tsx](apps/web/src/components/ui/citation-progress-ring.tsx) | 1 | `trackColor` SVG (cf §2.3) |

**Total Phase 5 (post-v4)** : 1 pivot + 6 wrappers partagés + 9 charts spécifiques + PlayerChips = **17 fichiers** (vs 8 en v3).

**Solution adoptée** : helper `getEChartsThemeColors()` qui résout les CSS vars au runtime + hook `useThemeVersion()` qui force le re-mémoization quand `data-theme` change. Voir Phase 5 §5.2.

### 2.6. Hardcodes restants à examiner

- [features/home/HomeSessionCarousel.tsx:244,248](apps/web/src/features/home/HomeSessionCarousel.tsx#L244) — `text-white` dans un `<p className="text-muted-foreground">` (playlist + mode dominants) → **invisible en light**.
- [features/home/HomeChallengesList.tsx:100](apps/web/src/features/home/HomeChallengesList.tsx#L100) — `'bg-white/90'` divider de section.
- [features/palmares/SeasonPassPage.tsx:80,82,97,99](apps/web/src/features/palmares/SeasonPassPage.tsx#L80) — chips `text-white` / `text-white/60` dans `PassContentSummary` (rendu en overlay sur image bg avec gradient sombre — possiblement OK, à valider visuellement).
- [features/palmares/SeasonPassPage.tsx:285](apps/web/src/features/palmares/SeasonPassPage.tsx#L285) — `text-white` titre overlay sur hero (OK, gradient `from-black/90` couvre).
- [features/palmares/BattlePassRewardCarousel.tsx:138,143](apps/web/src/features/palmares/BattlePassRewardCarousel.tsx#L138) — `text-white` sur `bg-success` / `bg-warning/90` → remplacer par `text-success-foreground` / `text-warning-foreground`.
- [features/palmares/BattlePassRewardCarousel.tsx:150](apps/web/src/features/palmares/BattlePassRewardCarousel.tsx#L150) — gradient fallback `rgba(15,23,42,0.92)` figé en sombre.
- [features/palmares/BattlePassRewardCarousel.tsx:173](apps/web/src/features/palmares/BattlePassRewardCarousel.tsx#L173) — `border-white/15` quand non-current.
- [features/match-view/PlayerDetailPanel.tsx:103](apps/web/src/features/match-view/PlayerDetailPanel.tsx#L103) — `text-white` sur badge coloré (probablement OK si fond coloré, à valider).

### 2.7. Cas particuliers (à arbitrer, pas refactor mécanique)

- **Backdrops modaux** (`bg-black/40`, `bg-black/70`, `bg-black/90`) sur [CompareDrawer](apps/web/src/features/compare/CompareDrawer.tsx), [MediaMatchPicker](apps/web/src/features/media/MediaMatchPicker.tsx), [CoverFlowModal](apps/web/src/features/media/CoverFlowModal.tsx), [BattlePassRewardLightbox](apps/web/src/features/palmares/BattlePassRewardLightbox.tsx) → **conserver `bg-black/X`** est OK : overlay neutre des deux côtés du thème (un dim noir au-dessus du contenu light reste lisible).
- **Text shadow** (`textShadow: '0 1px 6px rgba(0,0,0,0.85)'` sur banner identity) → reste valide tant que l'image de fond reste sombre (banner Halo officiel).
- **Image overlay gradients** (`bg-[radial-gradient(...rgba(15,23,42,1)...)]`) → fallback quand l'image BP n'est pas dispo. Doit suivre le thème → utiliser tokens via CSS vars `var(--background)` / `var(--card)`.
- **Spartan banner image** : la `<img>` de fond reste, mais le `bg-slate-950` de fallback (quand pas d'image) doit suivre le thème.
- **`text-white` text-on-colored-bg** : quand le texte est sur un `bg-success`, `bg-warning`, ou un `style={{ background: tokenCssVar('compare-a') }}`, `text-white` est habituellement remplaçable par le foreground correspondant (`text-success-foreground`, `text-warning-foreground`, `text-primary-foreground`). [SquadLayout.tsx:327](apps/web/src/features/squad/SquadLayout.tsx#L327) et [MedalDigest.tsx:58](apps/web/src/features/squad/MedalDigest.tsx#L58) ont déjà `// color-allow: blanc structurel sur fond joueur` — ces cas peuvent rester.

---

## 3. Mapping de substitution

| Hardcodé | Token cible | Justification |
|---|---|---|
| `bg-slate-950` | `bg-card` ou `bg-background` | Surface primaire selon contexte |
| `bg-slate-950/80` | `bg-card/80` | Surface translucide |
| `bg-slate-950/60` | `bg-muted/60` | Conteneur secondaire (badge holder) |
| `bg-slate-950/35` ou `/22` ou `/15` | `bg-muted/40` ou `bg-muted/20` | Fond translucide (carte peak) |
| `text-cyan-100` | `text-primary-foreground` ou `text-foreground` | Texte sur fond accentué |
| `text-cyan-100/70` à `/78` | `text-muted-foreground` | Texte secondaire |
| `text-cyan-50` | `text-foreground` | Texte sur surface card |
| `text-slate-300` | `text-muted-foreground` | Texte secondaire |
| `text-slate-200` | `text-foreground` | Texte description |
| `text-slate-400` | `text-muted-foreground` | Métadonnées |
| `border-cyan-300/60` | `border-primary/60` | Accent border (emblem ring) |
| `border-cyan-100/12` | `border-border` | Border subtle |
| `border-white/X` | `border-border` ou `border-border/X` | Border neutre |
| `text-white` (banner gamertag) | `text-foreground` (sur card) ou conserver si overlay sur image | Selon contexte |
| `text-white/85`, `/70`, `/60`, `/50`, `/30` | `text-foreground/X` ou `text-muted-foreground` | Texte translucide |
| `bg-white/5`, `/10`, `/90` | `bg-muted/X` ou `bg-background/X` | Fond translucide |
| `bg-black/40`, `/55`, `/60`, `/70`, `/85`, `/90` (backdrops modaux) | **Garder** (overlay neutre) | Convient en light/dark |
| `bg-black` (poster vidéo) | **Garder** (vidéo sur fond noir = standard) | Comportement attendu |
| `shadow-[0_X_X_rgba(8,15,28,0.X)]` | Conserver ou simplifier vers `shadow-md`/`shadow-lg` | Ombres dark-only acceptables |
| `bg-[radial-gradient(...rgba(15,23,42)...)]` | Tokens via CSS vars : `bg-[radial-gradient(circle_at_top,var(--primary)/.18,transparent_45%)]` | Gradient adaptatif |

---

## 4. Plan d'exécution — par phases (v2)

### Phase 0 — Composant partagé `MatchCard` (priorité haute)

**Fichier** : [components/ui/match-card.tsx](apps/web/src/components/ui/match-card.tsx)

C'est probablement la cause n°1 visible par l'utilisateur car la `MatchCard` est utilisée partout (Home, Match History, Squad…).

**Changements** :
1. L158 `bg-black/40 hover:bg-black/60` (favorite overlay sur image map) → **conserver** (overlay sur image, OK).
2. L264 `bg-white/5` panneau stats → `bg-muted/40`.
3. L282 `bg-white/20` séparateur vertical → `bg-border`.
4. L300, 329, 333, 337, 353, 386, 390, 402, 408, 420, 427 `text-white` (valeurs stats) → `text-foreground`.
5. L453, 484 `border-white/[0.06]` (séparateurs footer) → `border-border/40`.
6. L522 `border-white/10 bg-white/[0.02]` (footer) → `border-border bg-muted/30`.

**Tests** : aucun test ne semble cibler ces classes spécifiquement, mais re-runner [match-card.test.tsx](apps/web/src/components/ui/match-card.test.tsx) si présent.

### Phase 1 — Home (banner + cartes peak + battle pass + sessions)

**Fichiers** : [HomeSpartanIdentityBanner.tsx](apps/web/src/features/home/HomeSpartanIdentityBanner.tsx), [HomeSkillPeakCard.tsx](apps/web/src/features/home/HomeSkillPeakCard.tsx), [HomeBattlePassPanel.tsx](apps/web/src/features/home/HomeBattlePassPanel.tsx), [HomeSessionCarousel.tsx](apps/web/src/features/home/HomeSessionCarousel.tsx), [HomeChallengesList.tsx](apps/web/src/features/home/HomeChallengesList.tsx)

**Changements** :
1. **Banner identity (thématisé complet)** :
   - `bg-slate-950` (fallback sans image) → `bg-card`. L'image `<img>` de fond reste inchangée.
   - **Gestion lisibilité texte sur image** : ajouter un overlay gradient **adaptatif** au-dessus de l'image qui utilise les CSS vars du thème :
     ```tsx
     <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-background/30 via-background/10 to-background/40" aria-hidden="true" />
     ```
     En dark theme, l'overlay est sombre (image visible avec halo sombre). En light, l'overlay est clair (image atténuée vers le clair). Combiné avec `text-foreground`, le texte reste lisible dans les 2 thèmes même si l'image est saturée.
   - `text-white` gamertag → `text-foreground`. Le `text-shadow: '0 1px 6px rgba(0,0,0,0.85)'` doit être adapté : utiliser un `text-shadow` qui s'adapte (ou utiliser `drop-shadow` token-based). **Solution** : remplacer par une utility CSS `text-shadow-adaptive` (à créer dans `globals.css`) :
     ```css
     :root[data-theme='dark'] .text-shadow-adaptive { text-shadow: 0 1px 6px rgba(0,0,0,0.85); }
     :root[data-theme='light'] .text-shadow-adaptive { text-shadow: 0 1px 6px rgba(255,255,255,0.85); }
     ```
   - `text-cyan-50` Spartan ID → `text-foreground` + `text-shadow-adaptive`.
   - `text-cyan-100/70` identity unavailable → `text-muted-foreground`.
   - `bg-slate-950/60` emblem holder → `bg-card/80`.
   - `border-cyan-300/60` emblem ring → `border-primary/60`.
   - `text-cyan-100` monogramme → `text-primary-foreground`.
   - `bg-slate-950/15` career rank panel → `bg-background/40 backdrop-blur-sm`.
   - Empty state panel `bg-slate-950/22 border-white/10 text-white` → `bg-muted/40 border-border text-foreground`.
   - `text-cyan-100/72` description → `text-muted-foreground`.
2. **Skill peak card** : `bg-slate-950/35` → `bg-card`, `border-cyan-100/12` → `border-border`, `bg-slate-950/22` (no value) → `bg-muted/30`, `border-white/10` → `border-border`, `text-cyan-100/68` → `text-muted-foreground`, `text-cyan-100/78` → `text-muted-foreground`, `text-white` valeur → `text-foreground`, badge holder `bg-slate-950/60` → `bg-muted`.
3. **Battle Pass panel** : `bg-slate-950/80 border-white/15` frame image → `bg-card border-border`, `text-slate-300` placeholder → `text-muted-foreground`, gradient placeholder `bg-[radial-gradient(circle_at_top,rgba(14,165,233,0.18),transparent_45%),linear-gradient(135deg,rgba(15,23,42,1),rgba(51,65,85,0.95))]` → utiliser tokens via `bg-[radial-gradient(circle_at_top,var(--primary)/.18,transparent_45%),linear-gradient(135deg,var(--card),var(--muted))]` ou simplifier en `bg-muted` plat + image.
4. **HomeSessionCarousel** : L244, L248 `text-white` (playlist + mode dans paragraphe muted) → `text-foreground`.
5. **HomeChallengesList** : L100 `'bg-white/90'` divider → `'bg-border'` ou `'bg-muted-foreground/40'`. L173 (autre) `border-white/15` → `border-border`.

**Tests à mettre à jour** :
- [features/home/HomePage.test.tsx](apps/web/src/features/home/HomePage.test.tsx) — vérifier que les snapshots / queries (`text-cyan-*`, `bg-slate-*`) ne ciblent pas les classes désormais changées.

### Phase 2 — Media (lightbox + viewer + thumbnail)

**Fichiers** : [CoverFlowModal.tsx](apps/web/src/features/media/CoverFlowModal.tsx), [MediaViewer.tsx](apps/web/src/features/media/MediaViewer.tsx), [MediaToolbar.tsx](apps/web/src/features/media/MediaToolbar.tsx), [MediaMatchPicker.tsx](apps/web/src/features/media/MediaMatchPicker.tsx)

**Décision design** : CoverFlow et lightboxes thématisés comme le reste de l'app (validé utilisateur 2026-05-04). Plus de mode cinema spécial. Seuls les **backdrops** des modales (overlay au-dessus du contenu pour le dim) restent en `bg-black/X` — c'est un comportement standard d'overlay neutre, valide dans les 2 thèmes (un dim noir au-dessus d'une page claire reste lisible et convient au pattern modal).

**Changements** :
1. **CoverFlowModal (thématisé complet)** :
   - Backdrop `bg-black/90` → `bg-background/90` (modal overlay qui suit le thème). Le contenu de la modal lui-même devient une vraie surface card.
   - Header bar `bg-black/60 text-white` → `bg-card/95 text-foreground border-b border-border`.
   - `text-sm opacity-80` heading → conserver opacity (effet visuel) avec `text-foreground/80`.
   - Bouton réassociation `border-white/20 text-white/80 hover:border-white/50 hover:text-white` → `border-border text-foreground/80 hover:border-foreground/50 hover:text-foreground`.
   - Bouton autoChain : déjà sur `bg-success/20`, `bg-destructive/20` etc. (✅ tokens, OK).
   - Bouton close `text-white/70 hover:text-white` → `text-muted-foreground hover:text-foreground`.
   - Conteneur principal `bg-black` (image/video frame) → `bg-card` (le `<img>` ou `<video>` couvre la zone, mais en cas de letterbox le fond suit le thème).
   - Boutons prev/next `bg-black/60 text-white hover:bg-black/80` → `bg-card/90 text-foreground hover:bg-card border border-border`.
   - Player error fallback `bg-black text-white/80` → `bg-card text-foreground/80`. `text-white/40`, `/50`, `/30` (basename, sub-info) → `text-muted-foreground/X`.
   - `<img object-contain bg-black>` (lettrebox) → `bg-card`.
2. **MediaViewer** :
   - Like button compact `bg-black/55 text-rose-400` (liked) / `bg-black/45 text-white/50` (not liked) → conserver `text-rose-*` (exception § 20), passer `bg-black/55` → `bg-card/90`, `bg-black/45` → `bg-card/70`, `text-white/50` → `text-muted-foreground`.
   - Like button standard `border-white/20 bg-black/35 text-white/85` (not liked) → `border-border bg-card/80 text-foreground`.
   - `text-white/60` (svg play icon overlay sur thumb) → conserver (overlay sur image map = OK), ou passer en `text-foreground/60` si le contraste reste correct.
3. **MediaToolbar** : aucune action (bouton `liked-only` rose = exception § 20 valide).
4. **MediaMatchPicker** : backdrop `bg-black/70` → `bg-background/70` (cohérent avec décision CoverFlow).

**Tests** :
- [MediaViewer.test.tsx](apps/web/src/features/media/MediaViewer.test.tsx) — vérifier les classes ciblées.

### Phase 3 — Palmarès (lightbox récompense + carousel + page)

**Fichiers** : [BattlePassRewardLightbox.tsx](apps/web/src/features/palmares/BattlePassRewardLightbox.tsx), [BattlePassRewardCarousel.tsx](apps/web/src/features/palmares/BattlePassRewardCarousel.tsx), [SeasonPassPage.tsx](apps/web/src/features/palmares/SeasonPassPage.tsx)

**Changements** :
1. **Lightbox récompense (thématisé complet)** :
   - Backdrop `bg-black/85` → `bg-background/85` (cohérent avec décision modales).
   - Conteneur `bg-slate-950/95 border-white/10 text-white` → `bg-card border-border text-foreground`. Note : le glow par rareté reste (`rarityStyles.glow`).
   - Header `bg-black/40 border-white/10` → `bg-muted/60 border-border`.
   - `text-slate-400` subtitle → `text-muted-foreground`.
   - Close button `text-white/70 hover:bg-white/10 hover:text-white` → `text-muted-foreground hover:bg-muted hover:text-foreground`.
   - Image fallback `bg-black/60 bg-black/40 text-white` → `bg-muted/60 text-foreground`.
   - Boutons prev/next `bg-black/55 text-white/75 hover:bg-black/75 hover:text-white focus-visible:ring-white/50` → `bg-background/80 text-foreground/80 hover:bg-background hover:text-foreground focus-visible:ring-ring`.
   - Badges container `bg-black/40 border-white/10` → `bg-muted/40 border-border`.
   - `text-slate-200` description → `text-foreground`.
2. **Carousel** :
   - L138 `text-white` sur `bg-success` → `text-success-foreground` (token déjà défini).
   - L143 `text-white` sur `bg-warning/90` → `text-warning-foreground` (token déjà défini).
   - L150 gradient fallback `bg-[radial-gradient(circle_at_top,rgba(14,165,233,0.22),transparent_55%),linear-gradient(180deg,rgba(15,23,42,0.92),rgba(30,41,59,0.84))] text-white` → simplifier en `bg-muted text-foreground` plat (le rank en gros tient sans gradient).
   - L173 `border-white/15` (non-current) → `border-border`.
3. **SeasonPassPage** :
   - L80, 99 `font-semibold text-white tabular-nums` chips value → `font-semibold text-foreground tabular-nums` (sont sur card semi-transparent avec image bg, à valider visuel).
   - L82, 97 `text-white/60` chips label → `text-muted-foreground`.
   - L285 `text-white` titre overlay sur hero image → **conserver** (l'image hero a un gradient `from-black/90 via-black/50 to-transparent` au-dessus, garantit la lisibilité dans les 2 thèmes).
   - **Décision visuelle requise** : le `SeasonPassCard` a une image bg à `opacity-30` sur fond `bg-card/95`. En light theme, l'image (souvent sombre Halo) sera contrastée — `text-foreground` (texte sombre) sera lisible. En dark, l'image est cohérente avec la card sombre.

**Tests** :
- [BattlePassRewardLightbox.test.tsx](apps/web/src/features/palmares/BattlePassRewardLightbox.test.tsx) (à vérifier s'il existe)

### Phase 4 — Composants UI partagés (citation ring, star button) + cas isolés

**Fichiers** : [components/ui/citation-progress-ring.tsx](apps/web/src/components/ui/citation-progress-ring.tsx), [components/ui/star-button.tsx](apps/web/src/components/ui/star-button.tsx), [CompareDrawer.tsx](apps/web/src/features/compare/CompareDrawer.tsx), [TimeseriesCorrelationScatter.tsx](apps/web/src/features/timeseries/TimeseriesCorrelationScatter.tsx), [AccessibilityTab.tsx](apps/web/src/features/settings/AccessibilityTab.tsx), [PlayerDetailPanel.tsx](apps/web/src/features/match-view/PlayerDetailPanel.tsx)

**Changements** :
1. **citation-progress-ring** :
   - L28 `const trackColor = 'rgba(255,255,255,0.12)'` → utiliser `getComputedStyle(document.documentElement).getPropertyValue('--border')` ou plus simple : `var(--border)` directement comme valeur de `stroke`. **Solution recommandée** : `stroke="var(--border)"` (la CSS var bascule auto avec le thème).
   - L80 `bg-white/10` placeholder → `bg-muted/60`.
2. **star-button** :
   - L36 (compact) `bg-warning/90 text-white` (favori) → `bg-warning text-warning-foreground`. `bg-black/45 text-white/60 hover:bg-warning/70 hover:text-white` (non favori, overlay sur image map) → conserver (overlay sur image OK).
3. **CompareDrawer** : `bg-black/40` backdrop → `bg-background/40` (cohérent avec décision modales thématisées).
4. **TimeseriesCorrelationScatter** : `bg-white/90` (toggle bg) → `bg-card border-border` (à vérifier dans contexte palette swatch).
5. **AccessibilityTab** : `border-white/10` swatch → `border-border`.
6. **PlayerDetailPanel** : L103 `text-white` (badge coloré) → si fond coloré dynamique → conserver, sinon `text-primary-foreground`.

### Phase 5 — Charts ECharts (axes/labels invisibles en light) — stratégie v4

**Fichiers** (v4) : 1 pivot (`_utils.ts`) + 6 wrappers partagés (`components/charts/*`) + 9 charts spécifiques (`features/squad/charts/*` + 2 timeseries/engagement) + `PlayerChips.tsx` = **17 fichiers**.

#### 5.1 — Approche pilote (validée utilisateur, v4)

**Stratégie revue v4** : profiter du pivot `_utils.ts` pour refactorer en **2 vagues** (helper d'abord, puis chart pilote consommateur) :

1. **Vague 1 — helpers `themeColors.ts` + `useThemeVersion.ts`** (créés dans `lib/echarts/`) — Phase 5a
2. **Vague 2 — pivot `_utils.ts`** : transformer `GRID_COLOR`, `TEXT_COLOR`, `ZERO_LINE` en getters dynamiques + `axisBase`, `tooltipBase`, `legendBase` en factories `getAxisBase()`, etc. — Phase 5b
3. **Vague 3 — chart pilote** ([squadSynergyRadarChart](apps/web/src/features/squad/charts/squadSynergyRadarChart.ts) retenu — 4 hardcodes représentatifs : `axisName`, `splitLine`, `axisLine`, `splitArea`, instrumenté E2E via [squad-charts-render.spec.ts](apps/web/e2e/squad-charts-render.spec.ts)). — Phase 5c
4. **Vague 4 — test E2E Playwright** — Phase 5d
5. **🚦 Gate utilisateur** avant propagation
6. **Vague 5 — propagation** : 6 wrappers partagés branchent leur `useMemo(option)` sur `useThemeVersion()` (les bases `axisBase`/`tooltipBase` sont déjà dynamiques après Phase 5b) ; les 8 charts spécifiques restants sont migrés sur le pattern pilote ; `PlayerChips.tsx` aligné. — Phase 5e
7. **Vague 6 — extension du test E2E** à tous les charts. — Phase 5f

#### 5.2 — Création du helper

**Nouveau fichier** : `apps/web/src/lib/echarts/themeColors.ts`

```ts
/**
 * Résout les couleurs sémantiques du thème (light/dark) pour ECharts.
 * ECharts est rendu en <canvas> et ne résout PAS les CSS vars dans les configs.
 * On lit donc les valeurs computed via getComputedStyle au runtime.
 *
 * À appeler à chaque rendu de l'option ECharts. Combiné avec useThemeVersion()
 * pour re-render automatique sur switch de thème.
 */
export interface EChartsThemeColors {
  axisLabel: string         // var(--muted-foreground) — labels d'axe
  axisLine: string          // var(--border) — ligne d'axe
  splitLine: string         // var(--border) — gridlines
  splitAreaA: string        // bande paire splitArea radar (translucide)
  splitAreaB: string        // bande impaire splitArea radar (translucide légèrement plus opaque)
  text: string              // var(--foreground) — labels valeurs / tooltips
  tooltipBg: string         // var(--popover) — fond tooltip
  tooltipBorder: string     // var(--border)
}

export function getEChartsThemeColors(): EChartsThemeColors {
  const cs = getComputedStyle(document.documentElement)
  // Les vars sont en oklch(...) — ECharts accepte la valeur CSS string brute.
  // S'il y a un fallback nécessaire (test env sans CSS résolu), retourner des hex sûrs.
  const get = (name: string, fallback: string) =>
    cs.getPropertyValue(name).trim() || fallback
  // splitArea radar : bandes alternées avec opacité — utiliser muted avec rgba/oklch alpha
  // Stratégie : on retourne la CSS var avec un wrapper color-mix pour l'alpha.
  // Fallback : rgba neutre.
  return {
    axisLabel:     get('--muted-foreground', '#9ca3af'),
    axisLine:      get('--border', '#374151'),
    splitLine:     get('--border', '#374151'),
    splitAreaA:    `color-mix(in oklch, ${get('--muted', '#374151')} 15%, transparent)`,
    splitAreaB:    `color-mix(in oklch, ${get('--muted', '#374151')} 30%, transparent)`,
    text:          get('--foreground', '#f3f4f6'),
    tooltipBg:     get('--popover', 'rgba(20,24,30,0.92)'),
    tooltipBorder: get('--border', '#374151'),
  }
}
```

**Nouveau hook** : `apps/web/src/lib/echarts/useThemeVersion.ts`

```ts
import { useEffect, useState } from 'react'

/**
 * Renvoie un compteur qui s'incrémente quand `data-theme` change sur <html>.
 * À utiliser dans les `useMemo` de génération d'option ECharts pour
 * forcer le recalcul (et donc le re-render canvas) sur switch de thème.
 */
export function useThemeVersion(): number {
  const [version, setVersion] = useState(0)
  useEffect(() => {
    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        if (m.type === 'attributes' && m.attributeName === 'data-theme') {
          setVersion((v) => v + 1)
          return
        }
      }
    })
    observer.observe(document.documentElement, { attributes: true })
    return () => observer.disconnect()
  }, [])
  return version
}
```

**Pattern d'usage dans un chart** (à appliquer dans `squadSynergyRadarChart.ts` côté composant React qui consomme l'option, pas dans le builder pur) :

```ts
const themeVersion = useThemeVersion()
const option = useMemo(() => {
  const themeColors = getEChartsThemeColors()
  return buildSquadSynergyRadarOption(data, themeColors)
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [data, themeVersion])
```

Le builder `buildSquadSynergyRadarOption` accepte un nouveau paramètre `themeColors: EChartsThemeColors` et substitue tous les `rgba(255,255,255,X)` par les valeurs reçues.

#### 5.3 — Test E2E Playwright pilote

**Nouveau fichier** : `apps/web/e2e/theme-switch-charts.spec.ts`

```ts
/**
 * E2E — Switch de thème : vérifie que les couleurs des charts ECharts
 * suivent le thème (axes/labels lisibles en light comme en dark).
 *
 * Stratégie : on rend la page Escouade avec le radar synergy, on lit la
 * couleur computed des labels d'axe (via canvas → impossible directement,
 * donc on lit la CSS var du thème et on vérifie qu'elle est répercutée
 * dans l'option ECharts via le data-attribute exposé sur le wrapper).
 *
 * Approche alternative (plus robuste) : on lit la valeur computed de
 * --muted-foreground avant et après le toggle, et on s'assure que les
 * deux valeurs diffèrent (preuve que le thème a switché). Puis on
 * vérifie via le DOM que les charts sont toujours rendus (pas de
 * régression visuelle évidente).
 */
import { test, expect } from '@playwright/test'

const PLAYER_SLUG = 'JGtm'

test.describe('Theme switch — charts ECharts suivent le thème', () => {
  test('le radar synergy re-render à la couleur du nouveau thème', async ({ page }) => {
    await page.goto(`/players/${PLAYER_SLUG}/escouade`)
    await page.waitForSelector('[data-testid="squad-synergy-radar"]', { timeout: 15_000 })

    // Lire la couleur muted-foreground en thème courant
    const initialTheme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'))
    const initialMutedFg = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue('--muted-foreground').trim()
    )
    expect(initialMutedFg).not.toBe('')

    // Toggle thème via le ThemeToggle
    const toggleLabel = initialTheme === 'dark' ? 'Passer au thème clair' : 'Passer au thème sombre'
    await page.getByRole('switch', { name: toggleLabel }).click()

    // Attendre la propagation
    await page.waitForFunction(
      (prev) => document.documentElement.getAttribute('data-theme') !== prev,
      initialTheme,
      { timeout: 2000 },
    )

    const newTheme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'))
    const newMutedFg = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue('--muted-foreground').trim()
    )
    expect(newTheme).not.toBe(initialTheme)
    expect(newMutedFg).not.toBe(initialMutedFg)

    // Le radar est toujours visible — preuve que le re-render n'a pas crashé
    await expect(page.locator('[data-testid="squad-synergy-radar"]')).toBeVisible()

    // Optionnel — capture screenshot pour validation visuelle
    await page.locator('[data-testid="squad-synergy-radar"]').screenshot({
      path: `tests/e2e-results/squad-synergy-radar-${newTheme}.png`,
    })
  })

  test('dans les 2 thèmes, les couleurs --foreground/--muted-foreground/--border sont toutes définies', async ({ page }) => {
    await page.goto(`/players/${PLAYER_SLUG}/escouade`)

    const checkVars = async () => {
      const vars = await page.evaluate(() => {
        const cs = getComputedStyle(document.documentElement)
        return {
          fg:  cs.getPropertyValue('--foreground').trim(),
          mfg: cs.getPropertyValue('--muted-foreground').trim(),
          brd: cs.getPropertyValue('--border').trim(),
        }
      })
      expect(vars.fg).not.toBe('')
      expect(vars.mfg).not.toBe('')
      expect(vars.brd).not.toBe('')
    }

    await checkVars()
    const initialTheme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'))
    const toggleLabel = initialTheme === 'dark' ? 'Passer au thème clair' : 'Passer au thème sombre'
    await page.getByRole('switch', { name: toggleLabel }).click()
    await checkVars()
  })
})
```

**Critères d'acceptation du pilote** (avant propagation aux 9 autres charts) :
- ✅ Le radar synergy se re-render visuellement après toggle (capture screenshot light + dark différents).
- ✅ Aucune erreur console après le switch (pas de "Cannot read property of undefined" ECharts).
- ✅ Les CSS vars sont effectivement résolues (les couleurs ne sont pas vides).
- ✅ Le test E2E Playwright passe.
- ✅ Validation manuelle utilisateur : "le radar est lisible dans les 2 thèmes".

#### 5.4 — Propagation aux fichiers restants (v4)

Une fois le pilote validé, propager :

**A. Wrappers partagés `components/charts/`** — branchent `useMemo(option)` sur `useThemeVersion()` ; les bases (`axisBase`, `tooltipBase`, `legendBase`) sont déjà dynamiques via le pivot `_utils.ts` refactoré en Phase 5b. Reste à migrer les hardcodes inline spécifiques :

| Fichier | Substitutions inline |
|---|---|
| [components/charts/RadarChart.tsx](apps/web/src/components/charts/RadarChart.tsx) | 4 (axisName, splitArea ×2, splitLine, axisLine) |
| [components/charts/HistogramChart.tsx](apps/web/src/components/charts/HistogramChart.tsx) | 2 (nameTextStyle X+Y) |
| [components/charts/ScatterChart.tsx](apps/web/src/components/charts/ScatterChart.tsx) | 2 (nameTextStyle X+Y) |
| [components/charts/Heatmap2DChart.tsx](apps/web/src/components/charts/Heatmap2DChart.tsx) | 1 (visualMap.textStyle) |
| [components/charts/DonutChart.tsx](apps/web/src/components/charts/DonutChart.tsx) | 1 (label.color) |
| [components/charts/EngagementCurve.tsx](apps/web/src/components/charts/EngagementCurve.tsx) | 1 (nameTextStyle) |
| [components/PlayerChips.tsx](apps/web/src/components/PlayerChips.tsx) | 3 (à inventorier au refactor) |

**B. Charts spécifiques `features/squad/charts/` + timeseries/engagement** :

| Fichier | Substitutions |
|---|---|
| [squadSessionTimelineChart.ts](apps/web/src/features/squad/charts/squadSessionTimelineChart.ts) | 2 `rgba(...,0.6)` axisName |
| [squadIntensityHeatmapChart.ts](apps/web/src/features/squad/charts/squadIntensityHeatmapChart.ts) | 2 `rgba(...,0.7)` visualMap |
| [squadMapHeatmapChart.ts](apps/web/src/features/squad/charts/squadMapHeatmapChart.ts) | 2 `rgba(...,0.7)` visualMap |
| [squadPerMinuteChart.ts](apps/web/src/features/squad/charts/squadPerMinuteChart.ts) | 2 `rgba(...,0.85)` label |
| [squadWeaponKillsChart.ts](apps/web/src/features/squad/charts/squadWeaponKillsChart.ts) | 2 `rgba(...,0.92)` label |
| [squadFirstEventsChart.ts](apps/web/src/features/squad/charts/squadFirstEventsChart.ts) | 2 (à inventorier) **NOUVEAU v4** |
| [squadPerformanceLineCharts.ts](apps/web/src/features/squad/charts/squadPerformanceLineCharts.ts) | 2 (à inventorier, hors `'#aaa'` self-ghost à conserver) **NOUVEAU v4** |
| [TimeseriesKdaBars.tsx](apps/web/src/features/timeseries/TimeseriesKdaBars.tsx) | 1 `rgba(...,0.65)` nameTextStyle |
| [SquadEngagementView.tsx](apps/web/src/features/squad/v2/SquadEngagementView.tsx) | 1 `rgba(...,0.45)` nameTextStyle |

**Substitutions standards** :
- `rgba(255,255,255,0.65)` (axisName) → `themeColors.axisLabel`
- `rgba(255,255,255,0.85-0.92)` (label sur barre) → `themeColors.text`
- `rgba(255,255,255,0.1)` (splitLine, axisLine) → `themeColors.splitLine`
- `rgba(255,255,255,0.45-0.7)` (visualMap textStyle) → `themeColors.axisLabel`
- `rgba(255,255,255,0.02-0.05)` (splitArea radar — bandes alternées) → `themeColors.splitArea` (à ajouter au helper)

#### 5.5 — Étendre le test E2E

Une fois la propagation faite, étendre `theme-switch-charts.spec.ts` pour couvrir les autres charts :

```ts
test('tous les charts squad+timeseries restent rendus après toggle de thème', async ({ page }) => {
  // Naviguer Squad puis Timeseries puis Engagement
  // Pour chaque page : capture page complète light + dark, vérifier visibilité des testids
  const chartTestIds = [
    'squad-synergy-radar',
    'squad-session-timeline',
    'squad-intensity-heatmap',
    'squad-map-heatmap',
    'squad-per-minute-chart',
    'squad-weapon-kills',
    'timeseries-kda-bars',
    'squad-engagement-view',
  ]
  // ... toggle + assert visibility chacun
})
```

**Note** : les `'#aaa'` dans les tests unitaires (`squadPerformanceLineCharts.test.ts`, `squadFirstEventsChart.test.ts`) testent la couleur "self ghost" — c'est une couleur fixe pour distinguer un joueur "soi" sur un graphe d'équipe. **À conserver telle quelle** (pas une couleur sémantique de thème, c'est un identifiant visuel intentionnel).

**Tests unitaires** : tous les tests de charts qui matchent une couleur en dur via `toMatchObject({ color: 'rgba(...' })` devront être mis à jour pour matcher soit `expect.any(String)` (test plus laxiste sur la couleur), soit la valeur résolue du thème de test (mock `getComputedStyle` dans le setup Vitest).

### Phase 6 — Cleanup `color-allow:` annotations

Supprimer toutes les annotations `// color-allow: thématique Spartan UI` (devenues obsolètes après refactor). Conserver les annotations légitimes :
- `// color-allow: rose pour like` (rose / liked — exception § 20)
- `// color-allow: amber gold pour étoile favori` (amber — exception § 20)
- `// color-allow: rareté` (palmares/rarity.ts)
- `// color-allow: structurel SVG` (rank-progress-gauge)
- `// color-allow: blanc structurel sur fond joueur` (compare-a / medal digest avec fond coloré)
- `// color-allow: heatmap intensité` (heatmap squad)
- `// color-allow: fallback couleur citation` (couleur API)

### Phase 7 — Validation

1. **Build TypeScript** : `pnpm --filter web typecheck` ou équivalent.
2. **Tests unitaires** : `pnpm --filter web test` — corriger les snapshots/queries cassés.
3. **Test manuel light/dark** (chaque écran à valider dans les 2 thèmes) :
   - Home : banner identity + cartes CSR/LUSR + Battle Pass panel + Recent Matches (MatchCards) + Sessions carousel + Challenges list.
   - Palmarès : lightbox récompense ouverte (Common / Rare / Epic / Legendary / Mythic) + carousel de tiers + page complète (hero + StatCard).
   - Media : modal CoverFlow ouvert (image + vidéo) + thumbnail like states + toolbar `liked-only`.
   - Match View : MatchCard détail + PlayerDetailPanel.
   - Squad : tous les charts (radar synergy, intensity heatmap, map heatmap, session timeline, per minute, weapon kills) — vérifier que les axes/labels sont lisibles dans les 2 thèmes.
   - Compare drawer ouvert.
   - Settings → Accessibility → swatch palette.
4. **Linter custom couleurs** : vérifier qu'aucune nouvelle annotation `color-allow: thématique Spartan UI` n'existe.
5. **Audit final** : relancer le grep des hardcodes (cf. checklist § 8) pour confirmer le nettoyage.

---

## 5. Conventions à appliquer

1. **Un seul commit par phase** sur la branche unique `fix/theme-consistency-tokens`.
2. **Commits atomiques** par phase (préfixe `refactor(web):`) :
   - Phase 0 : `refactor(web): match-card — passer panneau stats aux tokens (text-foreground, bg-muted)`
   - Phase 1 : `refactor(web): home — passer banner+peaks+battlepass+sessions+challenges aux tokens (+ text-shadow-adaptive)`
   - Phase 2 : `refactor(web): media — thématiser CoverFlow + lightbox + viewer (plus de cinema mode forcé)`
   - Phase 3 : `refactor(web): palmares — thématiser reward lightbox+carousel+page`
   - Phase 4 : `refactor(web): cas isolés (citation-ring, star-button, compare, timeseries, settings, player-detail)`
   - Phase 5a : `feat(web): helper themeColors + useThemeVersion pour ECharts`
   - Phase 5b : `refactor(web): pivot _utils.ts — bases axisBase/tooltipBase/legendBase dynamiques (themeColors)`
   - Phase 5c : `refactor(web): radar synergy — couleurs axes via themeColors (chart pilote)`
   - Phase 5d : `test(web): E2E theme-switch-charts — pilote radar synergy`
   - **🚦 Gate utilisateur** : valider le pilote avant propagation.
   - Phase 5e : `refactor(web): propager themeColors aux wrappers partagés + 8 charts spécifiques + PlayerChips`
   - Phase 5f : `test(web): étendre E2E theme-switch-charts à tous les charts`
   - Phase 6 : `chore(web): supprimer les annotations color-allow:thématique-Spartan-UI obsolètes`
3. **Supprimer toutes les annotations `// color-allow: thématique Spartan UI`** (devenues obsolètes).
4. **Conserver** les `// color-allow: rose pour like…`, `// color-allow: amber gold (étoile favori)`, `// color-allow: rareté Halo`, `// color-allow: heatmap intensité`, `// color-allow: structurel SVG`, `// color-allow: blanc structurel sur fond compare-a/medal joueur` (toujours valides).
5. **thought_log** : entrée obligatoire à la fin du refactor (cf. CLAUDE.md § Workflow Agentique).

---

## 6. Risques & points d'attention

1. **Régression visuelle** : la palette light a un faible contraste avec certains glow/shadow conçus pour dark. Tester chaque écran en light après refactor.
2. **Tests qui matchent des classes Tailwind** : les `getByClassName(/text-cyan-/)` ou `expect(...).toHaveClass('bg-slate-950')` casseront — à mettre à jour. Lister à l'avance via `grep -r "text-cyan\|text-white\|bg-slate-\|bg-black" apps/web/src/**/*.test.*`.
3. **Banner image overlay (thématisé — décision validée)** : le gamertag/Spartan ID sont en overlay sur une `<img>`. Solution : ajouter un overlay gradient **adaptatif** `bg-gradient-to-b from-background/30 via-background/10 to-background/40` au-dessus de l'image, qui suit le thème, combiné avec `text-foreground` + utility CSS `text-shadow-adaptive` (text-shadow noir en dark, blanc en light). À valider visuellement : si en light theme l'image hero (souvent sombre Halo) reste très contrastée, augmenter l'opacité du gradient (`from-background/50`) ou le rendre directionnel.
4. **CoverFlow + lightbox récompense (thématisés — décision validée)** : modales plein écran avec vidéo / image entièrement thématisées. Backdrops en `bg-background/X` au lieu de `bg-black/X`. ⚠️ **Risque** : pour les vidéos en lecture (CoverFlow), un fond clair en light theme autour de la vidéo peut être visuellement gênant (lecteur vidéo flashy). Validation visuelle obligatoire en Phase 2.
5. **Charts ECharts (Phase 5)** : ECharts est rendu en `<canvas>` — les CSS vars (`var(--muted-foreground)`) **ne sont pas** automatiquement résolues par ECharts. Solution adoptée :
   - `getComputedStyle(document.documentElement).getPropertyValue('--muted-foreground')` au moment de générer l'option (helper `getEChartsThemeColors()` Phase 5a)
   - `MutationObserver` sur `data-theme` qui incrémente un compteur `useThemeVersion()` placé en dépendance du `useMemo` de l'option (Phase 5a)
   - Test E2E Playwright pilote (Phase 5c) avant propagation aux 9 autres charts
   - **Risque résiduel** : performance — chaque switch déclenche un re-render complet du chart (incluant transitions ECharts). Si visible/gênant, optimiser avec `chart.setOption(option, { lazyUpdate: true })` ou debounce sur le `useThemeVersion`.
6. **Linter custom** : si un linter ESLint check les classes Tailwind contre une whitelist (script `scripts/check_color_tokens.*`), mettre à jour la liste — chercher les références à `bg-slate-950` etc. dans les configs de lint.
7. **Snapshot tests** : si Vitest/Jest a des snapshots HTML avec les anciennes classes → régénérer (`pnpm test -u`).
8. **`text-white` sur backgrounds colorés tokens** : préférer le foreground correspondant (`text-{success,warning,destructive,primary}-foreground`) plutôt que `text-white` figé — cohérence avec le système.
9. **Performance ECharts** : si la solution choisie est de re-render le chart sur changement de thème, vérifier qu'il n'y a pas de flicker / coût visible (les charts sont parfois lourds). Throttle si besoin.
10. **Cohérence multi-titres** : le ThemeProvider est shared entre titres → le refactor profite à tous les titres (Halo Infinite et futurs).

---

## 7. Estimations (v4)

- **Phase 0 (MatchCard)** : 1 fichier, ~17 substitutions, ~25 min refactor + 15 min validation (tester sur Home, Match History, MatchView).
- **Phase 1 (Home)** : 5 fichiers + 1 utility CSS `text-shadow-adaptive`, ~22 substitutions, ~45 min + 25 min.
- **Phase 2 (Media)** : 4 fichiers, ~24 substitutions (thématisation complète CoverFlow), ~45 min + 20 min.
- **Phase 3 (Palmarès)** : 3 fichiers, ~18 substitutions, ~30 min + 20 min.
- **Phase 4 (cas isolés + composants UI)** : 6 fichiers, ~10 substitutions, ~25 min + 15 min.
- **Phase 5a (helper themeColors + useThemeVersion)** : 2 nouveaux fichiers, ~25 min.
- **Phase 5b (pivot `_utils.ts` dynamique)** : 1 fichier ; transformer 4 constantes + 3 bases en factories `getAxisBase()` etc. ; ajuster les callers (`axisBase` consommé tel quel devient `getAxisBase()`). ~45 min + 15 min run de tests `_utils.test.ts`. **NOUVEAU v4**
- **Phase 5c (chart pilote radar synergy)** : 1 fichier + composant React consommateur, ~30 min refactor + 20 min validation manuelle.
- **Phase 5d (test E2E pilote)** : 1 fichier de test, ~40 min écriture + ~10 min run.
- **🚦 Gate utilisateur** : 0–24h selon disponibilité.
- **Phase 5e (propagation 6 wrappers + 8 charts spécifiques + PlayerChips)** : 15 fichiers, ~22 substitutions + branchement consommateurs sur `useThemeVersion`, ~1h45 + 45 min validation visuelle. **DÉRIVE v4 (+45 min)**
- **Phase 5f (extension E2E)** : ~30 min.
- **Phase 6 (cleanup color-allow)** : ~15 min.
- **Phase 7 (validation + thought_log)** : ~30 min.

**Total v4** : ~8h30 hors imprévus + gate utilisateur (vs 7h en v3). La dérive vient de :
- Pivot `_utils.ts` ajouté en Phase 5b (+1h)
- 6 wrappers partagés + 2 squad charts manqués + `PlayerChips` ajoutés en Phase 5e (+45 min)
- Inverse positif : la Phase 5e profite de la base dynamique du pivot — pas de patch ligne par ligne sur les bases axes/tooltip/legend pour les 11 wrappers

---

## 8. Checklist finale (avant merge)

### Hardcodes nettoyés
- [ ] `grep -r "bg-slate-950" apps/web/src/features/ apps/web/src/components/` → 0 résultat
- [ ] `grep -r "text-cyan-" apps/web/src/features/ apps/web/src/components/` → 0 résultat
- [ ] `grep -r "border-cyan-" apps/web/src/features/ apps/web/src/components/` → 0 résultat
- [ ] `grep -r "text-slate-" apps/web/src/features/ apps/web/src/components/` → 0 résultat
- [ ] `grep -r "text-white\b" apps/web/src/features/ apps/web/src/components/` → uniquement les exceptions documentées (overlay sur image, fond coloré token)
- [ ] `grep -r "rgba(255,255,255" apps/web/src/features/squad/ apps/web/src/features/timeseries/ apps/web/src/components/charts/ apps/web/src/components/PlayerChips.tsx` → 0 résultat (remplacés par tokens)

### Annotations
- [ ] `grep -r "color-allow: thématique Spartan UI" apps/web/src/` → 0 résultat
- [ ] Toutes les annotations `color-allow:` restantes sont dans la liste autorisée (rarity, like-rose, amber-favori, structurel-svg, blanc-sur-fond-token, heatmap, citation-fallback)

### Build & tests
- [ ] `pnpm --filter web typecheck` passe
- [ ] `pnpm --filter web test` passe (snapshots régénérés si besoin)
- [ ] `pnpm --filter web lint` passe (si linter custom couleurs configuré)
- [ ] `npx playwright test e2e/theme-switch-charts.spec.ts` passe (test E2E charts)
- [ ] `npx playwright test` (suite complète) passe — pas de régression sur les autres specs (squad-charts-render, slice-5-home, slice-8-media, etc.)

### Test manuel light (plus exhaustif que v1)
- [ ] Home : banner identity + cartes peak + Battle Pass + Recent Matches + Sessions carousel + Challenges
- [ ] Match History : tableau de cartes
- [ ] Match View : détail + PlayerDetailPanel
- [ ] Squad : tous les charts (radar, heatmap, timeline, barres) avec axes lisibles
- [ ] Palmarès : page complète + lightbox récompense (toutes raretés) + carousel
- [ ] Media : grille thumbnails + CoverFlow modal (image + vidéo) + toolbar
- [ ] Compare drawer
- [ ] Settings → Accessibility

### Test manuel dark (régression nulle)
- [ ] Mêmes écrans qu'au-dessus → aucune régression visuelle vs avant le refactor

### Doc & process
- [ ] Entrée `.ai/thought_log.md` ajoutée
- [ ] Skill `delivery-checklist` invoqué avant le commit final
- [ ] Plan archivé dans `.ai/archive/` après merge

---

## 9. Tableau récapitulatif des fichiers (v4)

| Phase | Fichier | Type | Occ. |
|---|---|---|---|
| 0 | `components/ui/match-card.tsx` | Component partagé | 17 |
| 1 | `features/home/HomeSpartanIdentityBanner.tsx` | Feature | 8 + 4 white |
| 1 | `features/home/HomeSkillPeakCard.tsx` | Feature | 6 + 4 white |
| 1 | `features/home/HomeBattlePassPanel.tsx` | Feature | 2 + 1 white |
| 1 | `features/home/HomeSessionCarousel.tsx` | Feature | 2 white |
| 1 | `features/home/HomeChallengesList.tsx` | Feature | 1 white |
| 2 | `features/media/CoverFlowModal.tsx` | Feature | 10 white |
| 2 | `features/media/MediaViewer.tsx` | Feature | 3 + 3 white |
| 2 | `features/media/MediaToolbar.tsx` | Feature | 1 (rose, OK) |
| 2 | `features/media/MediaMatchPicker.tsx` | Feature | 1 white (backdrop) |
| 3 | `features/palmares/BattlePassRewardLightbox.tsx` | Feature | 3 + 9 white |
| 3 | `features/palmares/BattlePassRewardCarousel.tsx` | Feature | ~5 white |
| 3 | `features/palmares/SeasonPassPage.tsx` | Feature | 5 white |
| 4 | `components/ui/citation-progress-ring.tsx` | Component partagé | 1 + 1 rgba |
| 4 | `components/ui/star-button.tsx` | Component partagé | 1 white |
| 4 | `features/compare/CompareDrawer.tsx` | Feature | 1 white |
| 4 | `features/timeseries/TimeseriesCorrelationScatter.tsx` | Feature | 1 white |
| 4 | `features/settings/AccessibilityTab.tsx` | Feature | 1 white |
| 4 | `features/match-view/PlayerDetailPanel.tsx` | Feature | 1 white |
| 5a | `apps/web/src/lib/echarts/themeColors.ts` | **Nouveau helper** | — |
| 5a | `apps/web/src/lib/echarts/useThemeVersion.ts` | **Nouveau hook** | — |
| 5b | `components/charts/_utils.ts` | **Pivot ECharts** | 4 (constantes + tooltip) **NOUVEAU v4** |
| 5c | `features/squad/charts/squadSynergyRadarChart.ts` | Chart pilote | 4 rgba |
| 5d | `apps/web/e2e/theme-switch-charts.spec.ts` | **Nouveau test E2E** | — |
| 5e | `components/charts/RadarChart.tsx` | Wrapper partagé | 4 rgba **NOUVEAU v4** |
| 5e | `components/charts/HistogramChart.tsx` | Wrapper partagé | 2 rgba **NOUVEAU v4** |
| 5e | `components/charts/ScatterChart.tsx` | Wrapper partagé | 2 rgba **NOUVEAU v4** |
| 5e | `components/charts/Heatmap2DChart.tsx` | Wrapper partagé | 1 rgba **NOUVEAU v4** |
| 5e | `components/charts/DonutChart.tsx` | Wrapper partagé | 1 rgba **NOUVEAU v4** |
| 5e | `components/charts/EngagementCurve.tsx` | Wrapper partagé | 1 rgba **NOUVEAU v4** |
| 5e | `components/PlayerChips.tsx` | Composant partagé | 3 rgba **NOUVEAU v4** |
| 5e | `features/squad/charts/squadSessionTimelineChart.ts` | Chart | 2 rgba |
| 5e | `features/squad/charts/squadIntensityHeatmapChart.ts` | Chart | 2 rgba |
| 5e | `features/squad/charts/squadMapHeatmapChart.ts` | Chart | 2 rgba |
| 5e | `features/squad/charts/squadPerMinuteChart.ts` | Chart | 2 rgba |
| 5e | `features/squad/charts/squadWeaponKillsChart.ts` | Chart | 2 rgba |
| 5e | `features/squad/charts/squadFirstEventsChart.ts` | Chart | 2 rgba **NOUVEAU v4** |
| 5e | `features/squad/charts/squadPerformanceLineCharts.ts` | Chart | 2 rgba **NOUVEAU v4** |
| 5e | `features/timeseries/TimeseriesKdaBars.tsx` | Chart | 1 rgba |
| 5e | `features/squad/v2/SquadEngagementView.tsx` | Chart | 1 rgba |

**Total v4** : 35 fichiers à modifier + 3 fichiers à créer (helper + hook + E2E spec) — vs 28+1 en v3.
