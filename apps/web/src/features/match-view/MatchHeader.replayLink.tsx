/**
 * MatchHeader.replayLink.tsx — la porte d'entrée du rejeu 2D, et la SEULE.
 *
 * Extrait de MatchHeader.card.tsx, qui suit déjà ce découpage (`.perfRank`, `.utils`) et
 * approchait le seuil de 500 lignes.
 */
import { Link } from '@tanstack/react-router'

import { ReplayGenerateAction } from '@/features/_shared/replayGeneration/ReplayGenerateAction'
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
 * ReplayLink rend le lien vers le rejeu quand l'artefact existe ; sinon l'action de génération sous la
 * capacité `can_generate_replays` (bouton, puis pastille d'attente : `ReplayGenerateAction`) ; sinon RIEN.
 *
 * C'est tout l'objet du champ `replay_available` : la route de rejeu répond 404 quand aucun
 * artefact n'a été construit, et un lien qui mène à une page vide se lit comme une panne. Sans artefact
 * ni capacité de générer : aucun lien, et rien à expliquer.
 */
export function ReplayLink({ available, matchId, playerSlug, locale }: ReplayLinkProps) {
  const t = MATCH_VIEW_TEXT[locale]
  const titleSlug = useTitleSlug()
  // Le thème LOCAL, celui que le store a déjà tranché (`dark` | `light`) : l'icône est un
  // raster à deux variantes, elle ne peut pas se teinter comme un SVG en `currentColor`.
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  if (!available) return <ReplayGenerateAction variant="inline" playerSlug={playerSlug} matchId={matchId} />
  return (
    <Link
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay"
      params={{ titleSlug, playerSlug, matchId }}
      title={t.replayTooltip}
      aria-label={t.replayTooltip}
      className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-transparent transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-5 w-auto" />
    </Link>
  )
}
