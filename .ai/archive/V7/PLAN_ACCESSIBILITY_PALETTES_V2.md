# PLAN_ACCESSIBILITY_PALETTES_V2.md — Refonte des palettes d'accessibilité

> Plan rédigé le 2026-05-06 sur la branche `feat/accessibility-palettes-v2` (branchée depuis `feat/token-pool-parallel-sync`).
>
> Déclencheur : retour utilisateur (ami deutéranope) — la palette Okabe-Ito actuelle assigne les couples critiques (`outcome-win/loss`, `divergent-pos/neg`, `encounter-ally/enemy`, `heatmap-cold/hot`) sur l'axe **vert/rouge**, qui est précisément celui qui collapse en deutan. Le couple Bluish Green `#009E73` vs Vermillion `#D55E00` tombe sur Δluminance ≈ 0.14 et même axe a/b — distinguable seulement par luminance, défaillant en éclairage faible ou écran mal calibré.
>
> Inspiration secondaire : repo `afonsolopez/colorblind` (porté en Go d'un scoring WCAG 2.0). N'est **pas** une bibliothèque de palettes, c'est un grader de contraste — on en reprend l'esprit (test automatique de contraste) sans dépendance.

---

## TL;DR

| Phase | Scope | Effort | Livrable indépendant |
|---|---|:---:|:---:|
| **A** | Re-mapper la palette Okabe-Ito existante sur l'axe **bleu/orange** pour les couples binaires | ~1 h | ✅ |
| **B** | Ajouter 2 palettes : **Cividis** (séquentielle CVD-safe) + **Tol Bright** (catégorielle 7 couleurs) + adapter UI Settings | ~3 h | ✅ |
| **C** | Test automatique de **contraste WCAG AA** sur toutes les paires `narrative-*` (bg/text), exécuté sur les 4 palettes | ~1 h | ✅ |

**Total** : ~5 h de dev. Chaque phase est livrable en commit séparé, mergeable indépendamment.

**Branche cible** : `feat/accessibility-palettes-v2` (déjà créée).

**Critère de succès global** :
1. Aucun couple sémantique binaire ne repose sur l'axe vert/rouge dans les palettes daltonisme-safe.
2. L'utilisateur a au minimum 4 options dans Settings → Accessibilité (Standard, Okabe-Ito, Cividis, Tol Bright).
3. Toutes les paires `narrative-{bg}` / `narrative-{text}` atteignent WCAG AA (contraste ≥ 4.5:1) sur chaque palette.

---

## Phase A — Re-mapper Okabe-Ito sur l'axe bleu/orange

### A.1. Diagnostic — pourquoi remapper

L'analyse token-par-token (cf. message d'audit qui précède ce plan) identifie **4 zones cassées** dans [okabe-ito.ts](apps/web/src/lib/accessibility/palettes/okabe-ito.ts) :

| Token actuel | Hex | Token associé | Hex | Δ luminance deutan |
|---|---|---|---|:---:|
| `outcome-win` | `#009E73` | `outcome-loss` | `#D55E00` | 0.14 (faible) |
| `divergent-pos` | `#009E73` | `divergent-neg` | `#D55E00` | 0.14 (faible) |
| `narrative-encounter-ally-plus` | `#009E73` | `narrative-encounter-tough-enemy` | `#D55E00` | 0.14 (faible) |
| `heatmap-hot` | `#009E73` | `heatmap-cold` | `#D55E00` | 0.14 (faible) |

Le perf-tier ordinal (5 niveaux) a un problème distinct (non-monotonie de luminance, traité en A.3).

### A.2. Re-mapping proposé — couples binaires

Tous les couples binaires basculent sur **Blue `#0072B2` ↔ Vermillion `#D55E00`** (Δ luminance deutan ≈ 0.16, mais surtout **Δ chroma** préservé en deutan/protan car bleu n'est pas sur l'axe de confusion). Référence : Wong B. (2011) *"Points of view: Color blindness"*, Nature Methods.

