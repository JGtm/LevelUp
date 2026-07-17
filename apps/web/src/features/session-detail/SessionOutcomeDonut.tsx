/**
 * SessionOutcomeDonut — répartition des issues (victoires / défaites / nuls / abandons)
 * de la session en donut, avec le TAUX DE VICTOIRE affiché au centre.
 *
 * Remplace la bande de séquence d'issues (SessionOutcomeTape) en vue single.
 * Slices colorés par token sémantique (outcome-win/loss/draw/dnf) ; libellés résolus
 * via useFieldMappings (i18n backend-driven).
 */
import { useMemo } from 'react'

import { DonutChart, type ChartPointDonut } from '@/components/charts/DonutChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SemanticToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { outcomeCodeToValue } from '@/lib/outcome'
import { useSessionT } from './_shared'

const OUTCOME_TOKEN: Record<string, SemanticToken> = {
  win: 'outcome-win',
  loss: 'outcome-loss',
  tie: 'outcome-draw',
  dnf: 'outcome-dnf',
}
// Ordre d'affichage stable des slices.
const OUTCOME_ORDER = ['win', 'loss', 'tie', 'dnf'] as const

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
  /** Colonne divisée (drawer) : % dans le donut, pas d'étiquette externe. */
  compact?: boolean
}

export function SessionOutcomeDonut({ title, matches, height = 260, compact }: Props) {
  const t = useSessionT()
  const { data: fieldMappings } = useFieldMappings()

  const { series, sliceColors, winRate, hasData } = useMemo(() => {
    const counts: Record<string, number> = { win: 0, loss: 0, tie: 0, dnf: 0 }
    let total = 0
    for (const m of matches) {
      const key = outcomeCodeToValue(m.outcome)
      if (!key) continue
      counts[key] += 1
      total += 1
    }
    const outcomeLabel = (key: string): string => fieldMappings?.outcomes?.[key]?.label ?? key
    const points: ChartPointDonut[] = OUTCOME_ORDER.filter((k) => counts[k] > 0).map((k) => ({
      name: outcomeLabel(k),
      value: counts[k],
    }))
    const colors: Record<string, SemanticToken> = {}
    for (const k of OUTCOME_ORDER) {
      if (counts[k] > 0) colors[outcomeLabel(k)] = OUTCOME_TOKEN[k]
    }
    const s: ChartSeries<ChartPointDonut>[] = [{ key: 'outcomes', datapoints: points }]
    const wr = total > 0 ? Math.round((counts.win / total) * 100) : 0
    return { series: s, sliceColors: colors, winRate: wr, hasData: total > 0 }
  }, [matches, fieldMappings])

  if (!hasData) return null

  return (
    <DonutChart
      title={title}
      series={series}
      sliceColors={sliceColors}
      height={height}
      compact={compact}
      centerValue={`${winRate} %`}
      centerLabel={t('session.detail.donut_winrate_center')}
    />
  )
}
