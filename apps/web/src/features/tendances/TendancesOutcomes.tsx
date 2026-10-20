/**
 * TendancesOutcomes — les deux graphiques en haltères de la page : « Moyenne par match en
 * défaite et en victoire » (sous le titre de section « Victoires et défaites ») et « Médailles
 * par match ». Un même cadre `ChartCard` (titre + bulle par `titleWithInfo`, état vide du
 * cadre) monte le builder d'haltères ; la hauteur suit le nombre de lignes.
 *
 * Les deux suivent l'horizon de la page et ne relisent rien : le web choisit le bloc de la
 * réponse qui porte cet horizon.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { SectionTitle } from '@/components/ui/detail-section'
import { titleWithInfo } from '@/components/ui/title-with-info'
import type { SemanticToken } from '@/lib/accessibility'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { useIndicatorLabeler } from './labels'
import { formatTrendValue, type Horizon } from './tendances.logic'
import {
  buildTendancesDumbbellOption,
  dumbbellHeight,
  type DumbbellRow,
} from './tendancesDumbbell.logic'
import { buildMedalRows, medalsBlock } from './tendancesMedals.logic'
import { buildWinLossRows, winLossBlock, winLossShortfall } from './tendancesWinLoss.logic'

interface DumbbellCardProps {
  testId: string
  title: string
  info: string
  rows: readonly DumbbellRow[]
  nameA: string
  nameB: string
  colorA: SemanticToken
  colorB: SemanticToken
  reference?: number
  xAxisLabel: ((value: number) => string) | null
  emptyTitle: string
  emptyMessage: string
  /** Remplit la cellule de grille que la voisine étire (rangée « Médailles » / « Types de partie »). */
  fluid?: boolean
}

function DumbbellCard(props: DumbbellCardProps) {
  const { rows, nameA, nameB, colorA, colorB, reference, xAxisLabel, testId } = props
  const series = useMemo<ChartSeries<DumbbellRow>[]>(
    () => (rows.length > 0 ? [{ key: testId, datapoints: [...rows] }] : []),
    [rows, testId],
  )
  const buildOption = useCallback(
    (s: ChartSeries<DumbbellRow>[]) =>
      buildTendancesDumbbellOption({
        rows: s[0]?.datapoints ?? [],
        nameA,
        nameB,
        colorA,
        colorB,
        reference,
        xAxisLabel,
      }),
    [nameA, nameB, colorA, colorB, reference, xAxisLabel],
  )
  return (
    <div className={props.fluid ? 'h-full' : undefined} data-testid={testId}>
      <ChartCard
        title={titleWithInfo(props.info)(props.title)}
        series={series}
        buildOption={buildOption}
        height={dumbbellHeight(rows.length)}
        emptyTitle={props.emptyTitle}
        emptyMessage={props.emptyMessage}
        fluid={props.fluid}
      />
    </div>
  )
}

export interface TendancesOutcomesProps {
  locale: Locale
  data: TrendsPageResponse
  horizon: Horizon
}

export function TendancesWinLoss({ locale, data, horizon }: TendancesOutcomesProps) {
  const t = getTendancesText(locale)
  const labelOf = useIndicatorLabeler(locale)
  const block = winLossBlock(data, horizon)
  const rows = useMemo(() => buildWinLossRows(block, locale, labelOf), [block, locale, labelOf])
  const { n, required } = winLossShortfall(block)
  return (
    <section className="space-y-2" data-testid="tendances-outcomes">
      <SectionTitle>{t.outcomesSection}</SectionTitle>
      <DumbbellCard
        testId="tendances-winloss"
        title={t.winLossTitle}
        info={t.winLossInfo}
        rows={rows}
        nameA={t.winLossSeriesLoss}
        nameB={t.winLossSeriesWin}
        colorA="outcome-loss"
        colorB="outcome-win"
        reference={0}
        xAxisLabel={null}
        emptyTitle={t.winLossEmptyTitle}
        emptyMessage={t.winLossEmptyDescription(n, required)}
      />
    </section>
  )
}

export function TendancesMedals({ locale, data, horizon }: TendancesOutcomesProps) {
  const t = getTendancesText(locale)
  const block = medalsBlock(data, horizon)
  const nameB = t.horizonLong(horizon)
  const nameA = t.medalsSeriesPrevious(horizon)
  const rows = useMemo(
    () => buildMedalRows(block, locale, { current: nameB, previous: nameA }),
    [block, locale, nameA, nameB],
  )
  const xAxisLabel = useCallback(
    (value: number) => formatTrendValue(value, 'number', 2, locale),
    [locale],
  )
  return (
    <DumbbellCard
      testId="tendances-medals"
      title={t.medalsTitle}
      info={t.medalsInfo}
      rows={rows}
      nameA={nameA}
      nameB={nameB}
      colorA="divergent-neutral"
      colorB="chart-series-1"
      xAxisLabel={xAxisLabel}
      emptyTitle={t.medalsEmptyTitle}
      emptyMessage={t.medalsEmptyDescription}
      fluid
    />
  )
}
