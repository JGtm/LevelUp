/**
 * _weaponRangeChart.test — les deux bâtons p10→p90 et leurs losanges, en pur.
 *
 * Ce que ces tests verrouillent : la projection (libellé par locale, côté absent normalisé
 * en `null`, ORDRE DU BACKEND conservé, pseudo-armes sans portée écartées), le libellé de
 * ligne réduit au nom de l'arme, l'inversion de l'axe Y au montage, la géométrie exacte du `renderItem` (deux
 * rectangles décalés de part et d'autre du centre de bande, un losange par médiane), le cas
 * « un seul côté mesuré » (un seul bâton, l'infobulle dit « aucune mesure »), et
 * l'échappement HTML des noms d'armes dans l'infobulle.
 */
import { describe, expect, it } from 'vitest'

import type { EChartsThemeColors } from '@/lib/echarts/themeColors'
import type { WeaponRangeRow, WeaponRangeSide } from '@/lib/api/types'

import {
  WEAPON_RANGE_ROW_PX,
  buildWeaponRangeOption,
  weaponRangeAxisMax,
  weaponRangeCategoryLabel,
  weaponRangeChartHeight,
  weaponRangeLines,
  type WeaponRangeLine,
} from './_weaponRangeChart'

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
    weapon_key: 'hinf_melee',
    label: 'Mêlée',
    label_en: 'Melee',
    kills: side({ measured: 133, p10: 0.9, median: 1.4, p90: 2.6 }),
    deaths: side({ measured: 96, p10: 0.9, median: 1.5, p90: 2.8 }),
  },
  {
    // Un seul côté mesuré, et pas de libellé résolu : les deux cas rares sur la même ligne.
    weapon_key: 'hinf_commando',
    kills: side({ measured: 64, p10: 6.3, median: 11.9, p90: 21.4 }),
  },
  {
    weapon_key: 'hinf_s7',
    label: 'Fusil de précision S7',
    label_en: 'S7 Sniper',
    deaths: side({ measured: 117, p10: 19.6, median: 33.1, p90: 52 }),
  },
]

function optionOf(lines: WeaponRangeLine[]) {
  return buildWeaponRangeOption({
    lines,
    tc: TC,
    killsColor: '#aa0000',
    deathsColor: '#00aa00',
    medianColor: '#0000aa',
    cardColor: '#ffffff',
    fmtDistance: (m) => `${m} m`,
    labels: {
      kills: 'Mes frags',
      deaths: 'Mes morts',
      percentiles: 'p10 · médiane · p90',
      noMeasure: 'aucune mesure',
    },
  }) as {
    xAxis: { max: number }
    yAxis: { data: string[] }
    tooltip: { formatter: (p: unknown) => string }
    series: {
      type: string
      data: number[][]
      renderItem: (p: { dataIndex: number }, api: FakeApi) => RenderedGroup
    }[]
  }
}

/** API `renderItem` factice : x doublé, une bande de 34 px par catégorie à partir de 100. */
interface FakeApi {
  coord(p: [number, number]): number[]
}
const API: FakeApi = { coord: ([v, i]) => [v * 2, 100 + i * WEAPON_RANGE_ROW_PX] }

interface RenderedChild {
  type: string
  shape: Record<string, number | number[][]>
  style: Record<string, unknown>
}
interface RenderedGroup {
  type: string
  children: RenderedChild[]
}

