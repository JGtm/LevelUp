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
 *
 * LA GÉOMÉTRIE VIT DANS `_cadence.ts` (registre 2026-09-05, N1) : empilements, moyennes
 * mobiles, pic et libellés d'abscisse y sont purs et testés. Ce fichier ne fait plus que
 * l'HABILLAGE — les couleurs, qui dépendent de la palette d'accessibilité au moment du
 * rendu, l'ordre de superposition et les libellés d'axe.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getLegendBase, getTooltipBase, type EChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { MatchScoreboardRow, MatchViewCadence } from '@/lib/api/types'
import { buildCadence, type Cadence } from './_cadence'
import type { MatchViewText } from './i18n'
import { resolveXuidMeta } from './xuidMeta'

interface Props {
  cadence: MatchViewCadence | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
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

  // Cascade « allié = même camp que moi » : source unique (xuidMeta.ts).
  const model = useMemo(
    () => buildCadence({ cadence, meXUID, xuidMeta: resolveXuidMeta(scoreboard, meXUID) }),
    [cadence, scoreboard, meXUID],
  )

  const buildOption = useCallback(
    (s: ChartSeries<unknown>[]): EChartsCoreOption => {
      if (s.length === 0 || !model) return { backgroundColor: CHART_BG }
      return cadenceOption(model, getEChartsThemeColors(), t)
    },
    [model, t],
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

/** Nom réservé des doublures d'épaisseur : hors légende, un seul item par moyenne mobile. */
const MA_OUTLINE_NAME = '__ma_outline__'
/** Épaisseur de la doublure et du trait, en pixels — l'écart des deux FAIT la bordure. */
const MA_OUTLINE_WIDTH = 6
const MA_INNER_WIDTH = 3

/**
 * cadenceOption — l'option ECharts, extraite du composant pour rester lisible.
 *
 * LES COULEURS SE RÉSOLVENT ICI, dans le builder : leur valeur calculée change avec la
 * palette d'accessibilité, et c'est ce rebuild qui la rafraîchit (cf. ChartCard).
 */
function cadenceOption(model: Cadence, tc: EChartsThemeColors, t: MatchViewText): EChartsCoreOption {
  const axis = getAxisBase(tc)
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
      data: model.categories,
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
    series: cadenceSeries(model, tc, t),
  }
}

/**
 * cadenceSeries — les six séries du graphe : deux empilements, deux doublures, deux moyennes.
 *
 * LES DOUBLURES (z=4) sont le MÊME tracé, plus large, à la couleur de premier plan du thème :
 * c'est ce qui détache les courbes des barres empilées sans poser de teinte en dur
 * (`tc.text` = `--foreground`, qui s'inverse clair/sombre).
 */
function cadenceSeries(
  model: Cadence,
  tc: EChartsThemeColors,
  t: MatchViewText,
): Record<string, unknown>[] {
  const colorTeam = resolveToken('team-ally')
  const colorEnemy = resolveToken('team-enemy')
  const colorPic = resolveToken('warning')
  const doublure = (data: number[]) => ({
    type: 'line',
    name: MA_OUTLINE_NAME,
    data,
    lineStyle: { color: tc.text, width: MA_OUTLINE_WIDTH, opacity: 0.95 },
    symbol: 'none',
    smooth: true,
    silent: true,
    tooltip: { show: false },
    legendHoverLink: false,
    z: 4,
  })
  const moyenne = (name: string, data: number[], color: string) => ({
    type: 'line',
    name,
    data,
    lineStyle: { color, width: MA_INNER_WIDTH, opacity: 1 },
    itemStyle: { color },
    symbol: 'none',
    smooth: true,
    z: 5,
  })
  return [
    {
      type: 'bar',
      stack: 'total',
      name: t.combatTeamLabel,
      data: model.ally,
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
        data: model.peak.total > 0 ? [{ coord: [model.peak.index, model.peak.total] }] : [],
      },
    },
    {
      type: 'bar',
      stack: 'total',
      name: t.combatEnemyLabel,
      data: model.enemy,
      itemStyle: { color: colorEnemy, opacity: 0.8, borderColor: colorEnemy, borderWidth: 1 },
    },
    doublure(model.allyMA),
    doublure(model.enemyMA),
    moyenne(`MA ${t.combatTeamLabel}`, model.allyMA, colorTeam),
    moyenne(`MA ${t.combatEnemyLabel}`, model.enemyMA, colorEnemy),
  ]
}
