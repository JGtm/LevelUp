/**
 * placementCharts.test.ts — « Distance au coéquipier le plus proche et frags, par vie » (carte de
 * chaleur) et « Part des vies par placement » (spécification §2) : bornes des axes, cases (part,
 * étiquette à partir de 1,5 %, infobulle), gros points (distance médiane, frags moyens),
 * repère du radar et frontière (seuils du bloc), quatre quarts dont un seul teinté, légende
 * native, couleurs par jeton, seuil de 8 %, infobulles FR et EN.
 */
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

import { renderOptionToSvg } from '@/components/charts/svgRender.testutil'
import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import type { SquadEmprisePlacement } from '@/lib/api/types'

import { squadPlayerPalette } from '../colors'
import { PLACEMENT_2209 } from './placement.fixtures'
import {
  buildPlacementLifeOption,
  buildPlacementQuartsOption,
  MEDIAN_SPREAD,
  medianOffsets,
  medianSymbolSize,
  placementFormats,
  QUADRANT_TOKENS,
  resolvePlacementColors,
  type PlacementColors,
} from './placementCharts'
import { PLACEMENT_TEXT } from './placementStrings'

const COLORS: PlacementColors = {
  player: (gt) => `joueur-${gt}`,
  quadrant: {
    in_range_productive: '#101010',
    isolated_productive: '#202020',
    in_range_costly: '#303030',
    isolated_costly: '#404040',
  },
  radar: 'radar',
  heat: '#60a5fa',
  theme: {
    axisLabel: 'axis',
    axisLine: 'line',
    splitLine: 'split',
    splitAreaA: 'a',
    splitAreaB: 'b',
    text: 'text',
    tooltipBg: 'bg',
    tooltipBorder: 'border',
    card: 'card',
    isDark: false,
  },
}

type Loc = 'fr' | 'en'
const opts = (locale: Loc = 'fr') => ({ t: PLACEMENT_TEXT[locale], formats: placementFormats(locale) })

type Datum = { value: [number, number]; symbolSize: number; tip: string }
type HeatDatum = { value: [number, number, number]; tip: string }
type Item = {
  name?: string
  xAxis?: number
  yAxis?: number
  itemStyle?: { color: string }
  label?: { color?: string; position?: string }
}
type Series = {
  type: string
  name?: string
  z?: number
  data: (Datum | HeatDatum | number)[]
  itemStyle?: { color: string; borderColor: string; borderWidth: number }
  silent?: boolean
  stack?: string
  barWidth?: number
  label?: { formatter: (p: { value: number }) => string; color: string }
  markLine?: {
    data: { xAxis?: number; yAxis?: number }[]
    lineStyle: { color: string; type: string }
    label: { show: boolean; formatter?: string; color?: string; position?: string; fontSize?: number }
  }
  markArea?: { data: Item[][]; label: { color: string; fontSize: number } }
}
type Axis = {
  min: number
  max: number
  interval?: number
  name?: string
  nameLocation?: string
  data?: string[]
  axisTick?: { customValues?: number[] }
  axisLabel: { formatter: (v: number) => string; fontWeight?: number; customValues?: number[] }
}
interface Opt {
  series: Series[]
  xAxis: Axis
  yAxis: Axis
  legend: { bottom: number; left: string; data: { name: string; itemStyle: { color: string } }[] }
  tooltip: { formatter: (p: unknown) => string }
}

const life = (block: SquadEmprisePlacement = PLACEMENT_2209, locale: Loc = 'fr') =>
  buildPlacementLifeOption(block, COLORS, opts(locale)) as unknown as Opt
const quarts = (block: SquadEmprisePlacement = PLACEMENT_2209, locale: Loc = 'fr') =>
  buildPlacementQuartsOption(block, COLORS, opts(locale)) as unknown as Opt

const dotsOf = (o: Opt, name: string) => o.series.filter((s) => s.type === 'scatter' && s.name === name)
const heatOf = (o: Opt) => o.series.find((s) => s.type === 'custom')!
const tip = (o: Opt, d: { tip: string }) => o.tooltip.formatter({ data: d })
const overlay = (o: Opt) => o.series.filter((s) => s.silent && s.type === 'line')
const radarOf = (o: Opt) => overlay(o).find((s) => s.markLine?.data[0]?.xAxis != null)!
const frontierOf = (o: Opt) => overlay(o).find((s) => s.markLine?.data[0]?.yAxis != null)!

