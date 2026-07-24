/**
 * SessionIntensityProfile — « Intensité » sur les matchs d'UNE session (solo).
 *
 * Panneau unique : médiane des parts de frags par phase de match + enveloppe
 * interquartile P25–P75 (irrégularité). Réutilise le builder multi-grilles P1
 * (`buildSquadIntensityProfileOption`) en N=1 — aucune duplication de géométrie.
 *
 * Les phases par match viennent du payload session (`intensity_rows` /
 * `compare_intensity_rows`), calculées côté Go en MIROIR du chart Timeseries
 * (buildIntensityRows). Le builder omet `series` si aucune manche exploitable →
 * `hasExploitableMatch` sert l'état vide dans le bloc titré.
 */
import { useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { resolveToken } from '@/lib/accessibility'
import { phaseShares } from '@/lib/charts/phaseProfile'
import { useAppShellStore } from '@/stores/appShellStore'
import type { IntensityMatchRow } from '@/lib/api/types'
import {
  buildSquadIntensityProfileOption,
  intensityAxisLabels,
  type IntensityPanelInput,
} from '@/features/squad/charts/squadIntensityProfileChart'

import { useSessionT } from './_shared'

interface Props {
  title: string
  rows: IntensityMatchRow[]
  height?: number
}

export function SessionIntensityProfile({ title, rows, height = 300 }: Props) {
  const t = useSessionT()
  const locale = useAppShellStore((s) => s.locale)

  // Panneau solo unique : couleur série standard du chart, titre de panneau vide
  // (le titre de la carte identifie déjà la session). Absent si aucune manche
  // exploitable (Σ phases > 0) → ChartCard affiche l'état vide.
  const panels = useMemo<IntensityPanelInput[]>(() => {
    if (!rows.some((r) => phaseShares(r.phases) !== null)) return []
    return [
      {
        key: 'solo',
        label: '',
        color: resolveToken('chart-series-2'),
        rows: rows as Array<{ phases: number[] | null }>,
      },
    ]
  }, [rows])

  const series = useMemo<ChartSeries<IntensityPanelInput>[]>(
    () => panels.map((p) => ({ key: p.key, datapoints: [p] })),
    [panels],
  )

  return (
    <ChartCard
      title={
        <div className="flex flex-col gap-0.5">
          <span className="flex items-center gap-1.5">
            {title}
            <InfoTooltip content={t('session.detail.chart_intensity_tooltip')} />
          </span>
          <span className="text-xs font-normal text-muted-foreground">
            {t('session.detail.chart_intensity_subtitle')}
          </span>
        </div>
      }
      series={series}
      height={height}
      emptyMessage={t('session.detail.chart_empty')}
      buildOption={() =>
        buildSquadIntensityProfileOption({
          panels,
          medianLabel: t('session.detail.chart_intensity_median'),
          envelopeLabel: t('session.detail.chart_intensity_envelope'),
          refLabel: t('session.detail.chart_intensity_ref'),
          axisLabels: intensityAxisLabels(locale),
        })
      }
    />
  )
}