describe('weaponRangeLines — la projection du contrat', () => {
  it('conserve l’ordre du backend, résout le libellé et normalise le côté absent en null', () => {
    const fr = weaponRangeLines(ROWS, 'fr')
    expect(fr.map((l) => l.weaponKey)).toEqual(['hinf_melee', 'hinf_commando', 'hinf_s7'])
    expect(fr.map((l) => l.label)).toEqual(['Mêlée', 'hinf_commando', 'Fusil de précision S7'])
    expect(fr[1].deaths).toBeNull()
    expect(fr[2].kills).toBeNull()
    expect(weaponRangeLines(ROWS, 'en')[0].label).toBe('Melee')
  })

  it('accepte une liste absente (le contrat rend `weapons` nullable)', () => {
    expect(weaponRangeLines(null, 'fr')).toEqual([])
    expect(weaponRangeLines(undefined, 'fr')).toEqual([])
  })

  it('écarte les pseudo-armes dont la DISTANCE n’a pas de sens (chute et environnement)', () => {
    // Sans ce filtre la ligne « Chute et environnement » occupait une bande de l'axe avec
    // une « portée » qui mesure la géométrie du décor, pas un engagement.
    const rows: WeaponRangeRow[] = [
      { weapon_key: 'hinf_environment', label: 'Chute et environnement', kills: side({}) },
      ...ROWS,
    ]
    expect(weaponRangeLines(rows, 'fr').map((l) => l.weaponKey)).toEqual([
      'hinf_melee',
      'hinf_commando',
      'hinf_s7',
    ])
  })
})

describe('weaponRangeCategoryLabel — le nom de l’arme, nu', () => {
  it('n’écrit QUE le libellé — les effectifs vivent dans l’infobulle et le tableau', () => {
    const [melee, commando, s7] = weaponRangeLines(ROWS, 'fr')
    expect(weaponRangeCategoryLabel(melee)).toBe('Mêlée')
    // Repli sur la clé quand le registre n'a rien résolu : le trou doit rester VISIBLE.
    expect(weaponRangeCategoryLabel(commando)).toBe('hinf_commando')
    expect(weaponRangeCategoryLabel(s7)).toBe('Fusil de précision S7')
  })
})

describe('weaponRangeChartHeight / weaponRangeAxisMax', () => {
  it('une bande de 34 px par arme, plus la place des axes', () => {
    expect(weaponRangeChartHeight(0)).toBe(48)
    expect(weaponRangeChartHeight(10)).toBe(48 + 34 * 10)
  })

  it('l’axe est COMMUN aux deux côtés : il couvre le plus grand p90, arrondi aux 5 m', () => {
    expect(weaponRangeAxisMax(weaponRangeLines(ROWS, 'fr'))).toBe(55)
  })

  it('plancher de 5 m : un corpus tout au contact ne dégénère pas l’axe', () => {
    expect(weaponRangeAxisMax([])).toBe(5)
    expect(weaponRangeAxisMax(weaponRangeLines([{ weapon_key: 'x', kills: side({ p90: 1.2 }) }], 'fr'))).toBe(5)
  })
})

describe('buildWeaponRangeOption — l’axe et la série', () => {
  it('la première arme du backend est EN HAUT : l’axe Y inverse la liste', () => {
    const o = optionOf(weaponRangeLines(ROWS, 'fr'))
    expect(o.yAxis.data).toEqual([
      'Fusil de précision S7',
      'hinf_commando',
      'Mêlée',
    ])
    expect(o.xAxis.max).toBe(55)
    expect(o.series[0].type).toBe('custom')
    // Une donnée par ligne, porteuse de la borne d'axe (dimensions encodées en x).
    expect(o.series[0].data).toEqual([
      [0, 0, 55],
      [1, 0, 55],
      [2, 0, 55],
    ])
  })
})

