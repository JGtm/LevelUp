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

/**
 * useDismissable — la fermeture d'une couche (popover, menu, panneau, média agrandi) : Échap et,
 * selon `options.outside`, un clic hors de sa zone. Les écouteurs ne vivent que couche ouverte.
 *
 * `onClose` reçoit la raison de la fermeture (rendre le focus au déclencheur sur Échap seulement,
 * par exemple). Il est lu à chaque événement : un `onClose` neuf ne rebranche pas les écouteurs,
 * donc l'ordre d'écoute entre deux couches reste celui de leur ouverture.
 */
export function useDismissable<T extends HTMLElement = HTMLDivElement>(
  open: boolean,
  onClose: (reason: DismissReason) => void,
  options: DismissableOptions<T> = {},
): RefObject<T | null> {
  const ownRef = useRef<T>(null)
  const ref = options.ref ?? ownRef
  const outsideEvent = options.outside === undefined ? 'mousedown' : options.outside
  const latest = useRef({ onClose, options })
  useEffect(() => {
    latest.current = { onClose, options }
  })

  useEffect(() => {
    if (!open) return

    function onKeyDown(event: KeyboardEvent) {
      if (!isEscapeKey(event)) return
      const { innerLayerSelector } = latest.current.options
      if (innerLayerSelector && otherLayerOpen(innerLayerSelector, ref.current)) return
      latest.current.onClose('escape')
    }
    function onOutside(event: Event) {
      const target = event.target as Node | null
      if (!target || !ref.current) return
      if (ref.current.contains(target)) return
      if (latest.current.options.ignore?.current?.contains(target)) return
      latest.current.onClose('outside')
    }

    window.addEventListener('keydown', onKeyDown)
    if (outsideEvent) document.addEventListener(outsideEvent, onOutside)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      if (outsideEvent) document.removeEventListener(outsideEvent, onOutside)
    }
  }, [open, ref, outsideEvent])

  return ref
}
