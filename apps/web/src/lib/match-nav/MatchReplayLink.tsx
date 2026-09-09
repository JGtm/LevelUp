/**
 * MatchReplayLink — l'icône « Rejeu » d'une ligne de tableau de matchs, et le lien
 * interne vers la page de rejeu 2D du match.
 *
 * UN SEUL composant pour les deux tableaux (Explorer et Synergies escouade) : deux
 * copies divergeraient sur la route ou sur la règle d'affichage, et une ligne qui
 * pointe vers un rejeu inexistant se lit comme une panne.
 *
 * DEUX PORTES (règle du 2026-09-05, registre L5) :
 *   1. LE TITRE — capability `replay` : un titre sans décodeur de film n'a pas de page de
 *      rejeu (ses routes /replay* rendent 503). Rien n'est rendu, nulle part.
 *   2. LE MATCH — `available` (`has_replay`) : l'artefact existe-t-il pour CE match ? La
 *      route répond 404 sans lui — c'est le sens du champ servi par l'API, résolu en un
 *      seul listing de dossier par requête (jamais un accès disque par ligne).
 *
 * La porte 1 est ici parce que ce composant est LE point unique de l'icône (Explorer,
 * Synergies escouade, carte de match) : les tableaux masquent EN PLUS leur colonne, pour
 * ne pas laisser un en-tête sans contenu possible.
 *
 * Variante « bouton » de la page match : features/match-view/MatchHeader.replayLink.tsx
 * (même route, présentation différente : bouton avec libellé, pas une cellule).
 */
import { Link } from '@tanstack/react-router'

import { useCapability } from '@/lib/capabilities/capabilities'
import { themedIconSrc } from '@/lib/themedIcon'
import { useTitleSlug } from '@/lib/title-routing'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

/**
 * Présentation du lien. Les DEUX portes, la route et l'icône sont communes : seule
 * change l'enveloppe. Un composant séparé par présentation ferait diverger la règle
 * d'affichage (CLAUDE.md n°6).
 *   - `icon`   : cellule de tableau, boîte de 20x16 px sans cadre (Explorer, Synergies).
 *   - `button` : bouton cadré de 28 px, pour une tuile de match où l'icône nue passait
 *                inaperçue (retour utilisateur 2026-09-09).
 */
type ReplayLinkVariant = 'icon' | 'button'

/** Enveloppe du lien et taille de l'icône, par présentation. */
const VARIANT_CLASS: Record<ReplayLinkVariant, { link: string; img: string }> = {
  icon: {
    link: 'group inline-flex h-4 w-5 shrink-0 items-center justify-center text-muted-foreground hover:text-foreground transition-colors',
    img: 'h-full w-full shrink-0 object-contain opacity-60 group-hover:opacity-100 transition-opacity',
  },
  button: {
    link: 'group inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border bg-transparent text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
    img: 'h-3.5 w-[18px] shrink-0 object-contain opacity-80 group-hover:opacity-100 transition-opacity',
  },
}

interface MatchReplayLinkProps {
  /** `has_replay` de la ligne : un artefact de rejeu existe pour ce match. */
  available: boolean
  matchId: string
  playerSlug: string
  /** aria-label + tooltip, fourni par l'i18n de la feature appelante (FR/EN). */
  label: string
  /** Présentation (défaut `icon` : la cellule de tableau, cas d'origine). */
  variant?: ReplayLinkVariant
}

export function MatchReplayLink({
  available,
  matchId,
  playerSlug,
  label,
  variant = 'icon',
}: MatchReplayLinkProps) {
  const titleSlug = useTitleSlug()
  const titreARejeu = useCapability('replay')
  // Thème LOCAL déjà tranché par le store (`dark` | `light`) : l'icône est un raster à
  // deux variantes, elle ne peut pas se teinter en `currentColor`.
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  if (!titreARejeu || !available) return null
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
      className={VARIANT_CLASS[variant].link}
    >
      <img
        src={themedIconSrc('replay', theme)}
        alt=""
        aria-hidden
        // width/height HTML : taille intrinsèque pour le moteur ; object-contain :
        // l'icône n'est pas carrée, le `fill` par défaut la déformerait.
        width={20}
        height={16}
        className={VARIANT_CLASS[variant].img}
      />
    </Link>
  )
}
