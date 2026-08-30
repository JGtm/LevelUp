/**
 * TimeseriesKdaTrend — chart timeseries.02 (KPIs : "Évolution Frags / Morts")
 *
 * Frags + Morts en barres groupées par match (étiquettes X `#N\nMap`).
 * La courbe FDA Y2 a été retirée (chart "FDA" dédié sur la même page).
 */
import { useMemo, useState, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  stackedAxisExtent,
} from '@/components/charts/_utils'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { resolveToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'
import { ChartFromOption } from './ChartFromOption'

export interface TimeseriesKdaTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: {
    kills: string
    deaths: string
    yAxis: string
    /**
     * Libellé de la série « Bonus » (assistances ÷ 3), masquée par défaut.
     *
     * COUPLAGE ASSUMÉ label/clé : dans une légende ECharts NATIVE, le `name` de
     * la série EST la clé de `legend.selected` et de l'événement
     * `legendselectchanged` — il n'existe pas de champ clé distinct (contrairement
     * à la légende React de SquadToggleLegendChart, où le 2.2 a pu garder la clé
     * 'Bonus' non localisée à côté d'un libellé traduit). Localiser le libellé
     * localise donc la clé. C'est SANS danger ici parce que la valeur est lue
     * d'une seule variable, utilisée aux quatre points de couplage (series.name,
     * legend.data, legend.selected, handler) : aucun littéral 'Bonus' ne subsiste
     * dans le composant. Ne JAMAIS réintroduire un 'Bonus' en dur à l'un de ces
     * points — la série deviendrait impossible à afficher dans l'autre langue.
     */
    bonus: string
    /** Aide ⓘ : ce que vaut une assistance dans le FDA (ADR 0006). */
    bonusInfo: string
  }
}

export function TimeseriesKdaTrend({ rows, height = 360, title, emptyMessage, labels }: TimeseriesKdaTrendProps) {
  const themeVersion = useThemeVersion()
  const [showBonus, setShowBonus] = useState(false)

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const colKills = resolveToken('chart-series-1')
    const colDeaths = resolveToken('outcome-loss')
    const colBonus = resolveToken('bonus') // bonus assistances — violet, distinct du bleu frags / rouge morts

    const categories = buildMatchCategories(rows)
    const kills = rows.map((r) => r.kills)
    const deaths = rows.map((r) => r.deaths)
    const bonus = rows.map((r) => r.assists / 3)
    // Étendue d'axe calculée sur le JEU COMPLET (bonus toujours inclus, qu'il
    // soit affiché ou masqué via la légende) — item 5, DEC-5 : sans ça, ECharts
    // recalcule l'axe sur les séries visibles à chaque render et le toggle
    // « Bonus » fait bouger toute l'échelle. Kills+bonus empilés (positif) ;
    // morts jamais négatives ici (pas de `stack`) → min fixé à 0.
    const { min: yMin, max: yMax } = stackedAxisExtent([[kills, bonus], [deaths]])

    return {
      backgroundColor: CHART_BG,
      grid: { top: 32, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      legend: {
        ...getLegendBase(tc),
        bottom: 0,
        data: [labels.kills, labels.deaths, labels.bonus],
        selected: { [labels.bonus]: showBonus },
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          interval: 0,
          fontSize: 9,
        },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
        name: labels.yAxis,
        nameTextStyle: { color: tc.axisLabel, fontSize: 11 },
        min: yMin,
        max: yMax,
        minInterval: 1,
      },
      series: [
        {
          name: labels.kills,
          type: 'bar',
          stack: 'kills',
          data: kills,
          itemStyle: { color: colKills, opacity: 0.85 },
          barGap: 0,
          barMaxWidth: 14,
        },
        {
          name: labels.bonus,
          type: 'bar',
          stack: 'kills',
          data: bonus,
          itemStyle: { color: colBonus, opacity: 0.85 },
          barMaxWidth: 14,
        },
        {
          name: labels.deaths,
          type: 'bar',
          data: deaths,
          itemStyle: { color: colDeaths, opacity: 0.85 },
          barMaxWidth: 14,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, labels, themeVersion, showBonus])

  const onEvents = useMemo(
    () => ({
      legendselectchanged: (params: unknown) => {
        const p = params as { selected: Record<string, boolean> }
        setShowBonus(p.selected[labels.bonus] ?? false)
      },
    }),
    [labels.bonus],
  )

  return (
    <ChartFromOption
      // L'aide ⓘ est accrochée au TITRE et non à l'item de légende : la légende
      // est rendue par ECharts (canvas), aucun nœud React ne peut y être inséré —
      // contrairement à la légende React du 2.2 (SquadToggleLegendChart). Même
      // placement que TimeseriesFdaGapTrend sur cette page.
      title={
        <span className="flex items-center gap-1.5">
          {title}
          <InfoTooltip content={labels.bonusInfo} />
        </span>
      }
      option={option}
      height={height}
      emptyMessage={emptyMessage}
      onEvents={onEvents}
    />
  )
}
