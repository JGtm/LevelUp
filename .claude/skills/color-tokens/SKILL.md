---
name: color-tokens
description: Système de couleurs sémantiques d'apps/web (jetons --ac-*, palettes, scales, famille stats de combat, lint anti-hex). À invoquer avant d'écrire ou de modifier toute couleur dans apps/web/src (features, components, lib/replay) ou un graphe ECharts.
---

# Skill : color-tokens — Système de couleurs sémantiques (apps/web)

## Règle fondamentale

**Aucun hex, aucune fonction de couleur CSS** (`oklch()`, `rgb()`, `hsl()`…) **ni classe Tailwind
de couleur** (`text-red-*`, `bg-green-*`, `bg-black`, `text-white`…) dans `apps/web/src/features/`,
`apps/web/src/components/` ni `apps/web/src/lib/replay/`.

Toute couleur sémantique passe par les helpers d'accessibilité.

## APIs selon le contexte

Import : `@/lib/accessibility` (barrel `apps/web/src/lib/accessibility/index.ts`).

| Contexte | Helper | Retour |
|---|---|---|
| JSX (className / style) | `tokenCssVar(token)` | `var(--ac-<token>)` |
| Hook React | `useColor(token)` | la même chaîne `var(--ac-<token>)`, sans état : la cascade CSS suit la palette |
| ECharts / canvas / SVG (hex résolu) | `resolveToken(token)` | hex lu sur `:root` |
| Série de N couleurs (cycle sur une liste de jetons) | `getSeriesColors(n, tokens[])` | `string[]` d'hex |
| Hex qui doit suivre un changement de palette | `useColorPaletteVersion()` + `resolveToken` | re-rendu au changement |
| Ton dérivé d'un jeton (même teinte, clarté décalée) | `tokenTone(color, step)` | couleur |
| Complément d'une couleur (barres négatives) | `hexComplement(hex)` | hex |

## Exemples

```tsx
// JSX — variable CSS
style={{ color: tokenCssVar('outcome-win') }}

// ECharts — valeur hex résolue
const color = resolveToken('chart-series-1');

// Série de N couleurs pour un chart
const colors = getSeriesColors(5, ['chart-series-1', 'chart-series-2', 'chart-series-3']);
```

## Fichiers source

```
apps/web/src/lib/accessibility/
  semantic-tokens.ts     — liste de tous les tokens disponibles (type SemanticToken, ALL_TOKENS)
  resolveToken.ts        — resolveToken()
  useColor.ts            — useColor(), useScaleColor()
  useColorPaletteVersion.ts — version de palette, pour les hex réactifs
  plotlyColorscale.ts    — getSeriesColors() (nom historique, sert ECharts)
  applyPalette.ts        — applyPalette()
  paletteForTheme.ts     — pose les jetons dépendants du thème (rôles d'impact)
  tokenTone.ts           — tokenTone()
  hexComplement.ts       — hexComplement()
  colorDistance.ts       — oklab, deltaE, simulateCvd
  palettes/              — default, okabe-ito (daltonisme), cividis, tol-bright, spartan,
                           _impactRoleColors, _encounterColors
  scales/                — makeCategoricalScale, makeOrdinalScale, makeDivergentScale,
                           instances (perfScale, outcomeScale…), fragClass
```

Les quatre palettes que l'utilisateur choisit sont dans `app/providers/palette-picker.ts`.
Les graphes suivent la palette (`components/charts/chartsFollowPalette.test.tsx`).

## Tokens sémantiques

**La liste complète est dans `semantic-tokens.ts`** (type `SemanticToken`, commentaires par
famille). Ne pas la recopier ici ni deviner un nom : `primary`, `secondary`, `danger`,
`series1` ou `neutral` ne sont PAS des jetons. Les noms exacts sont par exemple
`destructive`, `info`, `chart-series-1..8`, `divergent-neutral`, `zone-neutral`.

Familles à connaître : issues (`outcome-*`), paliers de performance (`perf-tier-*`),
divergents (`divergent-*`), équipes (`team-ally` / `team-enemy`), joueurs d'escouade
(`squad-player-1..4`), classes de frags (`frag-*`, `scales/fragClass.ts`), rôles (`range-role-*`,
`objective-role-*`, `impact-*`), ressources de carte (`resource-*`), heatmaps (`heatmap-*`, rampes
composées par `heatmapRampTokens` dans `components/charts/heatmapColors.ts`), accents génériques
(`legendary`, `extreme`, `bonus`), stats de combat (`stat-*`, `assist-*`, ci-dessous).

