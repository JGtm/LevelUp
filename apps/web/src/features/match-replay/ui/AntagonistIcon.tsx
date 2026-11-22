/**
 * AntagonistIcon — L'ICÔNE D'UN TITRE d'antagoniste (némésis, souffre-douleur), UNE pour toutes les
 * surfaces du rejeu qui la montrent (bande de la frise, section de la fiche).
 *
 * UN MASQUE SUR UN APLAT `currentColor`. Les deux images du dépôt sont noires sur fond transparent, à
 * alpha d'un bit : seule leur silhouette compte. On la prend comme masque d'un aplat qui suit la
 * couleur du texte, donc celle que la surface pose (`ink`, un jeton de camp) ou celle qui l'entoure,
 * sans aucune couleur écrite ici et sans variante par thème. Le chemin passe par `themedIconSrc`
 * (garde-rail `themedIcon.guard.test.ts`) : le thème « clair » désigne le fichier noir, celui dont
 * on prend l'alpha.
 *
 * DÉCORATIVE : `aria-hidden`. Le nom accessible est celui du bouton ou du libellé qui la porte.
 * Les images n'ont pas le même rapport (714 x 670 et 1099 x 1369) : la boîte est carrée et
 * l'image s'y loge entière (`contain`), centrée.
 */
import type { CSSProperties } from 'react'

import type { AntagonistRole } from '@/lib/antagonists/antagonists'
import { themedIconSrc } from '@/lib/themedIcon'

/** Le nom de l'image de chaque rôle, dans `public/icons/`. */
const ICON_NAME: Record<AntagonistRole, string> = { nemesis: 'nemesis', victim: 'victim' }

export interface AntagonistIconProps {
  role: AntagonistRole
  /** Côté de la boîte carrée, en pixels. */
  size: number
  /** L'encre de l'icône (un jeton, `var(--ac-…)`) ; absente = la couleur du texte qui l'entoure. */
  ink?: string
  className?: string
}

export function AntagonistIcon({ role, size, ink, className }: AntagonistIconProps) {
  const mask = `url(${themedIconSrc(ICON_NAME[role], 'light')})`
  const style: CSSProperties = {
    display: 'inline-block',
    flexShrink: 0,
    width: `${size}px`,
    height: `${size}px`,
    backgroundColor: 'currentColor',
    color: ink,
    maskImage: mask,
    WebkitMaskImage: mask,
    maskSize: 'contain',
    WebkitMaskSize: 'contain',
    maskRepeat: 'no-repeat',
    WebkitMaskRepeat: 'no-repeat',
    maskPosition: 'center',
    WebkitMaskPosition: 'center',
  }
  return <span aria-hidden="true" data-antagonist-icon={role} className={className} style={style} />
}
