/**
 * LeadShareMapChart — « Mainmise par carte » : la sélection contre l'historique de chaque carte.
 * L'option vit dans `leadShareMapOption.ts` (pur, testé) ; ce composant la pose dans un `ChartCard`.
 *
 * Aucune carte mesurée : l'état vide dit la cause, jamais une carte de zéros.
 */
import { useCallback, useMemo } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { MapBreakdownRow } from '@/lib/api/types'
import { useLeadShareText } from '@/lib/charts/leadShare/text'

import { ChartCard, type ChartSeries } from './ChartCard'
import { buildLeadShareMapOption, leadShareMapRows } from './leadShareMapOption'

interface Props {
  rows: MapBreakdownRow[]
  mapLabelOf: (mapUI: string) => string
  scopeLabel: string
  historyLabel: string
}

export function LeadShareMapChart({ rows, mapLabelOf, scopeLabel, historyLabel }: Props) {
  const text = useLeadShareText()
  const kept = useMemo(() => leadShareMapRows(rows), [rows])
  const series = useMemo<ChartSeries<MapBreakdownRow>[]>(
    () => (kept.length > 0 ? [{ key: 'lead-share-by-map', datapoints: kept }] : []),
    [kept],
  )
  const buildOption = useCallback(
    (s: ChartSeries<MapBreakdownRow>[]) =>
      buildLeadShareMapOption(s[0]?.datapoints ?? [], { text, mapLabelOf, scopeLabel, historyLabel }),
    [text, mapLabelOf, scopeLabel, historyLabel],
  )
  const height = Math.max(200, Math.min(600, Math.min(kept.length, 20) * 32 + 70))
  return (
    <ChartCard
      title={
        <span className="inline-flex items-center gap-2">
          {text.perMapTitle}
          <InfoTooltip content={text.help} />
        </span>
      }
      series={series}
      buildOption={buildOption}
      height={height}
      emptyMessage={text.emptyNoMeasure}
    />
  )
}
