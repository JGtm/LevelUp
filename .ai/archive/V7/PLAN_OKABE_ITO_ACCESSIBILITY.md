# Plan — Accessibilité daltoniens (palette Okabe-Ito)

> **Objectif** : rendre toute l'app `apps/web/` compatible avec la palette Okabe-Ito (8 couleurs distinguables par les principaux types de daltonisme : protanopie, deutéranopie, tritanopie). Toggle activable depuis une **section dédiée des Paramètres**.
>
> **Principes directeurs** :
> 1. **Modularité** — palette isolée dans une couche dédiée, jamais référencée directement par un composant ou un chart.
> 2. **Simplicité** — un seul mapping sémantique central ; aucun fork de composant.
> 3. **Séparation des responsabilités** — palette → tokens sémantiques → CSS variables → composants. Chaque couche ne connaît que celle directement en dessous.
> 4. **Réversibilité** — l'utilisateur peut toujours revenir à la palette par défaut sans rechargement.
> 5. **Zéro magic hex** dans les composants après migration. Tout passe par un token.

---

## 0. Contexte (état des lieux)

L'exploration de `apps/web/src/` a révélé :

| Élément | Localisation | État |
|---|---|---|
| Variables CSS racine (OKLCh) | `src/styles/globals.css` | ✅ Fondation saine, 48 tokens, supporte light/dark via `data-theme` |
| Theme provider | `src/app/providers/theme-provider.tsx` | ✅ Lit Zustand → applique attribut DOM |
| Store préférences UI | `src/stores/settingsDraftStore.ts` | ✅ Persistance localStorage `levelup-ui-prefs`, déjà type `UiTheme` |
| Page Settings (onglets) | `src/features/settings/SettingsPage.tsx` | ⚠️ Pas d'onglet « Accessibilité » à ce jour |
| Hex hardcodés (perf score) | `src/lib/perf-color.ts` | ❌ 5 hex codés en dur |
| Hex hardcodés (outcomes) | `src/lib/outcome-color.ts` | ❌ Win/Loss/Draw/DNF |
| Hex hardcodés (badges narratifs) | `src/components/ui/match-card-presentation.ts` | ❌ DOMINANT/HUMILIATION/REMONTADA/DÉBÂCLE/CONTRE-REMONTADA |
| Charts Plotly (3) | `src/features/squad/charts/{hsPk,timeline,heatmap}Chart.ts` | ❌ Palettes hex inline |
| Heatmap timeseries | `src/components/ui/timeseries-heatmap.tsx` | ❌ 2 colorscales hex |
| Système i18n | Décentralisé (`features/*/i18n.ts`) | OK pour ajouter les libellés |

Hex à risque identifiés (deutéranopie/protanopie surtout) :
- `#0072B2` navy (badge REMONTADA + heatmap intensité) — confondu avec violet/marron.
- `#F59E0B` ↔ `#10B981` (warning ↔ success) — confondus en deutéranopie.
- Échelle perf 5 niveaux (vert→cyan→ambre→orange→rouge) — collision rouge/vert.

---

## 1. Architecture cible (6 couches)

```
┌─────────────────────────────────────────────────────────────┐
│ 6. COMPOSANTS / CHARTS — appellent UNE scale                │
│    perfScale(score) → token → couleur                       │
└─────────────────────────────────────────────────────────────┘
                          ▲
┌─────────────────────────────────────────────────────────────┐
│ 5. SCALES — value → SemanticToken (logique métier ordinale  │
│    / divergente / catégorielle, seuils centralisés)         │
│    src/lib/accessibility/scales/                            │
│    ex: perfScale, kdScale, mmrDeltaScale, outcomeScale      │
└─────────────────────────────────────────────────────────────┘
                          ▲
┌─────────────────────────────────────────────────────────────┐
│ 4. CSS VARIABLES SÉMANTIQUES injectées sur :root            │
│    --color-win, --color-perf-tier-1..5, --chart-series-1..8 │
│    Réécrites quand le mode change (default ↔ okabe-ito)     │
└─────────────────────────────────────────────────────────────┘
                          ▲
┌─────────────────────────────────────────────────────────────┐
│ 3. RESOLVER — applique la palette active aux tokens         │
│    src/lib/accessibility/applyPalette.ts                    │
└─────────────────────────────────────────────────────────────┘
                          ▲
┌─────────────────────────────────────────────────────────────┐
│ 2. MAPPING SÉMANTIQUE — palette × rôle = couleur            │
│    src/lib/accessibility/semantic-tokens.ts                 │
│    type SemanticToken = 'win'|'loss'|'perf-tier-1'|…        │
└─────────────────────────────────────────────────────────────┘
                          ▲
┌─────────────────────────────────────────────────────────────┐
│ 1. PALETTES BRUTES — constantes pures, aucune dépendance    │
│    src/lib/accessibility/palettes/{default,okabe-ito}.ts    │
└─────────────────────────────────────────────────────────────┘
```

**Règle d'or** : un composant ne doit JAMAIS importer depuis les couches 1-4 directement. Il importe :
- la couche 5 (`perfScale(score)`, `outcomeScale('win')`) si la logique « valeur → couleur » est concernée,
- la couche 4 (CSS var `var(--color-win)`) uniquement pour des couleurs **statiques** dans des feuilles de style ou des layouts Plotly.

**Nouvelle responsabilité (couche 5)** : centraliser les **seuils** et la **logique de mapping**, pas seulement les couleurs. C'est l'étage qui élimine la duplication identifiée dans l'inventaire (§4 ci-dessous).

