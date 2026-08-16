/**
 * Tests — calloutsLayer (zones nommées : normalisation, zoneAt 3D, calque).
 *
 * CE QU'ILS PROTÈGENT : l'affectation d'un joueur à sa zone se fait en DISTANCE 3D (les
 * étages se confondent en 2D — règle du POC), et le calque écrit chaque nom UNE fois.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayMapCallouts } from '@/lib/api/types'

import { calloutLabel, drawCalloutsLayer, normalizeCallouts, zoneAt } from './calloutsLayer'

const ENTRY: ReplayMapCallouts = {
  module: 'ridgeline',
  provenance: 'decoupe',
  zones: [
    {
      volume_index: 10, name: 'ridgeline horses', en: 'Horseshoe', fr: 'Fer à cheval',
      x: 19, y: 10, z: 1, z_bottom: -0.2, z_top: 11, big: false,
      polygon: [[14, 7], [24, 7], [24, 15]],
    },
    {
      // Même emprise 2D, étage EN DESSOUS : seul le z la distingue.
      volume_index: 25, name: 'ridgeline lower', en: 'Lower Horseshoe', fr: 'Fer à cheval inférieur',
      x: 19, y: 10, z: -2, z_bottom: -3, z_top: -1, big: false,
      polygon: [[15, 8], [23, 8], [23, 14]],
    },
    {
      volume_index: 11, name: 'antenna', en: 'Antenna', fr: 'Antenne',
      x: -2, y: -5, z: 1, z_bottom: 0, z_top: 10, big: true,
      polygon: [[-19, 7], [4, -18], [4, 6]],
      parts: [[[5, 5], [6, 5], [6, 6]]],
      holes: [[[-1, -1], [0, -1], [0, 0]]],
    },
    {
      // Volume SECONDAIRE du même nom, sans polygone : interrogeable, pas dessiné.
      volume_index: 26, name: 'antenna', en: 'Antenna', fr: 'Antenne',
      x: -2, y: -5, z: -4, z_bottom: -5, z_top: -3,
    },
  ],
}

describe('normalizeCallouts', () => {
  it('résout la nullabilité du transport et garde TOUTES les zones, même sans polygone', () => {
    const zones = normalizeCallouts(ENTRY)
    expect(zones).toHaveLength(4)
    expect(zones[0].big).toBe(false)
    expect(zones[2].big).toBe(true)
    expect(zones[3].polygon).toHaveLength(0)
    expect(zones[2].parts).toHaveLength(1)
    expect(zones[2].holes).toHaveLength(1)
  })

  it('ancre le libellé au centroïde du contour, ou au centre du volume sans contour', () => {
    const zones = normalizeCallouts(ENTRY)
    // Triangle (14,7)(24,7)(24,15) : centroïde (62/3, 29/3).
    expect(zones[0].labelX).toBeCloseTo(62 / 3, 5)
    expect(zones[0].labelY).toBeCloseTo(29 / 3, 5)
    expect(zones[3].labelX).toBe(-2)
    expect(zones[3].labelY).toBe(-5)
  })

  it('entrée absente = liste vide, jamais une erreur', () => {
    expect(normalizeCallouts(null)).toHaveLength(0)
    expect(normalizeCallouts(undefined)).toHaveLength(0)
  })
})

describe('zoneAt — affectation 3D', () => {
  const zones = normalizeCallouts(ENTRY)

  it('départage deux étages superposés par le z', () => {
    // Même point 2D (19, 10) : en bas -> version inférieure, en haut -> version haute.
    expect(zoneAt(zones, 19, 10, -1.9)?.en).toBe('Lower Horseshoe')
    expect(zoneAt(zones, 19, 10, 0.9)?.en).toBe('Horseshoe')
  })

  it('les volumes secondaires comptent : l’étage bas d’Antenne est affecté à Antenne', () => {
    expect(zoneAt(zones, -2, -5, -4)?.en).toBe('Antenna')
  })

  it('liste vide = null', () => {
    expect(zoneAt([], 0, 0, 0)).toBeNull()
  })
})

describe('calloutLabel', () => {
  it('rend le libellé de la langue de l’interface', () => {
    const zones = normalizeCallouts(ENTRY)
    expect(calloutLabel(zones[0], 'fr')).toBe('Fer à cheval')
    expect(calloutLabel(zones[0], 'en')).toBe('Horseshoe')
  })
})

/** Contexte canvas enregistreur : jsdom n'implémente pas getContext. */
function mockCtx() {
  const calls: { method: string; args: unknown[] }[] = []
  const record = (method: string) => (...args: unknown[]) => {
    calls.push({ method, args })
  }
  const ctx = {
    globalAlpha: 1,
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    lineJoin: 'miter',
    miterLimit: 10,
    font: '',
    textAlign: 'left',
    beginPath: record('beginPath'),
    moveTo: record('moveTo'),
    lineTo: record('lineTo'),
    closePath: record('closePath'),
    fill: record('fill'),
    stroke: record('stroke'),
    setLineDash: record('setLineDash'),
    strokeText: record('strokeText'),
    fillText: record('fillText'),
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, calls }
}

