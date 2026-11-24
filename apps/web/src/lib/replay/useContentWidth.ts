/**
 * useContentWidth — LA LARGEUR UTILE d'une zone d'affichage : celle de son contenu, sans les marges
 * intérieures (`padding`) ni les bordures, en pixels entiers ; 0 tant que la zone n'est pas mesurée.
 *
 * Une vue de la carte qui laisse l'Auto choisir son orientation (`useAutoTurned`) a besoin de la place
 * que SA zone offre au plan : c'est cette largeur, et non celle de la fenêtre. Le hook rend un `ref` de
 * rappel (état d'élément) : il suit une zone qui n'est montée qu'après coup (une vue qui rend `null` tant
 * que ses données manquent) sans que la mesure reste à zéro. La mesure est lue avant la première peinture,
 * puis suivie par `useObservedWidth` (`lib/layout`), qui ne diffère de la mesure du corps du cadre que par ce qu'il lit.
 */
import { useObservedWidth } from '@/lib/layout/useObservedWidth'

/** La largeur du contenu de `el` : `clientWidth` (bordures exclues) moins les marges intérieures gauche et droite. */
function contentWidthOf(el: HTMLElement): number {
  const style = getComputedStyle(el)
  const inner = el.clientWidth - (parseFloat(style.paddingLeft) || 0) - (parseFloat(style.paddingRight) || 0)
  return Math.max(Math.floor(inner), 0)
}

export function useContentWidth(): { width: number; ref: (el: HTMLElement | null) => void } {
  return useObservedWidth(contentWidthOf)
}
