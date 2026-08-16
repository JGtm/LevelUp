/**
 * useReplayHeatmap — tout ce qu'il faut savoir pour peindre la carte de chaleur, cuit une
 * fois : la grille, la rampe du thème, et la lecture réellement servie.
 *
 * DÉLIBÉRÉMENT HORS DU COMPOSANT (même règle que `useReplaySettings` / `useReplaySound`) :
 * ReplayCanvas ne doit ni savoir d'où viennent les positions de mort, ni décider quelle
 * rampe le thème impose, ni arbitrer une préférence devenue impossible. Il reçoit un calque
 * prêt à cuire et le pose ; toute la logique se teste ici, sans canvas.
 */
import { useMemo } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { ReplayBounds } from '@/lib/api/types'

import { buildHeatmap, heatRamp, type HeatGrid, type HeatmapMode } from './heatmapLayer'
import type { KillFxEntry } from './killFx'
import type { XY } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

export interface ReplayHeatmap {
  /** La grille cuite — null quand le calque est éteint OU que rien n'est mesurable. */
  grid: HeatGrid | null
  /** Les paliers `rgba()` du thème. Vide = le thème ne donne pas la rampe : on ne peint pas. */
  ramp: string[]
  /**
   * La lecture EFFECTIVEMENT servie. Elle retombe sur la présence quand aucune mort du
   * match n'a pu être localisée : une préférence héritée d'un autre match ne doit pas
   * laisser l'utilisateur devant un calque vide sans explication.
   */
  mode: HeatmapMode
  /** Vrai si la lecture « éliminations » a de quoi se dessiner (le tiroir s'y adosse). */
  killsAvailable: boolean
}

export function useReplayHeatmap(
  doc: ReplayDocumentReady,
  bounds: ReplayBounds,
  killFx: KillFxEntry[],
  settings: { show: boolean; mode: HeatmapMode },
): ReplayHeatmap {
  // LES LIEUX DE MORT, relus une fois : la carte des éliminations compte les victimes là
  // où elles sont TOMBÉES. `deathX` existe dès que la victime est localisée, même sans
  // tueur — contrairement à `vx`/`vy`, qui n'orientent l'effet qu'en couple complet.
  const deaths = useMemo(() => {
    const out: XY[] = []
    for (const e of killFx) {
      if (e.deathX !== null && e.deathY !== null) out.push({ x: e.deathX, y: e.deathY })
    }
    return out
  }, [killFx])

  const killsAvailable = deaths.length > 0
  const mode: HeatmapMode = killsAvailable ? settings.mode : 'presence'

  // Cuite UNE fois par document et par lecture (patron `buildShotFx`), et seulement quand
  // le calque est allumé : c'est le calcul le plus lourd de la page (mesuré à 7 ms sur une
  // arène, 19 ms sur une carte BTB, pour 72 000 positions).
  const grid = useMemo(
    () => (settings.show ? buildHeatmap(doc, bounds, mode, deaths) : null),
    [settings.show, doc, bounds, mode, deaths],
  )

  // La rampe est précalculée PAR THÈME, jamais par cellule : 64 paliers indexés au dessin.
  //
  // RAMPE DE FRÉQUENCE, ET C'EST UN CHOIX. La grandeur mesurée est une intensité NEUTRE
  // (du temps passé, un nombre de morts), pas une performance : la rampe « à connotation »
  // du dépôt va du rouge au vert et dirait donc « bon » du couloir le plus meurtrier. Elle
  // est de surcroît iso-luminante en vision daltonienne, là où celle-ci reste monotone en
  // luminance dans TOUTES les palettes (cf. heatmapColors.ts, qui tranche cette question).
  const paletteVersion = useColorPaletteVersion()
  const ramp = useMemo(() => {
    void paletteVersion // re-résoudre au changement de thème (resolveToken lit le DOM)
    const [low, high] = heatmapRampTokens('frequency').map(resolveToken)
    return heatRamp(low ?? '', high ?? '')
  }, [paletteVersion])

  return { grid, ramp, mode, killsAvailable }
}
