/**
 * squadPerMinuteChart.test.ts — teammates.14.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { tipNumber } from '@/components/charts/tipNumber'
import {
  buildSquadDamagePerMinuteOption,
  buildSquadPerMinuteOption,
  squadHasDamagePerMinute,
} from './squadPerMinuteChart'
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
  metricLabels: { frags: 'Frags/min', deaths: 'Morts/min', assists: 'Assists/min' },
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

const DAMAGE_OPTS = {
  colorByPlayer: OPTS.colorByPlayer,
  metricLabels: { dealt: 'Dégâts infligés/min', taken: 'Dégâts subis/min' },
  perMinuteSuffix: ' /min',
}

function damageEntry(name: string, dealt?: number, taken?: number): SquadPerMinuteEntry {
  return {
    ...entry(name, 1, 1, 1),
    ...(dealt == null ? {} : { damage_dealt_per_minute: dealt }),
    ...(taken == null ? {} : { damage_taken_per_minute: taken }),
  }
}

interface BarSeries {
  name: string
  data: Array<{ value: number; itemStyle: { color: string } } | null>
  label: { formatter: (p: { value: unknown }) => string }
}

describe('buildSquadDamagePerMinuteOption', () => {
  it('rows vides ou sans aucun dégât : option minimale', () => {
    expect(buildSquadDamagePerMinuteOption([], DAMAGE_OPTS)).toMatchObject({ backgroundColor: 'transparent' })
    expect(buildSquadDamagePerMinuteOption(makeSeries([damageEntry('Me')]), DAMAGE_OPTS)).toMatchObject({
      backgroundColor: 'transparent',
    })
  })

  it('deux catégories ; dégâts subis NÉGATIFS en couleur complémentaire, infligés en couleur joueur', () => {
    const opt = buildSquadDamagePerMinuteOption(makeSeries([damageEntry('Me', 312.6, 240.4)]), DAMAGE_OPTS)
    expect((opt.xAxis as { data: string[] }).data).toEqual(['Dégâts infligés/min', 'Dégâts subis/min'])
    const [me] = opt.series as unknown as BarSeries[]
    expect(me.data[0]).toMatchObject({ value: 312.6, itemStyle: { color: PLAYER_COLOR } })
    expect(me.data[1]).toMatchObject({ value: -240.4, itemStyle: { color: hexComplement(PLAYER_COLOR) } })
  })

  it('un joueur sans dégâts connus laisse un trou, pas un zéro ; sans dégâts subis du tout, la catégorie disparaît', () => {
    const opt = buildSquadDamagePerMinuteOption(
      makeSeries([damageEntry('Me', 300), damageEntry('F1')]),
      DAMAGE_OPTS,
    )
    expect((opt.xAxis as { data: string[] }).data).toEqual(['Dégâts infligés/min'])
    const [me, f1] = opt.series as unknown as BarSeries[]
    expect(me.data).toHaveLength(1)
    expect(f1.data).toEqual([null])
    expect(f1.label.formatter({ value: null })).toBe('')
  })

  it('étiquette : entier à la langue, valeur absolue', () => {
    const opt = buildSquadDamagePerMinuteOption(makeSeries([damageEntry('Me', 312.6, 240.4)]), DAMAGE_OPTS)
    const [me] = opt.series as unknown as BarSeries[]
    expect(me.label.formatter({ value: -240.4 })).toBe(tipNumber(240.4, 0))
    expect(me.label.formatter({ value: 312.6 })).toBe(tipNumber(312.6, 0))
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