describe('buildPlacementLifeOption — axes', () => {
  it('X : 0 à 2 par pas de 0,25, titre centré sous l’axe, libellés à deux décimales (virgule en FR, point en EN)', () => {
    const o = life()
    expect(o.xAxis).toMatchObject({ min: 0, max: 2, interval: 0.25, nameLocation: 'middle' })
    expect(o.xAxis.name).toBe('distance médiane au coéquipier le plus proche pendant la vie, en portées de radar')
    expect(o.xAxis.axisLabel.formatter(0.25)).toBe('0,25')
    expect(o.xAxis.axisLabel.formatter(2)).toBe('2,00')
    expect(life(PLACEMENT_2209, 'en').xAxis.axisLabel.formatter(0.25)).toBe('0.25')
    expect(life(PLACEMENT_2209, 'en').xAxis.name).toBe('median distance to the nearest teammate during the life, in radar ranges')
  })

  it('Y : étendue −0,5 à 5,5, graduations et libellés posés sur 0 à 5 (« 5+ » à la dernière ligne), titre vertical', () => {
    const o = life()
    expect(o.yAxis).toMatchObject({ min: -0.5, max: 5.5, nameLocation: 'middle' })
    expect(o.yAxis.name).toBe('frags dans la vie')
    expect(o.yAxis.axisTick?.customValues).toEqual([0, 1, 2, 3, 4, 5])
    expect(o.yAxis.axisLabel.customValues).toEqual([0, 1, 2, 3, 4, 5])
    expect([0, 1, 2, 3, 4, 5].map((v) => o.yAxis.axisLabel.formatter(v))).toEqual(['0', '1', '2', '3', '4', '5+'])
  })

  it('Y : le rendu écrit les six libellés (les graduations calculées, sur les demi-unités, n’en écrivaient aucun)', () => {
    const svg = renderOptionToSvg(buildPlacementLifeOption(PLACEMENT_2209, COLORS, opts()), 900, 520)
    const texts = [...svg.matchAll(/<text[^>]*>([^<]*)<\/text>/g)].map((m) => m[1])
    for (const label of ['0', '1', '2', '3', '4', '5+', 'frags dans la vie']) expect(texts).toContain(label)
  })
})

