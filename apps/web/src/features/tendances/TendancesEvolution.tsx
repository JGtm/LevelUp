/**
 * TendancesEvolution — la section « Évolution » : un titre portant sa bulle d'information, la
 * bascule de pas à droite, puis les grilles de graphiques de la vue (Solo : deux grilles sous
 * leur intertitre ; Escouade : une grille de six graphiques, barres empilées comprises), en deux
 * colonnes sur écran large, une sur écran étroit.
 *
 * Seules les briques existantes : `SectionTitle`, `InfoTooltip`, `ChartCard` (hauteur 280, titre
 * + bulle par `titleWithInfo`, légende en bas et centrée dans le builder), `EmptyStateNotice`.
 * L'horizon et le pas viennent de la page ; changer l'un ou l'autre ne relit rien.
 */
import { useCallback, useMemo } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { useIndicatorLabeler } from './labels'
import { stepsForHorizon, type Horizon, type Step, type TendancesView } from './tendances.logic'
import { buildTendancesGrids, type TendancesChartDef } from './tendancesCharts.logic'
import { buildTendancesLinesOption } from './tendancesLines.logic'
import {
  buildTendancesSquadGrids,
  isStackedChart,
  type TendancesStackedChartDef,
} from './tendancesSquadCharts.logic'
import { TendancesSegmented } from './TendancesSegmented'

/** Hauteur d'un graphique, en pixels. */
const CHART_HEIGHT = 280

export interface TendancesEvolutionProps {
  locale: Locale
  data: TrendsPageResponse
  horizon: Horizon
  step: Step
  /** Vue de la page : `solo` (défaut) ou `squad` (une grille de six graphiques). */
  view?: TendancesView
  onStepChange: (step: Step) => void
}

function EvolutionChart({ def }: { def: TendancesChartDef }) {
  const series = useMemo<ChartSeries<unknown>[]>(
    () => [{ key: def.id, datapoints: [...def.input.curves] }],
    [def],
  )
  const buildOption = useCallback(() => buildTendancesLinesOption(def.input), [def])
  return (
    <div data-testid={`tendances-chart-${def.id}`}>
      <ChartCard
        title={titleWithInfo(def.info ?? null)(def.title)}
        series={series}
        buildOption={buildOption}
        height={CHART_HEIGHT}
      />
    </div>
  )
}

/** Un graphique en barres empilées, dans une carte de section comme « Matchs par type de partie ». */
function EvolutionStackedChart({
  def,
  emptyMessage,
}: {
  def: TendancesStackedChartDef
  emptyMessage: string
}) {
  return (
    <div data-testid={`tendances-chart-${def.id}`}>
      <SectionCard title={def.title} titleAdornment={titleWithInfo(def.info ?? null)}>
        <BarStackedChart
          series={def.stacked.series}
          height={CHART_HEIGHT}
          frameless
          componentColors={def.stacked.componentColors}
          componentOrder={def.stacked.componentOrder}
          tooltipHideZero
          emptyMessage={emptyMessage}
        />
      </SectionCard>
    </div>
  )
}

export function TendancesEvolution({
  locale,
  data,
  horizon,
  step,
  view = 'solo',
  onStepChange,
}: TendancesEvolutionProps) {
  const t = getTendancesText(locale)
  const labelOf = useIndicatorLabeler(locale)
  const grids = useMemo(
    () =>
      (view === 'squad' ? buildTendancesSquadGrids : buildTendancesGrids)({
        data,
        horizon,
        step,
        locale,
        labelOf,
      }),
    [view, data, horizon, step, locale, labelOf],
  )
  const stepOptions = stepsForHorizon(horizon).map((s) => ({ value: s, label: t.stepLabel(s) }))

  return (
    <section className="space-y-4" data-testid="tendances-evolution">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <SectionTitle className="flex items-center gap-1.5">
          {t.evolutionTitle}
          <InfoTooltip content={t.evolutionInfo} />
        </SectionTitle>
        <TendancesSegmented
          options={stepOptions}
          value={step}
          onChange={onStepChange}
          ariaLabel={t.stepAria}
        />
      </div>

      {grids.length === 0 ? (
        <EmptyStateNotice title={t.evolutionEmptyTitle} description={t.evolutionEmptyDescription} />
      ) : (
        grids.map((grid) => (
          <div key={grid.id} className="space-y-2" data-testid={`tendances-grid-${grid.id}`}>
            {grid.title ? <SectionTitle>{grid.title}</SectionTitle> : null}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              {grid.charts.map((def) => (
                isStackedChart(def) ? (
                  <EvolutionStackedChart key={def.id} def={def} emptyMessage={t.mixEmptyMessage} />
                ) : (
                  <EvolutionChart key={def.id} def={def} />
                )
              ))}
            </div>
          </div>
        ))
      )}
    </section>
  )
}
