/**
 * squadFdaGapChart — « Écart cumulé au FDA attendu » par joueur (onglet Dynamique,
 * à côté de « Balance des dégâts cumulée », même abscisse `xAxisLabels(n)`).
 *
 * Décisions D3/D5 du plan PLAN_EXPECTED_FDA_2026-07 ; forme revue par le lot L1 du
 * plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : la valeur de fin s'écrit au bout
 * de chaque courbe (signée, une décimale, couleur du joueur, étiquettes écartées
 * verticalement quand elles se chevauchent), la légende est en bas et centrée, et
 * aucun texte de synthèse ne vit sous le graphe.
 *
 * Pour chaque joueur : cumul du différentiel `kda − kda_expected` (FDA réel natif
 * ADR 0006 moins FDA attendu projeté backend) par `match_order` CROISSANT. 1 série
 * `line` par joueur, couleur via `colorByPlayer` (getSquadPlayerColors), markLine 0,
 * PAS d'aire (multi-séries).
 *
 * D5 : un match sans attendu (`kda_expected` NULL / non-fini, ou `kda` absent) ne
 * fait pas avancer le cumul — la courbe REPORTE la dernière valeur (jamais 0, jamais
 * de rupture). Les trous d'intersection (aucune ligne à un `match_order`) restent
 * `null` et sont pontés par `connectNulls`.
 *
 * Extrait de `squadPerformanceLineCharts.ts` (déjà > 500 L) : builder dédié plutôt
 * que gonfler le fichier voisin. Réutilise ses helpers partagés
 * (`orderedPlayers` / `maxLength` / `xAxisLabels`).
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { withEndPoint } from '@/components/charts/endPoint'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { cumulativeFdaGap, type FdaGapPair } from '@/lib/charts/cumulativeFdaGap'

import {
  maxLength,
  orderedPlayers,
  xAxisLabels,
  type CommonOpts,
} from './squadPerformanceLineCharts'

/**
 * Paire FDA réel / attendu d'un point de la série de performance escouade. FDA
 * réel = valeur native (`kda`, ADR 0006) ; FDA attendu = projeté backend
 * (`kda_expected`). Le helper canonique traite null/non-fini comme absent (D5).
 */
function toPair(p: SquadPerformanceSeriesPoint): FdaGapPair {
  return { real: p.kda ?? null, expected: p.kda_expected ?? null }
}

/**
 * Cumul du différentiel par `match_order` croissant pour UN joueur, délégué au
 * helper canonique `cumulativeFdaGap` (source unique du cumul, CLAUDE.md n°6).
 * D5 : un point sans attendu ne fait pas avancer le cumul (report), mais figure
 * quand même à la valeur courante. Les indices sans point (trou d'intersection)
 * restent `null`.
 */
export function cumulativeFdaGapSeries(
  points: SquadPerformanceSeriesPoint[],
  n: number,
): Array<number | null> {
  const ordered = points
    .filter((p) => p.match_order >= 0 && p.match_order < n)
    .sort((a, b) => a.match_order - b.match_order)
  const cum = cumulativeFdaGap(ordered.map(toPair))
  const data = new Array<number | null>(n).fill(null)
  ordered.forEach((p, i) => {
    data[p.match_order] = cum[i].cumulative
  })
  return data
}

export interface FdaGapCumulativeOpts extends CommonOpts {
  /** Décimales affichées (infobulle, axe Y, valeur de fin). Défaut 1. */
  decimals?: number
  /** Locale Intl des nombres (`fr-FR` → « +3,0 »). Défaut : locale de l'environnement. */
  intlLocale?: string
}

/**
 * Marge droite de la grille : la place de la valeur de fin (« +15,4 ») écrite au bout
 * des courbes. `containLabel` ne compte que les graduations, pas les `endLabel`.
 */
const END_LABEL_ROOM = 48

/**
 * Formateur d'écart signé (« +0,7 » / « -0,4 » / « 0,0 ») : un zéro arrondi ne porte
 * pas de signe (`exceptZero`), jamais « -0,0 ».
 */
function signedGapFormatter(decimals: number, intlLocale?: string): (v: number) => string {
  const nf = new Intl.NumberFormat(intlLocale, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
    signDisplay: 'exceptZero',
  })
  return (v: number) => nf.format(v)
}

/**
 * La série d'un joueur : sa courbe, le point final grossi (`withEndPoint`, source unique), la
 * valeur de fin au bout ; la ligne 0 posée sur la première série seulement.
 */
function fdaGapPlayerSeries(
  player: string,
  idx: number,
  cum: Array<number | null>,
  s: { color: string; tc: ReturnType<typeof getEChartsThemeColors>; fmtSigned: (v: number) => string },
) {
  const { color, tc, fmtSigned } = s
  return {
    name: player,
    type: 'line' as const,
    data: withEndPoint(cum, tc.card),
    lineStyle: { color, width: 2 },
    itemStyle: { color },
    symbol: 'circle' as const,
    symbolSize: 4,
    connectNulls: true,
    // Valeur de fin au bout de la courbe (précédent : squadRangeRolesChart). ECharts la
    // pose sur le dernier point NON nul ; `moveOverlap: 'shiftY'` écarte verticalement
    // les étiquettes de TOUTES les séries qui se chevauchent (vérifié au rendu SVG).
    endLabel: {
      show: true,
      color,
      fontSize: 11,
      fontWeight: 600,
      distance: 6,
      formatter: (p: { value?: unknown }) => (typeof p.value === 'number' ? fmtSigned(p.value) : ''),
    },
    labelLayout: { moveOverlap: 'shiftY' as const },
    // markLine 0 (parité cumulée : FDA réel = FDA attendu) rendue une seule fois,
    // attachée au premier joueur.
    ...(idx === 0
      ? {
          markLine: {
            silent: true,
            symbol: 'none',
            lineStyle: { color: tc.axisLabel, type: 'dashed' as const, width: 1 },
            label: { show: false },
            data: [{ yAxis: 0 }],
          },
        }
      : {}),
  }
}

export function buildFdaGapCumulativeOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: FdaGapCumulativeOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }

  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }

  const xLabels = opts.xLabels ?? xAxisLabels(n)
  const fmtSigned = signedGapFormatter(opts.decimals ?? 1, opts.intlLocale)
  const hiddenPlayers = opts.hiddenPlayers ?? new Set<string>()
  const emptyData = new Array<number | null>(n).fill(null)

  const series = players.map((player, idx) =>
    fdaGapPlayerSeries(player, idx, hiddenPlayers.has(player) ? emptyData : cumulativeFdaGapSeries(rows[player], n), {
      color: opts.colorByPlayer[player] ?? '#888', // color-allow: gris structurel pour joueur sans couleur attribuée
      tc,
      fmtSigned,
    }),
  )

  return {
    backgroundColor: CHART_BG,
    grid: { top: 20, bottom: 36, left: 8, right: END_LABEL_ROOM, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? fmtSigned(v) : '-'),
    },
    // Légende en bas (getLegendBase) et CENTRÉE, explicitement (spec S2 du plan).
    legend: { ...getLegendBase(tc), left: 'center', data: players },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: {
        ...axis.axisLabel,
        interval: n > 30 ? Math.floor(n / 12) : 0,
      },
    },
    yAxis: {
      ...axis,
      type: 'value',
      axisLabel: {
        ...axis.axisLabel,
        formatter: (v: number) => fmtSigned(v),
      },
    },
    series,
  }
}
