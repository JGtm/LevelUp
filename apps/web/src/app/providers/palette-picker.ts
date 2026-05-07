/**
 * palette-picker.ts — Sélection défensive de palette à partir d'une clé.
 *
 * Sorti de theme-provider.tsx pour respecter `react-refresh/only-export-components`
 * (un fichier .tsx ne doit exporter que des composants).
 *
 * Le merge handler de Zustand persist (settingsDraftStore) ne valide pas les
 * valeurs persistées en localStorage. Une valeur `colorPalette` invalide
 * (ex. après rollback d'une palette) doit retomber sur defaultPalette via le
 * `default:` du switch, pas planter.
 */
import {
  cividisPalette,
  defaultPalette,
  okabePalette,
  tolBrightPalette,
  type Palette,
} from '@/lib/accessibility'
import { log } from '@/lib/accessibility/_logger'
import type { ColorPalette } from '@/stores/settingsDraftStore'

export function pickPalette(key: ColorPalette): Palette {
  switch (key) {
    case 'okabe-ito':
      return okabePalette
    case 'cividis':
      return cividisPalette
    case 'tol-bright':
      return tolBrightPalette
    case 'default':
      return defaultPalette
    default:
      // Trace dédupliquée : valeur localStorage corrompue/obsolète (ex. après
      // rollback d'une palette). Fallback silencieux côté UX, traçable côté dev.
      log.warn(
        `pickPalette:unknown:${String(key)}`,
        `palette inconnue "${String(key)}" — fallback sur defaultPalette`,
      )
      return defaultPalette
  }
}
