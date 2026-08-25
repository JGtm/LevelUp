/**
 * MatchReplayLink — l'icône « Rejeu » d'une ligne de tableau de matchs, et le lien
 * interne vers la page de rejeu 2D du match.
 *
 * UN SEUL composant pour les deux tableaux (Explorer et Synergies escouade) : deux
 * copies divergeraient sur la route ou sur la règle d'affichage, et une ligne qui
 * pointe vers un rejeu inexistant se lit comme une panne.
 *
 * Règle d'affichage : rien n'est rendu quand `available` est faux. La route répond 404
 * sans artefact — c'est le sens du champ `has_replay` servi par l'API, résolu en un seul
 * listing de dossier par requête (jamais un accès disque par ligne).
 *
 * Variante « bouton » de la page match : features/match-view/MatchHeader.replayLink.tsx
 * (même route, présentation différente : bouton avec libellé, pas une cellule).
 */
import { Link } from '@tanstack/react-router'

import { themedIconSrc } from '@/lib/themedIcon'
import { useTitleSlug } from '@/lib/title-routing'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

interface MatchReplayLinkProps {
  /** `has_replay` de la ligne : un artefact de rejeu existe pour ce match. */
  available: boolean
  matchId: string
  playerSlug: string
  /** aria-label + tooltip, fourni par l'i18n de la feature appelante (FR/EN). */
  label: string
}

export function MatchReplayLink({ available, matchId, playerSlug, label }: MatchReplayLinkProps) {
  const titleSlug = useTitleSlug()
  // Thème LOCAL déjà tranché par le store (`dark` | `light`) : l'icône est un raster à
  // deux variantes, elle ne peut pas se teinter en `currentColor`.
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  if (!available) return null
  return (
    <Link
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay"
      params={{ titleSlug, playerSlug, matchId }}
      onClick={(e) => e.stopPropagation()}
      aria-label={label}
      title={label}
      // Boîte de taille FIXE, et non un flex qui négocie sa taille avec la cellule :
      // une image dans un flex au sein d'une cellule à largeur automatique est
      // dimensionnée différemment selon le moteur (invisible sous Firefox, correcte
      // sous Chrome — constaté 2026-07-26 sur la colonne Waypoint voisine).
      className="group inline-flex h-4 w-5 shrink-0 items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
    >
      <img
        src={themedIconSrc('replay', theme)}
        alt=""
        aria-hidden
        // width/height HTML : taille intrinsèque pour le moteur ; object-contain :
        // l'icône n'est pas carrée, le `fill` par défaut la déformerait.
        width={20}
        height={16}
        className="h-full w-full shrink-0 object-contain opacity-60 group-hover:opacity-100 transition-opacity"
      />
    </Link>
  )
}
