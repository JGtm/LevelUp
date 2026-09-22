import { describe, expect, it } from 'vitest'

import type { MatchElevationKill } from '@/lib/api/types'

import {
  ELEVATION_LEVEL_BAND_M,
  ELEVATION_MIN_HALF_SPAN_M,
  ELEVATION_SERIES_LOBBY,
  buildElevationOption,
  elevationBounds,
  elevationPoints,
  elevationTooltipLines,
  type ElevationPoint,
} from './_elevation'

const tc = {
  axisLabel: 'a', axisLine: 'b', splitLine: 'c', splitAreaA: 'd', splitAreaB: 'e',
  text: 'f', tooltipBg: 'g', tooltipBorder: 'h',
} as never

function kill(over: Partial<MatchElevationKill> = {}): MatchElevationKill {
  return {
    distance_m: 10, delta_z_m: 2, time_ms: 1000, side: 'kill',
    weapon: 'Fusil', weapon_en: 'Rifle', opponent: 'Ennemi', ...over,
  } as MatchElevationKill
}

function pt(x: number, y: number): ElevationPoint {
  return { value: [x, y], timeMs: 0, side: 'kill', weapon: '', opponent: '' }
}

function option(kills: ElevationPoint[], lobby: ElevationPoint[], median: number | null) {
  return buildElevationOption({
    kills, deaths: [], lobby, lobbyMedian: median, tc,
    colors: { kills: 'k', deaths: 'd', lobby: 'l', band: 'b', zero: 'z' },
    labels: {
      kills: 'Mes frags', deaths: 'Mes morts', lobby: 'Lobby',
      xAxis: 'Distance', yAxis: 'Hauteur (m)', lobbyMedian: (m) => `Médiane ${m}`,
    },
    tooltip: () => 'tt',
  }) as Record<string, never>
}

describe('elevationPoints', () => {
  it('prend le libellé d’arme de la locale', () => {
    expect(elevationPoints([kill()], 'fr')[0].weapon).toBe('Fusil')
    expect(elevationPoints([kill()], 'en')[0].weapon).toBe('Rifle')
  })

  it('écarte une ligne sans géométrie lisible plutôt que de la poser à zéro', () => {
    const bad = kill({ delta_z_m: Number.NaN })
    expect(elevationPoints([bad, kill()], 'fr')).toHaveLength(1)
  })

  it('rend une liste vide sans bloc', () => {
    expect(elevationPoints(undefined, 'fr')).toEqual([])
  })
})

describe('elevationBounds', () => {
  it('est symétrique autour de zéro et ne descend pas sous le plancher', () => {
    expect(elevationBounds([pt(4, 0.2)])).toEqual({ yMax: ELEVATION_MIN_HALF_SPAN_M, xMax: 4 })
  })

  it('prend l’extrême des deux signes, arrondi au mètre supérieur', () => {
    expect(elevationBounds([pt(1, -7.2)], [pt(30, 4)])).toEqual({ yMax: 8, xMax: 30 })
  })
})

describe('buildElevationOption', () => {
  it('cadre l’ordonnée symétriquement et pose la bande à ±1 m avec la ligne de zéro', () => {
    const o = option([pt(10, 5)], [], null)
    const y = o.yAxis as { min: number; max: number }
    expect([y.min, y.max]).toEqual([-5, 5])
    const s0 = (o.series as Record<string, never>[])[0] as unknown as {
      markArea: { data: [{ yAxis: number }, { yAxis: number }][] }
      markLine: { data: { yAxis: number }[] }
    }
    expect(s0.markArea.data[0].map((d) => d.yAxis)).toEqual([
      -ELEVATION_LEVEL_BAND_M, ELEVATION_LEVEL_BAND_M,
    ])
    expect(s0.markLine.data[0].yAxis).toBe(0)
  })

  it('n’ajoute la série de fond et le repère de médiane que si le lobby est demandé', () => {
    const sans = option([pt(10, 1)], [], 0.4)
    expect((sans.series as { id: string }[]).map((s) => s.id)).not.toContain(ELEVATION_SERIES_LOBBY)

    const avec = option([pt(10, 1)], [pt(20, -2)], 0.4)
    const series = avec.series as { id: string; silent?: boolean }[]
    expect(series[0].id).toBe(ELEVATION_SERIES_LOBBY)
    // Le fond ne capture ni survol ni clic : sinon les points du joueur deviennent
    // inatteignables sous 90 points gris.
    expect(series[0].silent).toBe(true)
    const morts = series[2] as unknown as { markLine: { data: { yAxis: number }[] } }
    expect(morts.markLine.data[0].yAxis).toBe(0.4)
  })
})

describe('elevationTooltipLines', () => {
  it('échappe les données non constantes et saute les lignes vides', () => {
    expect(elevationTooltipLines(['<b>x</b>', '', 'y'])).toBe('&lt;b&gt;x&lt;/b&gt;<br>y')
  })
})