const VIEW = { bounds: { minX: -20, minY: -20, maxX: 25, maxY: 20 }, width: 800, height: 480, pad: 24 }

describe('drawCalloutsLayer', () => {
  it('écrit chaque nom UNE fois (dédoublonné), en MAJUSCULES', () => {
    const { ctx, calls } = mockCtx()
    drawCalloutsLayer(ctx, normalizeCallouts(ENTRY), VIEW, {
      bigColors: ['#111111'],
      fineInk: '#222222',
      locale: 'fr',
    })
    const labels = calls.filter((c) => c.method === 'fillText').map((c) => c.args[0])
    expect(labels).toContain('FER À CHEVAL')
    expect(labels).toContain('ANTENNE')
    // « Fer à cheval inférieur » est un NOM DISTINCT : il s'écrit aussi. Aucun doublon.
    expect(labels).toHaveLength(new Set(labels).size)
    // Chaque écriture est cernée : autant de strokeText que de fillText.
    expect(calls.filter((c) => c.method === 'strokeText')).toHaveLength(labels.length)
  })

  it('le libellé reste PETIT : au plus la taille du nom de joueur + 1 px d écran', () => {
    // Planche du 16/08 : « police trop grande, limiter le débordement au maximum ». La
    // borne n'est pas un goût — un nom de zone ne doit pas peser plus lourd qu'un joueur sur
    // la carte (8,7 px d'écran). Une unité de dessin EST un pixel d'écran ici (le contexte
    // est déjà transformé au devicePixelRatio) : la valeur écrite dans `font` est la bonne.
    const { ctx } = mockCtx()
    drawCalloutsLayer(ctx, normalizeCallouts(ENTRY), VIEW, {
      bigColors: ['#111111'],
      fineInk: '#222222',
      locale: 'fr',
    })
    const px = Number(/(\d+(?:\.\d+)?)px/.exec(ctx.font)?.[1])
    expect(px).toBeGreaterThan(0)
    expect(px).toBeLessThanOrEqual(9.7)
  })

  it('remplit les grandes zones en règle PAIR-IMPAIR (parties et trous du découpage)', () => {
    const { ctx, calls } = mockCtx()
    drawCalloutsLayer(ctx, normalizeCallouts(ENTRY), VIEW, {
      bigColors: ['#111111'],
      fineInk: '#222222',
      locale: 'fr',
    })
    expect(calls.some((c) => c.method === 'fill' && c.args[0] === 'evenodd')).toBe(true)
  })

  it('les zones fines sont POINTILLÉES, jamais remplies en pair-impair', () => {
    const fines: ReplayMapCallouts = {
      ...ENTRY,
      zones: ENTRY.zones ? ENTRY.zones.filter((z) => !z.big) : [],
    }
    const { ctx, calls } = mockCtx()
    drawCalloutsLayer(ctx, normalizeCallouts(fines), VIEW, {
      bigColors: [],
      fineInk: '#222222',
      locale: 'fr',
    })
    expect(calls.some((c) => c.method === 'setLineDash' && Array.isArray(c.args[0]) && (c.args[0] as number[]).length === 2)).toBe(true)
    expect(calls.some((c) => c.method === 'fill' && c.args[0] === 'evenodd')).toBe(false)
  })
})