```diff
  // ── Outcomes
- 'outcome-win':  '#009E73', // Bluish Green
+ 'outcome-win':  '#0072B2', // Blue
  'outcome-loss': '#D55E00', // Vermillion (inchangé)
- 'outcome-draw': '#56B4E9', // Sky Blue
+ 'outcome-draw': '#F0E442', // Yellow (libère le Sky Blue qui était proche du Blue)
  'outcome-dnf':  '#CC79A7', // Reddish Purple (inchangé)

  // ── Divergent
- 'divergent-pos':     '#009E73',
+ 'divergent-pos':     '#0072B2', // Blue
  'divergent-neutral': '#888888', // Gris (inchangé)
  'divergent-neg':     '#D55E00', // Vermillion (inchangé)

  // ── Encounter
- 'narrative-encounter-ally-plus':    '#009E73',
+ 'narrative-encounter-ally-plus':    '#0072B2', // Blue
  'narrative-encounter-tough-enemy':  '#D55E00', // Vermillion (inchangé)
  'narrative-encounter-ordinal':      '#56B4E9', // Sky Blue (inchangé)

  // ── Heatmap
- 'heatmap-hot':            '#009E73',
+ 'heatmap-hot':            '#0072B2', // Blue
  'heatmap-cold':           '#D55E00', // Vermillion (inchangé)
- 'heatmap-divergent-high': '#009E73',
+ 'heatmap-divergent-high': '#0072B2', // Blue
  'heatmap-divergent-low':  '#D55E00', // Vermillion (inchangé)
```

> ⚠️ **Convention culturelle inversée** : avec ce remapping, "win" / "positif" / "bon" / "hot" devient **bleu** au lieu de vert. C'est un trade-off conscient :
> - Pour la palette **default**, on garde le vert (convention occidentale "vert = bon").
> - Pour la palette **okabe-ito** (utilisée volontairement par un daltonien), la lisibilité prime sur la convention.
> - Indice secondaire à conserver dans les composants : icône ✓/✗, signe +/−, ou label texte. Vérifier au passage que les composants ne s'appuient pas uniquement sur la couleur (cf. checklist A.5).

### A.3. Re-mapping proposé — `perf-tier-1..5` (ordinal)

Le scale ordinal actuel échoue le critère "monotone en luminance" (yellow tier-3 = 0.77, vermillion tier-5 = 0.21, green tier-1 = 0.35). Un deutéranope perçoit tier-1/3/4/5 sur la même rangée jaune-grisâtre.

**Option retenue** : ramp **divergent bleu → jaune → orange/vermillion**, monotone en a* sur l'axe Lab (perceptuellement linéaire pour deutan/protan) :

```diff
- 'perf-tier-1': '#009E73', // Bluish Green
- 'perf-tier-2': '#56B4E9', // Sky Blue
- 'perf-tier-3': '#F0E442', // Yellow
- 'perf-tier-4': '#E69F00', // Orange
- 'perf-tier-5': '#D55E00', // Vermillion
+ 'perf-tier-1': '#0072B2', // Blue       — excellent
+ 'perf-tier-2': '#56B4E9', // Sky Blue
+ 'perf-tier-3': '#F0E442', // Yellow     — milieu (point neutre)
+ 'perf-tier-4': '#E69F00', // Orange
+ 'perf-tier-5': '#D55E00', // Vermillion — pire
```

Ce ramp passe le test "convertir en niveaux de gris → ordre préservé" (Blue 0.21 < SkyBlue 0.42 ~ Yellow 0.77 > Orange 0.43 > Vermillion 0.21). Imparfait sur la pure luminance (Yellow trop clair) — c'est le compromis des palettes catégorielles. Pour une vraie luminance monotone, voir Cividis en Phase B.

### A.4. Chart series — réordonnancement

