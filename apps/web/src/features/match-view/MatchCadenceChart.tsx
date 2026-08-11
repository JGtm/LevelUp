/**
 * MatchCadenceChart — match_view.11 (Cadence kills par tranche de temps)
 *
 * Reproduction 1:1 du mock `.ai/charts_specs/_generated/match_view/mock-echarts.html`
 * (chart-10) :
 *  - Bar stacked absolu (mon équipe / adversaires) par phase intra-match.
 *  - 2 courbes lissées : moyenne mobile (window=3) sur chaque équipe.
 *  - markPoint « PIC » sur la valeur la plus haute (toutes équipes confondues).
 *
 * Source : `combat_tab.cadence` (Phase 1 MV2). Le team_side de chaque xuid
 * dans `dp.components` est résolu via le scoreboard (ally vs enemy).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getLegendBase, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { MatchScoreboardRow, MatchViewCadence } from '@/lib/api/types'
import { formatBinSeconds } from './_chartSeries'
import type { MatchViewText } from './i18n'
import { resolveXuidMeta } from './xuidMeta'

interface Props {
  cadence: MatchViewCadence | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

/** Moyenne mobile droite avec fenêtre expansive au démarrage (window=3).
 *  Les window-1 premiers points utilisent une fenêtre réduite plutôt que null
 *  pour que la courbe parte dès le premier segment. */
function movingAverage(values: number[], window = 3): number[] {
  return values.map((_, i) => {
    const start = Math.max(0, i - (window - 1))
    let sum = 0
    for (let j = start; j <= i; j++) sum += values[j]
    return Math.round((sum / (i - start + 1)) * 10) / 10
  })
}

