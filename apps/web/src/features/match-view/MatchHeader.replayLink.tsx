/**
 * MatchHeader.replayLink.tsx — la porte d'entrée du rejeu 2D, et la SEULE.
 *
 * Extrait de MatchHeader.card.tsx, qui suit déjà ce découpage (`.perfRank`, `.utils`) et
 * approchait le seuil de 500 lignes.
 */
import { Link } from '@tanstack/react-router'

import { themedIconSrc } from '@/lib/themedIcon'
import { useTitleSlug } from '@/lib/title-routing'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

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
  // Le thème LOCAL, celui que le store a déjà tranché (`dark` | `light`) : l'icône est un
  // raster à deux variantes, elle ne peut pas se teinter comme un SVG en `currentColor`.
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  if (!available) return null
  return (
    <Link
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay"
      params={{ titleSlug, playerSlug, matchId }}
      title={t.replayTooltip}
      aria-label={t.replayTooltip}
      className="inline-flex h-8 items-center justify-center gap-1.5 rounded-md border border-border bg-transparent px-3 text-xs font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-4 w-auto" />
      {t.replayShort}
    </Link>
  )
}
