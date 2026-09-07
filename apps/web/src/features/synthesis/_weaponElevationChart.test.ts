/**
 * _weaponElevationChart.test — les deux barres empilées à 100 %, en pur.
 *
 * Ce que ces tests verrouillent : SIX séries (trois classes × deux côtés) en deux piles
 * nommées, les MÊMES catégories et le MÊME ordre que le graphe de portée (les deux se lisent
 * ligne à ligne), le côté absent qui rend une barre VIDE et non un segment inventé, le seuil
 * d'inscription du pourcentage dans le segment, et l'infobulle qui dit « aucune mesure ».
 */
import { describe, expect, it } from 'vitest'

import type { EChartsThemeColors } from '@/lib/echarts/themeColors'
import type { WeaponRangeRow, WeaponRangeSide } from '@/lib/api/types'

import { ELEVATION_KEYS, buildWeaponElevationOption } from './_weaponElevationChart'
import { buildWeaponRangeOption, weaponRangeLines, type WeaponRangeLine } from './_weaponRangeChart'

const TC: EChartsThemeColors = {
  axisLabel: '#111111',
  axisLine: '#222222',
  splitLine: '#222222',
  splitAreaA: 'rgba(0,0,0,0.1)',
  splitAreaB: 'rgba(0,0,0,0.2)',
  text: '#333333',
  tooltipBg: '#444444',
  tooltipBorder: '#555555',
  card: '#666666',
  isDark: true,
}

const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 10,
  p10: 5,
  median: 7,
  p90: 10,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

const ROWS: WeaponRangeRow[] = [
  {
    weapon_key: 'hinf_br75',
    label: 'BR75',
    label_en: 'BR75',
    kills: side({ measured: 281, above_pct: 37, level_pct: 49, below_pct: 14 }),
    deaths: side({ measured: 402, above_pct: 10, level_pct: 52, below_pct: 38 }),
  },
  { weapon_key: 'hinf_commando', kills: side({ measured: 64, above_pct: 17, level_pct: 65, below_pct: 18 }) },
]

const LINES = weaponRangeLines(ROWS, 'fr')

interface Serie {
  name: string
  type: string
  stack: string
  barWidth: number
  barGap: string
  data: number[]
  itemStyle: { color: string; borderColor: string }
  label: { formatter: (p: { value: number }) => string }
}

function optionOf(lines: WeaponRangeLine[]) {
  return buildWeaponElevationOption({
    lines,
    tc: TC,
    colors: { above: '#111', level: '#777', below: '#eee' },
    cardColor: '#ffffff',
    fmtPercent: (v) => `${Math.round(v)} %`,
    labels: {
      kills: 'Mes frags',
      deaths: 'Mes morts',
      noMeasure: 'aucune mesure',
      segments: { above: "d'en haut", level: 'à niveau', below: "d'en bas" },
    },
  }) as {
    xAxis: { max: number }
    yAxis: { data: string[] }
    tooltip: { formatter: (p: unknown) => string }
    series: Serie[]
  }
}

describe('buildWeaponElevationOption — deux piles de trois segments', () => {
  it('six séries : les trois classes des frags, puis celles des morts', () => {
    const s = optionOf(LINES).series
    expect(s).toHaveLength(6)
    expect(s.map((x) => x.stack)).toEqual(['kills', 'kills', 'kills', 'deaths', 'deaths', 'deaths'])
    expect(s.map((x) => x.name)).toEqual([
      "d'en haut",
      'à niveau',
      "d'en bas",
      "d'en haut",
      'à niveau',
      "d'en bas",
    ])
    expect(ELEVATION_KEYS).toEqual(['above', 'level', 'below'])
    expect(s.every((x) => x.type === 'bar' && x.barWidth === 7 && x.barGap === '55%')).toBe(true)
  })

  it('les MÊMES catégories, dans le MÊME ordre que le graphe de portée', () => {
    const portee = buildWeaponRangeOption({
      lines: LINES,
      tc: TC,
      killsColor: '#a',
      deathsColor: '#b',
      medianColor: '#c',
      cardColor: '#d',
      fmtDistance: (m) => `${m} m`,
      labels: { kills: 'f', deaths: 'm', percentiles: 'p', noMeasure: 'n' },
    }) as { yAxis: { data: string[] } }
    expect(optionOf(LINES).yAxis.data).toEqual(portee.yAxis.data)
    expect(optionOf(LINES).yAxis.data).toEqual(['hinf_commando ×64/—', 'BR75 ×281/402'])
    expect(optionOf(LINES).xAxis.max).toBe(100)
  })

  it('les parts vont dans leur segment ; un côté absent rend une barre VIDE (0), pas un segment inventé', () => {
    // L'axe est inversé : index 0 = Commando (frags seuls), index 1 = BR75.
    const s = optionOf(LINES).series
    expect(s[0].data).toEqual([17, 37])
    expect(s[1].data).toEqual([65, 49])
    expect(s[2].data).toEqual([18, 14])
    expect(s[3].data).toEqual([0, 10])
    expect(s[4].data).toEqual([0, 52])
    expect(s[5].data).toEqual([0, 38])
  })

  it('le pourcentage n’est inscrit qu’à partir de 18 % — en dessous il déborderait', () => {
    const fmt = optionOf(LINES).series[0].label.formatter
    expect(fmt({ value: 18 })).toBe('18 %')
    expect(fmt({ value: 17.9 })).toBe('')
    expect(fmt({ value: 0 })).toBe('')
  })

  it('chaque segment est séparé de son voisin par le fond de carte, jamais par une teinte neuve', () => {
    expect(optionOf(LINES).series.every((s) => s.itemStyle.borderColor === '#ffffff')).toBe(true)
    expect(optionOf(LINES).series.map((s) => s.itemStyle.color)).toEqual([
      '#111',
      '#777',
      '#eee',
      '#111',
      '#777',
      '#eee',
    ])
  })
})

describe('infobulle du dénivelé', () => {
  const tooltip = (dataIndex: number) => optionOf(LINES).tooltip.formatter([{ dataIndex }])

  it('rend les trois parts des deux côtés', () => {
    const html = tooltip(1)
    expect(html).toContain('<b>BR75</b>')
    // Les libellés de classe passent par `escapeHtml` comme le reste : l'apostrophe
    // devient `&#39;` dans le HTML de l'infobulle (et se réaffiche telle quelle à l'écran).
    expect(html).toContain('Mes frags : 37 % d&#39;en haut · 49 % à niveau · 14 % d&#39;en bas')
    expect(html).toContain('Mes morts : 10 % d&#39;en haut · 52 % à niveau · 38 % d&#39;en bas')
  })

  it('dit « aucune mesure » pour le côté absent', () => {
    expect(tooltip(0)).toContain('Mes morts : aucune mesure')
  })

  it('échappe le nom d’arme et supporte un paramètre non tableau', () => {
    const hostile = weaponRangeLines([{ weapon_key: 'x', label: '<b>x</b>', kills: side({}) }], 'fr')
    const html = optionOf(hostile).tooltip.formatter({ dataIndex: 0 })
    expect(html).toContain('&lt;b&gt;x&lt;/b&gt;')
  })

  it('rend une chaîne vide hors des lignes connues', () => {
    expect(tooltip(42)).toBe('')
  })
})
