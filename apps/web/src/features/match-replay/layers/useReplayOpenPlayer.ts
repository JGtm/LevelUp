/**
 * useReplayOpenPlayer — LE PION CERCLÉ du joueur dont la fiche est ouverte : la relecture de sa position et
 * les deux encres de l'anneau (`openPlayerRing.ts`), résolues une fois par palette.
 *
 * `undefined` quand aucune fiche n'est ouverte, ou quand le joueur est MASQUÉ (son pion n'est pas à la
 * carte : un anneau sans pion le désignerait quand même). SEUL le laisse cerclé : il est le seul pion.
 * L'anneau est clair (`--foreground`) et son liseré est à l'encre du fond (`--background`) : deux encres de
 * mise en page du thème, lues par `readInk`, jamais un littéral ; l'export les lit dans le thème sombre.
 */
import { useMemo } from 'react'

import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'

import { useExportActive } from '../export/exportLayoutStore'
import type { PlayerFocus } from '../model/playerPanelLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { readInk } from './canvasInk'
import { openPlayerPosAt, type OpenPlayerRing } from './openPlayerRing'

export function useReplayOpenPlayer(doc: ReplayDocumentReady, focus: PlayerFocus | null): OpenPlayerRing | undefined {
  const paletteVersion = useColorPaletteVersion()
  const exporting = useExportActive()
  const player = focus && !focus.hide ? focus.player : null
  return useMemo(() => {
    void paletteVersion
    void exporting
    if (!player) return undefined
    return { posAt: openPlayerPosAt(doc, player), ink: readInk('--foreground'), outline: readInk('--background') }
  }, [doc, player, paletteVersion, exporting])
}
