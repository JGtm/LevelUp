/**
 * ActivityCalendarChart — calendrier d'activité 90 j (jours joués) de l'onglet
 * Réalisations (DEC-5/D3).
 *
 * Rendu type « GitHub contributions » : une colonne par semaine, une ligne par
 * jour de semaine (lundi en haut). Une case remplie = un jour joué ; l'intensité
 * (rampe NEUTRE de fréquence, CVD-safe) reflète le nombre de matchs. Les jours
 * non joués restent des cases vides (fond).
 *
 * Compose `ChartCard` (wrapper ECharts partagé — pas d'echarts nu dans la feature)
 * avec un `buildOption` dédié, sur le modèle d'ExplorerActivityHeatmapChart. Tokens
 * sémantiques uniquement (rampe centralisée `heatmapColors`), i18n FR/EN.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { dowLabels, calendarChartText } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'
import { getAscensionText } from './i18n'
import type { ActivityDay } from './types'

const DAY_MS = 86_400_000

interface ActivityCalendarChartProps {
  since: string
  until: string
  days: ActivityDay[]
  height?: number
}

// ── Géométrie pure du calendrier (semaine × jour) ────────────────────────────

/** Parse un jour UTC "YYYY-MM-DD" en Date UTC minuit (pas de dérive TZ locale). */
function toUTCDate(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(Date.UTC(y, (m || 1) - 1, d || 1))
}

/** Index du jour de semaine, lundi = 0 … dimanche = 6 (aligné backend `dow`). */
function mondayIndex(date: Date): number {
  return (date.getUTCDay() + 6) % 7
}

interface CalendarCell {
  week: number
  weekday: number
  count: number
  date: string
}

interface CalendarGrid {
  cells: CalendarCell[]
  weeks: number
  maxCount: number
}

/** Projette les jours joués sur une grille semaine × jour de semaine. */
// eslint-disable-next-line react-refresh/only-export-components
export function buildCalendarGrid(since: string, until: string, days: ActivityDay[]): CalendarGrid {
  const start = toUTCDate(since)
  // Origine = lundi de la semaine contenant `since` (colonne 0 alignée).
  const originMonday = new Date(start.getTime() - mondayIndex(start) * DAY_MS)
  const untilDiff = Math.round((toUTCDate(until).getTime() - originMonday.getTime()) / DAY_MS)
  const weeks = Math.max(1, Math.floor(untilDiff / 7) + 1)

  let maxCount = 0
  const cells: CalendarCell[] = []
  for (const d of days) {
    if (d.count <= 0) continue
    const diff = Math.round((toUTCDate(d.date).getTime() - originMonday.getTime()) / DAY_MS)
    if (diff < 0) continue
    cells.push({
      week: Math.floor(diff / 7),
      weekday: diff % 7,
      count: d.count,
      date: d.date,
    })
    if (d.count > maxCount) maxCount = d.count
  }
  return { cells, weeks, maxCount }
}

// ── Option ECharts ───────────────────────────────────────────────────────────

interface BuildOpts {
  since: string
  until: string
  days: ActivityDay[]
  locale: ManifestLocale
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildActivityCalendarOption(opts: BuildOpts): EChartsCoreOption {
  const { since, until, days, locale } = opts
  const tc = getEChartsThemeColors()
  const DOW = dowLabels(locale)
  const chartTxt = calendarChartText(locale)
  const t = getAscensionText(locale)

  const grid = buildCalendarGrid(since, until, days)
  const hasData = grid.cells.length > 0

  // data ECharts heatmap : [weekIndex, weekdayIndex, count] + date pour le tooltip.
  const data = grid.cells.map((c) => ({ value: [c.week, c.weekday, c.count], date: c.date }))
  const weekCategories = Array.from({ length: grid.weeks }, (_, i) => String(i))

  return {
    backgroundColor: CHART_BG,
    grid: { left: 44, right: 16, top: 12, bottom: 44, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data?: { value: [number, number, number]; date: string } }) => {
        const d = params.data
        if (!d) return ''
        return `${escapeHtml(d.date)}<br>${escapeHtml(chartTxt.matches)} : <b>${d.value[2]}</b>`
      },
    },
    xAxis: {
      type: 'category',
      data: weekCategories,
      // Colonnes de semaines : repères visuels sans étiquettes (calendrier sobre).
      axisLabel: { show: false },
      axisTick: { show: false },
      axisLine: { show: false },
      splitArea: { show: false },
    },
    yAxis: {
      type: 'category',
      data: DOW,
      inverse: true, // lundi en haut
      axisLabel: { color: tc.axisLabel, fontSize: 10 },
      axisTick: { show: false },
      axisLine: { show: false },
      splitArea: { show: false },
    },
    visualMap: hasData
      ? {
          min: 0,
          max: grid.maxCount,
          calculable: false,
          show: true,
          orient: 'horizontal',
          left: 'center',
          bottom: 4,
          itemWidth: 10,
          itemHeight: 90,
          // Rampe NEUTRE de fréquence (mono-teinte, luminance monotone, CVD-safe) :
          // intensité d'activité, pas une perf → rampe centralisée (heatmapColors).
          inRange: { color: heatmapRampTokens('frequency').map(resolveToken) },
          // Légende sobre : « Moins » … « Plus ».
          text: [t.activityCalendarLegendMore, t.activityCalendarLegendLess],
          textStyle: { color: tc.axisLabel, fontSize: 10 },
        }
      : undefined,
    series: [
      {
        type: 'heatmap',
        data,
        itemStyle: { borderColor: CHART_BG, borderWidth: 2 },
        emphasis: { itemStyle: { shadowBlur: 6 } },
      },
    ],
  }
}

export function ActivityCalendarChart({ since, until, days, height }: ActivityCalendarChartProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = getAscensionText(locale)

  // Série non vide uniquement s'il y a des jours joués (sinon ChartCard affiche
  // l'état vide propre).
  const series: ChartSeries<ActivityDay>[] =
    days.length > 0 ? [{ key: 'activity', datapoints: days }] : []

  const buildOption = useCallback(
    () => buildActivityCalendarOption({ since, until, days, locale }),
    [since, until, days, locale],
  )

  return (
    <section aria-label={t.activityCalendarAria}>
      <ChartCard
        title={t.activityCalendarTitle}
        series={series}
        emptyMessage={t.activityCalendarEmpty}
        buildOption={buildOption as (s: ChartSeries<ActivityDay>[]) => EChartsCoreOption}
        height={height ?? 180}
      />
    </section>
  )
}