describe('buildPlacementLifeOption — la carte de chaleur', () => {
  it('une série de cases sous les repères et sous les gros points, une case par couple (distance, frags) non vide', () => {
    const o = life()
    const heat = heatOf(o)
    expect(heat.z).toBeLessThan(radarOf(o).z ?? 0)
    expect(heat.z).toBeLessThan(dotsOf(o, 'JGtm')[0].z ?? 0)
    // 14 vies mesurées dans la soirée figée, réparties en cases : jamais plus de cases que de vies.
    const lives = PLACEMENT_2209.players!.reduce((n, p) => n + p.lives!.length, 0)
    expect(lives).toBe(14)
    expect(heat.data.length).toBeGreaterThan(0)
    expect(heat.data.length).toBeLessThanOrEqual(lives)
  })

  it('une case porte ses bornes en X (x0, x1) et sa ligne de frags', () => {
    const data = heatOf(life()).data as HeatDatum[]
    // JGtm : une vie à 0,3 portée et 2 frags -> colonne [0,25 ; 0,5[, ligne 2.
    expect(data.some((d) => d.value[0] === 0.25 && d.value[1] === 0.5 && d.value[2] === 2)).toBe(true)
    // La vie à 2,8 portées et 7 frags tombe dans la dernière case : [1,75 ; 2[, ligne 5.
    expect(data.some((d) => d.value[0] === 1.75 && d.value[1] === 2 && d.value[2] === 5)).toBe(true)
  })

  it('infobulle d’une case : part, nombre de vies, bornes (FR, pluriel et singulier)', () => {
    const o = life()
    const data = heatOf(o).data as HeatDatum[]
    const mid = data.find((d) => d.value[0] === 0.25 && d.value[2] === 2)!
    // 1 vie sur 14 : 7,1 %.
    expect(tip(o, mid)).toBe('7,1 % des vies de l’escouade · 1 vie<br>distance de 0,25 à 0,50 radar · 2 frags')
    const last = data.find((d) => d.value[0] === 1.75 && d.value[2] === 5)!
    expect(tip(o, last)).toBe('7,1 % des vies de l’escouade · 1 vie<br>distance de 1,75 radar ou plus · 5 frags ou plus')
  })

  it('infobulle d’une case : parallèle EN', () => {
    const o = life(PLACEMENT_2209, 'en')
    const last = (heatOf(o).data as HeatDatum[]).find((d) => d.value[0] === 1.75 && d.value[2] === 5)!
    expect(tip(o, last)).toBe('7.1% of the squad’s lives · 1 life<br>distance 1.75 radar or more · 5 kills or more')
  })

  it('le rendu écrit la part dans les cases qui atteignent 1,5 % (« 7 % » : une vie sur quatorze)', () => {
    const svg = renderOptionToSvg(buildPlacementLifeOption(PLACEMENT_2209, COLORS, opts()), 900, 520)
    const texts = [...svg.matchAll(/<text[^>]*>([^<]*)<\/text>/g)].map((m) => m[1])
    expect(texts).toContain('7 %')
  })

  it('aucune vie dans la grille : aucune case, jamais de division par zéro', () => {
    const block: SquadEmprisePlacement = {
      ...PLACEMENT_2209,
      players: [{ xuid: 'x', gamertag: 'Vide', lives_total: 2, lives_measured: 0, quadrants: [], lives: [] }],
    }
    expect(heatOf(life(block)).data).toHaveLength(0)
  })

  it('aucun hasard, aucune position décalée : deux constructions donnent la même option', () => {
    expect(JSON.stringify(life())).toBe(JSON.stringify(life()))
  })
})

describe('buildPlacementLifeOption — taille des gros points', () => {
  it('gros point : 18 + min(nombre de vies, 200) / 10', () => {
    expect(medianSymbolSize(0)).toBe(18)
    expect(medianSymbolSize(146)).toBeCloseTo(32.6, 10)
    expect(medianSymbolSize(200)).toBe(38)
    expect(medianSymbolSize(500)).toBe(38)
    const big = dotsOf(life(), 'JGtm')[0]
    expect((big.data[0] as Datum).symbolSize).toBeCloseTo(18 + 6 / 10, 10)
  })
})

describe('medianOffsets — les gros points qui se recouvrent sont écartés (Escouade OKLM, 22/09)', () => {
  // Les positions relevées sur la soirée du 22/09 : Joueur90002 et Joueur90001 à 0,0025 portée
  // l'un de l'autre, au même compte de frags — le second tracé cachait le premier. Sans liste de
  // vies, la médiane du contrat tient lieu de moyenne.
  const spot = (xuid: string, gamertag: string, ratio: number, kills: number) => ({
    ...PLACEMENT_2209.players![0],
    xuid,
    gamertag,
    median_radar_ratio: ratio,
    median_kills: kills,
    lives: [],
  })
  const oklm = [spot('xj', 'JGtm', 0.4394, 0), spot('xm', 'Joueur90002', 0.4256, 1), spot('xc', 'Joueur90001', 0.4231, 1)]

  it('deux gros points confondus : écartés de part et d’autre de leur valeur, dans l’ordre du bloc', () => {
    const off = medianOffsets(oklm)
    expect(off.get('xm')).toBeCloseTo(-MEDIAN_SPREAD / 2, 10)
    expect(off.get('xc')).toBeCloseTo(MEDIAN_SPREAD / 2, 10)
    // JGtm, une frag plus bas : seul, pas décalé.
    expect(off.get('xj')).toBe(0)
  })

  it('trois gros points confondus : un pas entre chacun, centré, gardé dans l’axe en bas', () => {
    const off = medianOffsets([spot('a', 'A', 0.4, 0), spot('b', 'B', 0.41, 0), spot('c', 'C', 0.42, 0)])
    const ys = ['a', 'b', 'c'].map((x) => off.get(x)!)
    expect(ys[1] - ys[0]).toBeCloseTo(MEDIAN_SPREAD, 10)
    expect(ys[2] - ys[1]).toBeCloseTo(MEDIAN_SPREAD, 10)
    expect(Math.min(...ys)).toBeGreaterThanOrEqual(-0.25 - 1e-9)
  })

  it('la carte pose le gros point décalé ; l’infobulle garde la vraie distance', () => {
    const block = { ...PLACEMENT_2209, players: oklm }
    const madina = dotsOf(life(block), 'Joueur90002')[0]
    const choco = dotsOf(life(block), 'Joueur90001')[0]
    const yM = (madina.data[0] as Datum).value[1]
    const yC = (choco.data[0] as Datum).value[1]
    expect(yC - yM).toBeCloseTo(MEDIAN_SPREAD, 10)
    expect((madina.data[0] as Datum).tip).toContain('Joueur90002')
  })
})