describe('renderItem — la géométrie des deux bâtons', () => {
  const lines = weaponRangeLines(ROWS, 'fr')
  // Après inversion, l'index 2 est la Mêlée (deux côtés mesurés).
  const melee = () => optionOf(lines).series[0].renderItem({ dataIndex: 2 }, API)

  it('deux rectangles et deux losanges, décalés de part et d’autre du centre de bande', () => {
    const g = melee()
    expect(g.type).toBe('group')
    expect(g.children.map((c) => c.type)).toEqual(['rect', 'polygon', 'rect', 'polygon'])
    const yCenter = 100 + 2 * WEAPON_RANGE_ROW_PX
    // Décalage = demi-hauteur de bâton (3,5) + demi-écart (2) = 5,5 px, frags AU-DESSUS.
    expect(g.children[0].shape.y).toBe(yCenter - 5.5 - 3.5)
    expect(g.children[2].shape.y).toBe(yCenter + 5.5 - 3.5)
    expect(g.children[0].style.fill).toBe('#aa0000')
    expect(g.children[2].style.fill).toBe('#00aa00')
  })

  it('le bâton court de p10 à p90 (coordonnées de l’API, jamais recalculées)', () => {
    const rect = melee().children[0]
    expect(rect.shape.x).toBe(0.9 * 2)
    expect(rect.shape.width).toBeCloseTo((2.6 - 0.9) * 2, 10)
    expect(rect.shape.height).toBe(7)
  })

  it('le losange est centré sur la médiane et bordé du fond de carte', () => {
    const losange = melee().children[1]
    const cy = 100 + 2 * WEAPON_RANGE_ROW_PX - 5.5
    expect(losange.shape.points).toEqual([
      [1.4 * 2, cy - 5],
      [1.4 * 2 + 5, cy],
      [1.4 * 2, cy + 5],
      [1.4 * 2 - 5, cy],
    ])
    expect(losange.style.stroke).toBe('#ffffff')
  })

  it('un côté non mesuré ne dessine RIEN de son côté (un seul bâton)', () => {
    const commando = optionOf(lines).series[0].renderItem({ dataIndex: 1 }, API)
    expect(commando.children.map((c) => c.type)).toEqual(['rect', 'polygon'])
    expect(commando.children[0].style.fill).toBe('#aa0000')
    const s7 = optionOf(lines).series[0].renderItem({ dataIndex: 0 }, API)
    expect(s7.children.map((c) => c.type)).toEqual(['rect', 'polygon'])
    expect(s7.children[0].style.fill).toBe('#00aa00')
  })

  it('une portée quasi ponctuelle garde 2 px de large : l’arme ne disparaît pas', () => {
    const plat = weaponRangeLines([{ weapon_key: 'x', kills: side({ p10: 4, median: 4, p90: 4 }) }], 'fr')
    const g = optionOf(plat).series[0].renderItem({ dataIndex: 0 }, API)
    expect(g.children[0].shape.width).toBe(2)
  })
})

describe('infobulle', () => {
  const tooltip = (dataIndex: number) => optionOf(weaponRangeLines(ROWS, 'fr')).tooltip.formatter({ dataIndex })

  it('donne les trois valeurs des deux côtés', () => {
    const html = tooltip(2)
    expect(html).toContain('<b>Mêlée</b> — p10 · médiane · p90')
    expect(html).toContain('Mes frags — 133 : 0.9 m · <b>1.4 m</b> · 2.6 m')
    expect(html).toContain('Mes morts — 96 : 0.9 m · <b>1.5 m</b> · 2.8 m')
  })

  it('dit « aucune mesure » pour le côté absent — jamais un zéro', () => {
    expect(tooltip(1)).toContain('Mes morts — aucune mesure')
    expect(tooltip(1)).not.toContain('Mes morts — 0')
  })

  it('échappe le nom d’arme (rendu en innerHTML par ECharts)', () => {
    const hostile = weaponRangeLines(
      [{ weapon_key: 'x', label: '<img src=x onerror=alert(1)>', kills: side({}) }],
      'fr',
    )
    const html = optionOf(hostile).tooltip.formatter({ dataIndex: 0 })
    expect(html).toContain('&lt;img src=x onerror=alert(1)&gt;')
    expect(html).not.toContain('<img')
  })

  it('rend une chaîne vide hors des lignes connues (ECharts survole large)', () => {
    expect(tooltip(99)).toBe('')
  })
})
