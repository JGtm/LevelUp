/**
 * useReliefInks — LES DEUX ENCRES DU FOND RELIEF, résolues par jeton pour la palette active : la teinte de base
 * de sa rampe (`map-relief-base`, une valeur par palette) et le filet clair du cerne des pions
 * (`marker-outline-rim`), qui sert de liseré à la zone jouable (blanc dans toutes les palettes : il se lit contre le bord teinté du relief
 * comme contre le vide au-delà). Elles sont relues quand la palette change.
 */
import { useMemo } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'

import type { ReliefViewInks } from './useReliefView'

export function useReliefInks(): ReliefViewInks {
  const paletteVersion = useColorPaletteVersion()
  return useMemo(() => {
    void paletteVersion
    return { base: resolveToken('map-relief-base'), outline: resolveToken('marker-outline-rim') }
  }, [paletteVersion])
}
