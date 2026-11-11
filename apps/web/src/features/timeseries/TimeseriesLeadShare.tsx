/**
 * TimeseriesLeadShare — « Mainmise par mois » : la moyenne des matchs mesurés de chaque mois, sur
 * la population solo complète (comme « Performance solo »). Posée juste après elle, sur la même
 * logique de lecture ; la carte par carte se pose plus bas, sous la paire Taux de victoire /
 * Performance par carte.
 */
import { useMemo } from 'react'

import { LeadShareChart } from '@/components/charts/LeadShareChart'
import type { LeadSharePeriodPoint } from '@/lib/api/types'
import { useLeadShareText } from '@/lib/charts/leadShare/text'
import { useAppShellStore } from '@/stores/appShellStore'

import { leadShareMonthPoints } from './timeseriesLeadShare.logic'

interface Props {
  months: LeadSharePeriodPoint[] | null | undefined
}

export function TimeseriesLeadShareByMonth({ months }: Props) {
  const text = useLeadShareText()
  const locale = useAppShellStore((s) => s.locale)
  const points = useMemo(() => leadShareMonthPoints(months, text, locale), [months, text, locale])
  return <LeadShareChart title={text.perMonthTitle} points={points} />
}
