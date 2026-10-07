/**
 * TendancesCalendar — la carte « Calendrier des résultats » : une case par jour joué sur
 * l'horizon, semaines en colonnes (lundi en tête), jours en lignes (lundi en haut), colorée du
 * taux de victoire du jour.
 *
 * Seules les briques existantes : `SectionCard` + `titleWithInfo` pour la carte, le wrapper de
 * grille canonique `Heatmap2DChart` sans cadre, rampe divergente bornée à 0..1 dont la légende
 * en dégradé est VERTICALE, à droite (norme des légendes en dégradé, demande utilisateur du
 * 2026-10-07), aucun nombre dans les cases, jours non joués masqués. Aucun jour joué : la
 * notice d'état vide.
 */
import { useMemo } from 'react'

import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { buildCalendarGrid, formatCalendarTooltip } from './tendancesCalendar.logic'
import { formatTrendValue, type Horizon } from './tendances.logic'

const VALUE_RANGE: [number, number] = [0, 1]
/**
 * Marges du tracé avec la légende VERTICALE à droite : l'orientation verticale du wrapper
 * réserve à gauche et en bas la place de titres d'axes que ce calendrier n'a pas.
 */
const GRID_OVERRIDE = { top: 16, bottom: 32, left: 64 }
/** Hauteur du graphique (sept lignes de jours), en pixels. */
const CHART_HEIGHT = 240

export interface TendancesCalendarProps {
  locale: Locale
  data: TrendsPageResponse
  horizon: Horizon
}

export function TendancesCalendar({ locale, data, horizon }: TendancesCalendarProps) {
  const t = getTendancesText(locale)
  const grid = useMemo(
    () =>
      buildCalendarGrid({
        calendar: data.calendar,
        asOf: data.as_of,
        timeZone: data.timezone,
        horizon,
        locale,
      }),
    [data.calendar, data.as_of, data.timezone, horizon, locale],
  )
  const series = useMemo(() => [{ key: 'calendar', datapoints: grid.points }], [grid.points])
  const axisTuning = useMemo(() => ({ xLabelInterval: grid.labelInterval }), [grid.labelInterval])
  const formatTooltip = useMemo(
    () => (point: ChartPointHeatmap) => formatCalendarTooltip(point, locale),
    [locale],
  )
  const formatLegend = useMemo(
    () => (value: number) => formatTrendValue(value, 'ratio', 2, locale),
    [locale],
  )

  return (
    <SectionCard title={t.calendarTitle} titleAdornment={titleWithInfo(t.calendarInfo)}>
      <div data-testid="tendances-calendar">
        {grid.playedDays === 0 ? (
          <div className="p-3">
            <EmptyStateNotice
              title={t.calendarEmptyTitle}
              description={t.calendarEmptyDescription}
            />
          </div>
        ) : (
          <Heatmap2DChart
            series={series}
            height={CHART_HEIGHT}
            frameless
            paletteMode="divergent"
            valueRange={VALUE_RANGE}
            visualMapOrient="vertical"
            visualMapFormatter={formatLegend}
            gridOverride={GRID_OVERRIDE}
            emptyCells="hidden"
            showCellLabel={false}
            yAxisInverse
            axisTuning={axisTuning}
            formatTooltip={formatTooltip}
          />
        )}
      </div>
    </SectionCard>
  )
}
