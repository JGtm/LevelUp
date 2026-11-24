/**
 * useObservedWidth — LA LARGEUR D'UNE ZONE, suivie à chaque changement de taille, en pixels entiers ; 0 tant que la
 * zone n'est pas mesurée.
 *
 * Le hook rend un `ref` de rappel (état d'élément) : il suit une zone qui n'est montée qu'après coup (une vue qui rend
 * `null` tant que ses données manquent) sans que la mesure reste à zéro. La mesure est lue avant la première peinture,
 * puis suivie par `observeSize`. CE QUI SE LIT dans la zone est `measure` : la largeur de la boîte par défaut
 * (`boxWidthOf`), ou celle de son contenu (`useContentWidth`). `measure` est une fonction de module, stable d'un rendu
 * à l'autre.
 */
import { useLayoutEffect, useState } from 'react'

import { observeSize } from './observeSize'

/** La largeur de la boîte de `el`, marges intérieures et bordures comprises, en pixels entiers. */
export function boxWidthOf(el: HTMLElement): number {
  return Math.floor(el.getBoundingClientRect().width)
}

export function useObservedWidth(measure: (el: HTMLElement) => number = boxWidthOf): {
  width: number
  ref: (el: HTMLElement | null) => void
} {
  const [el, setEl] = useState<HTMLElement | null>(null)
  const [width, setWidth] = useState(0)
  useLayoutEffect(() => {
    if (!el) return
    const read = () => setWidth(measure(el))
    read()
    return observeSize(el, read)
  }, [el, measure])
  return { width, ref: setEl }
}
