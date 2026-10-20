/**
 * planPaint — l'ordre des calques du plan de l'onglet Tactique : contours des zones nommées SOUS la
 * chaleur, noms AU-DESSUS, par LE peintre du rejeu 2D ; rien des zones quand la lecture n'en sert pas.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ReplayCalloutZone } from '@/lib/api/types'
import * as callouts from '@/lib/replay/calloutsPaint'
import * as chaleur from '@/lib/replay/heatPaint'

import { peindreLePlan } from './planPaint'
import type { RepereTactique } from './tacticalView.logic'

const ordre = vi.hoisted(() => [] as string[])
vi.mock('@/lib/replay/calloutsPaint', async (importOriginal) => {
  const reel = await importOriginal<typeof import('@/lib/replay/calloutsPaint')>()
  return {
    ...reel,
    drawCalloutsShapes: vi.fn((...a: Parameters<typeof reel.drawCalloutsShapes>) => {
      ordre.push('formes')
      reel.drawCalloutsShapes(...a)
    }),
    drawCalloutsLabels: vi.fn((...a: Parameters<typeof reel.drawCalloutsLabels>) => {
      ordre.push('noms')
      reel.drawCalloutsLabels(...a)
    }),
  }
})
vi.mock('@/lib/replay/heatPaint', async (importOriginal) => {
  const reel = await importOriginal<typeof import('@/lib/replay/heatPaint')>()
  return {
    ...reel,
    drawTacticalHeatmap: vi.fn((...a: Parameters<typeof reel.drawTacticalHeatmap>) => {
      ordre.push('chaleur')
      reel.drawTacticalHeatmap(...a)
    }),
  }
})

/** Un contexte 2D qui ENREGISTRE les appels de dessin (jsdom n'en fournit aucun). */
function contexte() {
  const appels: string[] = []
  const etat: Record<string, unknown> = {}
  const ctx = new Proxy(etat, {
    get(cible, nom: string) {
      if (nom in cible) return cible[nom]
      return () => {
        appels.push(nom)
      }
    },
    set(cible, nom: string, valeur: unknown) {
      cible[nom] = valeur
      return true
    },
  })
  return { ctx: ctx as unknown as CanvasRenderingContext2D, appels }
}

// Repère 0..100 x 0..50 m, pas de 10 m : un canvas de 200 px de large met 2 px par mètre.
const REPERE: RepereTactique = { minX: 0, maxX: 100, minY: 0, maxY: 50, pasM: 10 }
const GRILLE = chaleur.buildTacticalGrid(
  [{ col: 2, row: 1, value: 3 }],
  { cell: 10, nx: 10, ny: 5, minX: 0, minY: 0 },
  { lo: 1, hi: 5, signee: false },
  1,
)
const ZONES: ReplayCalloutZone[] = [
  { name: '', fr: 'Base', en: 'Base', x: 20, y: 40, z: 1, z_bottom: 0, z_top: 2, volume_index: 1, big: true, polygon: [[10, 30], [30, 30], [30, 50], [10, 50]] },
  { name: '', fr: 'Pont', en: 'Bridge', x: 60, y: 10, z: 5, z_bottom: 4, z_top: 6, volume_index: 2, polygon: [[50, 0], [70, 0], [70, 20], [50, 20]] },
]
const ENCRE = 'encre-du-texte'

function peindre(zones: readonly callouts.CalloutZoneReady[], repere: RepereTactique = REPERE) {
  const { ctx, appels } = contexte()
  peindreLePlan(ctx, repere, 200, { grid: GRILLE, ramp: ['rampe-1', 'rampe-2'], zones, locale: 'fr', encre: ENCRE })
  return appels
}

afterEach(() => {
  ordre.length = 0
  vi.clearAllMocks()
})

describe('peindreLePlan — les zones nommées du plan, par le peintre du rejeu', () => {
  it('zones servies : contours SOUS la chaleur, noms AU-DESSUS, à l’encre du texte', () => {
    const appels = peindre(callouts.normalizeCalloutZones(ZONES))
    expect(ordre).toEqual(['formes', 'chaleur', 'noms'])
    const style = vi.mocked(callouts.drawCalloutsShapes).mock.calls[0]?.[3]
    expect(style).toEqual({ bigColors: [ENCRE], fineInk: ENCRE, locale: 'fr' })
    // Contours tracés avant la première cellule de chaleur, noms écrits après la dernière.
    expect(appels.indexOf('stroke')).toBeGreaterThanOrEqual(0)
    expect(appels.indexOf('stroke')).toBeLessThan(appels.indexOf('fillRect'))
    expect(appels.lastIndexOf('fillRect')).toBeLessThan(appels.indexOf('fillText'))
    expect(appels.filter((a) => a === 'fillText')).toHaveLength(2)
  })

  it('la projection est celle de la chaleur : (minX, maxY) au coin haut-gauche, Y inversé', () => {
    peindre(callouts.normalizeCalloutZones(ZONES))
    const projection = vi.mocked(callouts.drawCalloutsShapes).mock.calls[0]?.[2]
    expect(projection?.({ x: 0, y: 50 })).toEqual({ x: 0, y: 0 })
    expect(projection?.({ x: 100, y: 0 })).toEqual({ x: 200, y: 100 })
    expect(vi.mocked(callouts.drawCalloutsLabels).mock.calls[0]?.[2]).toBe(projection)
  })

  it('zones absentes : la chaleur seule, aucun contour ni nom', () => {
    const appels = peindre([])
    expect(appels).toContain('fillRect')
    expect(appels.filter((a) => a === 'stroke' || a === 'fillText' || a === 'strokeText')).toEqual([])
  })

  it('repère inexploitable : rien n’est peint', () => {
    const appels = peindre(callouts.normalizeCalloutZones(ZONES), { ...REPERE, maxX: 0 })
    expect(ordre).toEqual([])
    expect(appels).toEqual([])
  })
})