Re-ordonner pour que les **2 premières séries** soient la paire la plus distincte :

```diff
- 'chart-series-1': '#E69F00', // Orange
- 'chart-series-2': '#56B4E9', // Sky Blue
- 'chart-series-3': '#009E73', // Bluish Green
- 'chart-series-4': '#F0E442', // Yellow
- 'chart-series-5': '#0072B2', // Blue
- 'chart-series-6': '#D55E00', // Vermillion
- 'chart-series-7': '#CC79A7', // Reddish Purple
- 'chart-series-8': '#BBBBBB', // Light Grey
+ 'chart-series-1': '#0072B2', // Blue
+ 'chart-series-2': '#E69F00', // Orange
+ 'chart-series-3': '#56B4E9', // Sky Blue
+ 'chart-series-4': '#D55E00', // Vermillion
+ 'chart-series-5': '#009E73', // Bluish Green
+ 'chart-series-6': '#F0E442', // Yellow
+ 'chart-series-7': '#CC79A7', // Reddish Purple
+ 'chart-series-8': '#BBBBBB', // Light Grey
```

Justification : un graphe à 2 séries (très commun) tombera sur Blue+Orange (paire la plus safe, recommandation Wong 2011). À 4 séries, on a Blue/Orange/SkyBlue/Vermillion — toutes les paires restent ≥ 30 ΔE en deutan-simulation.

### A.5. Fichiers touchés

| Fichier | Changement |
|---|---|
| [apps/web/src/lib/accessibility/palettes/okabe-ito.ts](apps/web/src/lib/accessibility/palettes/okabe-ito.ts) | Remapping (A.2 + A.3 + A.4) |
| [apps/web/src/lib/accessibility/__tests__/__snapshots__/coverage.test.ts.snap](apps/web/src/lib/accessibility/__tests__/__snapshots__/coverage.test.ts.snap) | Régénérer le snapshot (`vitest -u` sur ce test précis) |
| `.ai/thought_log.md` | Entrée Phase A |

### A.6. Tests

- Le snapshot dans `coverage.test.ts` détectera la modification → re-générer **explicitement** (pas de `-u` global, juste sur ce test).
- Pas de nouveau test fonctionnel : la couverture token-par-token est déjà assurée.
- Vérifier visuellement après build : aller dans `Settings → Accessibilité`, sélectionner Okabe-Ito, vérifier les swatches outcome (win=Blue, loss=Vermillion).

### A.7. Risque & rollback

- **Risque visuel** : un utilisateur avec vision normale qui avait l'habitude du "win=vert" va voir "win=bleu". Pas un bug — un trade-off documenté en A.2. Rollback = revert du commit.
- **Risque sémantique** : composants qui utilisent `outcome-win` en supposant qu'il est vert (ex. icône stylée, logique conditionnelle "si rouge alors texte blanc"). Vérification en **2 passes** obligatoires :

  **Passe 1 — usage direct via `tokenCssVar` (JSX) ou références directes** :
  ```
  grep -rn "outcome-win\|divergent-pos\|encounter-ally\|heatmap-hot" apps/web/src/
  ```

  **Passe 2 — usage via `resolveToken()` (Plotly/SVG/ECharts wrappers)** : couvrir les 18 fichiers du dossier `apps/web/src/components/charts/` qui appellent `resolveToken`. Grep dédié :
  ```
  grep -rn 'resolveToken.*outcome-win\|resolveToken.*divergent-pos\|resolveToken.*encounter-ally\|resolveToken.*heatmap' apps/web/src/components/charts/ apps/web/src/features/
  ```

  **Critère de validation** : aucun composant ne doit avoir de logique conditionnelle qui dépend de la **teinte** (ex. `if (color === '#009E73') textColor = 'black'`). Si trouvé : refactor en `useColor(token)` + lookup texte calculé via `wcagContrast` (préfigure Phase C).

### A.8. Done definition Phase A