describe('buildPlacementLifeOption — repères et quarts', () => {
  it('repère du radar : trait vertical pointillé au seuil du bloc (pas une constante), étiquette en haut à l’intérieur, aucune zone teintée', () => {
    const radar = radarOf(life({ ...PLACEMENT_2209, isolated_from_ratio: 1.25 }))
    expect(radar.markLine!.data).toEqual([{ xAxis: 1.25 }])
    expect(radar.markLine!.lineStyle).toMatchObject({ color: 'radar', type: 'dashed' })
    expect(radar.markLine!.label).toMatchObject({ show: true, formatter: 'portée du radar', color: 'radar', position: 'insideEndTop', fontSize: 11 })
    expect(radar.markArea).toBeUndefined()
    expect(radarOf(life()).markLine!.data).toEqual([{ xAxis: 1 }])
    expect(radarOf(life(PLACEMENT_2209, 'en')).markLine!.label.formatter).toBe('radar range')
  })

  it('frontière horizontale pointillée grise à « seuil de frags − 0,5 » (0,5 pour un frag)', () => {
    expect(frontierOf(life()).markLine!.data).toEqual([{ yAxis: 0.5 }])
    expect(frontierOf(life()).markLine!.lineStyle).toMatchObject({ color: 'axis', type: 'dashed' })
    expect(frontierOf(life()).markLine!.label.show).toBe(false)
    expect(frontierOf(life({ ...PLACEMENT_2209, productive_from_kills: 2 })).markLine!.data).toEqual([{ yAxis: 1.5 }])
  })

  it('quatre quarts nommés (titre en capitales, sous-titre), texte gris 12 px, aux coins extérieurs de la carte', () => {
    const area = life().series.find((s) => s.markArea)!.markArea!
    expect(area.label).toMatchObject({ color: 'axis', fontSize: 12 })
    expect(area.data.map((d) => d[0].name)).toEqual([
      'GROUPÉ ET RENTABLE\nsûr',
      'ISOLÉ ET RENTABLE\nflanqueur, surveiller la régularité',
      'GROUPÉ ET COÛTEUX\nduel à travailler, pas le placement',
      'ISOLÉ ET COÛTEUX\nvie donnée pour rien, seul',
    ])
    expect(area.data.map((d) => d[0].label!.position)).toEqual([
      'insideTopLeft',
      'insideTopRight',
      'insideBottomLeft',
      'insideBottomRight',
    ])
    expect(area.data.map((d) => [d[0].xAxis, d[0].yAxis, d[1].xAxis, d[1].yAxis])).toEqual([
      [0, 5.5, 1, 0.5],
      [1, 5.5, 2, 0.5],
      [0, 0.5, 1, -0.5],
      [1, 0.5, 2, -0.5],
    ])
    const en = life(PLACEMENT_2209, 'en').series.find((s) => s.markArea)!.markArea!
    expect(en.data.map((d) => d[0].name)).toEqual([
      'GROUPED AND PRODUCTIVE\nsafe',
      'ISOLATED AND PRODUCTIVE\nflanker, watch the consistency',
      'GROUPED AND COSTLY\na duel to work on, not the placement',
      'ISOLATED AND COSTLY\na life given away, alone',
    ])
  })

  it('les quarts suivent les seuils du bloc (isolé à 1,25 portée, rentable à 2 frags)', () => {
    const area = life({ ...PLACEMENT_2209, isolated_from_ratio: 1.25, productive_from_kills: 2 }).series.find((s) => s.markArea)!.markArea!
    expect(area.data.map((d) => [d[0].xAxis, d[0].yAxis, d[1].xAxis, d[1].yAxis])).toEqual([
      [0, 5.5, 1.25, 1.5],
      [1.25, 5.5, 2, 1.5],
      [0, 1.5, 1.25, -0.5],
      [1.25, 1.5, 2, -0.5],
    ])
  })

  it('seul « isolé et coûteux » est teinté (perf-tier-5 à 10 %, son texte dans cette couleur)', () => {
    const area = life().series.find((s) => s.markArea)!.markArea!
    const [a, b, c, d] = area.data.map((x) => x[0])
    for (const q of [a, b, c]) {
      expect(q.itemStyle).toBeUndefined()
      expect(q.label!.color).toBeUndefined()
    }
    expect(d.itemStyle!.color).toBe('rgba(64,64,64,0.1)')
    expect(d.label!.color).toBe('#404040')
  })

  it('pas de zone « isolé » teintée sur le radar : un seul markArea, celui des quatre quarts', () => {
    expect(life().series.filter((s) => s.markArea)).toHaveLength(1)
  })
})

