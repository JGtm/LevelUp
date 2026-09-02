/**
 * useReplayStaticLayers — LES CALQUES QUI NE BOUGENT PAS, cuits hors écran une fois.
 *
 * POURQUOI ILS SONT ENSEMBLE. Le sol, les zones nommées, la carte de chaleur et les objectifs
 * statiques ne dépendent NI de l'image courante NI de la lecture : ils ne changent que si leur
 * donnée, le cadrage ou leurs encres changent. Chacun se peint donc sur un canvas hors écran,
 * que la boucle d'animation se contente de RECOPIER — c'est ce qui rend 45 000 cellules de sol
 * indolores à 60 images par seconde.
 *
 * POURQUOI CE FICHIER EXISTE. Les quatre cuissons étaient quatre effets copiés dans
 * `ReplayCanvas.tsx`, avec quatre fois la même amorce (créer le canvas, lire la densité de
 * pixels, poser la transformation) et quatre recopies du cadrage `{bounds, width, height, pad}`
 * alors qu'il est déjà mémoïsé une fois. Le registre des reports en faisait la condition de la
 * prochaine addition au canvas (« extraire d'abord », 2026-08-16) : c'est fait ici, et
 * `cookLayer` porte l'amorce une seule fois.
 *
 * CE QUE LE HOOK NE FAIT PAS : dessiner à l'image. Il rend quatre références de canvas ; c'est
 * l'appelant qui les recopie dans son ordre de calques.
 */
import { useEffect, useRef, type RefObject } from 'react'

import { drawCalloutsLayer, type CalloutZoneReady } from './calloutsLayer'
import { drawHeatmapLayer, type HeatGrid } from './heatmapLayer'
import type { ReplayLocale } from './i18n'
import type { FloorGrid } from './mapFloor'
import { drawObjectivesLayer, type ObjectiveElementReady } from './objectivesLayer'
import { drawFloorLayer, type FloorStyle } from './replayDraw'
import type { CanvasView } from './replayMarkers'

/** Ce que chaque calque statique a besoin de savoir, regroupé par calque. */
export interface StaticLayersInput {
  /** Le cadrage PARTAGÉ (`canvasView`) : le dessin et le survol lisent la même projection. */
  view: CanvasView
  /** Repeindre la scène — appelé après chaque cuisson, et à l'extinction de la chaleur. */
  redraw: () => void
  /**
   * GELER LA CUISSON (2026-09-02, glisser). Pendant un glisser le cadrage change à chaque
   * mouvement de pointeur ; recuire à cette cadence est hors de question — le sol reconstruit
   * fait ~45 000 cellules. Mais un glisser est une TRANSLATION PURE : l'appelant recopie donc
   * le calque déjà cuit avec un décalage (`layerOffset`), ce qui est exact et gratuit. La
   * cuisson reprend au relâchement, quand `frozen` retombe.
   *
   * On ne gèle JAMAIS pendant un zoom : l'échelle change alors, et un décalage ne suffirait
   * plus à replacer une image cuite pour une autre échelle.
   */
  frozen?: boolean
  floor: { grid: FloorGrid | null; style: FloorStyle }
  zones: {
    zones: readonly CalloutZoneReady[]
    bigColors: string[]
    fineInk: string
    locale: ReplayLocale
  }
  heat: { grid: HeatGrid | null; ramp: string[] }
  objectives: {
    elements: readonly ObjectiveElementReady[]
    colorOfTeam: (team: number) => string
  }
}

/** Les quatre canvas hors écran, dans l'ordre où ils se recopient. */
export interface StaticLayers {
  /**
   * LE CADRAGE AVEC LEQUEL LES CALQUES ONT ÉTÉ CUITS — `null` tant qu'aucune cuisson n'a eu
   * lieu. C'est lui que l'appelant compare au cadrage courant pour savoir de combien décaler
   * sa recopie. Sans cette valeur, un calque gelé serait recopié à sa place d'origine, et le
   * décor resterait immobile pendant que les joueurs se déplacent.
   */
  cookedRef: RefObject<CanvasView | null>
  floorRef: RefObject<HTMLCanvasElement | null>
  zonesRef: RefObject<HTMLCanvasElement | null>
  heatRef: RefObject<HTMLCanvasElement | null>
  objectivesRef: RefObject<HTMLCanvasElement | null>
}

