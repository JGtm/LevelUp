/**
 * SegmentLabel — L'ÉCRITURE D'UN SEGMENT DE PISTE HTML, centrée sur la hauteur, à l'encre blanche
 * cernée d'un contour sombre (classe `segment-ink`, jetons de `segmentInk.ts`). Seul endroit où une piste pose du texte sur un aplat.
 *
 * Posé en `absolute inset-0` dans un segment positionné. Le conteneur est `flex` : l'ancre de
 * l'infobulle (`inline-flex`) en devient un enfant de flex. Dans un bloc ordinaire, elle se
 * calait sur la ligne de base et descendait de 2 à 3 px sous le centre du segment (constat du
 * 2026-10-15 : texte à +3,5 px du centre d'une barre de 22 px).
 *
 * L'étiquette porte `data-fit-label` (contrat de `useSegmentLabelFit`) ; `hidden` la rend
 * invisible sans la retirer du DOM, pour pouvoir la remesurer.
 */
import type { ReactNode } from 'react'

import { Tooltip } from '@/components/ui/tooltip'

const JUSTIFY = { start: 'justify-start pl-[7px]', center: 'justify-center', end: 'justify-end pr-[7px]' } as const

interface SegmentLabelProps {
  /** Infobulle de tout le segment. */
  tip: ReactNode
  align?: keyof typeof JUSTIFY
  hidden?: boolean
  /** Corps de texte (taille, graisse) ; l'encre et le centrage ne se règlent pas ici. */
  textClassName?: string
  children: ReactNode
}

export function SegmentLabel({ tip, align = 'center', hidden = false, textClassName = 'text-xs font-medium', children }: SegmentLabelProps) {
  return (
    <div className="absolute inset-0 flex">
      <Tooltip content={tip} className="h-full w-full">
        <div className={`flex h-full w-full cursor-help items-center overflow-hidden ${JUSTIFY[align]}`}>
          <span
            data-fit-label
            className={`segment-ink whitespace-nowrap tabular-nums leading-none ${textClassName}`}
            style={{ visibility: hidden ? 'hidden' : 'visible' }}
          >
            {children}
          </span>
        </div>
      </Tooltip>
    </div>
  )
}