export function MatchCadenceChart({ cadence, scoreboard, meXUID, t }: Props) {
  // La cadence ne se peuple qu'avec des kills (components = xuid→nb de frags par
  // phase). Des datapoints sans aucun kill → EmptyState plutôt qu'un canvas vide
  // titré (cas des matchs sans données de combat exploitables).
  const hasKills =
    !!cadence &&
    cadence.datapoints.some((dp) =>
      Object.values(dp.components ?? {}).some((c) => c > 0),
    )
  const series: ChartSeries<unknown>[] =
    cadence && cadence.datapoints.length > 0 && hasKills
      ? [{ key: cadence.key, datapoints: cadence.datapoints }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<unknown>[]): EChartsCoreOption => {
      if (s.length === 0 || !cadence || cadence.datapoints.length === 0) {
        return { backgroundColor: CHART_BG }
      }
      // Cascade « allié = même camp que moi » : source unique (xuidMeta.ts). Le `xuid ===
      // meXUID` est conservé pour le cas — anormal — où le joueur de la page n'a pas de
      // ligne au scoreboard : il reste son propre allié.
      const meta = resolveXuidMeta(scoreboard, meXUID)
      const isAlly = (xuid: string): boolean =>
        xuid === meXUID || (meta.get(xuid)?.ally ?? false)

      const phaseSeconds = (cadence.meta?.phase_seconds as number | undefined) ?? 30
      const categories = cadence.datapoints.map((_, i) => formatBinSeconds(i * phaseSeconds))
      const teamSeries: number[] = []
      const enemySeries: number[] = []
      for (const dp of cadence.datapoints) {
        let team = 0
        let enemy = 0
        for (const [xuid, count] of Object.entries(dp.components)) {
          if (isAlly(xuid)) team += count
          else enemy += count
        }
        teamSeries.push(team)
        enemySeries.push(enemy)
      }

      const teamMA = movingAverage(teamSeries, 3)
      const enemyMA = movingAverage(enemySeries, 3)

      // PIC = max global toutes phases / équipes confondues
      let peakIdx = 0
      let peakVal = 0
      for (let i = 0; i < teamSeries.length; i++) {
        const total = teamSeries[i] + enemySeries[i]
        if (total > peakVal) { peakVal = total; peakIdx = i }
      }

      const colorTeam = resolveToken('team-ally')
      const colorEnemy = resolveToken('team-enemy')
      const colorPic = resolveToken('warning')
      const tc = getEChartsThemeColors()
      const axis = getAxisBase(tc)

      // Border (theme-aware) autour des courbes MA pour les détacher des
      // bars empilées : duplicat des courbes en couleur foreground du thème,
      // un cran plus large, dessous → effet bordure.
      // `tc.text` = `--foreground` qui s'inverse light/dark, donc lisible
      // dans les 2 modes sans hex en dur.
      const maOutlineWidth = 6
      const maInnerWidth = 3
      const maOutlineName = '__ma_outline__'

      return {
        backgroundColor: CHART_BG,
        grid: { left: 40, right: 20, top: 50, bottom: 80, containLabel: true },
        tooltip: { ...getTooltipBase(tc), trigger: 'axis', axisPointer: { type: 'shadow' } },
        legend: {
          ...getLegendBase(tc),
          bottom: 5,
          // Exclut les outlines de la légende (un seul item par MA)
          data: [
            t.combatTeamLabel,
            t.combatEnemyLabel,
            `MA ${t.combatTeamLabel}`,
            `MA ${t.combatEnemyLabel}`,
          ],
        },
        xAxis: {
          ...axis,
          type: 'category',
          data: categories,
          name: 'Temps',
          nameTextStyle: { color: tc.text },
          axisLabel: { ...axis.axisLabel, rotate: -45, fontSize: 9 },
          splitLine: { show: false },
        },
        yAxis: {
          ...axis,
          type: 'value',
          name: t.combatKillsLabel,
          nameTextStyle: { color: tc.text },
          min: 0,
          splitLine: { show: true, lineStyle: { color: tc.splitLine } },
        },
        series: [
          {
            type: 'bar',
            stack: 'total',
            name: t.combatTeamLabel,
            data: teamSeries,
            itemStyle: { color: colorTeam, opacity: 0.8, borderColor: colorTeam, borderWidth: 1 },
            markPoint: {
              silent: true,
              symbolSize: 1,
              label: {
                show: true,
                formatter: 'Pic',
                position: 'top',
                color: colorPic,
                fontSize: 10,
                backgroundColor: 'rgba(0,0,0,0.5)',
                padding: [2, 4],
              },
              data: peakVal > 0 ? [{ coord: [peakIdx, peakVal] }] : [],
            },
          },
          {
            type: 'bar',
            stack: 'total',
            name: t.combatEnemyLabel,
            data: enemySeries,
            itemStyle: { color: colorEnemy, opacity: 0.8, borderColor: colorEnemy, borderWidth: 1 },
          },
          // Outlines (z=4) — même tracé, plus large, couleur foreground
          {
            type: 'line',
            name: maOutlineName,
            data: teamMA,
            lineStyle: { color: tc.text, width: maOutlineWidth, opacity: 0.95 },
            symbol: 'none',
            smooth: true,
            silent: true,
            tooltip: { show: false },
            legendHoverLink: false,
            z: 4,
          },
          {
            type: 'line',
            name: maOutlineName,
            data: enemyMA,
            lineStyle: { color: tc.text, width: maOutlineWidth, opacity: 0.95 },
            symbol: 'none',
            smooth: true,
            silent: true,
            tooltip: { show: false },
            legendHoverLink: false,
            z: 4,
          },
          // Courbes MA colorées (z=5)
          {
            type: 'line',
            name: `MA ${t.combatTeamLabel}`,
            data: teamMA,
            lineStyle: { color: colorTeam, width: maInnerWidth, opacity: 1 },
            itemStyle: { color: colorTeam },
            symbol: 'none',
            smooth: true,
            z: 5,
          },
          {
            type: 'line',
            name: `MA ${t.combatEnemyLabel}`,
            data: enemyMA,
            lineStyle: { color: colorEnemy, width: maInnerWidth, opacity: 1 },
            itemStyle: { color: colorEnemy },
            symbol: 'none',
            smooth: true,
            z: 5,
          },
        ],
      }
    },
    [cadence, scoreboard, meXUID, t.combatKillsLabel, t.combatTeamLabel, t.combatEnemyLabel],
  )

  return (
    <ChartCard
      title={t.combatCadenceTitle}
      series={series}
      height={320}
      buildOption={buildOption}
      emptyMessage={t.combatNoData}
    />
  )
}
