/**
 * sessionBarsTrendChart — LA frise « une soirée, un bâton », partagée.
 *
 * Hissée depuis `features/squad/charts/squadRiposteSessionsChart.ts` (lot J) le
 * 2026-09-22 : la page Séries temporelles demande la même frise sur deux sujets
 * (Riposte, Appui reçu) et la règle des deux copies interdit de la recopier. Le module
 * de l'Escouade n'est plus qu'un ADAPTATEUR — il traduit sa `FriseRiposte` en séries
 * génériques et n'écrit plus une seule clé d'option ECharts.
 *
 * GRAMMAIRE TENUE (celle de `squadSessionTimelineChart.ts`, désignée comme référence par
 * l'utilisateur) : bâtons à 18 px, courbe pleine 2 px NON lissée, légende nommant chaque
 * série, axes en 10 px gris.
 *
 * TROIS INVARIANTS, quel que soit le nombre de séries :
 *
 *   1. UN SEUL AXE Y. Tout est en points de pourcentage — taux d'une soirée, tendance,
 *      repère d'habituel : deux axes seraient une faute de lecture (et la règle dataviz
 *      « jamais deux échelles dans un graphe »).
 *   2. LE REPÈRE D'HABITUEL EST UNE LIGNE TIRETÉE, jamais un bâton : une valeur de
 *      comparaison n'est pas une mesure de la période. Chaque série porte le sien.
 *      EXCEPTION — le MODE ÉCART (`baseline`, D23-3 du 2026-09-22) : quand chaque série
 *      est tracée en ÉCART à son propre repère, les repères ne sont plus des valeurs de
 *      l'axe, ils valent tous zéro. Les tiretés cèdent alors la place à UNE ligne zéro en
 *      trait plein, dont l'étiquette NOMME les repères qu'elle remplace ; l'axe cesse de
 *      compter des parts et compte des POINTS d'écart, et l'infobulle porte les deux
 *      lectures — la valeur absolue ET l'écart.
 *   3. LE DÉNOMINATEUR VOYAGE AVEC LE TAUX : soit en second rang d'étiquettes sous l'axe
 *      des dates (`volumeAxis`), soit en lignes d'infobulle (`tooltipLines`). Une frise
 *      qui n'en montre aucun laisse lire une soirée à 3 morts comme une soirée à 80.
 *
 * UN BÂTON CREUX (`hollow`) dit « échantillon faible » : il est rendu en contour, jamais
 * absent — la trame du temps ne doit pas mentir — et l'appelant l'exclut de sa tendance.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from './_utils'

/** Le repère de comparaison d'une série : sa valeur en POURCENTS et son libellé. */
export interface SessionBarsUsual {
  valuePct: number
  label: string
  /** Couleur du trait et de son étiquette (défaut : le gris des axes). */
  color?: string
}

/** La moyenne glissante d'une série, déjà calculée par l'appelant. */
export interface SessionBarsTrend {
  valuesPct: (number | null)[]
  label: string
  color: string
}

/** Une série de bâtons : une grandeur, une couleur, son repère et sa tendance. */
export interface SessionBarsSeriesSpec {
  name: string
  color: string
  /** Taux par soirée, en POURCENTS, aligné sur `labels`. `null` = pas de bâton. */
  valuesPct: (number | null)[]
  /** Par soirée : bâton en contour (échantillon faible). */
  hollow?: boolean[]
  /**
   * Par soirée : bâton ATTÉNUÉ — il est dans la trame du temps mais hors du périmètre que
   * le lecteur a demandé.
   *
   * MÊME GRAMMAIRE QUE LE NUAGE DE LA PORTÉE (lot W, D23-4 du 2026-09-21) : la population
   * entière est tracée, et ce que le filtre retient est en ENCRE PLEINE. Retirer les
   * autres reviendrait à ne montrer qu'un point — et un point n'a pas de population où se
   * situer. Se combine avec `hollow` : la fiabilité et l'appartenance au filtre sont deux
   * choses distinctes.
   */
  dimmed?: boolean[]
  /** Nom de pile — deux séries de même pile occupent la même colonne (verdicts à trous). */
  stack?: string
  usual?: SessionBarsUsual
  trend?: SessionBarsTrend
}

/**
 * Le MODE ÉCART : chaque série tracée en écart à SON repère, autour d'un zéro commun.
 *
 * Sans lui, deux grandeurs qui vivent l'une vers 48 % et l'autre vers 25 % occupent deux
 * bandes qui ne se croisent jamais : l'axe commun ne porte rien. En écart, elles
 * partagent enfin une unité. Une série SANS `usual` garde sa valeur brute — on ne lui
 * invente pas un repère.
 */
export interface SessionBarsBaseline {
  /** Étiquette posée sur la ligne zéro : elle nomme les repères qu'elle remplace. */
  label: string
  /** Unité de l'écart en infobulle (« pts »), servie par l'appelant (FR/EN). */
  deltaUnit: string
}

