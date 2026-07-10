/**
 * squadFirstEventsChart — teammates.17 : butterfly premier frag / première mort.
 *
 * Spec : .ai/charts_specs/teammates/17_first_events_butterfly.yaml
 *
 *   - X axis : bins 15 s (label = borne droite : "15s", "30s", ..., "1m00s", ...).
 *   - Y axis : nombre de matchs ; symétrique autour de 0, labels en valeur ABSOLUE
 *     (le signe est porté par la position haut/bas).
 *   - 2 séries par joueur :
 *       • Premier frag → bars POSITIVES, couleur joueur.
 *       • Première mort → bars NÉGATIVES, couleur joueur opacity 0.45.
 *   - Séparateurs verticaux pointillés entre chaque bin (markLine sur la 1ère série).
 *   - Légende méta cliquable rendue côté React (cf. SquadFirstEventsChart) : on
 *     reçoit ici les ensembles `hiddenPlayers` / `hiddenTypes` et on vide les
 *     données des séries masquées (axes et séparateurs restent stables).
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadFirstEvents } from '@/lib/api/types'
import { hexComplement } from '@/lib/accessibility'

export interface SquadFirstEventsOpts {
  /** gamertag → couleur hex (cf. getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
  /** i18n labels pour le tooltip et la légende (frag / death). */
  fragLabel: string
  deathLabel: string
  /** i18n suffix tooltip ex: "matchs". */
  matchesSuffix: string
  /** Joueurs masqués via la légende (toutes leurs séries sont vidées). */
  hiddenPlayers?: Set<string>
  /** Types d'événement masqués via la légende ('frag' et/ou 'death'). */
  hiddenTypes?: Set<'frag' | 'death'>
}

export function buildSquadFirstEventsOption(
  data: SquadFirstEvents | null | undefined,
  opts: SquadFirstEventsOpts,
): EChartsCoreOption {
  const xLabels = data?.bin_labels ?? []
  const rows = data?.rows ?? []
  if (!data || xLabels.length === 0 || rows.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const series: Array<Record<string, unknown>> = []
  const hiddenPlayers = opts.hiddenPlayers ?? new Set<string>()
  const hiddenTypes = opts.hiddenTypes ?? new Set<'frag' | 'death'>()
  const empty = xLabels.map(() => 0)

  for (let pi = 0; pi < rows.length; pi += 1) {
    const row = rows[pi]
    const color = opts.colorByPlayer[row.player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const negColor = hexComplement(color) // hue +180° — même convention que butterfly et stats/min
    const isFirst = pi === 0
    const fragHidden = hiddenPlayers.has(row.player) || hiddenTypes.has('frag')
    const deathHidden = hiddenPlayers.has(row.player) || hiddenTypes.has('death')

    // Frags positifs.
    series.push({
      name: `${row.player} — ${opts.fragLabel}`,
      type: 'bar',
      stack: `frag-${row.player}`,
      barMaxWidth: 16,
      itemStyle: { color },
      data: fragHidden ? empty : (row.kill_counts ?? empty),
      // markLine sur la 1ère série uniquement → séparateurs verticaux entre les bins.
      ...(isFirst
        ? {
            markLine: {
              silent: true,
              symbol: 'none',
              label: { show: false },
              lineStyle: { color: tc.splitLine, width: 1, type: 'dotted' },
              data: xLabels.slice(0, -1).map((_, i) => ({ xAxis: i + 0.5 })),
            },
          }
        : {}),
    })

    // Morts négatives — opposé colorimétrique (hue +180°), même convention que butterfly.
    series.push({
      name: `${row.player} — ${opts.deathLabel}`,
      type: 'bar',
      stack: `death-${row.player}`,
      barMaxWidth: 16,
      itemStyle: { color: negColor },
      data: deathHidden ? empty : (row.death_counts ?? empty).map((v) => (v === 0 ? 0 : -v)),
    })
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const cat = (arr[0] as { name?: string }).name ?? ''
        const lines = arr
          .map((p) => {
            const point = p as { seriesName: string; value: unknown; color: string }
            const v = typeof point.value === 'number' ? Math.abs(point.value) : 0
            if (v === 0) return null
            return `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${point.color};margin-right:6px"></span>${escapeHtml(point.seriesName ?? '')}: ${v} ${opts.matchesSuffix}`
          })
          .filter(Boolean)
        if (lines.length === 0) return ''
        return `<strong>${escapeHtml(cat)}</strong><br/>${lines.join('<br/>')}`
      },
    },
    legend: { show: false },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: { ...axis.axisLabel, interval: 0 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      // Labels en valeur absolue (le signe est porté par la position haut/bas).
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => `${Math.abs(v)}` },
      axisLine: { lineStyle: { color: tc.text, width: 1.5 } },
    },
    series,
  }
}
