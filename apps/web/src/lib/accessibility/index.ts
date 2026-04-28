/**
 * index.ts — Barrel export du module accessibilité.
 *
 * Les composants importent depuis ce barrel :
 *   import { useScaleColor, tokenCssVar } from '@/lib/accessibility'
 *
 * Règle : n'importer depuis les sous-modules directs (palettes/, scales/)
 * que si on a une raison explicite. Les composants n'ont jamais besoin des
 * palettes brutes.
 */

// Tokens sémantiques — contrat central
export { tokenCssVar, tokenVar, ALL_TOKENS } from './semantic-tokens'
export type { SemanticToken, Palette } from './semantic-tokens'

// Application de palette (usage : ThemeProvider)
export { applyPalette } from './applyPalette'

// Résolution synchrone (usage : Plotly layouts, canvas)
export { resolveToken } from './resolveToken'

// Hooks React (usage : composants JSX)
export { useColor, useScaleColor } from './useColor'
export { useColorPaletteVersion } from './useColorPaletteVersion'

// Helper série-colors (usage : chart builders ECharts)
export { getSeriesColors } from './plotlyColorscale'

// Palettes brutes (usage : ThemeProvider uniquement)
export { defaultPalette } from './palettes/default'
export { okabePalette } from './palettes/okabe-ito'
