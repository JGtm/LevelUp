# Axe 5 — Color tokens & charts

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : apps/web/src/{features,components,lib/accessibility}/

## Synthèse

Le système de tokens (`apps/web/src/lib/accessibility/`) est propre, complet et bien testé : 40 SemanticToken, 2 palettes (default + okabe-ito), scales ordinales/divergentes/catégorielles avec snapshot test, helpers `tokenCssVar` / `resolveToken` / `getSeriesColors` correctement consommés par les 11 wrappers ECharts (`components/charts/*.tsx`). Les builders inline majeurs (`SquadEngagementView`, `TimeseriesKdaBars`, `TimeseriesCombatYield`, `SynthesisPage.buildBipolaireOption`) passent tous par `resolveToken`, sans aucun hex direct côté ECharts. Les fuites résiduelles sont localisées : un panel hex partagé `#1d2328` (3 sites), des fallbacks de couleur API (`?? '#a78bfa'`, `?? '#4B5563'`, `?? '#fff'`), 22 classes Tailwind colorées dispersées dans des features non strictement métier (notifications, watcher, media liked, daily/weekly badge), et la quasi-totalité du fichier `rank-progress-gauge.tsx` qui hardcode noir/gris SVG. Surtout, **la règle §20 n'est enforcée par aucun linter ni test** : le commentaire `eslint-disable-next-line no-restricted-syntax` dans `prestige.ts` cible une règle qui n'est pas configurée dans `eslint.config.js`. Le système est solide en design ; sa garde-fou automatique manque.

## Compteurs

- Hex `#RRGGBB` dans `features/` : **8** occurrences dans **6** fichiers
- Hex `#RRGGBB` dans `components/` : **14** occurrences dans **5** fichiers (dont 4 en commentaire dans `rank-progress-gauge.tsx`)
- Hex hors exceptions justifiées (rareté, structurel SVG documenté, fallback API) : **~10**
- Tailwind couleurs (`text-red-*`, etc.) dans `features/` : **74** occurrences dans **18** fichiers
- Tailwind couleurs dans `components/` : **9** occurrences dans **6** fichiers
- Tailwind hors exceptions (rareté `palettes/rarity.ts`, badges UI génériques liked/star/notif) : **~25-30**
- ECharts `color: '#...'` direct dans builders/wrappers : **0**
- Tokens définis (`semantic-tokens.ts`) : 40 — outcomes (4), perf-tier (5), divergent (3), statuts UI (4), compare (2), chart-series (8), narrative + text (10), heatmap (4)
- Tokens non utilisés détectés : `heatmap-divergent-low`, `heatmap-divergent-high` ne sont consommés par aucune scale et aucun composant — uniquement définis dans les palettes
- Plotly résiduel : **0** import `react-plotly` ; références sont des commentaires d'historique de migration

## Constats

### [BLOQUANT] Aucun linter ni test n'enforce la règle §20

`apps/web/eslint.config.js` ne configure que `@levelup/no-hardcoded-strings` (lignes 8-37), pas de règle `no-restricted-syntax` ni de plugin Tailwind couleur. Pourtant `apps/web/src/lib/prestige.ts:49,51,53,55` contient `// eslint-disable-next-line no-restricted-syntax -- couleurs identitaires Prestige` qui anticipe une règle inexistante. Aucun test Vitest ne scanne le filesystem pour détecter `#[0-9a-fA-F]{6}` ou les classes Tailwind couleur dans `features/components`. La doc `.claude/skills/color-tokens/SKILL.md:60-67` ne propose qu'un grep manuel. Conséquence : toute nouvelle régression est invisible.

### [DETTE] Panel hex `#1d2328` répété sur 3 composants core

- `apps/web/src/components/ui/match-card.tsx:125` : `bg-[#1d2328]`
- `apps/web/src/components/ui/delta-card.tsx:54` : `bg-[#1d2328]`
- `apps/web/src/features/match-history/MatchHistoryTable.tsx:144` : `bg-[#1d2328]`
- `apps/web/src/features/match-view/PlayerDetailPanel.tsx:36` : `bg-[#151a1f]/80` (variante)

C'est un design token de surface (panel sombre) utilisé sur les 3 cartes principales de l'app. Aucun token sémantique correspondant. Il devrait soit utiliser la variable `--card` (`globals.css:29 --card: oklch(0.205 0 0)`) via `bg-card`, soit avoir un nouveau token `surface-panel` injecté dans la palette.

### [DETTE] Fallbacks hex en hardcoded sur les couleurs servies par l'API

- `apps/web/src/features/match-view/PlayerDetailPanel.tsx:104` : `style={{ backgroundColor: c.color ?? '#4B5563' }}` (fallback gris pour citations)
- `apps/web/src/features/citations/CitationsPage.tsx:136` : `borderLeftColor: c.color ?? '#a78bfa'`
- `apps/web/src/features/career/CareerCitationsTab.tsx:108` : `borderLeftColor: c.color ?? '#a78bfa'` (duplicat exact du précédent)
- `apps/web/src/features/match-view/MatchViewPage.tsx:373,569` : `color: '#fff'` (texte des badges citations)

