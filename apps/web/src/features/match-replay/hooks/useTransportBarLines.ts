/**
 * useTransportBarLines — SUR COMBIEN DE LIGNES LA BARRE DE LECTURE SE RANGE. Le hook MESURE (la largeur de la
 * barre, la largeur de chaque groupe de commandes) et laisse la règle pure (`barLines`) décider ; il se souvient de
 * sa dernière décision pour la marge de bascule.
 *
 * LES MESURES SONT DES DIMENSIONS DE MISE EN PAGE (`offsetWidth`, `clientWidth`, marges calculées) : une
 * transformation CSS de la barre (sa forme fine) n'y change rien, la décision est donc la même pour les deux formes.
 * La frise (`data-replay-bar-timeline`) est le seul élément qui ne compte pas dans les groupes. La mesure est lue
 * avant la peinture, à chaque rendu (une commande qui apparaît ou change de libellé change la largeur requise) et
 * à chaque changement de taille de la barre (`ResizeObserver`, optionnel : jsdom ne le fournit pas).
 */
import { useLayoutEffect, useRef, useState } from 'react'

import { barLines, type BarLines } from '../model/transportBarLogic'

/** Le marqueur de la frise dans la barre : elle prend la largeur restante, elle n'est pas un groupe. */
export const TIMELINE_SLOT_ATTR = 'data-replay-bar-timeline'

/** La largeur d'un groupe : sa boîte et ses marges, en pixels de mise en page ; 0 s'il n'est pas affiché. */
function groupWidthOf(el: HTMLElement): number {
  if (el.offsetWidth === 0) return 0
  const style = getComputedStyle(el)
  return el.offsetWidth + (parseFloat(style.marginLeft) || 0) + (parseFloat(style.marginRight) || 0)
}

/** Les largeurs des groupes visibles de la rangée, hors frise. */
function groupWidthsOf(row: HTMLElement): number[] {
  return Array.from(row.children)
    .filter((el): el is HTMLElement => el instanceof HTMLElement && !el.hasAttribute(TIMELINE_SLOT_ATTR))
    .map(groupWidthOf)
    .filter((w) => w > 0)
}

export function useTransportBarLines(): {
  outerRef: React.RefObject<HTMLDivElement | null>
  rowRef: React.RefObject<HTMLDivElement | null>
  lines: BarLines
  /** La hauteur de la rangée en pixels de mise en page (non transformée) ; 0 tant qu'elle n'est pas mesurée. */
  rowHeight: number
} {
  const outerRef = useRef<HTMLDivElement>(null)
  const rowRef = useRef<HTMLDivElement>(null)
  const [lines, setLines] = useState<BarLines>(1)
  const [rowHeight, setRowHeight] = useState(0)

  // La mesure lit l'état du DOM courant : la ref porte toujours la version du dernier rendu.
  const measureRef = useRef<() => void>(() => {})
  useLayoutEffect(() => {
    measureRef.current = () => {
      const outer = outerRef.current
      const row = rowRef.current
      if (!outer || !row) return
      const input = { availableWidth: outer.clientWidth, groupWidths: groupWidthsOf(row) }
      setLines((previous) => barLines(input, previous))
      setRowHeight(row.offsetHeight)
    }
    measureRef.current()
  })
  useLayoutEffect(() => {
    const outer = outerRef.current
    if (!outer || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(() => measureRef.current())
    observer.observe(outer)
    return () => observer.disconnect()
  }, [])

  return { outerRef, rowRef, lines, rowHeight }
}
