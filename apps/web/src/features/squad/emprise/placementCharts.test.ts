/**
 * placementCharts.test.ts — « Placement et rendement de chaque vie » et « Part des vies par
 * placement » (lot V4 du plan PLAN_EMPRISE_VIES_2026-09-28, spécification §2) : bornes des axes,
 * plafonds d'affichage, tailles, décalage vertical déterministe, repère du radar et frontière
 * (seuils du bloc), quatre quarts dont un seul teinté, légende native, couleurs par jeton, seuil
 * de 8 %, infobulles FR et EN.
 */
import * as echarts from 'echarts'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'

import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import type { SquadEmprisePlacement } from '@/lib/api/types'

import { squadPlayerPalette } from '../colors'
import { PLACEMENT_2209 } from './placement.fixtures'
import {
  buildPlacementLifeOption,
  buildPlacementQuartsOption,
  JITTER_MAX,
  lifeJitter,
  lifeSymbolSize,
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
  ink: 'encre',
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
type Item = { name?: string; xAxis?: number; yAxis?: number; itemStyle?: { color: string }; label?: { color: string } }
type Series = {
  type: string
  name?: string
  z?: number
  data: (Datum | number)[]
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

const scatterOf = (o: Opt, name: string) => o.series.filter((s) => s.type === 'scatter' && s.name === name)
const tip = (o: Opt, d: Datum) => o.tooltip.formatter({ data: d })
const overlay = (o: Opt) => o.series.filter((s) => s.silent)
const radarOf = (o: Opt) => overlay(o).find((s) => s.markLine?.data[0]?.xAxis != null)!
const frontierOf = (o: Opt) => overlay(o).find((s) => s.markLine?.data[0]?.yAxis != null)!

describe('buildPlacementLifeOption — axes et plafonds', () => {
  it('X : 0 à 2 par pas de 0,25, titre centré sous l’axe, libellés à deux décimales (virgule en FR, point en EN)', () => {
    const o = life()
    expect(o.xAxis).toMatchObject({ min: 0, max: 2, interval: 0.25, nameLocation: 'middle' })
    expect(o.xAxis.name).toBe('distance médiane au coéquipier le plus proche pendant la vie, en portées de radar')
    expect(o.xAxis.axisLabel.formatter(0.25)).toBe('0,25')
    expect(o.xAxis.axisLabel.formatter(2)).toBe('2,00')
    expect(life(PLACEMENT_2209, 'en').xAxis.axisLabel.formatter(0.25)).toBe('0.25')
    expect(life(PLACEMENT_2209, 'en').xAxis.name).toBe('median distance to the nearest teammate during the life, in radar ranges')
  })

  it('Y : étendue −0,5 à 5,5, graduations et libellés posés sur 0 à 5 (« 5+ » au plafond), titre vertical', () => {
    const o = life()
    expect(o.yAxis).toMatchObject({ min: -0.5, max: 5.5, nameLocation: 'middle' })
    expect(o.yAxis.name).toBe('frags dans la vie')
    expect(o.yAxis.axisTick?.customValues).toEqual([0, 1, 2, 3, 4, 5])
    expect(o.yAxis.axisLabel.customValues).toEqual([0, 1, 2, 3, 4, 5])
    expect([0, 1, 2, 3, 4, 5].map((v) => o.yAxis.axisLabel.formatter(v))).toEqual(['0', '1', '2', '3', '4', '5+'])
  })

  it('Y : le rendu écrit les six libellés (les graduations calculées, sur les demi-unités, n’en écrivaient aucun)', () => {
    const chart = echarts.init(null, null, { renderer: 'svg', ssr: true, width: 900, height: 420 })
    chart.setOption(buildPlacementLifeOption(PLACEMENT_2209, COLORS, opts()))
    const texts = [...chart.renderToSVGString().matchAll(/<text[^>]*>([^<]*)<\/text>/g)].map((m) => m[1])
    chart.dispose()
    for (const label of ['0', '1', '2', '3', '4', '5+', 'frags dans la vie']) expect(texts).toContain(label)
  })

  it('une vie à 2,80 portées et 7 frags est posée à 2 et à 5 (± décalage) ; l’infobulle garde les vraies valeurs', () => {
    const o = life()
    const far = scatterOf(o, 'JGtm')[0].data[5] as Datum
    expect(far.value[0]).toBe(2)
    expect(far.value[1]).toBeGreaterThanOrEqual(5 - JITTER_MAX)
    expect(far.value[1]).toBeLessThanOrEqual(5 + JITTER_MAX)
    const html = tip(o, far)
    expect(html).toContain('2,80 radar')
    expect(html).toContain('7 frags')
    expect(html).toContain('2:30')
  })

  it('un point sous le plafond garde sa position vraie en X', () => {
    const jg = scatterOf(life(), 'JGtm')[0]
    expect((jg.data[0] as Datum).value[0]).toBe(0.3)
    expect((jg.data[3] as Datum).value[0]).toBe(1.2)
  })
})

describe('buildPlacementLifeOption — tailles', () => {
  it('point de vie : 6 + min(durée en s, 90) / 9', () => {
    expect(lifeSymbolSize(0)).toBe(6)
    expect(lifeSymbolSize(45_000)).toBe(11)
    expect(lifeSymbolSize(90_000)).toBe(16)
    expect(lifeSymbolSize(150_000)).toBe(16)
    const jg = scatterOf(life(), 'JGtm')[0]
    expect((jg.data[0] as Datum).symbolSize).toBeCloseTo(6 + 62 / 9, 10)
    expect((jg.data[5] as Datum).symbolSize).toBe(16)
  })

  it('gros point : 18 + min(nombre de vies, 200) / 10', () => {
    expect(medianSymbolSize(0)).toBe(18)
    expect(medianSymbolSize(146)).toBeCloseTo(32.6, 10)
    expect(medianSymbolSize(200)).toBe(38)
    expect(medianSymbolSize(500)).toBe(38)
    const big = scatterOf(life(), 'JGtm')[1]
    expect((big.data[0] as Datum).symbolSize).toBeCloseTo(18 + 6 / 10, 10)
  })
})

describe('buildPlacementLifeOption — décalage vertical déterministe', () => {
  it('même clé (match, xuid, début) : même décalage, toujours dans ± 0,25', () => {
    expect(lifeJitter('m1', 'x1', 1000)).toBe(lifeJitter('m1', 'x1', 1000))
    for (let i = 0; i < 200; i++) {
      expect(Math.abs(lifeJitter(`m${i}`, 'x1', i * 977))).toBeLessThanOrEqual(JITTER_MAX)
    }
  })

  it('la clé compte : match, joueur et début changent le décalage, qui décolle bien des points de même compte', () => {
    const base = lifeJitter('m1', 'x1', 1000)
    expect(lifeJitter('m2', 'x1', 1000)).not.toBe(base)
    expect(lifeJitter('m1', 'x2', 1000)).not.toBe(base)
    expect(lifeJitter('m1', 'x1', 1001)).not.toBe(base)
    const spread = new Set(Array.from({ length: 50 }, (_, i) => lifeJitter('m', 'x', i).toFixed(3)))
    expect(spread.size).toBeGreaterThan(40)
  })

  it('aucun hasard : deux constructions donnent la même option, Math.random n’est pas appelé', () => {
    const rnd = vi.spyOn(Math, 'random')
    expect(JSON.stringify(life())).toBe(JSON.stringify(life()))
    expect(rnd).not.toHaveBeenCalled()
    rnd.mockRestore()
  })

  it('l’ordonnée d’une vie est son compte plus le décalage de sa clé', () => {
    const jg = scatterOf(life(), 'JGtm')[0]
    const p = PLACEMENT_2209.players![0]
    const l = p.lives![0]
    expect((jg.data[0] as Datum).value[1]).toBeCloseTo(l.kills + lifeJitter(l.match_id, p.xuid, l.start_ms), 10)
  })
})

describe('medianOffsets — les gros points qui se recouvrent sont écartés (Escouade OKLM, 22/09)', () => {
  // Les médianes relevées sur la soirée du 22/09 : Madina97294 et Chocoboflor à 0,0025 portée
  // l'un de l'autre, au même compte de frags — le second tracé cachait le premier.
  const spot = (xuid: string, gamertag: string, ratio: number, kills: number) => ({
    ...PLACEMENT_2209.players![0],
    xuid,
    gamertag,
    median_radar_ratio: ratio,
    median_kills: kills,
  })
  const oklm = [spot('xj', 'JGtm', 0.4394, 0), spot('xm', 'Madina97294', 0.4256, 1), spot('xc', 'Chocoboflor', 0.4231, 1)]

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

  it('le nuage pose le gros point décalé ; l’infobulle garde la vraie médiane', () => {
    const block = { ...PLACEMENT_2209, players: oklm }
    const madina = scatterOf(life(block), 'Madina97294')[1]
    const choco = scatterOf(life(block), 'Chocoboflor')[1]
    const yM = (madina.data[0] as Datum).value[1]
    const yC = (choco.data[0] as Datum).value[1]
    expect(yC - yM).toBeCloseTo(MEDIAN_SPREAD, 10)
    expect((madina.data[0] as Datum).tip).toContain('Madina97294')
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

  it('quatre quarts nommés (titre en capitales, sous-titre), texte gris 12 px, aux coins du seuil et de la frontière', () => {
    const area = life().series.find((s) => s.markArea)!.markArea!
    expect(area.label).toMatchObject({ color: 'axis', fontSize: 12 })
    expect(area.data.map((d) => d[0].name)).toEqual([
      'À PORTÉE ET RENTABLE\nsûr',
      'ISOLÉ ET RENTABLE\nflanqueur, surveiller la régularité',
      'À PORTÉE ET COÛTEUX\nduel à travailler, pas le placement',
      'ISOLÉ ET COÛTEUX\nvie donnée pour rien, seul',
    ])
    expect(area.data.map((d) => [d[0].xAxis, d[0].yAxis, d[1].xAxis, d[1].yAxis])).toEqual([
      [0, 5.5, 1, 0.5],
      [1, 5.5, 2, 0.5],
      [0, 0.5, 1, -0.5],
      [1, 0.5, 2, -0.5],
    ])
    const en = life(PLACEMENT_2209, 'en').series.find((s) => s.markArea)!.markArea!
    expect(en.data.map((d) => d[0].name)).toEqual([
      'IN RANGE AND PRODUCTIVE\nsafe',
      'ISOLATED AND PRODUCTIVE\nflanker, watch the consistency',
      'IN RANGE AND COSTLY\na duel to work on, not the placement',
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
      expect(q.label).toBeUndefined()
    }
    expect(d.itemStyle!.color).toBe('rgba(64,64,64,0.1)')
    expect(d.label!.color).toBe('#404040')
  })

  it('pas de zone « isolé » teintée sur le radar : un seul markArea, celui des quatre quarts', () => {
    expect(life().series.filter((s) => s.markArea)).toHaveLength(1)
  })
})

describe('buildPlacementLifeOption — joueurs, légende, gros point', () => {
  it('un semis et un gros point par joueur, MÊME nom de série (le clic sur la légende isole les deux) ; le gros point est au-dessus', () => {
    const o = life()
    for (const gt of ['JGtm', 'Chocoboflor', 'Madina97294']) {
      const [semis, gros] = scatterOf(o, gt)
      expect(semis.data.length).toBeGreaterThan(0)
      expect(gros.data).toHaveLength(1)
      expect(gros.z).toBeGreaterThan(semis.z ?? 0)
      expect(semis.itemStyle).toMatchObject({ color: `joueur-${gt}`, borderColor: 'card', borderWidth: 1 })
      expect(gros.itemStyle).toMatchObject({ color: `joueur-${gt}`, borderColor: 'text', borderWidth: 2 })
    }
  })

  it('le gros point : médiane X × médiane des frags', () => {
    // Seul dans le nuage : aucun autre gros point ne le recouvre, rien ne l’écarte (medianOffsets).
    const choco = PLACEMENT_2209.players!.find((p) => p.gamertag === 'Chocoboflor')!
    const big = scatterOf(life({ ...PLACEMENT_2209, players: [choco] }), 'Chocoboflor')[1].data[0] as Datum
    // Vies à 0,4 / 0,5 / 1,1 / 1,4 : médiane 0,8 ; frags 0, 1, 0, 2 : médiane 0,5.
    expect(big.value[0]).toBeCloseTo(0.8, 10)
    expect(big.value[1]).toBe(0.5)
  })

  it('le gros point est posé aux plafonds (2 portées, 5 frags), l’infobulle garde les vraies médianes', () => {
    const far = { ...PLACEMENT_2209.players![0], median_radar_ratio: 2.5, median_kills: 8 }
    const o = life({ ...PLACEMENT_2209, players: [far] })
    const big = scatterOf(o, 'JGtm')[1].data[0] as Datum
    expect(big.value).toEqual([2, 5])
    expect(tip(o, big)).toContain('médiane 2,50 radar · 8 frags par vie')
  })

  it('légende native en bas, centrée, une entrée par joueur à la couleur du joueur', () => {
    const l = life().legend
    expect(l).toMatchObject({ bottom: 4, left: 'center' })
    expect(l.data.map((d) => d.name)).toEqual(['JGtm', 'Chocoboflor', 'Madina97294'])
    expect(l.data.map((d) => d.itemStyle.color)).toEqual(['joueur-JGtm', 'joueur-Chocoboflor', 'joueur-Madina97294'])
  })

  it('sans vie mesurée, un joueur garde son semis (vide) mais n’a pas de gros point', () => {
    const block: SquadEmprisePlacement = {
      ...PLACEMENT_2209,
      players: [
        PLACEMENT_2209.players![0],
        { xuid: 'x-vide', gamertag: 'Vide', lives_total: 2, lives_measured: 0, quadrants: [], lives: [] },
      ],
    }
    const o = life(block)
    expect(scatterOf(o, 'Vide')).toHaveLength(1)
    expect(scatterOf(o, 'Vide')[0].data).toHaveLength(0)
    expect(scatterOf(o, 'JGtm')).toHaveLength(2)
  })

  it('sans joueur : option vide, jamais un graphe à zéro série', () => {
    expect(buildPlacementLifeOption({ ...PLACEMENT_2209, players: null }, COLORS, opts())).toEqual({ backgroundColor: 'transparent' })
    expect(buildPlacementQuartsOption({ ...PLACEMENT_2209, players: [] }, COLORS, opts())).toEqual({ backgroundColor: 'transparent' })
  })
})

describe('buildPlacementLifeOption — infobulles', () => {
  it('vie : « Joueur · une vie de m:ss » puis distance, part hors radar, frags (FR, pluriel et singulier)', () => {
    const o = life()
    const d = scatterOf(o, 'JGtm')[0].data[3] as Datum
    expect(tip(o, d)).toBe('<b>JGtm</b> · une vie de 1:15<br>distance médiane 1,20 radar · 60 % de la vie hors radar · 3 frags')
    const one = scatterOf(o, 'JGtm')[0].data[1] as Datum
    expect(tip(o, one)).toMatch(/ · 1 frag$/)
  })

  it('vie : parallèle EN', () => {
    const o = life(PLACEMENT_2209, 'en')
    const d = scatterOf(o, 'JGtm')[0].data[3] as Datum
    expect(tip(o, d)).toBe('<b>JGtm</b> · a life of 1:15<br>median distance 1.20 radar · 60% of the life out of radar · 3 kills')
  })

  it('gros point : « Joueur · N vies », médiane, part des vies isolées et sans frag', () => {
    const o = life()
    // 6 vies mesurées, une isolée et sans frag (1,6 radar, 0 frag) : 17 %.
    expect(tip(o, scatterOf(o, 'JGtm')[1].data[0] as Datum)).toBe(
      '<b>JGtm</b> · 6 vies<br>médiane 1,05 radar · 1,5 frags par vie<br>17 % des vies isolées et sans frag',
    )
    const en = life(PLACEMENT_2209, 'en')
    expect(tip(en, scatterOf(en, 'JGtm')[1].data[0] as Datum)).toBe(
      '<b>JGtm</b> · 6 lives<br>median 1.05 radar · 1.5 kills per life<br>17% of lives isolated and without a kill',
    )
  })

  it('le gamertag est échappé', () => {
    const block: SquadEmprisePlacement = {
      ...PLACEMENT_2209,
      players: [{ ...PLACEMENT_2209.players![0], gamertag: '<i>x</i>' }],
    }
    const o = life(block)
    expect(tip(o, scatterOf(o, '<i>x</i>')[0].data[0] as Datum)).toContain('<b>&lt;i&gt;x&lt;/i&gt;</b>')
  })
})

describe('buildPlacementQuartsOption', () => {
  it('quatre segments empilés, dans l’ordre, aux teintes perf-tier-1, 2, 4 et 5 (jamais perf-tier-3), épaisseur 22, séparateur couleur de carte', () => {
    const s = quarts().series
    expect(s.map((x) => x.name)).toEqual(['à portée et rentable', 'isolé et rentable', 'à portée et coûteux', 'isolé et coûteux'])
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
    expect(o.yAxis.data).toEqual(['Madina97294', 'Chocoboflor', 'JGtm'])
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

  it('valeur « v % » dans le segment, encre sombre, seulement à partir de 8 %', () => {
    const l = quarts().series[0].label!
    expect(l.color).toBe('encre')
    expect(l.formatter({ value: 7.4 })).toBe('')
    expect(l.formatter({ value: 0 })).toBe('')
    expect(l.formatter({ value: 8 })).toBe('8 %')
    expect(l.formatter({ value: 33.33 })).toBe('33 %')
    expect(quarts(PLACEMENT_2209, 'en').series[0].label!.formatter({ value: 33.33 })).toBe('33%')
  })

  it('infobulle : « Joueur · N vies » puis une ligne « nom : v % » par quart', () => {
    const o = quarts()
    const html = o.tooltip.formatter([
      { axisValue: 'JGtm', marker: '<m/>', seriesName: 'à portée et rentable', value: 33.3 },
      { axisValue: 'JGtm', marker: '<m/>', seriesName: 'isolé et coûteux', value: 16.7 },
    ])
    expect(html).toBe('<b>JGtm</b> · 6 vies<br><m/> à portée et rentable : 33 %<br><m/> isolé et coûteux : 17 %')
    const en = quarts(PLACEMENT_2209, 'en').tooltip.formatter([{ axisValue: 'JGtm', marker: '<m/>', seriesName: 'isolated and costly', value: 16.7 }])
    expect(en).toBe('<b>JGtm</b> · 6 lives<br><m/> isolated and costly: 17%')
    expect(o.tooltip.formatter([{ axisValue: 'Inconnu', marker: '', seriesName: 'x', value: 1 }])).toBe('')
  })

  it('légende native en bas, centrée, une entrée par quart à sa couleur', () => {
    const l = quarts().legend
    expect(l).toMatchObject({ bottom: 4, left: 'center' })
    expect(l.data.map((d) => d.name)).toEqual(['à portée et rentable', 'isolé et rentable', 'à portée et coûteux', 'isolé et coûteux'])
    expect(l.data.map((d) => d.itemStyle.color)).toEqual(['#101010', '#202020', '#303030', '#404040'])
  })
})

describe('resolvePlacementColors', () => {
  beforeAll(() => applyPalette(defaultPalette, 'default'))
  afterAll(() => {
    _resetActivePalette()
    document.documentElement.style.removeProperty('--warning-foreground')
  })

  it('joueurs : palette de la page (ordre de la SÉLECTION, pas du bloc) ; quarts : jetons perf-tier ; repère : jeton', () => {
    // Sélection Madina97294 puis Chocoboflor : l’ordre du bloc (JGtm, Chocoboflor, Madina97294) ne compte pas.
    const palette = squadPlayerPalette('JGtm', ['Madina97294', 'Chocoboflor'])
    const c = resolvePlacementColors(palette.tokenOf)
    expect(c.player('JGtm')).toBe(defaultPalette['squad-player-1'])
    expect(c.player('Madina97294')).toBe(defaultPalette['squad-player-2'])
    expect(c.player('Chocoboflor')).toBe(defaultPalette['squad-player-3'])
    expect(c.player('madina97294')).toBe(palette.colorByPlayer.Madina97294)
    expect(c.player('Inconnu')).toBe(c.theme.text)
    expect(c.quadrant.in_range_productive).toBe(defaultPalette['perf-tier-1'])
    expect(c.quadrant.isolated_productive).toBe(defaultPalette['perf-tier-2'])
    expect(c.quadrant.in_range_costly).toBe(defaultPalette['perf-tier-4'])
    expect(c.quadrant.isolated_costly).toBe(defaultPalette['perf-tier-5'])
    expect(c.radar).toBe(defaultPalette.extreme)
  })

  it('encre des valeurs : la variable d’encre sombre du thème, à défaut la couleur du texte', () => {
    const { tokenOf } = squadPlayerPalette('JGtm', [])
    expect(resolvePlacementColors(tokenOf).ink).toBe(resolvePlacementColors(tokenOf).theme.text)
    document.documentElement.style.setProperty('--warning-foreground', 'rgb(1, 2, 3)')
    expect(resolvePlacementColors(tokenOf).ink).toBe('rgb(1, 2, 3)')
  })
})
