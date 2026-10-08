/**
 * paletteForTheme.ts — La palette effectivement appliquée pour un thème : la palette choisie,
 * et en thème sombre les jetons dont la valeur dépend du fond (rampes ordinales des rôles
 * d'impact, `_impactRoleColors.ts`). Les autres jetons gardent la même valeur dans les deux
 * thèmes.
 */
import { IMPACT_ROLE_COLORS_DARK } from './palettes/_impactRoleColors'
import type { Palette } from './semantic-tokens'

export type PaletteTheme = 'dark' | 'light'

export function paletteForTheme(palette: Palette, theme: PaletteTheme): Palette {
  return theme === 'dark' ? { ...palette, ...IMPACT_ROLE_COLORS_DARK } : palette
}
