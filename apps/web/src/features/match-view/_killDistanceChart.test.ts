/**
 * _killDistanceChart.test.ts — le bâton min→max et son losange de moyenne, en pur.
 *
 * Ce que ces tests verrouillent : la projection (libellé par locale, repli weapon_key),
 * l'ORDRE (backend en tête = HAUT de l'axe, donc liste inversée au montage), la technique
 * du bâton flottant (socle transparent = min, barre visible = max − min), et la moyenne
 * posée en scatter sur la même catégorie.
 */
import { describe, expect, it } from 'vitest'

import type { MatchKillDistanceWeapon } from '@/lib/api/types'

import { buildKillDistanceOption, killDistanceBars } from './_killDistanceChart'

const WEAPONS: MatchKillDistanceWeapon[] = [
  {
    weapon_key: 'hinf_br75',
    label: 'BR75',
    label_en: 'BR75',
    measured_kills: 3,
    avg_distance_m: 12.4,
    min_distance_m: 3.1,
    max_distance_m: 21.6,
  },
  {
    weapon_key: 'hinf_repulsor',
    label: '',
    label_en: '',
    measured_kills: 1,
    avg_distance_m: 5,
    min_distance_m: 5,
    max_distance_m: 5,
  },
]

const TC = {
  axisLine: '#000',
  splitLine: '#000',
  axisLabel: '#000',
  tooltipBg: '#000',
  tooltipBorder: '#000',
  tooltipText: '#000',
} as never

function option() {
  return buildKillDistanceOption({
    bars: killDistanceBars(WEAPONS, 'fr'),
    tc: TC,
    rangeColor: '#111111',
    avgColor: '#222222',
    fmtDistance: (m) => `${m} m`,
    labels: { kills: 'Frags mesurés', min: 'Plus proche', avg: 'Moyenne', max: 'Plus loin' },
  }) as {
    yAxis: { data: string[] }
    series: { type: string; data: unknown[]; itemStyle?: { color?: string } }[]
  }
}

describe('killDistanceBars — la projection du contrat', () => {
  it('résout le libellé par locale et replie sur weapon_key', () => {
    const fr = killDistanceBars(WEAPONS, 'fr')
    expect(fr.map((b) => b.label)).toEqual(['BR75', 'hinf_repulsor'])
    expect(fr[0]).toMatchObject({ kills: 3, min: 3.1, max: 21.6, avg: 12.4 })
  })
})

describe('buildKillDistanceOption — le bâton flottant', () => {
  it("l'arme la plus utilisée est EN HAUT : l'axe Y inverse l'ordre du backend", () => {
    // ECharts empile les catégories du bas vers le haut — la dernière de la liste est en haut.
    expect(option().yAxis.data).toEqual(['hinf_repulsor ×1', 'BR75 ×3'])
  })

  it('socle transparent = min, barre visible = max − min, moyenne en scatter', () => {
    const [socle, plage, moyenne] = option().series
    expect(socle.itemStyle?.color).toBe('transparent')
    expect(socle.data).toEqual([5, 3.1])
    expect(plage.data).toEqual([0, 21.6 - 3.1])
    expect(moyenne.type).toBe('scatter')
    expect(moyenne.data).toEqual([
      [5, 0],
      [12.4, 1],
    ])
  })

  it("un seul frag mesuré : le bâton dégénère en point (plage 0), le losange reste le témoin", () => {
    const [, plage, moyenne] = option().series
    expect(plage.data[0]).toBe(0)
    expect(moyenne.data[0]).toEqual([5, 0])
  })
})
