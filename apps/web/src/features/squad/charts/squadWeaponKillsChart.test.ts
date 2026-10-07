/**
 * squadWeaponKillsChart.test.ts — barres groupées par joueur (Outils de destruction,
 * Mécaniques de frag). Forme du lot L2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 :
 * compte au bout de chaque barre, pastille de classe devant le nom, part du joueur en
 * infobulle ; mode `share` conservé pour les Mécaniques de frag.
 */
import { describe, it, expect, vi } from 'vitest'
import { buildSquadWeaponKillsOption, type SquadBarRows } from './squadWeaponKillsChart'

vi.mock('@/lib/accessibility/scales', () => ({
  fragClassColor: (cls: string) => `hex:${cls}`,
}))

const COLORS = { Me: '#aaa', F1: '#bbb' }

function data(): SquadBarRows {
  return {
    players: ['Me', 'F1'],
    rows: [
      { key: 'weapon:sniper', label: 'Sniper', cls: 'heavy', killsByPlayer: { Me: 2, F1: 0 }, total: 2 },
      { key: 'weapon:br', label: 'BR75', cls: 'shoulder', killsByPlayer: { Me: 30, F1: 25 }, total: 55 },
      { key: 'melee', label: 'Mêlée', cls: 'melee', killsByPlayer: { Me: 80, F1: 60 }, total: 140 },
    ],
  }
}

type LabelFormatter = (p: { value: unknown; dataIndex: number }) => string
type Serie = { name: string; type: string; data: number[]; itemStyle: { color: string }; label: { formatter: LabelFormatter } }

