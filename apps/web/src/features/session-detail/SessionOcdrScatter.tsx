/**
 * SessionOcdrScatter — nuage de points Rendement offensif (x) × Résistance défensive (y),
 * 1 point par match, coloré par issue (win/loss/tie/dnf). Montre la répartition des matchs
 * (offensif / défensif) + la corrélation. Remplace le tableau/donut OC-DR en vue single.
 */
import { useMemo } from 'react'

import type { ChartSeries } from '@/components/charts/ChartCard'
import { ScatterChart, type ChartPointScatter } from '@/components/charts/ScatterChart'
import type { SemanticToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { outcomeIntToKey, useSessionT } from './_shared'

const OUTCOME_TOKEN: Record<string, SemanticToken> = {
  win: 'outcome-win',
  loss: 'outcome-loss',
  tie: 'outcome-draw',
  dnf: 'outcome-dnf',
}

const round2 = (n: number) => Math.round(n * 100) / 100

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionOcdrScatter({ title, matches, height = 280 }: Props) {
  const t = useSessionT()
  const { data: fieldMappings } = useFieldMappings()

  const { series, tokens } = useMemo(() => {
    const buckets: Record<string, ChartPointScatter[]> = {}
    for (const m of matches) {
      if (m.offensive_conversion == null || m.defensive_resistance == null) continue
      const key = outcomeIntToKey(m.outcome ?? null) ?? 'dnf'
      ;(buckets[key] ??= []).push({ x: round2(m.offensive_conversion), y: round2(m.defensive_resistance) })
    }
    const outcomeLabel = (key: string): string => fieldMappings?.outcomes?.[key]?.label ?? key
    const built: ChartSeries<ChartPointScatter>[] = Object.entries(buckets).map(([key, pts]) => ({
      key,
      meta: { gamertag: outcomeLabel(key) },
      datapoints: pts,
    }))
    const colorTokens: Record<string, SemanticToken> = {}
    for (const key of Object.keys(buckets)) colorTokens[key] = OUTCOME_TOKEN[key] ?? 'outcome-dnf'
    return { series: built, tokens: colorTokens }
  }, [matches, fieldMappings])

  return (
    <ScatterChart
      title={title}
      series={series}
      height={height}
      xAxisLabel={t('session.detail.ocdr_axis_oc')}
      yAxisLabel={t('session.detail.ocdr_axis_dr')}
      symbolSize={9}
      seriesColorTokens={tokens}
      seriesNameResolver={(s) => (s.meta?.gamertag as string | undefined) ?? s.key}
    />
  )
}
