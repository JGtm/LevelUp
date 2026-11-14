/**
 * useElementWidth — la largeur (px) d'un élément, suivie par ResizeObserver ; `fallback` avant la
 * première mesure et sans observateur (tests, rendu serveur). Sert le diagramme de flux, dont les
 * marges de libellés dépendent de la largeur réelle.
 */
import { useEffect, useState, type RefObject } from 'react'

export function useElementWidth(ref: RefObject<HTMLElement | null>, fallback: number): number {
  const [width, setWidth] = useState(fallback)
  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    setWidth(el.getBoundingClientRect().width || fallback)
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.contentRect.width > 0) setWidth(entry.contentRect.width)
      }
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref, fallback])
  return width
}
