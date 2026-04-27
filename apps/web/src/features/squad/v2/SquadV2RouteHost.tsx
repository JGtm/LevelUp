/**
 * SquadV2RouteHost — bridge entre la layout Squad legacy (SquadContext) et
 * la page V2 (chunk S12).
 *
 * Lit `confirmedGamertags` depuis SquadContext (peuplé par SquadLayout via
 * GamertagCombobox) et les passe à `<SquadV2Page>` comme `teammates`. Garde
 * SquadV2Page découplé du store / context pour rester testable seul.
 */
import { useParams } from '@tanstack/react-router'

import { useSquadContext } from '../SquadContext'

import { SquadV2Page } from './SquadV2Page'

export function SquadV2RouteHost() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { confirmedGamertags } = useSquadContext()
  return (
    <SquadV2Page
      playerSlug={playerSlug}
      teammates={confirmedGamertags}
      // Period sera derive du global filter store dans une iteration ulterieure
      // (S13). Pour S12 on prend "all" par defaut — la session active n'est pas
      // forcement filtrable cote endpoint V2 sans parsing de la session label.
      period="all"
    />
  )
}