describe('buildSquadWeaponKillsOption', () => {
  it('vide ou sans ligne → option minimale', () => {
    expect(buildSquadWeaponKillsOption(null, { colorByPlayer: COLORS })).toMatchObject({ backgroundColor: 'transparent' })
    expect(buildSquadWeaponKillsOption({ players: ['Me'], rows: [] }, { colorByPlayer: COLORS })).toMatchObject({
      backgroundColor: 'transparent',
    })
  })

  it('une série bar par joueur, valeurs alignées sur les lignes, 0 si absent', () => {
    const series = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS }).series as Serie[]
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series.every((s) => s.type === 'bar')).toBe(true)
    expect(series[0].data).toEqual([2, 30, 80])
    expect(series[1].data).toEqual([0, 25, 60])
  })

  it('couleur par joueur, repli gris structurel sans couleur', () => {
    const series = buildSquadWeaponKillsOption(data(), { colorByPlayer: { Me: '#aaa' } }).series as Serie[]
    expect(series[0].itemStyle.color).toBe('#aaa')
    expect(series[1].itemStyle.color).toBe('#888')
  })

  it('compte au bout de CHAQUE barre non nulle, même la plus petite (plus de seuil de part)', () => {
    const series = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS }).series as Serie[]
    const me = series[0].label.formatter
    expect(me({ value: 2, dataIndex: 0 })).toBe('2')
    expect(me({ value: 80, dataIndex: 2 })).toBe('80')
    expect(series[1].label.formatter({ value: 0, dataIndex: 0 })).toBe('')
  })

  it('pastille de la classe devant le nom de la ligne (texte riche), couleur de la classe', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const yAxis = opt.yAxis as {
      data: string[]
      splitArea: { show: boolean }
      axisLabel: { formatter: (v: string, i: number) => string; rich: Record<string, { backgroundColor?: string }> }
    }
    expect(yAxis.data).toEqual(['weapon:sniper', 'weapon:br', 'melee'])
    expect(yAxis.splitArea.show).toBe(true)
    expect(yAxis.axisLabel.formatter('weapon:br', 1)).toBe('{c_shoulder|}{name|BR75}')
    expect(yAxis.axisLabel.rich.c_shoulder.backgroundColor).toBe('hex:shoulder')
    expect(yAxis.axisLabel.rich.c_melee.backgroundColor).toBe('hex:melee')
  })

  it('ligne sans classe → nom seul ; un nom ne casse pas le texte riche', () => {
    const d: SquadBarRows = {
      players: ['Me'],
      rows: [{ key: 'x', label: 'A{b}|c', killsByPlayer: { Me: 1 }, total: 1 }],
    }
    const yAxis = buildSquadWeaponKillsOption(d, { colorByPlayer: COLORS }).yAxis as {
      axisLabel: { formatter: (v: string, i: number) => string }
    }
    expect(yAxis.axisLabel.formatter('x', 0)).toBe('A b  c')
  })

  it('xAxis caché, légende ECharts désactivée (légende des joueurs hors canvas)', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    expect((opt.xAxis as { show: boolean }).show).toBe(false)
    expect((opt.legend as { show: boolean }).show).toBe(false)
  })

  it('infobulle : compte ET part du total du joueur, texte fourni par l’appelant, zéros exclus', () => {
    const opt = buildSquadWeaponKillsOption(data(), {
      colorByPlayer: COLORS,
      valueText: (k, pct) => `${k} frags (${pct} % des siens)`,
    })
    const html = (opt.tooltip as { formatter: (raw: unknown) => string }).formatter([
      { seriesName: 'Me', value: 30, marker: '', dataIndex: 1 },
      { seriesName: 'F1', value: 0, marker: '', dataIndex: 1 },
    ])
    expect(html).toContain('BR75')
    expect(html).toContain('Me: 30 frags (27 % des siens)') // 30 / 112
    expect(html).not.toContain('F1')
  })

  it('un seul joueur (Sessions) : chaque barre à la couleur de sa classe, infobulle sans nom de joueur', () => {
    const d: SquadBarRows = {
      players: ['Me'],
      rows: [
        { key: 'weapon:br', label: 'BR75', cls: 'shoulder', killsByPlayer: { Me: 30 }, total: 30 },
        { key: 'x', label: 'Sans classe', killsByPlayer: { Me: 2 }, total: 2 },
      ],
    }
    const opt = buildSquadWeaponKillsOption(d, { colorByPlayer: COLORS, soloByClass: true })
    const [me] = opt.series as { data: unknown[] }[]
    expect(me.data).toEqual([{ value: 30, itemStyle: { color: 'hex:shoulder' } }, 2])
    const html = (opt.tooltip as { formatter: (raw: unknown) => string }).formatter([
      { seriesName: 'Me', value: 30, marker: '', dataIndex: 0 },
    ])
    expect(html).toContain('BR75')
    expect(html).not.toContain('Me')
  })

  describe('mode `share` (Mécaniques de frag, inchangé)', () => {
    it('part du total du joueur au bout des barres assez larges, masquée sous 5 %', () => {
      const series = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS, valueLabel: 'share' })
        .series as Serie[]
      const me = series[0].label.formatter
      expect(me({ value: 30, dataIndex: 1 })).toBe('27 %')
      expect(me({ value: 2, dataIndex: 0 })).toBe('')
    })

    it('dénominateur fourni (vue compacte de Sessions : la part de TOUS mes frags) et aucun seuil', () => {
      const series = buildSquadWeaponKillsOption(data(), {
        colorByPlayer: COLORS,
        valueLabel: 'share',
        shareTotals: { Me: 200 },
        minLabelShare: 0,
      }).series as Serie[]
      const me = series[0].label.formatter
      expect(me({ value: 30, dataIndex: 1 })).toBe('15 %')
      expect(me({ value: 2, dataIndex: 0 })).toBe('1 %')
    })

    it('infobulle par défaut : valeur en gras + part', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS, valueLabel: 'share' })
      const html = (opt.tooltip as { formatter: (raw: unknown) => string }).formatter([
        { seriesName: 'F1', value: 25, marker: '', dataIndex: 1 },
      ])
      expect(html).toContain('<b>25</b> (29 %)') // 25 / 85
    })
  })
})