---

## 2. Palette Okabe-Ito (référence)

8 couleurs publiées par Okabe & Ito (2008), distinguables par daltoniens :

| Nom | Hex | sRGB | Usage suggéré |
|---|---|---|---|
| Black | `#000000` | 0,0,0 | Texte / contour |
| Orange | `#E69F00` | 230,159,0 | Warning, perf-tier-2 |
| Sky Blue | `#56B4E9` | 86,180,233 | Info, série chart |
| Bluish Green | `#009E73` | 0,158,115 | Success/win, perf-tier-1 |
| Yellow | `#F0E442` | 240,228,66 | Highlight, série chart |
| Blue | `#0072B2` | 0,114,178 | Compare-A, série chart |
| Vermillion | `#D55E00` | 213,94,0 | Loss/destructive, perf-tier-5 |
| Reddish Purple | `#CC79A7` | 204,121,167 | Compare-B, série chart |

**Échelle ordinale 5 niveaux** (perf tier, K/D, intensité) — combiner luminosité ET teinte :
```
tier-1 (best)   #009E73  Bluish Green
tier-2          #56B4E9  Sky Blue
tier-3 (mid)    #F0E442  Yellow
tier-4          #E69F00  Orange
tier-5 (worst)  #D55E00  Vermillion
```
NB : pour les heatmaps, on combine cette échelle avec une variation d'**opacité** (10 %→100 %), ce qui restaure une lisibilité monochrome de secours.

---

## 3. Inventaire des tokens sémantiques

Fichier cible : `src/lib/accessibility/semantic-tokens.ts`

### 3.1 Outcomes (4)
`outcome-win`, `outcome-loss`, `outcome-draw`, `outcome-dnf`

### 3.2 Performance / qualité (5 tiers ordinaux)
`perf-tier-1` (excellent) → `perf-tier-5` (mauvais)
> Réutilisés par `perfScale`, `accuracyScale`, `kdScale`, `progressScale` — pas besoin de tokens dédiés par cas (c'est tout l'intérêt de la couche scales §5).

### 3.3 Divergent (3)
`divergent-pos`, `divergent-neutral`, `divergent-neg`
> Consommé par `mmrDeltaScale`, `skillDeltaScale`, et tout futur indicateur signé.

### 3.4 Statuts UI (4)
`success`, `warning`, `info`, `destructive`

### 3.5 Comparaisons (2)
`compare-a`, `compare-b`

### 3.6 Séries de charts (8 max — pile sur Okabe-Ito)
`chart-series-1` … `chart-series-8`
> Réutilisent les 8 couleurs Okabe-Ito en cycle. Une couleur peut être identique à un token sémantique (ex: `chart-series-1` ≈ `outcome-win`) — c'est sans conséquence puisque les contextes d'usage sont disjoints (une série de chart n'apparaît jamais à côté d'un badge outcome dans un même graphique).

### 3.7 Badges narratifs (5)
`narrative-dominant`, `narrative-humiliation`, `narrative-remontada`, `narrative-debacle`, `narrative-contre-remontada`

