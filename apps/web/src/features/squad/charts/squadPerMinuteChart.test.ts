/**
 * squadPerMinuteChart.test.ts — teammates.14.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { tipNumber } from '@/components/charts/tipNumber'
import { buildSquadPerMinuteOption, squadHasDamagePerMinute } from './squadPerMinuteChart'
import { hexComplement } from '@/lib/accessibility/hexComplement'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerMinuteEntry } from '@/lib/api/types'

vi.mock('@/lib/accessibility', async () => {
  // hexComplement est une fonction pure — on la passe en real impl dans le mock
  // pour que le chart l'utilise sans avoir besoin de DOM ou de palette active.
  const { hexComplement: hc } = await import('@/lib/accessibility/hexComplement')
  return {
    tokenCssVar: (token: string) => `color:${token}`,
    resolveToken: (token: string) => `hex(${token})`,
    hexComplement: hc,
  }
})

// Couleurs non-grises pour que hexComplement retourne un résultat distinct.
const PLAYER_COLOR = '#818cf8' // indigo — couleur positive du joueur "Me"

const OPTS = {
  colorByPlayer: { Me: PLAYER_COLOR, F1: '#00dc82', F2: '#f59e0b' },
  metricLabels: {
    frags: 'Frags/min',
    deaths: 'Morts/min',
    assists: 'Assists/min',
    dealt: 'Dégâts infligés/min',
    taken: 'Dégâts subis/min',
  },
  damageAxisName: 'Dégâts / min',
  perMinuteSuffix: ' /min',
}

function entry(name: string, k: number, d: number, a: number): SquadPerMinuteEntry {
  return { player: name, kills_per_minute: k, deaths_per_minute: d, assists_per_minute: a, match_count: 5 }
}

function makeSeries(rows: SquadPerMinuteEntry[]): ChartSeries<SquadPerMinuteEntry>[] {
  return [{ key: 'k', datapoints: rows }]
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadPerMinuteOption', () => {
  it('rows vides → option minimale', () => {
    expect(buildSquadPerMinuteOption([], OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 série bar par joueur (3 joueurs → 3 series)', () => {
    const opt = buildSquadPerMinuteOption(
      makeSeries([entry('Me', 1.2, 0.7, 0.3), entry('F1', 0.9, 0.5, 0.2), entry('F2', 1.5, 1.0, 0.5)]),
      OPTS,
    )
    const series = opt.series as Array<{ name: string; type: string }>
    expect(series).toHaveLength(3)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1', 'F2'])
    expect(series.every((s) => s.type === 'bar')).toBe(true)
  })

  it('xAxis = [Frags, Morts, Assists] selon i18n labels', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const xAxis = opt.xAxis as { data: string[] }
    expect(xAxis.data).toEqual(['Frags/min', 'Morts/min', 'Assists/min'])
  })

  it('values : frags positif, morts NÉGATIF (sous l\'axe), assists positif', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1.5, 0.7, 0.3)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ value: number }> }>
    const data = series[0].data
    expect(data[0].value).toBe(1.5)
    expect(data[1].value).toBe(-0.7) // morts négatives
    expect(data[2].value).toBe(0.3)
  })

  it('couleur complémentaire opaque sur morts, couleur joueur sur frags+assists', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string; opacity?: number } }> }>
    const data = series[0].data
    const expectedNeg = hexComplement(PLAYER_COLOR)
    expect(data[0].itemStyle.color).toBe(PLAYER_COLOR)      // frags = couleur joueur
    expect(data[0].itemStyle.opacity).toBeUndefined()
    expect(data[1].itemStyle.color).toBe(expectedNeg)        // morts = complémentaire
    expect(data[1].itemStyle.opacity).toBeUndefined()        // opaque (pas d'opacity)
    expect(data[1].itemStyle.color).not.toBe(PLAYER_COLOR)  // must differ from source
    expect(data[2].itemStyle.color).toBe(PLAYER_COLOR)      // assists = couleur joueur
  })

  it('joueur sans couleur mappée → fallback gris #888, complément de #888', () => {
    const opt = buildSquadPerMinuteOption(
      makeSeries([entry('Unknown', 1, 1, 1)]),
      OPTS,
    )
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>
    expect(series[0].data[0].itemStyle.color).toBe('#888') // frags = gris fallback
    // morts = complément du gris (#888888 → hue 0° → complément = même gris)
    expect(series[0].data[1].itemStyle.color).toBe(hexComplement('#888'))
  })

  it('label formatter retourne valeur ABSOLUE pour les morts (data index 1)', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 0.7, 0.3)]), OPTS)
    const series = opt.series as Array<{
      label: { formatter: (p: { value: number; dataIndex: number }) => string }
    }>
    const fmt = series[0].label.formatter
    expect(fmt({ value: -0.7, dataIndex: 1 })).toBe(tipNumber(0.7, 2))
    expect(fmt({ value: 1.5, dataIndex: 0 })).toBe(tipNumber(1.5, 2))
  })

  it('axe X = ligne foreground en gras (zero line emphasis)', () => {
    // Phase 5e (PLAN_THEME_CONSISTENCY) : la couleur de l'axe zéro est passée
    // de rgba(255,255,255,0.75) à tc.text (CSS var --foreground résolue au
    // runtime). On valide que la largeur est renforcée (2px) et qu'une
    // couleur non vide est posée.
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const xAxis = opt.xAxis as { axisLine: { lineStyle: { color: string; width: number } } }
    expect(xAxis.axisLine.lineStyle.width).toBe(2)
    expect(xAxis.axisLine.lineStyle.color).toBeTruthy()
  })
})

function damageEntry(name: string, dealt?: number, taken?: number): SquadPerMinuteEntry {
  return {
    ...entry(name, 1, 1, 1),
    ...(dealt == null ? {} : { damage_dealt_per_minute: dealt }),
    ...(taken == null ? {} : { damage_taken_per_minute: taken }),
  }
}

interface BarSeries {
  name: string
  yAxisIndex: number
  stack?: string
  data: Array<{ value: number; itemStyle: { color: string } } | null>
  label: { fontSize?: number; formatter: (p: { value: unknown; dataIndex: number }) => string }
}

interface AxisShape {
  min?: number
  max?: number
  position?: string
  splitLine?: { show: boolean }
  axisLabel: { formatter: (v: number) => string }
}

describe('buildSquadPerMinuteOption : étiquettes', () => {
  it("étiquette de barre en 10 px, comme avant l'ajout des dégâts", () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const [me] = opt.series as unknown as BarSeries[]
    expect(me.label.fontSize).toBe(10)
  })
})

describe("buildSquadPerMinuteOption : dégâts sur l'axe de droite", () => {
  it('sans dégâts : trois catégories, une série par joueur, un seul axe', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    expect((opt.xAxis as { data: string[] }).data).toHaveLength(3)
    expect(opt.series as unknown[]).toHaveLength(1)
    expect(opt.yAxis as unknown[]).toHaveLength(1)
  })

  it('avec dégâts : deux catégories de plus dans la même grille, une série par axe et par joueur', () => {
    const opt = buildSquadPerMinuteOption(
      makeSeries([damageEntry('Me', 312.6, 240.4), damageEntry('F1', 200, 150)]),
      OPTS,
    )
    expect((opt.xAxis as { data: string[] }).data).toEqual([
      'Frags/min',
      'Morts/min',
      'Assists/min',
      'Dégâts infligés/min',
      'Dégâts subis/min',
    ])
    const series = opt.series as unknown as BarSeries[]
    expect(series.map((s) => [s.name, s.yAxisIndex])).toEqual([
      ['Me', 0],
      ['Me', 1],
      ['F1', 0],
      ['F1', 1],
    ])
    // Les deux séries d'un joueur partagent une colonne (même pile) ; deux joueurs, deux colonnes.
    expect(series[0].stack).toBe('Me')
    expect(series[1].stack).toBe('Me')
    expect(series[2].stack).toBe('F1')
  })

  it("dégâts infligés au-dessus (couleur joueur), subis sous le zéro (complément), trous de l'autre axe", () => {
    const opt = buildSquadPerMinuteOption(makeSeries([damageEntry('Me', 312.6, 240.4)]), OPTS)
    const [left, right] = opt.series as unknown as BarSeries[]
    expect(left.data.map((d) => d?.value ?? null)).toEqual([1, -1, 1, null, null])
    expect(right.data.map((d) => d?.value ?? null)).toEqual([null, null, null, 312.6, -240.4])
    expect(right.data[3]).toMatchObject({ itemStyle: { color: PLAYER_COLOR } })
    expect(right.data[4]).toMatchObject({ itemStyle: { color: hexComplement(PLAYER_COLOR) } })
  })

  it('zéros alignés, axe de droite sans grille', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([damageEntry('Me', 312.6, 240.4)]), OPTS)
    const [l, r] = opt.yAxis as unknown as AxisShape[]
    expect(r.position).toBe('right')
    expect(r.splitLine).toEqual({ show: false })
    const share = (a: AxisShape) => -a.min! / (a.max! - a.min!)
    expect(share(l)).toBeCloseTo(share(r), 10)
    expect(r.max!).toBeGreaterThanOrEqual(312.6)
    expect(r.min!).toBeLessThanOrEqual(-240.4)
  })

  it('sans dégâts subis (titre qui ne les fournit pas) : une seule catégorie de dégâts', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([damageEntry('Me', 300)]), OPTS)
    expect((opt.xAxis as { data: string[] }).data).toEqual([
      'Frags/min',
      'Morts/min',
      'Assists/min',
      'Dégâts infligés/min',
    ])
  })

  it('un joueur sans dégâts connus laisse un trou, pas un zéro', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([damageEntry('Me', 300, 100), damageEntry('F1')]), OPTS)
    const series = opt.series as unknown as BarSeries[]
    const f1Right = series[3]
    expect(f1Right.name).toBe('F1')
    expect(f1Right.data.map((d) => d?.value ?? null)).toEqual([null, null, null, null, null])
    expect(f1Right.label.formatter({ value: null, dataIndex: 3 })).toBe('')
  })

  it('un joueur sans frags, morts ni assistances connus laisse un trou sur ces catégories, pas un zéro', () => {
    const sansComptes: SquadPerMinuteEntry = { player: 'F1', match_count: 0 }
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1.2, 0.7, 0.3), sansComptes]), OPTS)
    const [me, f1] = opt.series as unknown as BarSeries[]
    expect(me.data.map((d) => d?.value ?? null)).toEqual([1.2, -0.7, 0.3])
    expect(f1.name).toBe('F1')
    expect(f1.data.map((d) => d?.value ?? null)).toEqual([null, null, null])
  })

  it('décimales : entier pour les dégâts, deux pour les comptes (étiquette et axe)', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([damageEntry('Me', 312.6, 240.4)]), OPTS)
    const [left, right] = opt.series as unknown as BarSeries[]
    expect(left.label.formatter({ value: -1.234, dataIndex: 1 })).toBe(tipNumber(1.234, 2))
    expect(right.label.formatter({ value: -240.4, dataIndex: 4 })).toBe(tipNumber(240.4, 0))
    const [l, r] = opt.yAxis as unknown as AxisShape[]
    expect(l.axisLabel.formatter(-1.5)).toBe(tipNumber(1.5, 2))
    expect(r.axisLabel.formatter(-300)).toBe(tipNumber(300, 0))
  })
})

describe('squadHasDamagePerMinute', () => {
  it('vrai dès qu un joueur porte une cadence de dégâts', () => {
    expect(squadHasDamagePerMinute([damageEntry('Me'), damageEntry('F1', undefined, 100)])).toBe(true)
  })
  it('faux sans dégâts', () => {
    expect(squadHasDamagePerMinute([damageEntry('Me')])).toBe(false)
    expect(squadHasDamagePerMinute([])).toBe(false)
  })
})
