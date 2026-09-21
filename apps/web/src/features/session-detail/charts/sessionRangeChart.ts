/**
 * sessionRangeChart — l'option ECharts de la carte « Portée des engagements » de la session
 * (lot O, D22-4) : UN BÂTON PAR MATCH, hauteur = mon écart à la médiane du lobby.
 *
 * PAS `BarStackedChart` : ce graphe n'empile rien, et il porte trois choses que le wrapper
 * n'expose pas — un `itemStyle` PAR BARRE (le bâton creux sous le plancher de mesure), les
 * `markArea` des trois bandes de rôle et la `markLine` de la médiane de session. Même
 * pattern que `squadRangeRolesChart` : composer `<ChartCard>` avec un `buildOption` custom.
 *
 * COULEURS — jetons sémantiques uniquement : `divergent-pos` au-dessus de la ligne du lobby,
 * `divergent-neg` en dessous (le VERDICT, pas une teinte de série), la ligne du lobby et les
 * bandes en encres de thème. Aucun hex.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'

import type { BatonPortee, RoleDePortee, SeuilsRoles } from '../sessionRange.logic'

export interface SessionRangeChartOpts {
  categories: string[]
  seuils: SeuilsRoles | null
  /** Médiane des écarts de la session (bâtons pleins) — `null` = pas de ligne. */
  medianeM: number | null
  /**
   * Bornes Y IMPOSÉES (mode comparaison) : les deux colonnes partagent l'axe, sinon un
   * même écart se dessine deux hauteurs différentes et la comparaison ment. Absent =
   * bornes calculées sur les bâtons de cette session.
   */
  yDomain?: [number, number]
  libelles: {
    yAxis: string
    lobbyLine: string
    medianLine: string
    bandes: Record<RoleDePortee, string>
    /** L'infobulle d'un bâton, déjà composée par l'appelant (donc déjà traduite). */
    tooltip: (b: BatonPortee) => string
  }
}

/** La donnée d'une barre : sa hauteur, son encre, et le bâton BRUT pour l'infobulle. */
interface EchartBarDatum {
  value: number
  itemStyle: Record<string, unknown>
  raw: BatonPortee
}

/** Bornes de l'axe Y : l'amplitude observée, jamais plus serrée que ±`MIN_AMPLITUDE_M`. */
const MIN_AMPLITUDE_M = 5

export function bornesEcart(batons: readonly BatonPortee[]): { min: number; max: number } {
  const amplitude = Math.max(MIN_AMPLITUDE_M, ...batons.map((b) => Math.abs(b.ecartM)), 0)
  const borne = Math.ceil(amplitude) + 1
  return { min: -borne, max: borne }
}

/** Les trois bandes de rôle, posées sur l'échelle relative. */
function bandesRoles(
  seuils: SeuilsRoles | null,
  bornes: { min: number; max: number },
  libelles: SessionRangeChartOpts['libelles'],
  couleurBande: string,
): Record<string, unknown> | undefined {
  if (!seuils) return undefined
  const bande = (bas: number, haut: number, nom: string) => [
    {
      yAxis: bas,
      name: nom,
      itemStyle: { color: couleurBande, opacity: 0.07 },
      label: { position: 'insideEndTop' },
    },
    { yAxis: haut },
  ]
  return {
    silent: true,
    label: { show: true, fontSize: 10, position: 'insideEndTop' },
    data: [
      bande(bornes.min, seuils.bas, libelles.bandes.front),
      bande(seuils.bas, seuils.haut, libelles.bandes.polyvalent),
      bande(seuils.haut, bornes.max, libelles.bandes.sniper),
    ],
  }
}

export function buildSessionRangeOption(
  batons: BatonPortee[],
  opts: SessionRangeChartOpts,
): EChartsCoreOption {
  if (batons.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const bornes = opts.yDomain
    ? { min: opts.yDomain[0], max: opts.yDomain[1] }
    : bornesEcart(batons)
  const encrePos = resolveToken('divergent-pos')
  const encreNeg = resolveToken('divergent-neg')

  const data: EchartBarDatum[] = batons.map((b) => {
    const encre = b.ecartM >= 0 ? encrePos : encreNeg
    return {
      value: b.ecartM,
      // BÂTON CREUX sous le plancher : contour pointillé, aucun remplissage. Il reste
      // visible (une mesure faible n'est pas une absence) mais ne se confond jamais avec
      // une médiane tenue.
      itemStyle: b.plein
        ? { color: encre }
        : { color: 'transparent', borderColor: encre, borderWidth: 2, borderType: 'dashed' },
      raw: b,
    }
  })

  const markLineData: Record<string, unknown>[] = [
    {
      yAxis: 0,
      lineStyle: { type: 'dashed', color: tc.axisLabel },
      label: { formatter: opts.libelles.lobbyLine, position: 'insideStartTop' },
    },
  ]
  if (opts.medianeM != null) {
    markLineData.push({
      yAxis: opts.medianeM,
      lineStyle: { type: 'solid', color: resolveToken('warning'), width: 1.5 },
      label: { formatter: opts.libelles.medianLine, position: 'insideEndTop' },
    })
  }

  const serie: Record<string, unknown> = {
    type: 'bar',
    data,
    barMaxWidth: 34,
    markLine: {
      silent: true,
      symbol: 'none',
      label: { show: true, color: tc.axisLabel, fontSize: 10 },
      data: markLineData,
    },
  }
  const bandes = bandesRoles(opts.seuils, bornes, opts.libelles, tc.axisLine)
  if (bandes) serie.markArea = bandes

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 62, left: 56, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown): string => {
        const raw = (params as { data?: EchartBarDatum }).data?.raw
        if (!raw) return ''
        const titre = escapeHtml(opts.categories[batons.indexOf(raw)] ?? `#${raw.ordre + 1}`)
        return `<b>${titre}</b><br/>${escapeHtml(opts.libelles.tooltip(raw))}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: opts.categories,
      axisLabel: {
        ...(axis.axisLabel as Record<string, unknown>),
        interval: 0,
        rotate: opts.categories.length > 8 ? 40 : 0,
        fontSize: 10,
      },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: opts.libelles.yAxis,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 11 },
      min: bornes.min,
      max: bornes.max,
    },
    series: [serie],
  }
}