export interface SessionBarsTrendOpts {
  /** Les soirées, dans l'ordre du temps. */
  labels: string[]
  series: SessionBarsSeriesSpec[]
  yAxisLabel: string
  /** Second rang d'étiquettes sous l'axe des dates (un volume par soirée). */
  volumeAxis?: { label: string; values: string[] }
  /** Lignes ajoutées à l'infobulle d'une soirée (ses dénominateurs). */
  tooltipLines?: (index: number) => string[]
  /** Présent = MODE ÉCART : valeur tracée = `value - usual`, une seule ligne à zéro. */
  baseline?: SessionBarsBaseline
  /** Témoin de légende du bâton creux (échantillon faible) : une pastille, aucune donnée. */
  hollowLegend?: { label: string; color: string }
}

function round1(v: number): number {
  return parseFloat(v.toFixed(1))
}

/** Un écart SIGNÉ : sans le « + » porté, un +3 et un -3 ne se distinguent qu'au signe absent. */
function signe(v: number): string {
  return v > 0 ? `+${round1(v)}` : `${round1(v)}`
}

/**
 * shortSessionLabel réduit un libellé de session à sa DATE.
 *
 * Le libellé complet (« 13/10/2025 22:27–22:46 (3) ») porte la plage horaire et le nombre
 * de matchs : posé sur quarante graduations d'axe, il se chevauche et devient illisible.
 * L'axe dit QUAND — la soirée —, pas le détail. Un libellé sans espace est rendu tel
 * quel : on ne coupe jamais à l'aveugle.
 */
export function shortSessionLabel(label: string): string {
  const espace = label.indexOf(' ')
  return espace > 0 ? label.slice(0, espace) : label
}

/** Le repère soustrait à une série en mode écart ; zéro partout ailleurs. */
function decalageDe(spec: SessionBarsSeriesSpec, ecart: boolean): number {
  return ecart && spec.usual ? spec.usual.valuePct : 0
}

/**
 * Opacité d'un bâton hors du périmètre filtré. Assez basse pour que l'encre pleine du
 * filtre se détache au premier coup d'œil, assez haute pour que la forme de la population
 * reste lisible — c'est elle qui justifie de tracer ces bâtons.
 */
const OPACITE_HORS_FILTRE = 0.3

function barData(spec: SessionBarsSeriesSpec, decalage: number): unknown[] {
  return spec.valuesPct.map((v, i) => {
    if (v == null) return null
    const value = round1(v - decalage)
    const creux = spec.hollow?.[i] === true
    const attenue = spec.dimmed?.[i] === true
    if (!creux && !attenue) return value
    // Bâton CREUX : contour de la couleur de la série sur un fond transparent. Le bâton
    // reste à sa place et à sa hauteur — c'est sa FIABILITÉ qui est dite, pas sa valeur.
    const itemStyle: Record<string, unknown> = creux
      ? { color: 'transparent', borderColor: spec.color, borderWidth: 1.5 }
      : {}
    if (attenue) itemStyle.opacity = OPACITE_HORS_FILTRE
    return { value, itemStyle }
  })
}

function markLineOf(usual: SessionBarsUsual, axisColor: string): Record<string, unknown> {
  const couleur = usual.color ?? axisColor
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed', color: couleur, width: 1 },
    label: { formatter: usual.label, color: couleur, fontSize: 10, position: 'insideEndTop' },
    data: [{ yAxis: round1(usual.valuePct) }],
  }
}

/**
 * markLineZero — LE repère unique du mode écart.
 *
 * Trait PLEIN : un tireté dirait « valeur de comparaison », alors qu'ici c'est l'origine
 * de l'axe. Son étiquette nomme les deux repères confondus en elle — sans ce libellé, le
 * lecteur ne saurait plus à quoi il compare.
 */
function markLineZero(label: string, couleur: string): Record<string, unknown> {
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'solid', color: couleur, width: 1.6 },
    label: { formatter: label, color: couleur, fontSize: 10, position: 'insideStartTop' },
    data: [{ yAxis: 0 }],
  }
}

interface LigneInfobulle {
  marker: string
  seriesName: string
  value: number | null
}

/**
 * ligneInfobulle — la ligne d'une série dans l'infobulle d'une soirée.
 *
 * En mode écart, LES DEUX LECTURES : la valeur absolue — elle a quitté l'axe, elle ne
 * doit pas quitter le graphe — ET l'écart au repère, qui est ce que le bâton dessine.
 */
function ligneInfobulle(
  r: LigneInfobulle,
  decalages: Map<string, number>,
  baseline: SessionBarsBaseline | undefined,
): string {
  const v = r.value as number
  if (!baseline) return `${r.marker}${r.seriesName} : ${v} %`
  const absolu = round1(v + (decalages.get(r.seriesName) ?? 0))
  return `${r.marker}${r.seriesName} : ${absolu} % (${signe(v)} ${baseline.deltaUnit})`
}

