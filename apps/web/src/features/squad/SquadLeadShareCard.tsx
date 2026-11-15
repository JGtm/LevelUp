/**
 * SquadLeadShareCard — « Mainmise, match par match » de la section Dynamique : la part de chaque
 * match du périmètre passée en tête du score par l'équipe du joueur principal, en courbe (le
 * périmètre peut compter des dizaines de matchs), sur l'axe normalisé de 0 à 100 %.
 *
 * Posée après la rangée « Vies nettes / Écart cumulé » et avant « Engagement ». N'existe que sur un
 * titre qui décode le film ; sans match mesuré, l'état vide dit la cause.
 */
import { useMemo } from 'react'

import { LeadShareChart } from '@/components/charts/LeadShareChart'
import type { SquadLeadSharePoint } from '@/lib/api/types'
import { useLeadShareText } from '@/lib/charts/leadShare/text'

import { squadLeadSharePoints } from './squadLeadShare.logic'

interface Props {
  series: SquadLeadSharePoint[] | null | undefined
}

export function SquadLeadShareCard({ series }: Props) {
  const text = useLeadShareText()
  const points = useMemo(() => squadLeadSharePoints(series), [series])
  return <LeadShareChart title={text.perMatchTitle} points={points} variant="line" />
}