### 3.8 Heatmaps (2 axes)
`heatmap-cold` → `heatmap-hot` (générée par interpolation OKLCh à partir de 3 stops Okabe-Ito)
`heatmap-divergent-low` ↔ `heatmap-divergent-high` (autour d'un neutre)

**Total** : ~33 tokens. Chaque palette (default, okabe-ito) doit fournir une valeur pour chacun.

---

## 4. Inventaire des color schemes existants (état des lieux)

Audit exhaustif (avril 2026) — **17 systèmes de coloration** identifiés, dont la majorité sont des variantes des 3 mêmes archétypes. Plusieurs **incohérences** documentées.

### 4.1 Échelles ordinales (mauvais → bon)

| # | Cas | Fichier:ligne | Tiers | Seuils | Statut |
|---|---|---|---|---|---|
| 1 | Performance score | `lib/perf-color.ts` | 5 | 80/65/50/35 | ✅ Centralisé |
| 2 | K/D ratio (HomePage) | `features/home/HomePage.tsx:801` | 3 | 1 / 0 | ❌ Inline (Tailwind) |
| 3 | K/D ratio (match-card) | `components/ui/match-card.tsx:349` | 3 | 1 / 0 | ❌ Inline (Tailwind) |
| 4 | K/D ratio (nemesis) | `match-view/MatchStatCards.tsx:157` | 2 | 1.0 | ❌ Inline (hex `#00DC82`/`#FF4B4B`) |
| 5 | Accuracy (%) | `features/home/HomePage.tsx:900-903` | 3 | 55 / 40 | ❌ Inline (Tailwind) |
| 6 | Win rate heatmap | `squad/charts/heatmapChart.ts:6-12` | 5 | 35/50/65/100 | ⚠️ Miroir de #1 réimplémenté |
| 7 | Progress gauge | `components/ui/rank-progress-gauge.tsx:23-28` | 4 | 25/50/75 | ⚠️ Local |
| 8 | Combat yield bar | `components/ui/combat-yield-bar.tsx` | 2 | p80 dynamique | ⚠️ Local, calcul ad hoc |
| 9 | Highlights value (PERF_HEX) | `features/home/HomePage.tsx:44-64` | 5 | identiques à #1 | ❌ **Recodage local de #1** |

**Constat critique** : K/D est codé **3 fois différemment** (2 Tailwind + 1 hex), et perf-color est **dupliqué** dans HomePage (`PERF_HEX`).

### 4.2 Échelles divergentes (négatif ↔ neutre ↔ positif)

| # | Cas | Fichier:ligne | Bande neutre | Couleurs | Statut |
|---|---|---|---|---|---|
| 10 | Delta MMR | `components/ui/match-card.tsx:418-434` | ±10 | green/blue/red | ❌ Inline |
| 11 | Delta CSR/LUSR | `match-view/MatchStatCards.tsx:105-109` | strict 0 | `#00DC82`/`#FF4B4B`/muted | ❌ Inline |
| 12 | Skill Δ | `components/ui/match-card.tsx:296` | strict 0 | `#22c55e`/`#ef4444` | ❌ Inline |
| 13 | Delta-card (générique) | `components/ui/delta-card.tsx:31` | strict 0 | `#00DC82`/muted/`#FF4B4B` | ✅ Helper local existe |
| 14 | Intensity heatmap K/D | `components/ui/timeseries-heatmap.tsx:71-75` | 0.5 médian | red/black/green | ⚠️ Gradient local |

**Constat critique** : 5 implémentations légèrement divergentes du **même pattern « signe → couleur »**, avec des bandes neutres incohérentes (±10, strict 0, médian).

### 4.3 Mappings catégoriels

| # | Cas | Fichier:ligne | Cardinalité | Statut |
|---|---|---|---|---|
| 15 | Outcome (win/loss/draw/dnf) | `lib/outcome-color.ts` | 4 | ✅ Centralisé |
| 16 | Narrative badges | `components/ui/match-card-presentation.ts:42-48` | 5 | ✅ Centralisé (mais réimporté à `career/CareerTopMatchesTable.tsx:18-24`) |
| 17 | Outcome bar segments (KPI home) | `features/home/HomePage.tsx:149-152` | 4 | ❌ Hex inline (mais cohérent avec #15) |
| — | KDA bars (K/A/D segments) | `components/ui/match-card.tsx:308-330` | 3 | ⚠️ Couleurs structurelles, pas une échelle |
| — | Citation progress (normal/mastered) | `components/ui/citation-progress-ring.tsx:26-27` | 2 | ⚠️ État binaire, pas une échelle |

### 4.4 Synthèse

- **3 archétypes** suffisent à couvrir 14 des 17 cas : ordinal, divergent, catégoriel.
- **Duplication actuelle** : ~40-50% du code couleur ordinal est réimplémenté localement.
- **Incohérences à corriger** : K/D (3 variantes), bande neutre divergente (±10 vs strict 0).
- **3 cas spécifiques** restent en dehors des 3 archétypes : (a) `combat-yield-bar` à seuils dynamiques p80, (b) `KDA bars` (segments structurels K/A/D, pas une échelle), (c) `citation-progress-ring` (état binaire normal/mastered). **Ils consommeront quand même la palette via tokens** (couches 1-4), juste pas les scales (couche 5).

---

## 5. Couche « Scales » — la vraie source de vérité

### 5.1 Trois constructeurs génériques

Fichier : `src/lib/accessibility/scales/index.ts`

```ts
// Ordinale — N tiers ordonnés, seuils décroissants
makeOrdinalScale<T extends number>(config: {
  tiers: SemanticToken[];      // ex: ['perf-tier-1', …, 'perf-tier-5']
  thresholds: number[];         // longueur = tiers.length - 1, décroissant
  // tiers[0] = meilleur ; value >= thresholds[0] → tiers[0]
}): (value: T) => SemanticToken;

// Divergente — autour d'une bande neutre
makeDivergentScale<T extends number>(config: {
  positive: SemanticToken;      // au-dessus de la bande
  neutral: SemanticToken;       // dans la bande
  negative: SemanticToken;      // en-dessous
  neutralBand: [number, number]; // ex: [-10, 10] ou [0, 0]
}): (value: T) => SemanticToken;

// Catégorielle — clé typée → token
makeCategoricalScale<K extends string>(
  map: Record<K, SemanticToken>
): (key: K) => SemanticToken;
```

Chaque scale retourne **un SemanticToken**, pas une couleur — le résolveur (couches 3-4) fait la traduction. Cela garantit qu'un changement de palette (default ↔ okabe-ito) est totalement transparent pour les scales.

### 5.2 Instances pré-configurées (singletons exportés)

Fichier : `src/lib/accessibility/scales/instances.ts`

```ts
export const perfScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-2', 'perf-tier-3', 'perf-tier-4', 'perf-tier-5'],
  thresholds: [80, 65, 50, 35],
});

export const accuracyScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-3', 'perf-tier-5'],
  thresholds: [55, 40],
});

export const kdScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-3', 'perf-tier-5'],
  thresholds: [1.0, 0.0], // décision §9.7 : >=1 / [0,1[ / <0
});

export const progressScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-2', 'perf-tier-4', 'perf-tier-5'],
  thresholds: [75, 50, 25],
});

export const mmrDeltaScale = makeDivergentScale({
  positive: 'divergent-pos',
  neutral: 'divergent-neutral',
  negative: 'divergent-neg',
  neutralBand: [-10, 10],
});

export const skillDeltaScale = makeDivergentScale({
  positive: 'divergent-pos',
  neutral: 'divergent-neutral',
  negative: 'divergent-neg',
  neutralBand: [0, 0], // strict
});

export const outcomeScale = makeCategoricalScale({
  win: 'outcome-win',
  loss: 'outcome-loss',
  draw: 'outcome-draw',
  dnf: 'outcome-dnf',
});

export const narrativeScale = makeCategoricalScale({
  dominant: 'narrative-dominant',
  humiliation: 'narrative-humiliation',
  remontada: 'narrative-remontada',
  debacle: 'narrative-debacle',
  contre_remontada: 'narrative-contre-remontada',
});
```

**Conséquences directes** :
- Changer le seuil "K/D bon" passe de 1.0 à 1.2 → **un seul edit** dans `instances.ts`.
- Harmoniser la bande neutre divergente (aujourd'hui ±10 / strict 0) → **une décision documentée**, plus de divergence accidentelle.
- Plus aucun composant n'écrit `if (kd > 1) return 'text-green-400'`.

### 5.3 Helper de consommation

Pour les composants React :

```ts
// src/lib/accessibility/useColor.ts
export function useColor(token: SemanticToken): string;
// retourne la couleur résolue via les CSS vars (live, réagit au changement de palette)

// Ou via l'API plus haut-niveau :
export function useScaleColor<V>(scale: (v: V) => SemanticToken, value: V): string;
```

Usage typique :
```tsx
const color = useScaleColor(perfScale, score);
return <span style={{ color }}>{score}</span>;
```

Pour Plotly (hors React) : `resolveToken(token)` synchrone qui lit la CSS var.

---

## 6. Plan en 8 phases

Chaque phase = **1 commit atomique**, testable et rollbackable. Branche unique : `feat/accessibility-okabe-ito`.

### Phase 1 — Fondation : couches 1 → 4 (sans toggle UI)
**Objectif** : poser l'infrastructure, sans changer le rendu visible.

Fichiers créés :
- `src/lib/accessibility/palettes/okabe-ito.ts` — 8 couleurs brutes
- `src/lib/accessibility/palettes/default.ts` — extraction des hex actuels (perf-color.ts, outcome-color.ts, etc.) en un seul endroit
- `src/lib/accessibility/semantic-tokens.ts` — type `SemanticToken` + record `Palette = Record<SemanticToken, string>`
- `src/lib/accessibility/applyPalette.ts` — fonction `applyPalette(palette: Palette): void` qui fait `document.documentElement.style.setProperty('--color-win', palette.win)` etc.
- `src/lib/accessibility/index.ts` — barrel export

CSS :
- Ajout dans `src/styles/globals.css` du **bloc de tokens sémantiques par défaut** (sous `:root`) — sert de fallback si JS désactivé.

Tests :
- `palettes/__tests__/coverage.test.ts` — chaque palette définit bien tous les `SemanticToken`.
- `applyPalette.test.ts` — vérifie l'écriture des CSS variables (jsdom).

**Aucun fichier existant n'est modifié à cette phase.** Risque = 0.

---

### Phase 2 — Couche scales (constructeurs + instances)
**Objectif** : poser les 3 archétypes (`makeOrdinalScale`, `makeDivergentScale`, `makeCategoricalScale`) et instancier les **8 scales nommées** identifiées dans §5.2 (`perfScale`, `accuracyScale`, `kdScale`, `progressScale`, `mmrDeltaScale`, `skillDeltaScale`, `outcomeScale`, `narrativeScale`).

Fichiers créés :
- `src/lib/accessibility/scales/makeOrdinalScale.ts`
- `src/lib/accessibility/scales/makeDivergentScale.ts`
- `src/lib/accessibility/scales/makeCategoricalScale.ts`
- `src/lib/accessibility/scales/instances.ts` — **source de vérité unique** des seuils
- `src/lib/accessibility/useColor.ts` + `resolveToken.ts` (helpers React + Plotly)

**Décisions appliquées** (cf. §9) :
- Bande neutre divergente : strict `[0, 0]` par défaut, override `[-10, 10]` pour `mmrDeltaScale`.
- Seuils K/D : 3 tiers `[1.0, 0.0]` (≥1 / [0,1[ / <0).

Tests :
- `scales/__tests__/makeOrdinalScale.test.ts` — bornes, tiers, valeurs aux seuils exacts.
- `scales/__tests__/makeDivergentScale.test.ts` — valeurs dans/hors bande neutre.
- `scales/__tests__/instances.test.ts` — snapshot des seuils (détecte tout drift accidentel).

**Aucun fichier de feature n'est modifié à cette phase.** Les composants existants continuent de fonctionner sur leur ancien code.

---

### Phase 3 — Migration des helpers JS existants vers les scales (couche 6, partie 1)
**Objectif** : faire converger tous les helpers actuels vers les scales de la phase 2.

Fichiers modifiés :
- `src/lib/perf-color.ts` → wrapper fin autour de `perfScale` + `resolveToken`
- `src/lib/outcome-color.ts` → wrapper autour de `outcomeScale`
- `src/components/ui/match-card-presentation.ts` → utilise `outcomeScale` + `narrativeScale`
- `src/components/ui/delta-card.tsx` → utilise `skillDeltaScale` (cas générique)

Migration des call-sites identifiés en §4 :
- `features/home/HomePage.tsx` :
  - `PERF_HEX` (l. 44-64) **supprimé**, remplacé par `perfScale`
  - K/D (l. 801) → `kdScale`
  - Accuracy (l. 900-903) → `accuracyScale`
  - Outcome bar segments (l. 149-152) → `outcomeScale`
- `components/ui/match-card.tsx` :
  - K/D (l. 349) → `kdScale`
  - Delta MMR (l. 418-434) → `mmrDeltaScale`
  - Skill Δ (l. 296) → `skillDeltaScale`
- `match-view/MatchStatCards.tsx` :
  - Delta CSR/LUSR (l. 105-109) → `mmrDeltaScale` ou `skillDeltaScale` (à trancher)
  - Nemesis K/D (l. 157) → `kdScale`
- `components/ui/rank-progress-gauge.tsx` (l. 23-28) → `progressScale`
- `features/career/CareerTopMatchesTable.tsx` (l. 18-24) → `narrativeScale`

Tests :
- Snapshot de chaque composant modifié — couleur résolue identique (en mode `default`).
- Test de cohérence : `kdScale(1.5)` retourne le même token, peu importe le call-site.

Garde-fou (eslint custom rule) :
- Règle `no-raw-hex-color` activée en **warning** (whitelist : `src/lib/accessibility/palettes/`).
- Règle `no-tailwind-color-class` (`text-(red|green|blue|amber|emerald|...)-[0-9]+`) en **warning** dans `features/` et `components/ui/` (whitelist : composants strictement structurels).

---

### Phase 4 — Migration des charts Plotly (couche 6, partie 2)
**Objectif** : tous les charts consomment des tokens via les helpers Plotly.

Helpers créés (extension de `plotlyColorscale.ts`) :
- `buildOrdinalColorscale(tokens: SemanticToken[])` → format Plotly `[[0, hex], [0.25, hex], …]`
- `getChartSeries(n: number)` → tableau de `n` couleurs résolues depuis `chart-series-1..n`
- Tous résolvent via `resolveToken()` au moment de la construction du layout.

Fichiers modifiés :
- `src/features/squad/charts/hsPkChart.ts` → `CHART_COLORS = getChartSeries(3)`
- `src/features/squad/charts/timelineChart.ts` → vérifier qu'il consomme bien `perfScale` après Phase 3 (devrait déjà être correct)
- `src/features/squad/charts/heatmapChart.ts` → `colorscale = buildOrdinalColorscale(['perf-tier-5', 'perf-tier-4', 'perf-tier-3', 'perf-tier-2', 'perf-tier-1'])`
- `src/components/ui/timeseries-heatmap.tsx` → 2 colorscales reconstruites (intensité + K/D divergent)
- Audit ciblé : tous les fichiers `**/charts/**/*.ts` + `**/timeseries/**/*.tsx`

**Réactivité au changement de palette** : Plotly ne consomme pas de CSS vars dans son layout (couleurs résolues à la construction). Pour réagir au toggle, le composant React doit **re-construire le layout** via une dépendance sur `colorPalette` du store (hook `useColorPaletteVersion()` à exposer).

Tests :
- Pour chaque chart : construire le layout avec palette `default` puis `okabe-ito`, vérifier que les hex finaux **changent bien** (et correspondent aux palettes).
- Snapshot du layout résolu (objet) pour chaque palette × chart.

---

### Phase 5 — Store & persistance (couche 3 réactive)
**Objectif** : brancher le toggle au store, sans UI encore.

Fichier modifié :
- `src/stores/settingsDraftStore.ts` :
  - Ajouter `colorPalette: 'default' | 'okabe-ito'` (défaut `'default'`)
  - Ajouter action `setColorPalette()`
  - Inclure dans le `partialize` de la persistance

Fichier modifié :
- `src/app/providers/theme-provider.tsx` : observe aussi `colorPalette` et appelle `applyPalette()` à chaque changement (et au mount).

**À noter** : la palette est **orthogonale** au theme dark/light. Les 4 combinaisons doivent fonctionner. La couche 4 (CSS vars sémantiques) est définie une fois ; les variables structurelles (`--background`, etc.) ne bougent pas.

Tests :
- Test d'intégration : changer `colorPalette` dans le store → vérifier les CSS vars sur `document.documentElement`.
- Vérifier qu'au reload, la valeur persistée est bien réappliquée avant le premier render (éviter un FOUC).

---

### Phase 6 — UI : section « Accessibilité » dans Paramètres
**Objectif** : exposer le toggle à l'utilisateur (cf. décision §9.1 : onglet dédié).

Fichier modifié :
- `src/features/settings/SettingsPage.tsx` :
  - Ajouter onglet `accessibility` (entre `general` et `sync`)
  - Composant `AccessibilityTab` (nouveau fichier `AccessibilityTab.tsx`)

Contenu de l'onglet :
1. **Palette de couleurs** (radio group) :
   - ☐ Par défaut
   - ☐ Okabe-Ito (adaptée aux daltonismes)
2. **Aperçu live** : une mini-grille montrant les 8 couleurs sémantiques principales (win/loss/perf-tiers) qui change instantanément selon la sélection.
3. **Lien doc** : bouton « En savoir plus » → ouvre une modale ou un lien vers `/help` avec une nouvelle entrée glossaire.

Anticipation : prévoir l'ajout futur de :
- Réduction des animations (`prefers-reduced-motion`)
- Augmentation contraste
- Taille de texte
→ structure l'onglet en sections pour qu'on puisse étendre sans refactor.

i18n :
- Étendre `src/features/settings/i18n.ts` avec les libellés FR/EN.

Tests :
- Test de rendu : onglet visible, radio fonctionnel, click → store mis à jour.

---

### Phase 7 — Audit de couverture par feature
**Objectif** : passer le lint en `error` et corriger toutes les violations résiduelles.

Méthode :
1. **Activer eslint `no-raw-hex-color` et `no-tailwind-color-class` en `error`** (whitelist : `src/lib/accessibility/palettes/` + composants strictement structurels listés explicitement).
2. Pour chaque erreur reportée par le lint : remplacer par un token / une scale, ou ajouter à la whitelist avec justification en commentaire.
3. Vérification croisée par grep manuel (filet de sécurité) :
   - `grep -rn '#[0-9a-fA-F]\{3,8\}' apps/web/src/features/ apps/web/src/components/`
   - `grep -rnE 'text-(red|green|blue|amber|emerald|cyan|orange|violet|purple)-[0-9]+' apps/web/src/features/ apps/web/src/components/`

Features à auditer en priorité (chart-heavy, identifiées en exploration) :
- `career` — progression charts, encounters, top matches
- `squad` — synergies, contributions, charts (déjà en phase 3 mais re-vérifier les sous-composants)
- `home` — KPI grid, hero banner, recent playlists
- `match-view` — scoreboard, badges
- `timeseries` — toutes les viz
- `synthesis` — relations preview
- `palmares` — saisons, season pass
- `leaderboard` — sorting visuel

Livrable : tableau de couverture (par feature, # hex restants → 0).

---

### Phase 8 — Tests visuels & QA daltoniens
**Objectif** : valider empiriquement.

1. **Outils automatisés** :
   - WebAIM Contrast Checker sur chaque combinaison fond/texte des tokens.
   - Plugin `eslint-plugin-jsx-a11y` (si pas déjà actif).
2. **Simulation** : Color Oracle (desktop) ou DevTools Chrome → Rendering → Emulate vision deficiencies (Protanopia, Deuteranopia, Tritanopia, Achromatopsia).
3. **Captures avant/après** sur 5 pages clés : Home, Career, Squad, Match View, Timeseries. Stockées dans `.ai/screenshots/okabe-ito/`.
4. **Test manuel utilisateur** : si possible, faire valider par une personne daltonienne.

Critères de succès :
- Aucun cas où Win/Loss sont indiscernables en deutéranopie.
- Échelle perf 5 niveaux reste ordonnée (lisible) sous chaque simulation.
- Heatmaps gardent un gradient lisible (combiner teinte + opacité).
- Contraste texte/fond ≥ 4.5:1 partout.

---

## 7. Structure de fichiers finale

```
apps/web/src/lib/accessibility/
├── index.ts                    # barrel
├── palettes/
│   ├── default.ts              # couleurs actuelles consolidées
│   └── okabe-ito.ts            # 8 couleurs + extensions ordinales
├── semantic-tokens.ts          # type SemanticToken + Palette
├── applyPalette.ts             # écrit les CSS vars sur :root
├── resolveToken.ts             # token → couleur résolue (lit CSS var)
├── useColor.ts                 # hooks React (useColor, useScaleColor)
├── plotlyColorscale.ts         # helpers Plotly (buildOrdinalColorscale, getChartSeries)
├── _logger.ts                  # logger namespacé 'accessibility'
├── scales/
│   ├── makeOrdinalScale.ts
│   ├── makeDivergentScale.ts
│   ├── makeCategoricalScale.ts
│   ├── instances.ts            # ★ source de vérité unique des seuils
│   └── __tests__/
│       ├── makeOrdinalScale.test.ts
│       ├── makeDivergentScale.test.ts
│       └── instances.test.ts
└── __tests__/
    ├── coverage.test.ts
    ├── applyPalette.test.ts
    └── plotlyColorscale.test.ts

apps/web/src/features/settings/
├── SettingsPage.tsx            # +onglet 'accessibility'
├── AccessibilityTab.tsx        # nouveau
└── i18n.ts                     # +libellés
```

---

## 8. Effort estimé

| Phase | Effort | Risque | Bloquant pour suivante |
|---|---|---|---|
| 1. Fondation (palettes + tokens + CSS vars) | S (½ j) | Très faible | Oui pour 2 |
| 2. Scales (constructeurs + instances) | S (½ j) | Faible | Oui pour 3 |
| 3. Migration helpers + call-sites | M (1–1½ j) | Moyen (volume) | Non |
| 4. Charts Plotly | M (1 j) | Moyen (snapshots) | Non |
| 5. Store + provider | S (¼ j) | Faible | Oui pour 6 |
| 6. UI Settings | S (½ j) | Faible | Non |
| 7. Audit features | S (½ j) | Faible (mécanique) | Non |
| 8. QA visuelle | M (½–1 j) | — | — |
| **Total** | **~5 j** | | |

> Note : l'effort total augmente d'~1 j vs estimation initiale, mais l'**économie est nette** côté maintenance — toute évolution future de seuil/couleur devient un edit unique au lieu de N edits dispersés.

---

## 9. Décisions arbitrées (validées 2026-04-25)

1. ✅ **Onglet dédié « Accessibilité »** dans Settings (pas une sous-section de General).
2. ✅ **Persistance local-only** (localStorage via `settingsDraftStore`, comme `theme`).
3. ✅ **Opt-in** (pas de détection auto).
4. ✅ **Default palette = baseline actuelle** ; Okabe-Ito reste un choix explicite.
5. ✅ **Lint `no-raw-hex-color` : warning en phase 3, error en phase 7**.
6. ✅ **Bande neutre divergente** : strict 0 par défaut, override `±10` justifié pour `mmrDeltaScale` uniquement (bruit MMR < 10 non significatif).
7. ✅ **Seuils K/D unifiés — 3 tiers** : `kd < 0` → tier-5, `0 ≤ kd < 1` → tier-3, `kd ≥ 1` → tier-1. Configuré comme `kdScale = makeOrdinalScale({ tiers: ['perf-tier-1', 'perf-tier-3', 'perf-tier-5'], thresholds: [1.0, 0.0] })`.

---

## 10. Stratégie de tests

**Objectif** : couverture maximale sur les couches stables (palettes, tokens, scales, resolver) + non-régression visuelle sur les composants migrés.

### 10.1 Tests unitaires (couches 1 → 5)

| Couche | Fichier | Tests obligatoires |
|---|---|---|
| 1. Palettes brutes | `palettes/__tests__/coverage.test.ts` | • Chaque palette (default, okabe-ito) définit **tous** les `SemanticToken` (zéro `undefined`)<br>• Chaque hex est un format valide `#RRGGBB`<br>• Aucun token n'est partagé entre 2 catégories sémantiques différentes (ex: `outcome-win` ≠ `success` même si valeur identique)<br>• Snapshot stable (détecte tout changement non-intentionnel) |
| 2. Tokens sémantiques | `__tests__/semantic-tokens.test.ts` | • Type `SemanticToken` exhaustif (test compile-time via `satisfies Record<SemanticToken, …>`) |
| 3. Resolver | `__tests__/applyPalette.test.ts` | • Écrit chaque token comme CSS var sur `:root` (jsdom)<br>• Idempotent : appel 2x = même état<br>• Cleanup : `applyPalette(default)` après `applyPalette(okabeIto)` rétablit l'état initial<br>• Pas de fuite : aucune CSS var hors namespace `--color-*` modifiée |
| 4. CSS vars | `__tests__/css-vars.test.ts` | • Toutes les vars listées en §3 sont définies dans `:root` (parsing CSS) |
| 5. Scales | `scales/__tests__/makeOrdinalScale.test.ts` | • Valeurs aux bornes exactes (égal au seuil = tier supérieur)<br>• Valeurs à `±epsilon` autour des seuils<br>• `NaN`, `±Infinity`, `null/undefined` → comportement défensif documenté<br>• Erreur explicite si `tiers.length !== thresholds.length + 1`<br>• Erreur explicite si `thresholds` non décroissants |
| 5. Scales | `scales/__tests__/makeDivergentScale.test.ts` | • Bande neutre inclusive aux bornes<br>• Cas strict `[0,0]` vs bande `[-10,10]`<br>• Cas dégénérés (bande inversée, `NaN`) |
| 5. Scales | `scales/__tests__/makeCategoricalScale.test.ts` | • Clé absente → erreur explicite (pas de fallback silencieux) |
| 5. Instances | `scales/__tests__/instances.test.ts` | • **Snapshot des seuils** (toute modif d'un seuil métier = revue obligée)<br>• Vérification croisée : `kdScale(1.0) === perfScale(80)` token attendu<br>• Cas concrets historiques (table de valeurs réelles → token attendu) |

### 10.2 Tests de non-régression sur la migration des call-sites

Pour chaque call-site listé en §4 et migré en Phase 3-4 :
- **Snapshot du composant en mode `default`** avant migration → après migration : **identique pixel-près** (couleur résolue inchangée).
- Test paramétré : `kdScale(value)` retourne le **même token** quel que soit le call-site (Home / match-card / nemesis).
- Test de cohérence palette : pour 5 valeurs métier réelles (perf=42, kd=1.3, mmrΔ=-15, …), vérifier qu'en mode `default` la couleur résolue est exactement celle de la baseline (table `__tests__/regression-baseline.ts`).

### 10.3 Tests d'intégration (Phases 5-6)

- Store : `setColorPalette('okabe-ito')` → `applyPalette` appelé → CSS vars mises à jour → re-render → composants reflètent la nouvelle couleur (test React Testing Library + jsdom).
- Persistance : reload simulé (re-mount du provider) → palette restaurée depuis localStorage avant le premier paint (pas de FOUC).
- UI Settings : `<AccessibilityTab>` clic radio → store muté → aperçu live mis à jour.

### 10.4 Tests d'accessibilité (Phase 8)

- **Contraste WCAG AA** automatisé : pour chaque combinaison token texte / token fond utilisée dans l'app, ratio ≥ 4.5:1 (Axe ou `wcag-contrast`).
- **Distinguabilité daltoniens** : test custom qui simule les 3 types (protanopie / deutéranopie / tritanopie) via matrices LMS et vérifie que les couleurs des tokens d'une même scale restent distinguables (`ΔE2000 ≥ seuil`).
- Snapshots Playwright sous chaque mode de simulation pour 5 pages clés.

### 10.5 Cibles de couverture

- **100 %** lignes/branches sur `src/lib/accessibility/**` (couches 1-5 — code pur, aucun excuse à un trou).
- **≥ 90 %** sur les helpers migrés (`perf-color.ts`, `outcome-color.ts`, `match-card-presentation.ts`, `delta-card.tsx`).
- Pas de seuil imposé sur les composants de feature (reposent sur les couches déjà couvertes).

### 10.6 CI

- Tests accessibilité bloquants (échec contraste = fail CI).
- Snapshots seuils bloquants (modification accidentelle d'un threshold métier = revue obligatoire).
- Lint `no-raw-hex-color` en error en Phase 7 → bloquant après merge phase 7.

---

## 11. Stratégie de logging

**Principe** : logger **les transitions d'état** (changements palette, application au boot, erreurs de résolution), pas les opérations par-frame (zéro log dans les hot paths comme `useColor` ou `resolveToken` appelés par render).

### 11.1 Couches concernées

| Couche | Événement à logger | Niveau | Quand |
|---|---|---|---|
| Resolver | `applyPalette('okabe-ito')` | `info` | Chaque application |
| Resolver | Token demandé inexistant dans la palette | `error` | À chaque résolution échouée (dédupliqué : 1× par token) |
| Resolver | Token résolu vers une chaîne vide ou invalide | `warn` | Idem |
| Store | Changement `colorPalette` (user action) | `info` | À chaque mutation, avec `from → to` |
| Store | Restauration depuis localStorage au boot | `info` | 1× au mount du provider |
| Store | Détection localStorage corrompu / migration de schéma | `warn` | Au boot uniquement |
| Scales | Valeur hors-bornes inattendue (`NaN`, `Infinity`) | `warn` | 1× par scale + valeur (dédupliqué) |
| Scales | Catégorielle : clé inconnue | `error` | À chaque appel (signal bug applicatif) |
| UI | Onglet Accessibilité ouvert | `debug` | Pour analytics si pertinent |
| UI | Aperçu live re-render | — | **NE PAS logger** (haute fréquence) |

### 11.2 Mécanisme

Utiliser le logger existant de l'app (vraisemblablement un wrapper console + niveau, à confirmer via `grep -rn "logger\|createLogger\|console\." apps/web/src/lib/`). Si aucun logger structuré n'existe, créer un logger dédié minimal :

```ts
// src/lib/accessibility/_logger.ts
const log = createLogger('accessibility'); // namespace
```

Le namespace `accessibility` permet de filtrer/désactiver via les prefs dev existantes.

### 11.3 Anti-patterns à proscrire

- ❌ `console.log` dans les hooks `useColor` / `useScaleColor` (appelés des centaines de fois par render).
- ❌ Logger la valeur résolue à chaque appel (volume + bruit).
- ❌ Logger sans namespace (devient noyé dans le reste de la console).
- ❌ Logger en `info` ce qui devrait être `debug` (pollue prod).

### 11.4 Tests du logging

- Vérifier qu'un changement de palette émet **exactement 1** log `info` (pas 0, pas 5).
- Vérifier qu'un token inexistant émet **1 seul** log par token sur toute la session (déduplication).
- Vérifier qu'aucun log n'est émis pendant 100 résolutions consécutives de tokens valides.
- Mocker le logger dans les tests pour assertions.

### 11.5 Télémétrie (optionnel, hors-scope MVP)

Si une couche analytics existe, instrumenter :
- Adoption du mode Okabe-Ito (% d'utilisateurs ayant activé)
- Bascules dans une session (signal d'hésitation UX)

À traiter dans un sprint ultérieur, pas dans ce plan.

---

## 12. Hors-scope (mais proches)

À ne PAS embarquer dans ce plan, mais à garder en tête pour des plans futurs :
- Mode haut-contraste (différent d'Okabe-Ito : il vise le contraste, pas la distinguabilité)
- Réduction mouvement (`prefers-reduced-motion`)
- Taille de texte / zoom dédié
- Lecteur d'écran / `aria-label` audit
- Navigation clavier complète

L'onglet « Accessibilité » créé en phase 6 sera leur point d'entrée naturel.

---

## 13. Checklist pré-merge

**Couverture code**
- [ ] `grep -rn '#[0-9a-fA-F]\{3,8\}' apps/web/src/features/ apps/web/src/components/` retourne 0 (hors assets SVG)
- [ ] `grep -rnE 'text-(red|green|blue|amber|emerald|cyan|orange|violet|purple)-[0-9]+' apps/web/src/features/ apps/web/src/components/` retourne 0 (hors composants structurels whitelistés)
- [ ] Aucun call-site identifié au §4 ne contient encore de couleur ou seuil inline
- [ ] Les 8 scales de §5.2 sont les **seuls** lieux où les seuils existent

**Tests** (cf. §10)
- [ ] Couverture **100 %** sur `src/lib/accessibility/**`
- [ ] Couverture **≥ 90 %** sur les helpers migrés
- [ ] Snapshot `instances.test.ts` vert (seuils stables et revus)
- [ ] Tests de non-régression visuelle verts sur tous les call-sites du §4
- [ ] Tests d'accessibilité (contraste WCAG AA + distinguabilité daltoniens) verts en CI
- [ ] Snapshots Playwright des 5 pages clés sous chaque mode de simulation

**Logging** (cf. §11)
- [ ] Logger namespace `accessibility` opérationnel
- [ ] 1 log `info` par changement de palette (vérifié par test)
- [ ] Déduplication des warnings/erreurs vérifiée
- [ ] Aucun log dans les hot paths (`useColor`, `resolveToken`)

**QA**
- [ ] Toutes les phases mergées
- [ ] Aperçu Plotly en preview manuelle vérifié
- [ ] Validation simulation Chrome DevTools (protanopie, deutéranopie, tritanopie)
- [ ] Captures avant/après sur 5 pages clés stockées dans `.ai/screenshots/okabe-ito/`

**Doc**
- [ ] `features/help/i18n.ts` : entrée glossaire utilisateur
- [ ] `.ai/thought_log.md` : entrée dédiée
- [ ] `.ai/project_map.md` : ajout du module `lib/accessibility/` (avec sous-dossier `scales/`)
- [ ] CLAUDE.md : ajouter une règle « Aucun hex/Tailwind color class hors `lib/accessibility/palettes/` »