describe('buildPlacementLifeOption — joueurs, légende, gros point', () => {
  it('un gros point par joueur, à la couleur du joueur, au-dessus des cases et des repères', () => {
    const o = life()
    for (const gt of ['JGtm', 'Joueur90001', 'Joueur90002']) {
      const dots = dotsOf(o, gt)
      expect(dots).toHaveLength(1)
      expect(dots[0].data).toHaveLength(1)
      expect(dots[0].z).toBeGreaterThan(heatOf(o).z ?? 0)
      expect(dots[0].z).toBeGreaterThan(radarOf(o).z ?? 0)
      expect(dots[0].itemStyle).toMatchObject({ color: `joueur-${gt}`, borderColor: 'text', borderWidth: 2 })
    }
  })

  it('le gros point : distance médiane en X, MOYENNE des frags par vie en Y', () => {
    // Seul sur la carte : aucun autre gros point ne le recouvre, rien ne l’écarte (medianOffsets).
    const choco = PLACEMENT_2209.players!.find((p) => p.gamertag === 'Joueur90001')!
    const big = dotsOf(life({ ...PLACEMENT_2209, players: [choco] }), 'Joueur90001')[0].data[0] as Datum
    // Vies à 0,4 / 0,5 / 1,1 / 1,4 : médiane 0,8 ; frags 0, 1, 0, 2 : moyenne 0,75.
    expect(big.value[0]).toBeCloseTo(0.8, 10)
    expect(big.value[1]).toBeCloseTo(0.75, 10)
  })

  it('le gros point est posé aux plafonds (2 portées, 5 frags), l’infobulle garde les vraies valeurs', () => {
    const far = {
      ...PLACEMENT_2209.players![0],
      median_radar_ratio: 2.5,
      lives: [{ ...PLACEMENT_2209.players![0].lives![0], kills: 8 }],
    }
    const o = life({ ...PLACEMENT_2209, players: [far] })
    const big = dotsOf(o, 'JGtm')[0].data[0] as Datum
    expect(big.value).toEqual([2, 5])
    expect(tip(o, big)).toContain('distance médiane 2,50 radar · 8 frags par vie en moyenne')
  })

  it('légende native en bas, centrée, une entrée par joueur à la couleur du joueur', () => {
    const l = life().legend
    expect(l).toMatchObject({ bottom: 4, left: 'center' })
    expect(l.data.map((d) => d.name)).toEqual(['JGtm', 'Joueur90001', 'Joueur90002'])
    expect(l.data.map((d) => d.itemStyle.color)).toEqual(['joueur-JGtm', 'joueur-Joueur90001', 'joueur-Joueur90002'])
  })

  it('sans vie mesurée, un joueur n’a pas de gros point (série vide), les autres gardent le leur', () => {
    const block: SquadEmprisePlacement = {
      ...PLACEMENT_2209,
      players: [
        PLACEMENT_2209.players![0],
        { xuid: 'x-vide', gamertag: 'Vide', lives_total: 2, lives_measured: 0, quadrants: [], lives: [] },
      ],
    }
    const o = life(block)
    expect(dotsOf(o, 'Vide')[0].data).toHaveLength(0)
    expect(dotsOf(o, 'JGtm')[0].data).toHaveLength(1)
  })

  it('sans joueur : option vide, jamais un graphe à zéro série', () => {
    expect(buildPlacementLifeOption({ ...PLACEMENT_2209, players: null }, COLORS, opts())).toEqual({ backgroundColor: 'transparent' })
    expect(buildPlacementQuartsOption({ ...PLACEMENT_2209, players: [] }, COLORS, opts())).toEqual({ backgroundColor: 'transparent' })
  })
})

