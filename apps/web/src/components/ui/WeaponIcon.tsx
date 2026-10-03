/**
 * WeaponIcon — rendu title-agnostic d'une icône d'arme.
 *
 * Deux modes, sélectionnés UNIQUEMENT par le champ `tinted` que le back publie
 * (`image_tinted`) — jamais par le titre, jamais par la forme de l'URL :
 *
 *  - MASQUE (`tinted`) : l'icône est un dessin porté par l'alpha, sans couleur
 *    propre (les icônes extraites d'Halo Infinite sont blanches à 100 %). On la
 *    rend comme un `mask-image` sur un aplat de couleur : elle prend alors la
 *    couleur qu'on lui donne, donc reste lisible en thème clair comme en sombre,
 *    et peut porter une couleur d'équipe sans produire un asset par couleur.
 *  - IMAGE (défaut) : visuel fini en couleur (icônes officielles Halo 5, les deux
 *    PNG dessinés à la main d'Infinite) → `<img>` classique. Le teindre
 *    l'aplatirait en silhouette.
 *
 * La teinte est soit la couleur de texte héritée (`currentColor` — le thème décide),
 * soit un TOKEN sémantique via `tokenCssVar`. Jamais un hex, jamais une classe
 * Tailwind de couleur. Comme `--ac-team-ally` / `--ac-team-enemy` sont écrasés par
 * la couleur d'équipe choisie en jeu, une icône teintée avec ces tokens suit la
 * VRAIE couleur du joueur, sans produire un asset par couleur.
 */
import type { CSSProperties } from 'react'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { tokenCssVar } from '@/lib/accessibility'

interface WeaponIconProps {
  /** URL servie par le back. Vide/absente → le composant ne rend rien. */
  imageUrl?: string | null
  /** L'icône est un masque à teindre (champ `image_tinted` du back). */
  tinted?: boolean
  /** Texte alternatif (nom de l'arme). */
  label: string
  /**
   * Token de teinte, ignoré hors mode masque. Absent → `currentColor` : l'icône
   * prend la couleur de texte héritée, donc suit le thème sans code. Passer un
   * token (`team-ally`/`team-enemy`…) quand la teinte porte un SENS.
   */
  token?: SemanticToken
  /** Largeur du slot : nombre = px, chaîne = valeur CSS telle quelle ('100%'…). */
  width?: number | string
  /** Hauteur du slot : même convention. */
  height?: number | string
  className?: string
  style?: CSSProperties
  /** Appelé si l'image ne charge pas (mode image seul — un masque n'a pas d'onError). */
  onError?: () => void
}

export function WeaponIcon({
  imageUrl,
  tinted = false,
  label,
  token,
  width = 56,
  height = 24,
  className,
  style,
  onError,
}: WeaponIconProps) {
  if (!imageUrl) return null

  const cssSize = (v: number | string) => (typeof v === 'number' ? `${v}px` : v)
  const box: CSSProperties = { width: cssSize(width), height: cssSize(height), ...style }

  if (tinted) {
    const mask = `url(${imageUrl})`
    return (
      <span
        role="img"
        aria-label={label}
        className={className}
        style={{
          ...box,
          display: 'inline-block',
          backgroundColor: token ? tokenCssVar(token) : 'currentColor',
          maskImage: mask,
          WebkitMaskImage: mask,
          maskSize: 'contain',
          WebkitMaskSize: 'contain',
          maskRepeat: 'no-repeat',
          WebkitMaskRepeat: 'no-repeat',
          maskPosition: 'center',
          WebkitMaskPosition: 'center',
        }}
      />
    )
  }

  return (
    <img
      src={imageUrl}
      alt={label}
      className={className}
      style={{ ...box, objectFit: 'contain' }}
      loading="lazy"
      decoding="async"
      onError={onError}
    />
  )
}
