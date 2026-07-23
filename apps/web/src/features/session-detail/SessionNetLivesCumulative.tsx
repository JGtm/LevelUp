/**
 * SessionNetLivesCumulative — « Balance des dégâts cumulée » sur les matchs d'une
 * session (P3). Somme CUMULÉE de la balance `netLives = (dégâts infligés − dégâts
 * subis) / PV-pour-tuer` (en vies), match après match dans l'ordre chronologique.
 *
 * Un match sans dégâts subis/infligés est SAUTÉ : la courbe reporte la dernière
 * valeur cumulée (D5, délégué au helper générique `cumulativeSigned`).
 *
 * Même pattern visuel que `SessionFdaGapCumulative` : aire signée divergente
 * ancrée à 0 (`divergentZeroGradient`, PAS de visualMap) + markLine 0 + pastille
 * KPI « balance moyenne par match ». Masqué par `useProvidesDamageTaken()`
 * (capability `damage_taken` — Halo 5 n'a pas de dégâts subis).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { cumulativeSigned, meanOfValid } from '@/lib/charts/cumulativeSeries'
import { netLives } from '@/lib/charts/netLives'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useEffectiveHpToKill, substituteHpToken, useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

export interface NetLivesPoint {
  /** Étiquette d'axe X du match (`#N` + carte/mode). */
  label: string
  /** Balance du match (vies nettes), null si dégâts absents (D5). */
  value: number | null
  /** Balance cumulée (reporte la dernière valeur si le match n'a pas de dégâts). */
  cumulative: number
}

/**
 * Cumul de la balance des dégâts sur les matchs, TRIÉS chronologiquement puis
 * délégué au helper générique `cumulativeSigned` (source unique du cumul,
 * CLAUDE.md n°6). D5 : un match sans dégâts n'ajoute rien au cumul (report) mais
 * figure quand même sur l'axe.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function computeCumulativeNetLives(
  matches: SessionDetailMatchRow[],
  hp: number,
): NetLivesPoint[] {
  const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
  const cum = cumulativeSigned(sorted.map((m) => netLives(m.damage_dealt, m.damage_taken, hp)))
  return sorted.map((m, i) => ({
    label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
    value: cum[i].value,
    cumulative: cum[i].cumulative,
  }))
}

export interface NetLivesLabels {
  seriesLabel: string
  matchLabel: string
  yDomain?: [number, number]
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionNetLivesOption(
  series: ChartSeries<NetLivesPoint>[],
  opts: NetLivesLabels,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  const values = points.map((p) => p.cumulative)
  const divergentColor = divergentZeroGradient(values)
  const fmt = (v: number | null, signed = false) =>
    v == null ? '—' : signed && v >= 0 ? `+${v}` : `${v}`

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ dataIndex?: number }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = points[params[0]?.dataIndex ?? 0]
        if (!p) return ''
        const cat = escapeHtml(p.label.replace('\n', ' · '))
        return (
          `${cat}<br/>` +
          `${escapeHtml(opts.seriesLabel)}: <b>${fmt(p.cumulative, true)}</b><br/>` +
          `${escapeHtml(opts.matchLabel)}: ${fmt(p.value, true)}`
        )
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    yAxis: {
      ...axis,
      type: 'value',
      ...(opts.yDomain ? { min: opts.yDomain[0], max: opts.yDomain[1] } : {}),
    },
    series: [
      {
        name: opts.seriesLabel,
        type: 'line',
        data: values,
        showSymbol: false,
        // Ligne + aire divergentes (vert = balance positive cumulée, rouge négative),
        // aire ancrée à 0 (même dégradé, bascule pile sur 0).
        lineStyle: { width: 2, color: divergentColor },
        areaStyle: { color: divergentColor, opacity: 0.18, origin: 0 },
        // Ligne de référence à 0 (équilibre : dégâts infligés = dégâts subis).
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed', width: 1 },
          label: { show: false },
          data: [{ yAxis: 0 }],
        },
      },
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
  /** Domaine Y [min, max] partagé A/B en mode comparaison (sinon auto-scale). */
  yDomain?: [number, number]
}

export function SessionNetLivesCumulative({ title, matches, height = 280, yDomain }: Props) {
  const providesDamageTaken = useProvidesDamageTaken()
  const hp = useEffectiveHpToKill()
  const locale = useAppShellStore((s) => s.locale)
  const t = useSessionT()

  const points = useMemo(
    () => (matches.length === 0 ? [] : computeCumulativeNetLives(matches, hp)),
    [matches, hp],
  )
  const series = useMemo<ChartSeries<NetLivesPoint>[]>(
    () => (points.length === 0 ? [] : [{ key: 'net_lives', datapoints: points }]),
    [points],
  )
  const mean = useMemo(() => meanOfValid(points.map((p) => p.value)), [points])

  // Titre sans dégâts subis (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!providesDamageTaken) return null

  const nf = new Intl.NumberFormat(locale === 'en' ? 'en-US' : 'fr-FR', {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
    signDisplay: 'always',
  })

  return (
    <ChartCard
      title={
        <span className="flex items-center gap-1.5">
          {title}
          <InfoTooltip content={substituteHpToken(t('session.detail.net_lives_tooltip'), hp)} />
        </span>
      }
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionNetLivesOption(s, {
          seriesLabel: t('session.detail.net_lives_series'),
          matchLabel: t('session.detail.net_lives_match'),
          yDomain,
        })
      }
    >
      <div
        className="flex flex-col gap-1.5 border-t border-border px-3 py-2"
        data-testid="net-lives-kpi"
      >
        <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t('session.detail.net_lives_average_caption')}
        </span>
        <span className="tabular-nums font-semibold text-foreground">
          {mean == null
            ? '—'
            : `${nf.format(mean)} ${t('session.detail.net_lives_average_unit')}`}
        </span>
      </div>
    </ChartCard>
  )
}