**Voile et texte sur voile** : `scrim` (voile noir, `bg-scrim/60`) et `on-scrim` (texte blanc
sur voile, `text-on-scrim/80`). Identiques dans les deux thèmes, définis dans
`styles/globals.css`. Ils remplacent `bg-black` / `text-white` (et variantes d'opacité) dans
`features/` et `components/` : le garde-rail `tools/lint-no-hardcoded-colors.mjs` les refuse.

## Stats de combat — famille dédiée (2026-09-17)

Toute couleur qui dit « c'est un frag / une mort / une assistance / ce sens d'assistance »
passe par ces cinq jetons, dans toutes les pages :

| Jeton | Rôle |
|---|---|
| `stat-kills` | frags |
| `stat-deaths` | morts |
| `stat-assists` | assistances (y compris le bonus assistances/3 et la marque du rejeu) |
| `assist-received` | un autre joueur t'assiste (« il te sert ») |
| `assist-given` | tu assistes un autre joueur (« tu le sers ») |

- **Part de dégâts d'une assistance** : jamais une autre teinte. Trois tons de la couleur du
  sens pour les tranches < 25 %, 25-50 %, > 50 % — trois CLARTÉS OKLCH du jeton, montant vers
  le premier plan du thème, PAS des opacités (remplacées le 2026-09-19 : elles noyaient les
  tons faibles dans la piste ; le jeton lui-même ne peut pas s'éclaircir, il est retenu par le
  contraste clair et la séparation deutéranopie) — `components/assists/assistTierTone.ts`
  (le calcul du ton est dans `lib/accessibility/tokenTone.ts`), bornes
  `domain.AssistTierLowMaxPct` / `AssistTierMidMaxPct` (`domain/relation_assists.go`).
- **Ne relèvent PAS de la famille** (garder la couleur de leur rôle) : couleur d'équipe par
  camp (`team-*`), couleur de joueur (`squad-player-*`), classe d'arme (`frag-*`), issue du
  match, échelle de qualité, accent de sous-type de frag (frags parfaits, tirs à la tête…).
- **Garde-fous** : `lib/accessibility/combatStatTokens.test.ts` (contraste + écarts, valeurs
  par palette) et `lib/accessibility/combatStatNoBorrow.guard.test.ts` (ratchet : un jeton
  emprunté sur une ligne qui nomme une stat échoue, sauf usage justifié dans `KEPT`).
- **Distance de couleur** dans un test : importer `deltaE` / `simulateCvd` depuis
  `lib/accessibility/colorDistance.ts` (copie de la matrice OKLab interdite par ratchet).

## Rôles d'impact — seuls jetons qui dépendent du THÈME (2026-10-08)

`impact-gain-1..4` et `impact-loss-1..3` : rampes ordinales des rôles d'impact (graphe « Points
d'impact par soirée et par rôle », Escouade › Contributions). Pas 1 = plus fort barème, contre
l'axe, le plus contrasté sur la carte. Valeurs dans `palettes/_impactRoleColors.ts`, une paire
de rampes (claire + jumelle sombre) PAR PALETTE, dans les teintes de ses `divergent-pos` /
`divergent-neg` : vert / rouge (défaut), bleu / vermillon (Okabe-Ito, Cividis), bleu / rouge Tol
(Tol Bright). La palette porte la rampe claire ; `paletteForTheme` (ThemeProvider) pose la jumelle
sombre, c'est le seul mécanisme de jeton thème-dépendant : un nouveau jeton qui en a besoin s'y
ajoute, jamais par un `isDark` dans un graphe. Garde-fous : `lib/accessibility/impactRoleTokens.test.ts`
(par palette et par thème : ordre des contrastes, pas pâle ≥ 2:1, voisins ΔE ≥ 15 en vision normale
et sous daltonisme, teinte) et `app/providers/theme-provider-impact.test.tsx` (valeurs écrites sur
`:root` au basculement).

## Fond « Relief » et cerne des pions — trois jetons

- `map-relief-base` : teinte de base de la rampe du fond Relief, UNE valeur par palette et la même dans les
  deux thèmes (Graphite `#6B7078` par défaut, Violet `#7559B5` Okabe-Ito et Tol Bright, Vert `#3F8C4F`
  Cividis). La rampe de clarté s'en tire par `lib/accessibility/reliefRamp.ts` (`reliefRampHex`) : pas de
  jeton par palier.
- `marker-outline-band` (bande sombre) et `marker-outline-rim` (filet clair) : le cerne deux tons des
  pions du rejeu (`features/match-replay/layers/markerOutline.ts`), mêmes valeurs partout
  (`palettes/_mapMarkerOutline.ts`, étalées dans chaque palette). Le noyau du pion garde la couleur d'équipe.
- Garde-fou : `lib/accessibility/mapReliefTokens.test.ts` (par palette, thème et vision : paliers voisins de la
  rampe à ΔE ≥ 6, meilleur des deux tons du cerne à 3:1 contre la rampe du Relief, le fond Plan, le fond Rendu
  et la carte du thème).

## Exceptions tolérées

Une exception est un commentaire `color-allow(<genre>[, <genre>]): <raison>` sur la MÊME ligne.
Les genres sont `hex`, `tailwind`, `fonction-couleur`. Le marqueur n'excuse que les genres
qu'il nomme. Un marqueur posé sur une ligne sans couleur à excuser est une violation
`marqueur-inerte` (plafond 0, jamais excusable).

Fichiers tolérés en entier (`tools/lint-no-hardcoded-colors.lib.mjs`, `ALLOWED_FILES`) :
`lib/accessibility/palettes/`, `features/palmares/rarity.ts` (couleurs de rareté Halo),
les tests, `e2e/` et les fichiers générés.

Cas tolérés aujourd'hui avec marqueur : le rose du like (`features/media`), l'ambre
d'état ou de gravité (`features/ascension`, `components/ui/match-card.tsx`). Garde :
`lib/accessibility/lintNoHardcodedColors.guard.test.ts`.

## Outils de vérification

```bash
# Le lint de référence (features/, components/, lib/replay/)
cd apps/web && npm run lint:colors     # = node ../../tools/lint-no-hardcoded-colors.mjs
```