Le fallback `#a78bfa` est dupliqué dans 2 fichiers ; `#4B5563` n'a pas de cohérence avec le reste. Devraient être `tokenCssVar('chart-series-3')` / `tokenCssVar('outcome-draw')` ou un nouveau token `citation-default`. Le `color: '#fff'` est un cas d'usage récurrent (texte sur fond coloré dynamique) qui mériterait un helper `getContrastTextColor(bgColor)` ou un token `text-on-color`.

### [DETTE] `rank-progress-gauge.tsx` hardcode noir/gris SVG hors exception structurelle claire

`apps/web/src/components/ui/rank-progress-gauge.tsx`:
- ligne 62 : `stroke="#2d3748"` (piste de fond)
- ligne 96 : `fill="#e2e8f0"` (texte titre)

La piste de fond peut tomber sous l'exception "couleur structurelle SVG", mais le fill du texte est sémantique (couleur de typographie) et devrait être `var(--foreground)` ou `tokenCssVar('chart-series-1')`. Aucun commentaire justificatif. À noter : la palette dynamique elle-même est correctement gérée via `progressScale(pct * 100)` et `tokenCssVar` (lignes 23-29) — exemple à suivre.

### [DETTE] Classes Tailwind sky/amber dans badges Solo/Escouade et liked

- `apps/web/src/components/ui/match-card.tsx:246-247` : `'rgba(56,189,248,0.15)', color: '#38bdf8'` (Escouade) / `'rgba(168,85,247,0.15)', color: '#a855f7'` (Solo) — hex + rgba
- `apps/web/src/components/ui/match-card.tsx:501,505` : `text-sky-400`, `text-yellow-400` (citations sur card)
- `apps/web/src/features/home/HomeChallengesList.tsx:57,59` : `border-amber-500/25 bg-amber-500/8` / `border-sky-500/25 bg-sky-500/8` (daily/weekly)
- `apps/web/src/features/timeseries/TimeseriesCorrelationScatter.tsx:128-129` : `bg-cyan-500/20 text-cyan-300 ring-cyan-500/40`

Solo/Escouade et daily/weekly sont des distinctions métier qui mériteraient des tokens dédiés (`squad-with-friends`, `squad-solo`, `cadence-daily`, `cadence-weekly`). Couleurs sky/cyan et amber/yellow sont aujourd'hui assumées arbitrairement sans correspondre à `chart-series-1..8`.

### [DETTE] `cover-flow-modal.tsx` utilise `bg-green-500/20` et `bg-red-500/20`

`apps/web/src/components/ui/cover-flow-modal.tsx:307-308`:
```
? 'bg-green-500/20 border-green-500/50 text-green-400'
: 'bg-red-500/20 border-red-500/50 text-red-400'
```

C'est un toggle auto-chain (état actif/inactif) — pas une exception "liked/warning". Devrait utiliser `success` / `destructive` tokens (déjà dans la palette).

### [DETTE] WatcherCard.tsx — 7 occurrences `text-green-*` et `text-amber-*`

`apps/web/src/features/settings/WatcherCard.tsx:62, 71, 112, 171, 193, 204, 213` : `text-green-600 dark:text-green-400`, `text-amber-600`, `bg-green-500`. Composant entier sur une logique success/warning/info qui devrait s'aligner sur `success` / `warning` / `info` tokens (cohérence avec le reste). Tolérable comme exception "badge d'état système" mais 7 occurrences éparpillées sans commentaire = dette.

### [DETTE] Tokens heatmap-divergent-* non utilisés

Les tokens `heatmap-divergent-low` / `heatmap-divergent-high` (`semantic-tokens.ts:68-69`) sont définis dans les 2 palettes mais aucune scale ni aucun composant ne les consomme (commentaire palette `default.ts:67-68 // K/D bas/haut` mais la K/D scale utilise `perf-tier-1/3/5`). Soit retirer, soit créer une scale `kdHeatmapScale` qui les utilise.

### [AMÉLIORATION] `index.css` ancienne configuration legacy non utilisée

`apps/web/src/index.css:1-122` définit une CSS complète old-school (palette `--text`, `--bg`, `--accent: #aa3bff`, etc.) sans préfixe `--ac-`. Ne semble plus chargée par l'app (l'entrée canonique est `src/styles/globals.css`). À supprimer si confirmé inutile pour clarifier la source de vérité (`globals.css` est la seule source utilisée, lignes 189-238 fallback `--ac-*`).

### [AMÉLIORATION] `index.css:7,67-68` couleurs `#aa3bff`, `#f8fafc` inutilisées si fichier déprécié

Voir constat précédent. Si le fichier est encore servi, ces hex tombent sous l'exception "structurel layout SVG" mais devraient être documentés.

### [AMÉLIORATION] `_utils.ts` couleurs `rgba(255,255,255,...)` inline dans tooltip/grid

