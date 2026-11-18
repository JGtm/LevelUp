/**
 * useContentWidth — LA LARGEUR UTILE d'une zone d'affichage : celle de son contenu, sans les marges
 * intérieures (`padding`) ni les bordures, en pixels entiers ; 0 tant que la zone n'est pas mesurée.
 *
 * Une vue de la carte qui laisse l'Auto choisir son orientation (`useAutoTurned`) a besoin de la place
 * que SA zone offre au plan : c'est cette largeur, et non celle de la fenêtre. Le hook rend un `ref` de
 * rappel (état d'élément) : il suit une zone qui n'est montée qu'après coup (une vue qui rend `null` tant
 * que ses données manquent) sans que la mesure reste à zéro. La mesure est lue avant la première peinture,
 * puis suivie par un `ResizeObserver` (optionnel : jsdom ne le fournit pas, la mesure initiale reste).
 */
import { useLayoutEffect, useState } from 'react'

/** La largeur du contenu de `el` : `clientWidth` (bordures exclues) moins les marges intérieures gauche et droite. */
function contentWidthOf(el: HTMLElement): number {
  const style = getComputedStyle(el)
  const inner = el.clientWidth - (parseFloat(style.paddingLeft) || 0) - (parseFloat(style.paddingRight) || 0)
  return Math.max(Math.floor(inner), 0)
}

export function useContentWidth(): { width: number; ref: (el: HTMLElement | null) => void } {
  const [el, setEl] = useState<HTMLElement | null>(null)
  const [width, setWidth] = useState(0)
  useLayoutEffect(() => {
    if (!el) return
    const read = () => setWidth(contentWidthOf(el))
    read()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(read)
    observer.observe(el)
    return () => observer.disconnect()
  }, [el])
  return { width, ref: setEl }
}
