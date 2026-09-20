import { describe, expect, it } from 'vitest'

import type { WeaponDistanceRecordRow } from '@/lib/api/types'

import {
  RULER_LABEL_MIN_GAP_PX,
  RULER_LABEL_ROW_PX,
  RULER_PADDING_X,
  formatMeters,
  labelBaselineY,
  resolveRecordLabel,
  rulerMaxMeters,
  staggerLabels,
  weaponRecordsLayout,
} from './weaponRecords_logic'

const row = (key: string, record: number, label = key): WeaponDistanceRecordRow => ({
  weapon_key: key,
  label,
  label_en: label,
  class: 'shoulder',
  measured: 10,
  median_m: record / 2,
  record_m: record,
  record: { match_id: 'm1', time_ms: 1000 },
})

describe('resolveRecordLabel', () => {
  it('prend le libellé de la locale et retombe sur la clé, jamais sur un nom inventé', () => {
    expect(resolveRecordLabel({ weapon_key: 'hinf_br75', label: 'BR75', label_en: 'BR75 EN' }, 'fr')).toBe('BR75')
    expect(resolveRecordLabel({ weapon_key: 'hinf_br75', label: 'BR75', label_en: 'BR75 EN' }, 'en')).toBe('BR75 EN')
    expect(resolveRecordLabel({ weapon_key: 'hinf_x', label: '  ', label_en: null }, 'fr')).toBe('hinf_x')
  })
})

describe('formatMeters', () => {
  it('une décimale, séparateur de la locale', () => {
    expect(formatMeters(96.4, 'fr')).toBe('96,4 m')
    expect(formatMeters(96.4, 'en')).toBe('96.4 m')
    expect(formatMeters(10, 'en')).toBe('10.0 m')
  })
})

describe('rulerMaxMeters', () => {
  it('arrondit le record le plus lointain à la dizaine supérieure, plancher un pas', () => {
    expect(rulerMaxMeters([])).toBe(10)
    expect(rulerMaxMeters([{ record_m: 3.2 }])).toBe(10)
    expect(rulerMaxMeters([{ record_m: 96.4 }, { record_m: 12 }])).toBe(100)
    expect(rulerMaxMeters([{ record_m: 100 }])).toBe(100)
    expect(rulerMaxMeters([{ record_m: 100.1 }])).toBe(110)
  })
})

describe('staggerLabels', () => {
  it('deux boîtes qui se touchent prennent deux rangs, une boîte éloignée revient au rang 0', () => {
    const ranks = staggerLabels(
      [
        { x0: 0, x1: 60 },
        { x0: 50, x1: 110 }, // chevauche la première
        { x0: 70, x1: 130 }, // chevauche les deux
        { x0: 300, x1: 360 }, // loin : rang 0 libre
      ],
      RULER_LABEL_MIN_GAP_PX,
    )
    expect(ranks).toEqual([0, 1, 2, 0])
  })

  it("respecte l'écart minimal : deux boîtes séparées de moins que l'écart s'étagent", () => {
    expect(staggerLabels([{ x0: 0, x1: 60 }, { x0: 70, x1: 120 }], 14)).toEqual([0, 1])
    expect(staggerLabels([{ x0: 0, x1: 60 }, { x0: 74, x1: 120 }], 14)).toEqual([0, 0])
  })
})

describe('weaponRecordsLayout', () => {
  it("ne retrie pas : l'ordre du backend est l'ordre des items", () => {
    const layout = weaponRecordsLayout([row('a', 10), row('b', 50), row('c', 30)], 'fr', 1000)
    expect(layout.items.map((i) => i.row.weapon_key)).toEqual(['a', 'b', 'c'])
  })

  it('place chaque losange à son record sur une échelle linéaire bornée par la dizaine supérieure', () => {
    const layout = weaponRecordsLayout([row('a', 25), row('b', 96.4)], 'fr', 1000)
    expect(layout.maxMeters).toBe(100)
    expect(layout.x(0)).toBe(RULER_PADDING_X)
    expect(layout.x(100)).toBe(1000 - RULER_PADDING_X)
    expect(layout.items[0].cx).toBeCloseTo(RULER_PADDING_X + 0.25 * (1000 - 2 * RULER_PADDING_X))
    expect(layout.ticks).toEqual([0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100])
  })

  it('aucun libellé ne chevauche un autre du même rang, et la hauteur suit le nombre de rangs', () => {
    // Dix armes serrées entre 9 et 28 m : impossible sur un seul rang à 1000 px.
    const rows = [9.8, 12.4, 15.2, 18.3, 19.4, 22.9, 26.1, 27.5, 33.8, 96.4].map((m, i) =>
      row(`w${i}`, m, `Arme numero ${i}`),
    )
    const layout = weaponRecordsLayout(rows, 'fr', 1000)
    expect(layout.rowCount).toBeGreaterThan(1)
    const byRank = new Map<number, typeof layout.items>()
    for (const it of layout.items) byRank.set(it.rank, [...(byRank.get(it.rank) ?? []), it])
    for (const items of byRank.values()) {
      const sorted = [...items].sort((a, b) => a.labelX - b.labelX)
      for (let i = 1; i < sorted.length; i += 1) {
        const prev = sorted[i - 1]
        const cur = sorted[i]
        const half = (s: (typeof sorted)[number]) =>
          (Math.max(s.label.length, s.valueText.length) * 6.8 + 12) / 2
        expect(cur.labelX - half(cur)).toBeGreaterThanOrEqual(prev.labelX + half(prev) + RULER_LABEL_MIN_GAP_PX - 1e-6)
      }
    }
    const one = weaponRecordsLayout([row('a', 50)], 'fr', 1000)
    expect(layout.height - one.height).toBe((layout.rowCount - 1) * RULER_LABEL_ROW_PX)
  })

  it('recentre un libellé qui sortirait de la vue, sans déplacer le losange', () => {
    const layout = weaponRecordsLayout([row('a', 0.5, 'Un libellé vraiment long pour déborder')], 'fr', 400)
    const item = layout.items[0]
    expect(item.cx).toBeCloseTo(layout.x(0.5))
    const half = (Math.max(item.label.length, item.valueText.length) * 6.8 + 12) / 2
    expect(item.labelX - half).toBeGreaterThanOrEqual(0)
  })

  it('une liste vide donne une règle sans rang, jamais une exception', () => {
    const layout = weaponRecordsLayout([], 'fr', 800)
    expect(layout.items).toEqual([])
    expect(layout.rowCount).toBe(0)
    expect(layout.height).toBeGreaterThan(0)
  })

  it('les rangs montent depuis l axe', () => {
    const layout = weaponRecordsLayout([row('a', 10)], 'fr', 800)
    expect(labelBaselineY(layout, 0)).toBeLessThan(layout.axisY)
    expect(labelBaselineY(layout, 1)).toBe(labelBaselineY(layout, 0) - RULER_LABEL_ROW_PX)
  })
})