describe('buildPlacementLifeOption — infobulle du gros point', () => {
  it('« Joueur · N vies », distance médiane et frags moyens, part des vies isolées et sans frag', () => {
    const o = life()
    // 6 vies mesurées, une isolée et sans frag (1,6 radar, 0 frag) : 17 % ; frags 2, 1, 0, 3, 0, 7 : moyenne 2,17 (plafonnée à 5 sur la carte seulement).
    expect(tip(o, dotsOf(o, 'JGtm')[0].data[0] as Datum)).toBe(
      '<b>JGtm</b> · 6 vies<br>distance médiane 1,05 radar · 2,17 frags par vie en moyenne<br>17 % des vies isolées et sans frag',
    )
    const en = life(PLACEMENT_2209, 'en')
    expect(tip(en, dotsOf(en, 'JGtm')[0].data[0] as Datum)).toBe(
      '<b>JGtm</b> · 6 lives<br>median distance 1.05 radar · 2.17 kills per life on average<br>17% of lives isolated and without a kill',
    )
  })

  it('le gamertag est échappé', () => {
    const block: SquadEmprisePlacement = {
      ...PLACEMENT_2209,
      players: [{ ...PLACEMENT_2209.players![0], gamertag: '<i>x</i>' }],
    }
    const o = life(block)
    expect(tip(o, dotsOf(o, '<i>x</i>')[0].data[0] as Datum)).toContain('<b>&lt;i&gt;x&lt;/i&gt;</b>')
  })
})

