/**
 * MedalIcon — rendu title-agnostic d'une icône de médaille.
 *
 * Deux modes, sélectionnés UNIQUEMENT par la présence des champs sprite :
 *  - SPRITE (ex. Halo 5) : la médaille est une découpe d'une feuille de sprites.
 *    On rend un `<div role="img">` avec background-image + background-position
 *    (même technique que asset-drawer/AssetCard.tsx). Le sprite a une taille
 *    native (`spriteWidth`/`spriteHeight`, ~74px) ; on le met à l'échelle du
 *    slot d'affichage via `transform: scale(size / nativeWidth)` pour qu'il
 *    remplisse `size` px sans déformation.
 *  - IMG (ex. Halo Infinite) : pas de sprite → `<img src={imageUrl}>` classique,
 *    avec masquage onError pour préserver le comportement HINF actuel (les 404
 *    de PNG manquants disparaissent au lieu d'afficher une icône cassée).
 *
 * AUCUNE logique par titre : on regarde `spriteSheet` et c'est tout.
 */
import type { CSSProperties } from 'react'

interface MedalIconProps {
  imageUrl?: string | null
  spriteSheet?: string
  spriteLeft?: number
  spriteTop?: number
  spriteWidth?: number
  spriteHeight?: number
  /** Texte alternatif (nom de la médaille). */
  label: string
  /** Taille d'affichage cible en px (côté du carré). Défaut 40. */
  size?: number
  /** Classes appliquées au conteneur (taille/glow gérés par le caller). */
  className?: string
  /** Style inline additionnel (ex. drop-shadow par rareté). */
  style?: CSSProperties
}

/** Taille native par défaut d'une cellule de sprite (sheet Halo 5). */
const NATIVE_SPRITE_PX = 74

export function MedalIcon({
  imageUrl,
  spriteSheet,
  spriteLeft = 0,
  spriteTop = 0,
  spriteWidth,
  spriteHeight,
  label,
  size = 40,
  className,
  style,
}: MedalIconProps) {
  if (spriteSheet) {
    const nativeW = spriteWidth || NATIVE_SPRITE_PX
    const nativeH = spriteHeight || NATIVE_SPRITE_PX
    // On rend la cellule à sa taille native, puis on la réduit pour qu'elle
    // remplisse `size` px. Échelle calée sur la largeur ; transform-origin
    // top-left pour que la cellule reste alignée dans la boîte size×size.
    const scale = size / nativeW
    return (
      <div
        role="img"
        aria-label={label}
        className={className}
        style={{
          width: `${size}px`,
          height: `${nativeH * scale}px`,
          overflow: 'hidden',
          ...style,
        }}
      >
        <div
          className="shrink-0"
          style={{
            width: `${nativeW}px`,
            height: `${nativeH}px`,
            backgroundImage: `url(${spriteSheet})`,
            backgroundPosition: `-${spriteLeft}px -${spriteTop}px`,
            backgroundRepeat: 'no-repeat',
            transform: `scale(${scale})`,
            transformOrigin: 'top left',
          }}
        />
      </div>
    )
  }

  if (!imageUrl) return null

  return (
    <img
      src={imageUrl}
      alt={label}
      className={className}
      style={{ width: `${size}px`, height: `${size}px`, ...style }}
      loading="lazy"
      decoding="async"
      onError={(e) => {
        ;(e.currentTarget as HTMLImageElement).style.display = 'none'
      }}
    />
  )
}
