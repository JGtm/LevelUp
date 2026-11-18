/**
 * feedGlyphs.test.ts — les pictogrammes d'une ligne de prise : quatre natures, quatre tracés ; l'étoile ne
 * touche jamais le tracé principal ; la nature d'une prise du fil ajoute le cas « équipement qui change la
 * partie » sans toucher à la famille des socles de la carte.
 */
import { describe, expect, it } from 'vitest'

import { FEED_FILTER_GLYPH, FEED_GLYPH_PATH, PICKUP_NATURE_GLYPH } from './feedGlyphs'
import { pickupNatureFrom } from './pickupFeedLogic'
import { PAD_EQUIPMENT_FAMILIES, POWER_PAD_KEYS, padFamilyOf } from './weaponPadFamilies'

type Point = readonly [number, number]
type Segment = readonly [Point, Point]

/** Les segments d'un tracé écrit avec M, L, H, V (absolus ou relatifs) et Z : tout ce que ces pictogrammes emploient. */
function segmentsOf(d: string): Segment[] {
  const out: Segment[] = []
  let cur: Point = [0, 0]
  let start: Point = [0, 0]
  for (const [, cmd, args] of d.matchAll(/([MmLlHhVvZz])([^A-Za-z]*)/g)) {
    const n = (args.match(/-?\d*\.?\d+/g) ?? []).map(Number)
    const rel = cmd === cmd.toLowerCase()
    const go = (to: Point) => {
      out.push([cur, to])
      cur = to
    }
    switch (cmd.toUpperCase()) {
      case 'M':
        cur = rel ? [cur[0] + n[0], cur[1] + n[1]] : [n[0], n[1]]
        start = cur
        for (let i = 2; i < n.length; i += 2) go(rel ? [cur[0] + n[i], cur[1] + n[i + 1]] : [n[i], n[i + 1]])
        break
      case 'L':
        for (let i = 0; i < n.length; i += 2) go(rel ? [cur[0] + n[i], cur[1] + n[i + 1]] : [n[i], n[i + 1]])
        break
      case 'H':
        go([rel ? cur[0] + n[0] : n[0], cur[1]])
        break
      case 'V':
        go([cur[0], rel ? cur[1] + n[0] : n[0]])
        break
      default:
        go(start)
    }
  }
  return out
}

function distance(p: Point, [a, b]: Segment): number {
  const dx = b[0] - a[0]
  const dy = b[1] - a[1]
  const t = Math.max(0, Math.min(1, ((p[0] - a[0]) * dx + (p[1] - a[1]) * dy) / (dx * dx + dy * dy)))
  return Math.hypot(p[0] - a[0] - t * dx, p[1] - a[1] - t * dy)
}

/** La plus petite distance entre deux jeux de segments, sur un échantillonnage fin du premier. */
function gapBetween(a: Segment[], b: Segment[]): number {
  let min = Number.POSITIVE_INFINITY
  for (const [p, q] of a) {
    for (let i = 0; i <= 200; i++) {
      const pt: Point = [p[0] + ((q[0] - p[0]) * i) / 200, p[1] + ((q[1] - p[1]) * i) / 200]
      for (const s of b) min = Math.min(min, distance(pt, s))
    }
  }
  return min
}

const STROKE = 2
/** Une unité de grille libre entre deux traits, épaisseur comprise : à 15 px de haut (24 unités), 0,6 px. */
const MIN_CLEAR = 1

describe('les pictogrammes d’une prise', () => {
  it('quatre natures, quatre tracés distincts', () => {
    const paths = Object.values(PICKUP_NATURE_GLYPH).map((g) => FEED_GLYPH_PATH[g])
    expect(paths).toHaveLength(4)
    expect(new Set(paths).size).toBe(4)
  })

  it('l’arme de râtelier est la flèche, l’arme spéciale la même flèche et l’étoile ; le menu de filtre prend la flèche', () => {
    const fleche = FEED_GLYPH_PATH[PICKUP_NATURE_GLYPH.classic]
    expect(FEED_GLYPH_PATH[PICKUP_NATURE_GLYPH.power].startsWith(fleche)).toBe(true)
    expect(FEED_GLYPH_PATH[PICKUP_NATURE_GLYPH.power]).not.toBe(fleche)
    expect(FEED_GLYPH_PATH[FEED_FILTER_GLYPH.pickups]).toBe(fleche)
  })

  it('l’équipement est la caisse, l’équipement qui change la partie la même caisse et l’étoile (plus de bouclier)', () => {
    const caisse = FEED_GLYPH_PATH[PICKUP_NATURE_GLYPH.powerup]
    expect(FEED_GLYPH_PATH[PICKUP_NATURE_GLYPH.powerupGameChanger].startsWith(caisse)).toBe(true)
    expect(FEED_GLYPH_PATH[PICKUP_NATURE_GLYPH.powerupGameChanger]).not.toBe(caisse)
    expect(caisse).not.toContain('c0 5')
  })

  it('l’étoile laisse au moins une unité de grille libre entre son trait et celui de la flèche, puis de la caisse', () => {
    const etoile = (clef: 'pickupPower' | 'pickupEquipmentPower', base: 'pickup' | 'pickupEquipment') =>
      segmentsOf(FEED_GLYPH_PATH[clef].slice(FEED_GLYPH_PATH[base].length))
    for (const [clef, base] of [['pickupPower', 'pickup'], ['pickupEquipmentPower', 'pickupEquipment']] as const) {
      const jeu = gapBetween(etoile(clef, base), segmentsOf(FEED_GLYPH_PATH[base])) - STROKE
      expect(jeu, `${clef}`).toBeGreaterThanOrEqual(MIN_CLEAR)
    }
  })

  it('tous les tracés restent dans la grille de 24, trait compris', () => {
    for (const clef of ['pickup', 'pickupPower', 'pickupEquipment', 'pickupEquipmentPower'] as const) {
      for (const [a, b] of segmentsOf(FEED_GLYPH_PATH[clef])) {
        for (const v of [...a, ...b]) {
          expect(v - STROKE / 2).toBeGreaterThanOrEqual(0)
          expect(v + STROKE / 2).toBeLessThanOrEqual(24)
        }
      }
    }
  })
})

describe('la nature d’une prise du fil', () => {
  it('un équipement dont la clé figure parmi les socles de puissance change la partie', () => {
    for (const clef of Object.keys(PAD_EQUIPMENT_FAMILIES)) {
      expect(POWER_PAD_KEYS).toContain(clef)
      expect(pickupNatureFrom(padFamilyOf(clef, undefined), clef)).toBe('powerupGameChanger')
    }
  })

  it('un équipement hors de cette liste reste une caisse ordinaire', () => {
    expect(pickupNatureFrom('powerup', 'powerup_inconnu')).toBe('powerup')
  })

  it('une arme garde la famille de son socle : spéciale ou râtelier', () => {
    expect(pickupNatureFrom('power', 'hinf_s7_sniper')).toBe('power')
    expect(pickupNatureFrom('classic', 'hinf_br75')).toBe('classic')
  })
})