describe('buildPlacementQuartsOption', () => {
  it('quatre segments empilés, dans l’ordre, aux teintes perf-tier-1, 2, 4 et 5 (jamais perf-tier-3), épaisseur 22, séparateur couleur de carte', () => {
    const s = quarts().series
    expect(s.map((x) => x.name)).toEqual(['groupé et rentable', 'isolé et rentable', 'groupé et coûteux', 'isolé et coûteux'])
    expect(s.map((x) => x.itemStyle!.color)).toEqual(['#101010', '#202020', '#303030', '#404040'])
    for (const x of s) {
      expect(x.type).toBe('bar')
      expect(x.stack).toBe('quarts')
      expect(x.barWidth).toBe(22)
      expect(x.itemStyle).toMatchObject({ borderColor: 'card', borderWidth: 1 })
    }
    expect(Object.values(QUADRANT_TOKENS)).toEqual(['perf-tier-1', 'perf-tier-2', 'perf-tier-4', 'perf-tier-5'])
  })

  it('premier joueur en haut (axe catégoriel inversé), noms en gras, axe de 0 à 100 %', () => {
    const o = quarts()
    expect(o.yAxis.data).toEqual(['Joueur90002', 'Joueur90001', 'JGtm'])
    expect(o.yAxis.axisLabel.fontWeight).toBe(600)
    expect(o.xAxis).toMatchObject({ min: 0, max: 100 })
    expect(o.xAxis.axisLabel.formatter(50)).toBe('50 %')
  })

  it('les parts viennent du contrat, en pourcentage, dans l’ordre inversé des joueurs', () => {
    const s = quarts().series
    const pct = (i: number) => PLACEMENT_2209.players![i].quadrants!.map((q) => (q.share ?? 0) * 100)
    expect(s.map((x) => (x.data as number[])[2])).toEqual(pct(0))
    expect(s.map((x) => (x.data as number[])[1])).toEqual(pct(1))
    expect(s.map((x) => (x.data as number[])[0])).toEqual(pct(2))
    expect(pct(0)[3]).toBeCloseTo(100 / 6, 10)
  })

  it('valeur « v % » dans le segment, en blanc à contour sombre, centrée, seulement à partir de 8 %', () => {
    const l = quarts().series[0].label!
    expect(l.color).toBe('#ffffff')
    expect((l as { verticalAlign?: string }).verticalAlign).toBe('middle')
    expect(l.formatter({ value: 7.4 })).toBe('')
    expect(l.formatter({ value: 0 })).toBe('')
    expect(l.formatter({ value: 8 })).toBe('8 %')
    expect(l.formatter({ value: 33.33 })).toBe('33 %')
    expect(quarts(PLACEMENT_2209, 'en').series[0].label!.formatter({ value: 33.33 })).toBe('33%')
  })

  it('infobulle : « Joueur · N vies » puis une ligne « nom : v % » par quart', () => {
    const o = quarts()
    const html = o.tooltip.formatter([
      { axisValue: 'JGtm', marker: '<m/>', seriesName: 'groupé et rentable', value: 33.3 },
      { axisValue: 'JGtm', marker: '<m/>', seriesName: 'isolé et coûteux', value: 16.7 },
    ])
    expect(html).toBe('<b>JGtm</b> · 6 vies<br><m/> groupé et rentable : 33 %<br><m/> isolé et coûteux : 17 %')
    const en = quarts(PLACEMENT_2209, 'en').tooltip.formatter([{ axisValue: 'JGtm', marker: '<m/>', seriesName: 'isolated and costly', value: 16.7 }])
    expect(en).toBe('<b>JGtm</b> · 6 lives<br><m/> isolated and costly: 17%')
    expect(o.tooltip.formatter([{ axisValue: 'Inconnu', marker: '', seriesName: 'x', value: 1 }])).toBe('')
  })

  it('légende native en bas, centrée, une entrée par quart à sa couleur', () => {
    const l = quarts().legend
    expect(l).toMatchObject({ bottom: 4, left: 'center' })
    expect(l.data.map((d) => d.name)).toEqual(['groupé et rentable', 'isolé et rentable', 'groupé et coûteux', 'isolé et coûteux'])
    expect(l.data.map((d) => d.itemStyle.color)).toEqual(['#101010', '#202020', '#303030', '#404040'])
  })
})

describe('resolvePlacementColors', () => {
  beforeAll(() => applyPalette(defaultPalette, 'default'))
  afterAll(() => {
    _resetActivePalette()
    document.documentElement.style.removeProperty('--warning-foreground')
  })

  it('joueurs : palette de la page (ordre de la SÉLECTION, pas du bloc) ; quarts : jetons perf-tier ; repère : jeton ; cases : haut de la rampe de fréquence', () => {
    // Sélection Joueur90002 puis Joueur90001 : l’ordre du bloc (JGtm, Joueur90001, Joueur90002) ne compte pas.
    const palette = squadPlayerPalette('JGtm', ['Joueur90002', 'Joueur90001'])
    const c = resolvePlacementColors(palette.tokenOf)
    expect(c.player('JGtm')).toBe(defaultPalette['squad-player-1'])
    expect(c.player('Joueur90002')).toBe(defaultPalette['squad-player-2'])
    expect(c.player('Joueur90001')).toBe(defaultPalette['squad-player-3'])
    expect(c.player('joueur90002')).toBe(palette.colorByPlayer.Joueur90002)
    expect(c.player('Inconnu')).toBe(c.theme.text)
    expect(c.quadrant.in_range_productive).toBe(defaultPalette['perf-tier-1'])
    expect(c.quadrant.isolated_productive).toBe(defaultPalette['perf-tier-2'])
    expect(c.quadrant.in_range_costly).toBe(defaultPalette['perf-tier-4'])
    expect(c.quadrant.isolated_costly).toBe(defaultPalette['perf-tier-5'])
    expect(c.radar).toBe(defaultPalette.extreme)
    expect(c.heat).toBe(defaultPalette['heatmap-freq-high'])
  })
})
