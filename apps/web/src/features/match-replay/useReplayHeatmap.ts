/**
 * useReplayHeatmap — tout ce qu'il faut savoir pour peindre la carte de chaleur, cuit une
 * fois : la grille, la rampe du thème, et la lecture réellement servie.
 *
 * DÉLIBÉRÉMENT HORS DU COMPOSANT (même règle que `useReplaySettings` / `useReplaySound`) :
 * ReplayCanvas ne doit ni savoir d'où viennent les positions de mort, ni décider quelle
 * rampe le thème impose, ni arbitrer une préférence devenue impossible. Il reçoit un calque
 * prêt à cuire et le pose ; toute la logique se teste ici, sans canvas.
 */
import { useEffect, useMemo, useState, type RefObject } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { ReplayBounds } from '@/lib/api/types'

import {
  buildHeatmap,
  heatRamp,
  type HeatDeath,
  type HeatGrid,
  type HeatmapMode,
  type HeatmapSpan,
} from './heatmapLayer'
import type { KillFxEntry } from './killFx'
import { msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/**
 * PAS DE RECUISSON PAR IMAGE — le pas de la portée `live`, en millisecondes de MATCH.
 *
 * Cuire la grille coûte 6 à 20 ms selon la carte (mesuré le 2026-08-18 sur trois films :
 * 5 445 cellules en 6 ms, 15 232 en 20 ms). À soixante images par seconde ce serait tout le
 * budget d'animation pour une carte qui ne change pas à l'œil d'une image à l'autre : deux
 * secondes de jeu déplacent un joueur de quelques mètres, soit une poignée de cellules sur
 * dix mille. Le pas est donc en temps de MATCH et non en temps réel — à 4× la carte se
 * remplit quatre fois plus vite à l'écran, ce qui est exactement ce qu'on veut voir.
 */
export const HEAT_LIVE_STEP_MS = 2_000

/** Cadence de SURVEILLANCE de l'image courante, en ms réelles. Ne cuit rien : lit une ref. */
const HEAT_LIVE_POLL_MS = 250

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

/** Ce que le hook a besoin de savoir du réglage, et d'où lire l'image courante. */
export interface ReplayHeatmapSettings {
  show: boolean
  mode: HeatmapMode
  /** Portée de temps (V2, 2026-08-18) : toute la partie, ou jusqu'à l'image courante. */
  span: HeatmapSpan
  /**
   * L'image courante, PAR RÉFÉRENCE. Le canvas ne publie pas la frame dans un état React (il
   * la garde dans une ref pour ne pas re-rendre le DOM soixante fois par seconde) : le hook
   * vient donc la LIRE, et seulement quand la portée `live` est demandée.
   */
  frameRef: RefObject<number>
}

/**
 * useLiveFrameBucket — l'image courante ARRONDIE au pas de recuisson, ou null hors portée
 * `live`.
 *
 * ARRONDIE, ET C'EST TOUT L'INTÉRÊT : rendre la frame exacte ferait un rendu React et une
 * cuisson par sondage. Le seau ne change qu'une fois par `HEAT_LIVE_STEP_MS` de match, donc
 * `useMemo` de la grille ne se déclenche qu'alors. Un retour en arrière (barre de lecture,
 * boucle de fin) change le seau vers le bas : la carte se REDESSINE à ce qu'elle était, elle
 * ne garde pas une chaleur qui n'a plus eu lieu.
 *
 * AUCUN `setState` DANS LE CORPS DE L'EFFET, et ce n'est pas qu'une question de lint : un état
 * posé à l'armement déclenche un rendu en cascade à chaque bascule du réglage. La valeur
 * initiale vient donc de l'initialiseur de `useState`, et les suivantes du seul minuteur. Prix
 * assumé : en allumant la portée `live` en cours de lecture, le premier seau peut dater d'un
 * quart de seconde — le temps d'un battement de sondage, invisible sur une carte qui se remplit.
 */
function useLiveFrameBucket(
  doc: ReplayDocumentReady,
  settings: ReplayHeatmapSettings,
): number | null {
  const live = settings.show && settings.span === 'live'
  const step = useMemo(() => Math.max(1, msToFrames(HEAT_LIVE_STEP_MS, doc)), [doc])
  const { frameRef } = settings
  const [bucket, setBucket] = useState(() => Math.floor(frameRef.current / step) * step)
  useEffect(() => {
    if (!live) return
    const id = window.setInterval(() => {
      const next = Math.floor(frameRef.current / step) * step
      setBucket((prev) => (prev === next ? prev : next))
    }, HEAT_LIVE_POLL_MS)
    return () => window.clearInterval(id)
  }, [live, step, frameRef])
  return live ? bucket : null
}

export function useReplayHeatmap(
  doc: ReplayDocumentReady,
  bounds: ReplayBounds,
  killFx: KillFxEntry[],
  settings: ReplayHeatmapSettings,
): ReplayHeatmap {
  // LES LIEUX DE MORT, relus une fois : la carte des éliminations compte les victimes là
  // où elles sont TOMBÉES. `deathX` existe dès que la victime est localisée, même sans
  // tueur — contrairement à `vx`/`vy`, qui n'orientent l'effet qu'en couple complet.
  const deaths = useMemo(() => {
    const out: HeatDeath[] = []
    for (const e of killFx) {
      if (e.deathX !== null && e.deathY !== null) {
        out.push({ x: e.deathX, y: e.deathY, frame: e.frame })
      }
    }
    return out
  }, [killFx])

  const killsAvailable = deaths.length > 0
  const mode: HeatmapMode = killsAvailable ? settings.mode : 'presence'

  // Cuite UNE fois par document et par lecture (patron `buildShotFx`), et seulement quand
  // le calque est allumé : c'est le calcul le plus lourd de la page (mesuré à 7 ms sur une
  // arène, 19 ms sur une carte BTB, pour 72 000 positions).
  const until = useLiveFrameBucket(doc, settings)
  const grid = useMemo(
    () => (settings.show ? buildHeatmap(doc, bounds, mode, deaths, until ?? undefined) : null),
    [settings.show, doc, bounds, mode, deaths, until],
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
