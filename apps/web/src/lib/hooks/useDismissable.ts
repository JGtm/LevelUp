import { useEffect, useRef, type RefObject } from 'react'

/** Ce qui a fermé la couche : la touche Échap, ou un geste hors de la couche. */
export type DismissReason = 'escape' | 'outside'

export interface DismissableOptions<T extends HTMLElement> {
  /** La zone intérieure de la couche. Absente : le hook en tient une et la rend. */
  ref?: RefObject<T | null>
  /**
   * Le geste qui compte comme « dehors » : `mousedown` (défaut) ou `pointerdown` (la couche part
   * au geste, pas au relâché) ; `null` : aucun clic dehors, seul Échap ferme.
   */
  outside?: 'mousedown' | 'pointerdown' | null
  /**
   * `false` : la couche ne prend pas Échap (sa place dans la pile est laissée aux autres) ; seul le clic
   * dehors ferme. Pour une couche dont Échap est lu par un champ de saisie qu'elle contient.
   */
  escape?: boolean
  /** Un second élément dont le clic ne ferme pas la couche (son déclencheur). */
  ignore?: RefObject<HTMLElement | null>
  /**
   * Sélecteur des couches d'Échap de la page. Tant qu'une couche AUTRE que `ref` est ouverte,
   * Échap ne ferme pas celle-ci : la couche intérieure se ferme la première, par son propre écouteur.
   */
  innerLayerSelector?: string
}

/** La touche qui ferme une couche : celle qu'un menu laisse passer jusqu'au hook. */
export function isEscapeKey(event: Pick<KeyboardEvent, 'key'>): boolean {
  return event.key === 'Escape'
}

/** Une autre couche que `self` correspond-elle à `selector` ? */
function otherLayerOpen(selector: string, self: Element | null): boolean {
  return Array.from(document.querySelectorAll(selector)).some((layer) => layer !== self)
}

/** Ce que le registre d'Échap lit d'une couche ouverte, à chaque frappe. */
interface EscapeLayer {
  ref: RefObject<HTMLElement | null>
  current: () => { onClose: (reason: DismissReason) => void; innerLayerSelector?: string }
}

/**
 * Les couches ouvertes, de la plus ancienne à la plus récente. UN SEUL écouteur d'Échap sert
 * toutes les couches : à une frappe, la couche la plus récente se ferme SEULE, et la frappe est
 * consommée. Un écouteur par couche fermerait aussi celles du dessous, ou, selon l'ordre
 * d'abonnement, la mauvaise d'abord (React vide ses mises à jour entre deux écouteurs).
 */
const escapeLayers: EscapeLayer[] = []

function onEscapeKeyDown(event: KeyboardEvent) {
  if (!isEscapeKey(event)) return
  for (let at = escapeLayers.length - 1; at >= 0; at--) {
    const layer = escapeLayers[at]
    const { onClose, innerLayerSelector } = layer.current()
    // Une couche qui s'efface devant une couche intérieure hors registre laisse celle du dessous
    // se fermer : la frappe ne se perd que si aucune couche ne la prend.
    if (innerLayerSelector && otherLayerOpen(innerLayerSelector, layer.ref.current)) continue
    onClose('escape')
    return
  }
}

function registerEscapeLayer(layer: EscapeLayer): () => void {
  if (escapeLayers.length === 0) window.addEventListener('keydown', onEscapeKeyDown)
  escapeLayers.push(layer)
  return () => {
    const at = escapeLayers.indexOf(layer)
    if (at >= 0) escapeLayers.splice(at, 1)
    if (escapeLayers.length === 0) window.removeEventListener('keydown', onEscapeKeyDown)
  }
}

/**
 * useDismissable — la fermeture d'une couche (popover, menu, panneau, média agrandi) : Échap et,
 * selon `options.outside`, un clic hors de sa zone. Les écouteurs ne vivent que couche ouverte.
 *
 * ÉCHAP NE FERME QUE LA COUCHE DU DESSUS : la plus récemment ouverte parmi celles qui utilisent ce
 * hook (hors celles que `innerLayerSelector` retient), et la frappe s'arrête là. Une frappe
 * suivante ferme la couche d'en dessous.
 *
 * `onClose` reçoit la raison de la fermeture (rendre le focus au déclencheur sur Échap seulement,
 * par exemple). Il est lu à chaque événement : un `onClose` neuf ne rebranche pas les écouteurs,
 * donc l'ordre des couches reste celui de leur ouverture.
 */
export function useDismissable<T extends HTMLElement = HTMLDivElement>(
  open: boolean,
  onClose: (reason: DismissReason) => void,
  options: DismissableOptions<T> = {},
): RefObject<T | null> {
  const ownRef = useRef<T>(null)
  const ref = options.ref ?? ownRef
  const outsideEvent = options.outside === undefined ? 'mousedown' : options.outside
  const takesEscape = options.escape !== false
  const latest = useRef({ onClose, options })
  useEffect(() => {
    latest.current = { onClose, options }
  })

  useEffect(() => {
    if (!open || !takesEscape) return
    return registerEscapeLayer({
      ref,
      current: () => ({
        onClose: latest.current.onClose,
        innerLayerSelector: latest.current.options.innerLayerSelector,
      }),
    })
  }, [open, ref, takesEscape])

  useEffect(() => {
    if (!open || !outsideEvent) return
    function onOutside(event: Event) {
      const target = event.target as Node | null
      if (!target || !ref.current) return
      if (ref.current.contains(target)) return
      if (latest.current.options.ignore?.current?.contains(target)) return
      latest.current.onClose('outside')
    }
    document.addEventListener(outsideEvent, onOutside)
    return () => document.removeEventListener(outsideEvent, onOutside)
  }, [open, ref, outsideEvent])

  return ref
}
