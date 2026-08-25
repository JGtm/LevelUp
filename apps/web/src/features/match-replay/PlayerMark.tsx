/**
 * PlayerMark — le GLYPHE d'identité devant un nom : « ami », deux silhouettes.
 *
 * La même marque dans les fiches et dans le fil (décision D5, plan d'habillage 2026-08-16) :
 * on AJOUTE un signe, on ne change pas la couleur du nom, qui dit déjà l'équipe ou la mort.
 * `currentColor` : le glyphe prend l'encre du texte qu'il précède.
 *
 * LE GLYPHE « MOI » A ÉTÉ SUPPRIMÉ le 2026-08-24 (demande utilisateur : « supprimer le point
 * qui indique qui est le joueur actif de partout ») : la marque `me` ne dessine plus RIEN
 * dans les fiches ni dans le fil. Sur la carte, le marqueur du joueur de la page garde sa
 * FORME distinctive (double anneau, replayMarkers.ts) — c'est une demande distincte du
 * 2026-08-18, qui n'est pas ce « point ».
 */
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { PlayerMarkKind } from './playerMarks'

interface Props {
  kind: PlayerMarkKind | undefined
  locale: ReplayLocale
  /** Côté du glyphe en px (défaut 10 : la hauteur d'une lettre de fiche). */
  size?: number
}

export function PlayerMark({ kind, locale, size = 10 }: Props) {
  if (kind !== 'friend') return null
  const t = REPLAY_TEXT[locale]
  const label = t.markFriend
  return (
    <svg
      role="img"
      aria-label={label}
      width={size}
      height={size}
      viewBox="0 0 10 10"
      className="inline-block shrink-0 align-[-1px]"
      fill="currentColor"
    >
      <title>{label}</title>
      <circle cx="3.4" cy="3" r="1.7" />
      <path d="M0.4 8.6c0-1.9 1.3-3.1 3-3.1s3 1.2 3 3.1z" />
      <circle cx="7.2" cy="3.2" r="1.4" opacity="0.75" />
      <path d="M6.2 8.6c0-1.6 1-2.6 2.2-2.6 0.6 0 1.1 0.3 1.4 0.7v1.9z" opacity="0.75" />
    </svg>
  )
}