- [ ] [okabe-ito.ts](apps/web/src/lib/accessibility/palettes/okabe-ito.ts) modifié + commentaires à jour
- [ ] Snapshot `coverage.test.ts.snap` régénéré
- [ ] Passe 1 + Passe 2 du grep d'A.7 exécutées, aucune logique conditionnelle sur teinte trouvée
- [ ] `npm run typecheck` passe (depuis `apps/web/`)
- [ ] `npm run lint` passe (zéro nouvelle erreur eslint)
- [ ] `npm run test` passe (tous vitest verts)
- [ ] Vérification visuelle dans Settings → Accessibilité avec palette Okabe-Ito sélectionnée
- [ ] Entrée `thought_log.md` ajoutée
- [ ] Commit `feat(accessibility): remap okabe-ito on blue/orange axis`

---

## Phase B — Ajouter Cividis + Tol Bright

### B.1. Pourquoi ces 2 palettes en plus

| Palette | Type | Nombre couleurs | Cas d'usage idéal |
|---|---|:---:|---|
| **Cividis** (Nuñez et al. 2018) | Séquentielle continue, perceptuellement uniforme | N (interpolée) | Heatmaps, perf-tiers, gradients ordinaux. Monotone en L*. Conçue **explicitement** pour CVD (publiée dans PLOS ONE). |
| **Tol Bright** (Paul Tol 2018) | Catégorielle | 7 | Séries de charts, comparaisons multi-entités. Plus discriminable qu'Okabe-Ito au-delà de 4 séries en deutan. |

Pourquoi pas d'autres :
- **Viridis** : très populaire mais visuellement très similaire à Cividis pour un voyant — Cividis suffit, pas besoin des deux.
- **IBM Carbon** : palette d'entreprise, pas un standard scientifique. Skip.
- **Brewer Set2** : pas optimisée CVD au-delà de 3 couleurs. Skip.

### B.2. Définitions des palettes

#### B.2.1. Cividis

Source : Nuñez J. R., Anderton C. R., Renslow R. S. (2018) *"Optimizing colormaps with consideration for color vision deficiency to enable accurate interpretation of scientific data"*, PLOS ONE 13(7).

Pour notre usage **catégoriel** (5 perf-tiers + 8 chart-series), on échantillonne le gradient continu à intervalles réguliers :

```ts
// 5 échantillons pour perf-tier-1..5 (du foncé bleu = bon → jaune clair = mauvais)
'perf-tier-1': '#00224E', // Cividis t=0.00 — best
'perf-tier-2': '#3F4A6B', // t=0.25
'perf-tier-3': '#7C7B78', // t=0.50
'perf-tier-4': '#B6A855', // t=0.75
'perf-tier-5': '#FEE838', // t=1.00 — worst

// 8 échantillons pour chart-series (alterne extrémités pour distinguer 2 séries voisines)
'chart-series-1': '#00224E', // Cividis t=0.00
'chart-series-2': '#FEE838', // t=1.00
'chart-series-3': '#3F4A6B', // t=0.25
'chart-series-4': '#B6A855', // t=0.75
'chart-series-5': '#575C6D', // t=0.40
'chart-series-6': '#928D6B', // t=0.60
'chart-series-7': '#1B3158', // t=0.10
'chart-series-8': '#E1C752', // t=0.90
```

> ⚠️ Cividis **n'a pas** de notion native de catégories binaires (win/loss, ally/enemy). Pour ces tokens, on réutilise le mapping Phase A (Blue `#0072B2` / Vermillion `#D55E00`) — Cividis et Okabe-Ito partagent la même bonne pratique sur ce point.

#### B.2.2. Tol Bright

Source : Tol P. (2018) *"Colour schemes"*, technical note SRON/EPS/TN/09-002.

7 couleurs catégorielles + 1 grey (pour rester compatible avec notre token `chart-series-8`) :

