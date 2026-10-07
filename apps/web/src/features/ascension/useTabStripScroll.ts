/**
 * useTabStripScroll — la barre d'onglets d'Ascension défile à l'horizontale quand la fenêtre est
 * plus étroite que ses onglets (sous ~620 px, « Tactique » sortait de l'écran sans moyen de
 * l'atteindre).
 *
 *   - l'onglet ACTIF est ramené dans la partie visible à chaque changement d'onglet et de
 *     largeur (défilement de la seule barre, jamais de la page) ;
 *   - un fondu au bord qui cache encore des onglets dit qu'il y a de la suite (la barre de
 *     défilement est masquée, motif du carrousel).
 */
import { useCallback, useEffect, useRef, useState, type CSSProperties, type RefObject } from 'react'

/** Largeur du fondu de bord (px). */
const FADE_PX = 24

/** Le défilement qui rend visible l'étendue [left, right] dans une fenêtre [scrollLeft, +width]. */
export function scrollLeftToReveal(left: number, right: number, scrollLeft: number, width: number): number {
  if (left < scrollLeft) return left
  if (right > scrollLeft + width) return right - width
  return scrollLeft
}

/** Les bords qui cachent encore des onglets. */
export interface HiddenEdges {
  before: boolean
  after: boolean
}

export function hiddenEdges(scrollLeft: number, width: number, scrollWidth: number): HiddenEdges {
  return { before: scrollLeft > 1, after: scrollLeft + width < scrollWidth - 1 }
}

/** Le masque de fondu des bords cachés (alpha seulement : la couleur n'y joue pas). */
export function edgeFadeMask({ before, after }: HiddenEdges): CSSProperties | undefined {
  if (!before && !after) return undefined
  const opaque = 'var(--foreground)'
  const start = before ? `transparent, ${opaque} ${FADE_PX}px` : `${opaque}, ${opaque}`
  const end = after ? `${opaque} calc(100% - ${FADE_PX}px), transparent` : `${opaque}`
  const mask = `linear-gradient(to right, ${start}, ${end})`
  return { maskImage: mask, WebkitMaskImage: mask }
}

export function useTabStripScroll(activeKey: string): {
  ref: RefObject<HTMLElement | null>
  edges: HiddenEdges
  onScroll: () => void
} {
  const ref = useRef<HTMLElement | null>(null)
  const [edges, setEdges] = useState<HiddenEdges>({ before: false, after: false })

  const measure = useCallback(() => {
    const strip = ref.current
    if (!strip) return
    const next = hiddenEdges(strip.scrollLeft, strip.clientWidth, strip.scrollWidth)
    setEdges((prev) => (prev.before === next.before && prev.after === next.after ? prev : next))
  }, [])

  const revealActive = useCallback(() => {
    const strip = ref.current
    const active = strip?.querySelector<HTMLElement>('[aria-selected="true"]')
    if (!strip || !active) return
    // La barre est positionnée (`relative`) : `offsetLeft` d'un onglet se lit depuis elle.
    const left = active.offsetLeft
    strip.scrollLeft = scrollLeftToReveal(left, left + active.offsetWidth, strip.scrollLeft, strip.clientWidth)
    measure()
  }, [measure])

  useEffect(() => {
    revealActive()
  }, [activeKey, revealActive])

  useEffect(() => {
    const strip = ref.current
    if (!strip || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(() => revealActive())
    observer.observe(strip)
    return () => observer.disconnect()
  }, [revealActive])

  return { ref, edges, onScroll: measure }
}