`apps/web/src/components/charts/_utils.ts:18-20` :
```
export const GRID_COLOR = 'rgba(255,255,255,0.06)'
export const TEXT_COLOR = 'rgba(255,255,255,0.45)'
export const ZERO_LINE = 'rgba(255,255,255,0.15)'
```

Et le tooltipBase (ligne 32) : `backgroundColor: 'rgba(20,24,30,0.92)'`. Centralisés (bon point) mais pas exposés comme tokens — assument un thème sombre uniquement, aucun support du toggle dark/light. Si le toggle `:root[data-theme='light']` (`globals.css:79`) est actif, les charts ECharts auraient des grilles invisibles. À convertir en tokens `chart-grid` / `chart-text-muted` / `chart-tooltip-bg` ou en `var(--muted-foreground)` etc.

### [AMÉLIORATION] Commentaire de fallback pour les hex API

Plusieurs sites passent `c.color` (servi par l'API Go, axe 1) sans pouvoir savoir si la couleur est elle-même un token. Idéalement, l'API doit retourner un `colorToken: SemanticToken` au lieu d'un hex (`apps/web/src/components/PlayerChips.tsx:100` montre le bon pattern : `accent = tokenCssVar(player.colorToken)`). Constat partiel hors-axe (couvert axe 1).

## Cartographie : flux d'une couleur (ex: outcome-win)

```
1. Source de vérité — palette
   apps/web/src/lib/accessibility/palettes/default.ts:23
     'outcome-win': '#10B981'
   apps/web/src/lib/accessibility/palettes/okabe-ito.ts:35
     'outcome-win': '#009E73'

2. Injection en CSS variable
   applyPalette() → document.documentElement.style.setProperty('--ac-outcome-win', value)
   Fallback statique : globals.css:198 --ac-outcome-win: #10B981

3a. Côté JSX — tokenCssVar
   apps/web/src/lib/outcome-color.ts:10
     OUTCOME_COLORS.win = tokenCssVar('outcome-win') = 'var(--ac-outcome-win)'
   Consommé via style={{ color: ... }} ou className arbitraire

3b. Côté ECharts — resolveToken
   apps/web/src/components/charts/_utils.ts:50
     outcomeColor('win') → resolveToken('outcome-win')
     → getComputedStyle(:root).getPropertyValue('--ac-outcome-win').trim()
     → '#10B981' (hex résolu, requis car ECharts ne supporte pas var())

3c. Cas séries N — getSeriesColors
   apps/web/src/lib/accessibility/plotlyColorscale.ts:11
     getSeriesColors(n, ['chart-series-1','chart-series-2',...])
     → cycle modulo + resolveToken pour chaque

4. Rendu
   - JSX : navigateur résout la cascade CSS, réagit dynamiquement à applyPalette()
   - ECharts : valeur figée au moment du build d'option (re-render requis sur changement palette,
     géré par useColorPaletteVersion()).
```

Le contrat est solide et bidirectionnel (CSS-first pour JSX, hex-résolu pour canvas). Le seul maillon faible est l'absence de garde-fou automatique pour empêcher de bypasser ce flux.

## Constats hors-axe

- **Axe 1 (déjà couvert)** : les fallbacks `c.color ?? '#fff'` reflètent que l'API Go sert toujours des hex au lieu de tokens. Cf. axe 1.
- **Axe 4 (front React)** : `index.css` legacy à confirmer mort et supprimer.
- **Axe 4 (typage)** : `MatchNarrativeSection.test.tsx:38 outcome_color: '#22c55e'` — fixture de test, pas de problème en soi mais le contrat domain expose `outcome_color: string` sans union/branding ; pourrait gagner à être typé `outcome_color: SemanticToken | HexString`.

## Suivi recommandé

1. **Ajouter une garde-fou automatique** (ESLint custom rule ou test Vitest qui scanne `apps/web/src/{features,components}/` à la recherche de `/#[0-9a-fA-F]{6}/` et des regex Tailwind couleur, avec allowlist explicite par fichier). Activer en `error` sur la CI. Modèle : `eslint-rules/no-hardcoded-strings.js` existe déjà comme template.
2. **Créer un token `surface-panel`** (ou réutiliser `--card`) pour remplacer les 3 occurrences de `bg-[#1d2328]` et la variante `bg-[#151a1f]/80` ; centraliser la couleur de fond des cards stats.
3. **Convertir les `*_COLOR` rgba/hex de `_utils.ts`** en tokens (`chart-grid`, `chart-text-muted`, `chart-tooltip-bg`) pour permettre un vrai toggle dark/light côté ECharts.
4. **Nettoyer les tokens morts** (`heatmap-divergent-low`/`high`) ou créer la scale qui les consomme.
5. **Remplacer les fallbacks `c.color ??` par des tokens** (`citation-default`, `text-on-color`) pour gagner en cohérence et préparer un futur passage de l'API à des `colorToken: SemanticToken`.