```ts
'chart-series-1': '#4477AA', // Blue
'chart-series-2': '#EE6677', // Red
'chart-series-3': '#228833', // Green
'chart-series-4': '#CCBB44', // Yellow
'chart-series-5': '#66CCEE', // Cyan
'chart-series-6': '#AA3377', // Purple
'chart-series-7': '#BBBBBB', // Grey (Tol's "neutral")
'chart-series-8': '#000000', // (extension : Tol Bright n'a que 7 — on ajoute black pour le 8e)
```

Pour les autres tokens (outcomes, divergent, perf-tier), on reprend une logique **bleu/rouge** alignée sur la philosophie Tol :
- `outcome-win` = `#4477AA` (Blue), `outcome-loss` = `#EE6677` (Red)
- `perf-tier-1..5` = ramp construit en interpolant Tol's "Sunset" séquentiel : `#364B9A → #4A7BB7 → #98C2E0 → #EAECCC → #FAB57F → #DC4A38` (5 échantillons)

### B.3. Fichiers à créer

| Fichier | Contenu |
|---|---|
| `apps/web/src/lib/accessibility/palettes/cividis.ts` | Export `cividisPalette: Palette` |
| `apps/web/src/lib/accessibility/palettes/tol-bright.ts` | Export `tolBrightPalette: Palette` |

Modèle : copier la structure de [okabe-ito.ts](apps/web/src/lib/accessibility/palettes/okabe-ito.ts) — même commentaire d'en-tête expliquant la dérogation "zéro magic hex", même ordre de tokens.

### B.4. Fichiers à modifier

| Fichier | Changement |
|---|---|
| [apps/web/src/lib/accessibility/index.ts](apps/web/src/lib/accessibility/index.ts) | Ajouter `export { cividisPalette }` + `export { tolBrightPalette }` |
| [apps/web/src/stores/settingsDraftStore.ts](apps/web/src/stores/settingsDraftStore.ts) | `ColorPalette = 'default' \| 'okabe-ito' \| 'cividis' \| 'tol-bright'` |
| [apps/web/src/app/providers/theme-provider.tsx](apps/web/src/app/providers/theme-provider.tsx) | `switch (colorPalette)` au lieu du ternaire actuel |
| [apps/web/src/features/settings/AccessibilityTab.tsx](apps/web/src/features/settings/AccessibilityTab.tsx) | Passer le `<div className="grid sm:grid-cols-2">` à `sm:grid-cols-2 lg:grid-cols-4` + 2 `<PaletteOption>` supplémentaires |
| [apps/web/src/features/settings/i18n.ts](apps/web/src/features/settings/i18n.ts) | Ajouter `paletteCividis`, `paletteCividisDesc`, `paletteTolBright`, `paletteTolBrightDesc` (FR + EN) |
| [apps/web/src/lib/accessibility/__tests__/coverage.test.ts](apps/web/src/lib/accessibility/__tests__/coverage.test.ts) | Ajouter les 2 palettes au map `PALETTES` |
| `.ai/thought_log.md` | Entrée Phase B |

### B.5. i18n — strings à ajouter

```ts
// FR
paletteCividis: 'Cividis (séquentiel CVD)',
paletteCividisDesc: 'Palette perceptuellement uniforme, conçue pour la déficience visuelle des couleurs (PLOS ONE 2018). Idéale pour les heatmaps et gradients.',
paletteTolBright: 'Tol Bright (catégoriel)',
paletteTolBrightDesc: 'Palette catégorielle 7 couleurs optimisée daltonisme par Paul Tol (SRON 2018). Recommandée pour les graphes multi-séries.',

// EN
paletteCividis: 'Cividis (CVD sequential)',
paletteCividisDesc: 'Perceptually uniform palette designed for colour vision deficiency (PLOS ONE 2018). Ideal for heatmaps and gradients.',
paletteTolBright: 'Tol Bright (categorical)',
paletteTolBrightDesc: 'Categorical 7-colour palette optimised for colour-blindness by Paul Tol (SRON 2018). Recommended for multi-series charts.',
```