/**
 * cookLayer — l'amorce commune : un canvas à la densité de l'écran, la transformation posée,
 * puis le tracé. Rend null quand le contexte n'est pas disponible (le canvas est alors éteint
 * plutôt que laissé dans un état intermédiaire).
 */
function cookLayer(
  view: CanvasView,
  paint: (ctx: CanvasRenderingContext2D, dpr: number) => void,
): HTMLCanvasElement | null {
  const dpr = window.devicePixelRatio || 1
  const off = document.createElement('canvas')
  off.width = Math.round(view.width * dpr)
  off.height = Math.round(view.height * dpr)
  const ctx = off.getContext('2d')
  if (!ctx) return null
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  paint(ctx, dpr)
  return off
}

export function useReplayStaticLayers({
  view,
  redraw,
  floor,
  zones,
  heat,
  objectives,
  frozen = false,
}: StaticLayersInput): StaticLayers {
  const cookedRef = useRef<CanvasView | null>(null)
  // Le cadrage de la DERNIÈRE cuisson. Il ne bouge pas tant que le gel dure : c'est ce qui
  // permet de calculer le décalage de recopie, et c'est aussi ce qui le rend juste.
  useEffect(() => {
    if (!frozen) cookedRef.current = view
  }, [view, frozen])
  const floorRef = useRef<HTMLCanvasElement | null>(null)
  const zonesRef = useRef<HTMLCanvasElement | null>(null)
  const heatRef = useRef<HTMLCanvasElement | null>(null)
  const objectivesRef = useRef<HTMLCanvasElement | null>(null)

  const { grid: floorGrid, style: floorStyle } = floor
  useEffect(() => {
    // GELÉ : on garde le calque tel quel, l'appelant le recopiera décalé (cf. `frozen`).
    if (frozen) return
    if (!floorGrid || view.width === 0) {
      floorRef.current = null
      return
    }
    floorRef.current = cookLayer(view, (ctx) => drawFloorLayer(ctx, floorGrid, view, floorStyle))
    redraw()
  }, [floorGrid, floorStyle, view, redraw, frozen])

  const { zones: zoneList, bigColors, fineInk, locale } = zones
  useEffect(() => {
    if (frozen) return
    if (zoneList.length === 0 || view.width === 0) {
      zonesRef.current = null
      return
    }
    zonesRef.current = cookLayer(view, (ctx) =>
      drawCalloutsLayer(ctx, [...zoneList], view, { bigColors, fineInk, locale }),
    )
    redraw()
  }, [zoneList, bigColors, fineInk, locale, view, redraw, frozen])

  const { grid: heatGrid, ramp } = heat
  useEffect(() => {
    if (frozen) return
    if (!heatGrid || view.width === 0 || ramp.length === 0) {
      heatRef.current = null
      // REPEINDRE APRÈS AVOIR ÉTEINT : à l'arrêt, rien d'autre ne déclenche de redessin
      // (contrairement au calque des zones, dont la bascule figure dans les dépendances de
      // `draw`) — sans cette ligne, la carte de chaleur resterait à l'écran après extinction.
      redraw()
      return
    }
    heatRef.current = cookLayer(view, (ctx, dpr) =>
      drawHeatmapLayer(ctx, heatGrid, view, { ramp, k: dpr }),
    )
    redraw()
  }, [heatGrid, ramp, view, redraw, frozen])

  const { elements, colorOfTeam } = objectives
  useEffect(() => {
    if (frozen) return
    if (elements.length === 0 || view.width === 0) {
      objectivesRef.current = null
      return
    }
    objectivesRef.current = cookLayer(view, (ctx) =>
      drawObjectivesLayer(ctx, [...elements], view, { colorOfTeam }),
    )
    redraw()
  }, [elements, colorOfTeam, view, redraw, frozen])

  return { floorRef, zonesRef, heatRef, objectivesRef, cookedRef }
}
