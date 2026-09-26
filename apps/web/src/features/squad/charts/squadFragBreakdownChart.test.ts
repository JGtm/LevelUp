/**
 * squadFragBreakdownChart.test.ts — MODÈLE de la « Répartition des frags » par joueur :
 * segments PAR CLASSE (ordre canonique), échelle commune, repli S3 et ton de l'écriture.
 */
import { describe, it, expect } from 'vitest'
import {
  buildFragBreakdownRows,
  fragBreakdownClasses,
  repliOffsetPct,
  segmentTextTone,
} from './squadFragBreakdownChart'
import type { FragClassEntry } from '@/lib/api/types'

function cls(className: string, kills: number): FragClassEntry {
  return { class: className, kills, authoritative: false }
}

const ORDER = ['Me', 'F1']

describe('buildFragBreakdownRows', () => {
  it('vide, ou aucune classe > 0 → aucune barre', () => {
    expect(buildFragBreakdownRows({})).toEqual([])
    expect(buildFragBreakdownRows({ Me: [cls('shoulder', 0)] }, ['Me'])).toEqual([])
  })

  it('segments dans l’ordre canonique des classes, zéros omis, total par joueur', () => {
    const rows = buildFragBreakdownRows(
      {
        Me: [cls('unattributed', 10), cls('melee', 6), cls('shoulder', 18)],
        F1: [cls('grenade', 4), cls('heavy', 5)],
      },
      ORDER,
    )
    expect(rows.map((r) => r.player)).toEqual(['Me', 'F1'])
    expect(rows[0].segments.map((s) => s.cls)).toEqual(['shoulder', 'melee', 'unattributed'])
    expect(rows[0].segments.map((s) => s.kills)).toEqual([18, 6, 10])
    expect(rows[0].total).toBe(34)
    expect(rows[1].segments.map((s) => s.cls)).toEqual(['heavy', 'grenade'])
    expect(rows[1].total).toBe(9)
  })

  it('échelle COMMUNE : le plus gros total = 100 %, segments contigus', () => {
    const rows = buildFragBreakdownRows(
      { Me: [cls('shoulder', 30), cls('melee', 10)], F1: [cls('shoulder', 20)] },
      ORDER,
    )
    const [me, f1] = rows
    expect(me.segments[0]).toMatchObject({ leftPct: 0, widthPct: 75 })
    expect(me.segments[1]).toMatchObject({ leftPct: 75, widthPct: 25 })
    expect(f1.segments[0]).toMatchObject({ leftPct: 0, widthPct: 50 })
  })

  it('agrège plusieurs entrées d’une même classe ; classe H5 « Capacités spartanes » placée avant le résidu', () => {
    const rows = buildFragBreakdownRows(
      { Me: [cls('shoulder', 4), cls('shoulder', 3), cls('unattributed', 1), cls('spartan_ability', 2)] },
      ['Me'],
    )
    expect(rows[0].segments.map((s) => [s.cls, s.kills])).toEqual([
      ['shoulder', 7],
      ['spartan_ability', 2],
      ['unattributed', 1],
    ])
  })

  it('soirée du 22/09 (maquette C3EW) : totaux 65 / 63 / 121', () => {
    const soiree = {
      JGtm: [cls('shoulder', 28), cls('sidearm', 13), cls('heavy', 11), cls('melee', 6), cls('grenade', 5), cls('unattributed', 2)],
      Chocoboflor: [cls('shoulder', 24), cls('sidearm', 19), cls('heavy', 4), cls('melee', 13), cls('grenade', 2), cls('environmental', 1)],
      Madina97294: [cls('shoulder', 45), cls('sidearm', 37), cls('heavy', 16), cls('melee', 16), cls('grenade', 3), cls('environmental', 1), cls('unattributed', 3)],
    }
    const rows = buildFragBreakdownRows(soiree, ['JGtm', 'Chocoboflor', 'Madina97294'])
    expect(rows.map((r) => r.total)).toEqual([65, 63, 121])
    const last = rows[2].segments[rows[2].segments.length - 1]
    expect(last.leftPct + last.widthPct).toBeCloseTo(100)
  })
})

describe('fragBreakdownClasses', () => {
  it('union des classes présentes, ordre canonique (même source que les segments)', () => {
    expect(
      fragBreakdownClasses({ Me: [cls('melee', 1), cls('shoulder', 2)], F1: [cls('heavy', 1)] }, ORDER),
    ).toEqual(['shoulder', 'heavy', 'melee'])
  })
})

describe('repliOffsetPct (S3)', () => {
  const segments = [
    { cls: 'shoulder', kills: 30, leftPct: 0, widthPct: 60 },
    { cls: 'melee', kills: 2, leftPct: 60, widthPct: 4 },
    { cls: 'grenade', kills: 1, leftPct: 64, widthPct: 2 },
  ]
  it('la ligne de repli s’aligne sur le PREMIER segment masqué', () => {
    expect(repliOffsetPct(segments, (c) => c !== 'shoulder')).toBe(60)
    expect(repliOffsetPct(segments, (c) => c === 'grenade')).toBe(64)
  })
  it('tout tient → pas de ligne de repli', () => {
    expect(repliOffsetPct(segments, () => false)).toBeNull()
  })
})

describe('segmentTextTone', () => {
  it('écriture sombre sur un aplat clair, claire sur un aplat sombre', () => {
    // Couleurs de test (pas de charte) : un jaune très clair et un bleu nuit.
    expect(segmentTextTone('#fde68a')).toBe('dark') // color-allow: valeur de test
    expect(segmentTextTone('#1e3a8a')).toBe('light') // color-allow: valeur de test
  })
  it('valeur non hex (variable CSS) → clair', () => {
    expect(segmentTextTone('var(--x)')).toBe('light')
  })
})