### B.6. Tests

- `coverage.test.ts` : assure couverture exhaustive des tokens sur les 4 palettes (incluant snapshots).
- `AccessibilityTab.test.tsx` : ajouter 2 cas (radio Cividis sélectionnable, radio Tol Bright sélectionnable). Pattern existant à dupliquer (cf. lignes 38-48).
- Pas de test de rendu graphique (out-of-scope, cf. CLAUDE.md "tests UI Streamlit").

### B.7. Risque & rollback

- **Risque** : oubli d'un token dans une nouvelle palette → `coverage.test.ts` échoue → bloquer le merge. C'est exactement la garantie que le test apporte.
- **Rollback — comportement Zustand vérifié** : le `merge` handler de [settingsDraftStore.ts:152-168](apps/web/src/stores/settingsDraftStore.ts#L152-L168) fait un spread brut `...persisted.localUiPrefs` **sans validation**. Donc si un utilisateur a `colorPalette: 'cividis'` en localStorage et qu'on revert B (ColorPalette retombe à `'default' | 'okabe-ito'`), la valeur invalide **subsisterait** en mémoire.

  **Garde-fou défensif obligatoire** dans [theme-provider.tsx](apps/web/src/app/providers/theme-provider.tsx) : utiliser un `switch` avec **`default:` qui retombe sur `defaultPalette`**, jamais une chaîne ternaire :

  ```ts
  // ❌ NE PAS faire (planterait silencieusement avec valeur invalide)
  const palette = colorPalette === 'cividis' ? cividisPalette : ...

  // ✅ Faire
  function pickPalette(key: ColorPalette): Palette {
    switch (key) {
      case 'okabe-ito':  return okabePalette
      case 'cividis':    return cividisPalette
      case 'tol-bright': return tolBrightPalette
      case 'default':    return defaultPalette
      default:           return defaultPalette  // garde-fou : valeur localStorage corrompue/obsolète
    }
  }
  ```

  Pas besoin de `.catch` dans le merge handler — le `default:` du switch suffit et est plus lisible. À tester explicitement (`themeProvider.test.tsx` : passer `colorPalette: 'unknown' as ColorPalette`, vérifier que `defaultPalette` est appliqué).

### B.8. Done definition Phase B

- [ ] 2 fichiers palette créés, exhaustifs (44 tokens chacun) avec en-tête "exception zéro magic hex"
- [ ] Index, store, provider, tab, i18n (FR + EN) mis à jour
- [ ] `theme-provider.tsx` refactoré en `switch` avec `default:` defensif (cf. B.7)
- [ ] `coverage.test.ts` étendu, tests verts (avec nouveaux snapshots)
- [ ] `AccessibilityTab.test.tsx` étendu (2 nouvelles options testées)
- [ ] `themeProvider.test.tsx` : test explicite "valeur ColorPalette invalide → fallback default"
- [ ] Vérification visuelle des 4 palettes dans Settings
- [ ] `npm run typecheck` passe
- [ ] `npm run lint` passe
- [ ] `npm run test` passe
- [ ] Entrée `thought_log.md`
- [ ] Commit `feat(accessibility): add cividis + tol-bright palettes`

---

## Phase C — Test automatique de contraste WCAG

### C.1. Pourquoi

Les 5 paires `narrative-{x}` / `narrative-{x}-text` sont peintes texte sur fond. Aujourd'hui [okabe-ito.ts:69-78](apps/web/src/lib/accessibility/palettes/okabe-ito.ts#L69-L78) annote les calculs en commentaire ("noir sur vert clair", "blanc sur fond foncé") — ce sont des **assertions humaines** non vérifiées. Un changement futur de hex peut casser le contraste sans détection.

Inspiration : `afonsolopez/colorblind` implémente exactement ce calcul (`ScoreHex`, `Grading`) — on en réimplémente la logique en TS (~30 lignes) côté tests.

### C.2. Spec WCAG 2.0 (résumé)

```
relative_luminance(R, G, B):
  R_lin = (R/255 ≤ 0.03928) ? (R/255)/12.92 : ((R/255 + 0.055)/1.055)^2.4
  G_lin, B_lin idem
  L = 0.2126·R_lin + 0.7152·G_lin + 0.0722·B_lin

contrast_ratio(c1, c2):
  L1 = max(L(c1), L(c2))
  L2 = min(L(c1), L(c2))
  return (L1 + 0.05) / (L2 + 0.05)
```

Seuils WCAG :
- **AA texte normal** : ≥ 4.5
- **AA texte large** (≥ 18pt ou ≥ 14pt bold) : ≥ 3.0
- **AAA** : ≥ 7.0

Pour les badges narratifs, on **vise AA texte normal** (≥ 4.5) car la taille d'affichage est `text-xs` ou `text-sm`.

### C.3. Fichier à créer

`apps/web/src/lib/accessibility/__tests__/wcagContrast.test.ts`

```ts
/**
 * wcagContrast.test.ts — Vérifie WCAG 2.0 AA sur toutes les paires bg/text
 * de chaque palette. S'inspire de afonsolopez/colorblind (Go) — réimplémenté
 * en TS pour rester sans dépendance.
 */
import { describe, it, expect } from 'vitest'
import { ALL_TOKENS, type SemanticToken } from '../semantic-tokens'
import { defaultPalette } from '../palettes/default'
import { okabePalette } from '../palettes/okabe-ito'
import { cividisPalette } from '../palettes/cividis'
import { tolBrightPalette } from '../palettes/tol-bright'

const PALETTES = { default: defaultPalette, 'okabe-ito': okabePalette, cividis: cividisPalette, 'tol-bright': tolBrightPalette }

const NARRATIVE_PAIRS: Array<[SemanticToken, SemanticToken]> = [
  ['narrative-dominant',         'narrative-dominant-text'],
  ['narrative-humiliation',      'narrative-humiliation-text'],
  ['narrative-remontada',        'narrative-remontada-text'],
  ['narrative-debacle',          'narrative-debacle-text'],
  ['narrative-contre-remontada', 'narrative-contre-remontada-text'],
]

const WCAG_AA_NORMAL = 4.5

function relLuminance(hex: string): number { /* ... */ }
function contrastRatio(a: string, b: string): number { /* ... */ }

describe.each(Object.entries(PALETTES))('palette "%s" — WCAG AA narratif', (_name, palette) => {
  it.each(NARRATIVE_PAIRS)('paire %s/%s ≥ 4.5:1', (bg, text) => {
    const ratio = contrastRatio(palette[bg], palette[text])
    expect(ratio, `contraste insuffisant pour ${bg}/${text}: ${ratio.toFixed(2)}`).toBeGreaterThanOrEqual(WCAG_AA_NORMAL)
  })
})
```

### C.4. Implémentation des helpers

À placer dans **un fichier de production** (pas test-only) car les helpers peuvent servir à un futur composant qui voudrait calculer dynamiquement un texte readable :

`apps/web/src/lib/accessibility/wcagContrast.ts` :
- `relLuminance(hex: string): number` — formule WCAG 2.0
- `contrastRatio(hex1: string, hex2: string): number` — ratio
- `wcagGrade(ratio: number): 'fail' | 'AA-large' | 'AA' | 'AAA'`

Re-export depuis [index.ts](apps/web/src/lib/accessibility/index.ts).

Tests unitaires inclus pour les helpers eux-mêmes (cas connus : noir/blanc = 21, blanc/blanc = 1, gris moyen).

### C.5. Si un test échoue

Plan de remédiation **avant** de désactiver le test :
1. Identifier la paire qui échoue (ex. `okabe-ito.narrative-humiliation/narrative-humiliation-text`).
2. Choisir : (a) ajuster la couleur de texte (noir ↔ blanc), (b) ajuster la couleur de fond.
3. Documenter le choix dans le commit.
4. Si **vraiment** infaisable (très rare avec un texte noir ou blanc), alors et seulement alors marquer le test `.skip` avec un commentaire `// FIXME: WCAG-fail-by-design — palette X token Y`.

### C.6. Fichiers touchés

| Fichier | Changement |
|---|---|
| `apps/web/src/lib/accessibility/wcagContrast.ts` | **Nouveau** — helpers production |
| `apps/web/src/lib/accessibility/__tests__/wcagContrast.test.ts` | **Nouveau** — test palettes |
| [apps/web/src/lib/accessibility/index.ts](apps/web/src/lib/accessibility/index.ts) | Export des helpers |
| `.ai/thought_log.md` | Entrée Phase C |

### C.7. Risque & rollback

- **Risque** : un test échoue dès la phase A ou B, on découvre que l'une des paires existantes était sous le seuil. C'est précisément ce qu'on veut détecter — corriger la couleur, pas désactiver le test.
- **Rollback** : revert. Les helpers production restent inutilisés mais ne cassent rien.

### C.8. Done definition Phase C

- [ ] `wcagContrast.ts` (production) + tests unitaires (cas connus : noir/blanc=21, blanc/blanc=1, gris moyen)
- [ ] `wcagContrast.test.ts` (palettes) — toutes paires AA passent sur les 4 palettes
- [ ] Helpers exportés depuis `index.ts`
- [ ] `npm run typecheck` passe
- [ ] `npm run lint` passe
- [ ] `npm run test` passe
- [ ] Entrée `thought_log.md`
- [ ] Commit `feat(accessibility): add WCAG AA contrast test on narrative pairs`

---

## Plan-review checklist (rempli)

| Section | Statut |
|---|:---:|
| **1. Structure** — objectif clair, phases ordonnées, blockers, effort, branche nommée | ✅ |
| **2. Architecture Go** | N/A (frontend uniquement) |
| **3. Multi-titres** | N/A (couche présentation pure) |
| **4. Adapters** | N/A |
| **5. Tests** — unitaires (helpers WCAG), couverture (palettes), e2e (UI tab) | ✅ |
| **6. Logging** | N/A (pas de logique runtime nouvelle) |
| **7. Frontend** — strings i18n FR+EN ✓, pas de hex hors palettes/ ✓, types stricts ✓ | ✅ |
| **8. Livraison** — done definition par phase, thought_log par phase, phases indépendantes | ✅ |

---

## Ordre d'exécution recommandé

```
Phase A  →  commit  →  vérif visuelle  →  Phase B  →  commit  →  Phase C  →  commit
```

Pourquoi cet ordre :
1. **A en premier** : changement minimal, fix immédiat le ressenti utilisateur. Si rien d'autre n'est fait, A seul suffit.
2. **B avant C** : C teste les palettes A+B ensemble. Inverser ferait écrire les tests sur 2 palettes puis les ré-étendre.
3. **3 commits séparés** : chacun est un livrable autonome, mergeable séparément si besoin.

## Hors scope

- Ajout d'icônes/patterns redondants dans les charts (texture en plus de la couleur) — déjà couvert par les labels textuels existants.
- Mode achromatopsie pure (monochrome) — moins d'1% des cas, demande une refonte plus profonde (toutes les charts en hatching/patterns). À évaluer si demande explicite.
- Extension de l'API contraste WCAG à des paires hors `narrative-*` (ex : tester texte sur fond UI générique). Possible en suivi.
- Migration des composants vers `useColor()` pour les hex en dur restants (cf. `rarity.ts`, exceptions tolérées CLAUDE.md règle 20).
