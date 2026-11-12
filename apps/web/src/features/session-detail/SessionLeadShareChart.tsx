/**
 * SessionLeadShareChart — « Mainmise, match par match » : la part de chaque match passée en tête
 * du score, une colonne par match, sous « Vies nettes » et avant « Intensité ».
 *
 * NORMALISÉE : l'axe va toujours de 0 à 100 % et le repère à 50 % tombe au même endroit dans les
 * deux colonnes d'une comparaison de sessions — la comparaison se lit sans échelle partagée à
 * calculer. Un match non mesuré reste en pointillé (jamais zéro). La moyenne des matchs mesurés,
 * et leur nombre, se lisent sous le graphe.
 */
import { useMemo } from 'react'

import { LeadShareChart } from '@/components/charts/LeadShareChart'
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'
import { useLeadShareText } from '@/lib/charts/leadShare/text'

import { sessionLeadSharePoints, sessionLeadShareSummary } from './sessionLeadShare.logic'

interface Props {
  matches: SessionDetailMatchRow[]
  entry: SessionCompareEntry | null
  /** Colonne divisée (drawer de comparaison) : colonnes et étiquettes plus serrées. */
  compact?: boolean
}

export function SessionLeadShareChart({ matches, entry, compact = false }: Props) {
  const text = useLeadShareText()
  const points = useMemo(() => sessionLeadSharePoints(matches), [matches])
  const summary = sessionLeadShareSummary(entry, text)
  return <LeadShareChart title={text.perMatchTitle} points={points} compact={compact} summary={summary} />
}