/** Les bâtons d'une série, avec le repère qui lui revient selon le mode. */
function barreDe(
  s: SessionBarsSeriesSpec,
  index: number,
  ctx: { baseline: SessionBarsBaseline | undefined; decalage: number; axisColor: string },
): Record<string, unknown> {
  const repere = ctx.baseline
    ? index === 0
      ? { markLine: markLineZero(ctx.baseline.label, ctx.axisColor) }
      : {}
    : s.usual
      ? { markLine: markLineOf(s.usual, ctx.axisColor) }
      : {}
  return {
    type: 'bar',
    name: s.name,
    ...(s.stack ? { stack: s.stack } : {}),
    barMaxWidth: 18,
    itemStyle: { borderRadius: [3, 3, 0, 0] },
    color: s.color,
    data: barData(s, ctx.decalage),
    ...repere,
  }
}

/** La courbe de tendance d'une série : pleine en mode valeur, pointillée fine en écart. */
function tendanceDe(
  trend: SessionBarsTrend,
  decalage: number,
  ecart: boolean,
): Record<string, unknown> {
  return {
    type: 'line',
    name: trend.label,
    data: trend.valuesPct.map((v) => (v == null ? null : round1(v - decalage))),
    smooth: false,
    // La tendance d'un écart est un RAPPEL, pas une mesure : pointillé fin et sans
    // marqueurs, pour qu'elle ne dispute pas la lecture aux bâtons.
    lineStyle: ecart
      ? { width: 1.6, type: 'dashed', color: trend.color, opacity: 0.75 }
      : { width: 2, color: trend.color },
    itemStyle: { color: trend.color },
    symbol: ecart ? 'none' : 'circle',
    symbolSize: 6,
  }
}

function axeDesVolumes(
  volumeAxis: NonNullable<SessionBarsTrendOpts['volumeAxis']>,
  axisColor: string,
): Record<string, unknown> {
  // Le rang des volumes : un axe de catégories SANS ligne ni graduation, posé sous le
  // premier. Il ne porte aucune série — seulement ses étiquettes.
  return {
    type: 'category',
    data: volumeAxis.values,
    position: 'bottom',
    offset: 22,
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { show: false },
    name: volumeAxis.label,
    nameLocation: 'end',
    nameGap: 8,
    nameTextStyle: { color: axisColor, fontSize: 10 },
    axisLabel: { color: axisColor, fontSize: 10 },
  }
}

/**
 * buildSessionBarsTrendOption — l'option ECharts complète de la frise.
 *
 * ORDRE DE RENDU : tous les bâtons dans l'ordre des séries, puis toutes les tendances.
 * Une courbe doit passer AU-DESSUS des bâtons, y compris de ceux d'une série voisine.
 */
export function buildSessionBarsTrendOption(opts: SessionBarsTrendOpts): EChartsCoreOption {
  if (opts.labels.length === 0 || opts.series.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const ecart = opts.baseline != null
  const decalages = new Map(opts.series.map((s) => [s.name, decalageDe(s, ecart)]))

  const barres = opts.series.map((s, i) =>
    barreDe(s, i, {
      baseline: opts.baseline,
      decalage: decalages.get(s.name) ?? 0,
      axisColor: tc.axisLabel,
    }),
  )
  const tendances = opts.series
    .filter((s) => s.trend != null)
    .map((s) => tendanceDe(s.trend as SessionBarsTrend, decalages.get(s.name) ?? 0, ecart))
  // Le témoin « échantillon faible » : une série SANS donnée, présente pour sa seule
  // pastille de légende — son contour y dit ce que disent les bâtons creux.
  const temoins = opts.hollowLegend
    ? [
        {
          type: 'bar',
          name: opts.hollowLegend.label,
          silent: true,
          data: [] as unknown[],
          itemStyle: {
            color: 'transparent',
            borderColor: opts.hollowLegend.color,
            borderWidth: 1.5,
          },
        },
      ]
    : []

  const xAxis: Record<string, unknown>[] = [{ ...axis, type: 'category', data: opts.labels }]
  if (opts.volumeAxis) xAxis.push(axeDesVolumes(opts.volumeAxis, tc.axisLabel))

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: opts.volumeAxis ? 64 : 44, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Le DÉNOMINATEUR rejoint chaque infobulle : un taux sans son volume se lit faux.
      formatter: (params: unknown) => {
        const rows = Array.isArray(params)
          ? (params as (LigneInfobulle & { axisValue: string; dataIndex: number })[])
          : []
        if (rows.length === 0) return ''
        const lignes = rows
          .filter((r) => r.value != null)
          .map((r) => ligneInfobulle(r, decalages, opts.baseline))
        lignes.push(...(opts.tooltipLines?.(rows[0].dataIndex) ?? []))
        return [rows[0].axisValue, ...lignes].join('<br/>')
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: [
        ...barres.map((b) => b.name as string),
        ...tendances.map((l) => l.name as string),
        ...temoins.map((l) => l.name),
      ],
    },
    xAxis,
    yAxis: {
      ...axis,
      type: 'value',
      // En mode écart, l'axe DOIT descendre sous zéro : l'ancrer à 0 amputerait la moitié
      // du propos — « la soirée a fait MOINS bien que son repère ».
      ...(ecart ? {} : { min: 0 }),
      name: opts.yAxisLabel,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: {
        ...axis.axisLabel,
        formatter: ecart ? (v: number) => signe(v) : '{value} %',
      },
    },
    series: [...barres, ...tendances, ...temoins],
  }
}
