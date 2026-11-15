/**
 * useSeriesColors — LES COULEURS DE SÉRIE du canvas : une par joueur (option « couleurs distinctes »)
 * et une par grande zone nommée. Extraites de `ReplayCanvas.tsx`, sans changement de comportement :
 * le fichier est au plafond de taille, et le panneau de joueur y ajoute son câblage.
 *
 * Les deux relisent les tokens à chaque version de palette (`paletteVersion`) : une couleur de
 * série est un token résolu en valeur concrète, qui change avec le thème et la palette.
 */
import { useMemo } from 'react'

import { getSeriesColors } from '@/lib/accessibility/plotlyColorscale'

import type { CalloutZoneReady } from '../../../lib/replay/calloutsPaint'
import { SERIES_TOKENS } from './replayCanvasConfig'

/**
 * Une série stable par joueur à la place de la couleur d'équipe. `count` est le nombre de joueurs
 * du roster quand l'option est active, `null` sinon (alors aucune série : `null`).
 */
export function useDistinctPlayerColors(count: number | null, paletteVersion: number): string[] | null {
  return useMemo(() => {
    void paletteVersion
    return count === null ? null : getSeriesColors(count, SERIES_TOKENS)
  }, [count, paletteVersion])
}

/** Une couleur de série PAR grande zone : la rotation de teinte du POC, en tokens. */
export function useZoneSeriesColors(zones: readonly CalloutZoneReady[], paletteVersion: number): string[] {
  return useMemo(() => {
    void paletteVersion
    const nBig = zones.reduce((n, z) => n + (z.big ? 1 : 0), 0)
    return nBig > 0 ? getSeriesColors(nBig, SERIES_TOKENS) : []
  }, [zones, paletteVersion])
}
