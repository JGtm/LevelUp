/**
 * MatchHeader.replayLink.tsx — la porte d'entrée du rejeu 2D, et la SEULE.
 *
 * Extrait de MatchHeader.card.tsx, qui suit déjà ce découpage (`.perfRank`, `.utils`) et
 * approchait le seuil de 500 lignes.
 */
import { Link } from '@tanstack/react-router'

import { useTitleSlug } from '@/lib/title-routing'

import { MATCH_VIEW_TEXT, type MatchViewLocale } from './i18n'

interface ReplayLinkProps {
  /** `header.replay_available` : la PRÉSENCE de l'artefact, résolue par un os.Stat côté API. */
  available: boolean
  matchId: string
  playerSlug: string
  locale: MatchViewLocale
}

/**
 * ReplayLink ne rend RIEN quand l'artefact n'existe pas.
 *
 * C'est tout l'objet du champ `replay_available` : la route de rejeu répond 404 quand aucun
 * artefact n'a été construit, et un lien qui mène à une page vide se lit comme une panne. En
 * production aucun artefact n'est produit aujourd'hui — donc aucun lien, et rien à expliquer.
 */
export function ReplayLink({ available, matchId, playerSlug, locale }: ReplayLinkProps) {
  const t = MATCH_VIEW_TEXT[locale]
  const titleSlug = useTitleSlug()
  if (!available) return null
  return (
    <Link
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay"
      params={{ titleSlug, playerSlug, matchId }}
      title={t.replayTooltip}
      aria-label={t.replayTooltip}
      className="inline-flex h-8 items-center justify-center gap-1.5 rounded-md border border-border bg-transparent px-3 text-xs font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25m18 0V12a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 12V5.25" />
      </svg>
      {t.replayShort}
    </Link>
  )
}
