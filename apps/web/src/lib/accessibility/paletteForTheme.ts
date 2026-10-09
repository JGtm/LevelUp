/**
 * paletteForTheme.ts — La palette effectivement appliquée pour un thème : la palette choisie,
 * et en thème sombre les jetons dont la valeur dépend du fond (rampes ordinales des rôles
 * d'impact, `_impactRoleColors.ts`). Les autres jetons gardent la même valeur dans les deux
 * thèmes.
 *
 * La rampe sombre est la jumelle de la rampe claire que la palette porte : chaque palette garde
 * ainsi ses propres teintes de gain / perte dans les deux thèmes. Une palette dont la rampe claire
 * n'appartient à aucune paire connue est rendue telle quelle (trace dédupliquée).
 */
import { log } from './_logger'
import { IMPACT_ROLE_RAMPS, type ImpactRoleRamp, type ImpactRoleToken } from './palettes/_impactRoleColors'
import type { Palette } from './semantic-tokens'

export type PaletteTheme = 'dark' | 'light'

function darkTwin(palette: Palette): ImpactRoleRamp | undefined {
  return IMPACT_ROLE_RAMPS.find(({ light }) =>
    (Object.keys(light) as ImpactRoleToken[]).every((t) => palette[t] === light[t]),
  )?.dark
}

export function paletteForTheme(palette: Palette, theme: PaletteTheme): Palette {
  if (theme !== 'dark') return palette
  const dark = darkTwin(palette)
  if (!dark) {
    log.warn(
      `paletteForTheme:no-dark-impact:${palette['impact-gain-1']}`,
      'rampe claire des rôles d’impact sans jumelle sombre — valeurs claires gardées en sombre',
    )
    return palette
  }
  return { ...palette, ...dark }
}
