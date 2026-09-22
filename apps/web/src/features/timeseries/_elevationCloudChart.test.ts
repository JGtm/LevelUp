/**
 * _elevationCloudChart.test — la géométrie du nuage « distance × dénivelé » (D25, T5).
 *
 * Ce que ces tests cadenassent :
 *
 *  1. L'AXE DES DÉNIVELÉS EST SYMÉTRIQUE. Un axe cadré sur les données ferait paraître un
 *     joueur « toujours en hauteur » parce que sa pire chute est courte.
 *  2. Le halo est le rectangle p25-p75 DU CÔTÉ, et il n'existe pas sans quartile.
 *  3. Le signe reçu est affiché tel quel — jamais réinversé côté web.
 *  4. L'infobulle ne montre AUCUNE clé d'arme brute.
 */
import { describe, expect, it } from 'vitest'

import type { ElevationCloudBlock, TimeseriesMatchRow } from '@/lib/api/types'

import {
  ELEVATION_LEVEL_BAND_M,
  buildElevationCloudOption,
  elevationAxisBounds,
  elevationHalo,
  elevationMatchLabels,
} from './_elevationCloudChart'

const SUMMARY_VIDE = { n: 0 }

const BLOCK: ElevationCloudBlock = {
  kills: [
    { distance_m: 12, delta_z_m: 3.5, match_id: 'm1', time_ms: 1000, weapon: 'br' },
    { distance_m: 30, delta_z_m: 0.2, match_id: 'm2', time_ms: 2000, weapon: 'inconnue' },
  ],
  deaths: [{ distance_m: 8, delta_z_m: -2.5, match_id: 'm1', time_ms: 3000, weapon: 'br' }],
  kills_summary: {
    distance_p25: 10,
    distance_p50: 14,
    distance_p75: 22,
    delta_z_p25: 0.5,
    delta_z_p50: 2.4,
    delta_z_p75: 4,
    n: 2,
  },
  deaths_summary: { ...SUMMARY_VIDE },
  weapon_labels: { br: { label: 'Fusil de combat BR75', label_en: 'BR75 Battle Rifle' } },
  measured_kills: 2,
  total_kills: 10,
  measured_deaths: 1,
  total_deaths: 8,
}

const THEME = {
  axisLabel: '#888888',
  splitLine: '#eeeeee',
  card: '#ffffff',
  text: '#111111',
  tooltipBg: '#ffffff',
  tooltipBorder: '#dddddd',
} as unknown as Parameters<typeof buildElevationCloudOption>[0]['tc']

const INPUT = {
  block: BLOCK,
  tc: THEME,
  colors: { kills: '#2e7d32', deaths: '#c62828', card: '#ffffff' },
  matchLabels: { m1: '#3 · Aquarius' },
  weaponLabels: { br: 'Fusil de combat BR75' },
  fmt: {
    distance: (m: number) => `${m.toFixed(1)} m`,
    signedDistance: (m: number) => `${m > 0 ? '+' : ''}${m.toFixed(1)} m`,
  },
  labels: {
    kills: 'Mes frags',
    deaths: 'Mes morts',
    medianKills: 'médiane frags +2,4 m',
    medianDeaths: 'médiane morts -1,9 m',
    xAxis: "Distance de l'engagement",
    yAxis: 'Hauteur (m)',
    levelBand: 'À niveau (± 1 m)',
  },
}

describe('elevationAxisBounds', () => {
  it("l'axe des dénivelés est SYMÉTRIQUE, des deux côtés de la ligne zéro", () => {
    const { yAbs, xMax } = elevationAxisBounds(BLOCK)
    // Le dénivelé le plus extrême vaut +3,5 m ; la borne le couvre, marge comprise, et elle
    // vaut autant en dessous qu'au-dessus — c'est l'option qui pose min = -yAbs.
    expect(yAbs).toBeGreaterThanOrEqual(3.5)
    expect(xMax).toBeGreaterThanOrEqual(30)
    const option = buildElevationCloudOption(INPUT) as {
      yAxis: { min: number; max: number }
    }
    expect(option.yAxis.min).toBe(-option.yAxis.max)
  })

  it('un nuage entièrement plat contient quand même la bande à niveau', () => {
    const plat: ElevationCloudBlock = {
      ...BLOCK,
      kills: [{ distance_m: 5, delta_z_m: 0, match_id: 'm1', time_ms: 1, weapon: 'br' }],
      deaths: [],
    }
    expect(elevationAxisBounds(plat).yAbs).toBeGreaterThanOrEqual(ELEVATION_LEVEL_BAND_M)
  })
})

describe('elevationHalo', () => {
  it('rend le rectangle p25-p75 du côté', () => {
    expect(elevationHalo(BLOCK.kills_summary)).toEqual({ x0: 10, x1: 22, y0: 0.5, y1: 4 })
  })

  it("un côté SANS quartile n'a pas de halo — jamais un rectangle collé à l'origine", () => {
    expect(elevationHalo(BLOCK.deaths_summary)).toBeNull()
  })
})

describe('buildElevationCloudOption', () => {
  it('un point par frag, avec son côté, son match et son arme — rien n’est agrégé', () => {
    const option = buildElevationCloudOption(INPUT) as { series: { name?: string; data: unknown[] }[] }
    const frags = option.series.find((s) => s.name === 'Mes frags')
    expect(frags?.data).toHaveLength(2)
    expect(frags?.data[0]).toEqual([12, 3.5, 'kills', 'm1', 'br'])
  })

  it('la médiane d’un côté vide n’a AUCUN point (pas de zéro dessiné)', () => {
    const option = buildElevationCloudOption(INPUT) as { series: { name?: string; data: unknown[] }[] }
    expect(option.series.find((s) => s.name === 'médiane morts -1,9 m')?.data).toHaveLength(0)
    expect(option.series.find((s) => s.name === 'médiane frags +2,4 m')?.data).toEqual([[14, 2.4]])
  })

  it("l'infobulle nomme le côté, les deux mesures, l'arme et le match", () => {
    const option = buildElevationCloudOption(INPUT) as {
      tooltip: { formatter: (p: unknown) => string }
    }
    const html = option.tooltip.formatter({ value: [12, 3.5, 'kills', 'm1', 'br'] })
    expect(html).toContain('Mes frags')
    expect(html).toContain('12.0 m')
    // Le signe est celui du contrat : il n'est jamais réinversé ici.
    expect(html).toContain('+3.5 m')
    expect(html).toContain('Fusil de combat BR75')
    expect(html).toContain('#3 · Aquarius')
  })

  it('une arme inconnue du dictionnaire ne montre PAS sa clé brute', () => {
    const option = buildElevationCloudOption(INPUT) as {
      tooltip: { formatter: (p: unknown) => string }
    }
    const html = option.tooltip.formatter({ value: [30, 0.2, 'kills', 'm2', 'inconnue'] })
    expect(html).not.toContain('inconnue')
    // Le match non plus n'est pas inventé : m2 n'a pas d'étiquette dans cette fixture.
    expect(html).not.toContain('m2')
  })
})

describe('elevationMatchLabels', () => {
  it('reprend la numérotation de la page, et la carte quand elle existe', () => {
    const rows = [
      { match_id: 'a', index: 3, map_name: 'Aquarius', map_name_fr: 'Aquarius' },
      { match_id: 'b', index: 4 },
    ] as unknown as TimeseriesMatchRow[]
    expect(elevationMatchLabels(rows)).toEqual({ a: '#3 · Aquarius', b: '#4' })
  })
})
