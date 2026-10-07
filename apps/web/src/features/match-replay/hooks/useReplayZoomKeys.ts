/**
 * useReplayZoomKeys — LE ZOOM AU CLAVIER sur un plan qui reprend le zoom du rejeu sans en être un
 * (Vue match « Occupation du terrain », Tactique). Même table de touches que le lecteur
 * (`zoomKeyCommand` : + / = grossir, − / _ réduire, 0 toute la carte), mêmes actions du cadrage.
 *
 * # ACTIF SEULEMENT SUR LE PLAN : SURVOLÉ, OU LE FOCUS DEDANS
 *
 * Ces plans vivent dans une page qui défile, avec ses champs et ses autres raccourcis (la Vue
 * match change de match aux flèches) : une frappe ne zoome que si le pointeur est sur le cadre du
 * plan, ou si le focus y est (la commande ⌂ / + / − qu'on vient de cliquer, par exemple). Jamais
 * depuis un champ de saisie, jamais avec Ctrl / Cmd / Alt (Ctrl + « + » reste le zoom du
 * navigateur). Le cadre est le PARENT de la toile : il porte aussi la commande de cadrage.
 *
 * LE REJEU N'EN A PAS BESOIN, et ne doit pas l'appeler : ses raccourcis (`useReplayShortcuts`)
 * écoutent déjà ces touches sur toute la page du lecteur ; deux écouteurs zoomeraient deux fois.
 */
import { useEffect, useRef, type RefObject } from 'react'

import { applyZoomKey, isTypingTarget, zoomKeyCommand } from './useReplayShortcuts'
import type { ReplayZoom } from './useReplayZoom'

/** Le plan est-il visé : pointeur dessus, ou focus à l'intérieur ? */
export function planIsTargeted(frame: Element, hovered: boolean, active: Element | null): boolean {
  return hovered || (active != null && frame.contains(active))
}

export function useReplayZoomKeys(canvasRef: RefObject<HTMLCanvasElement | null>, zoom: ReplayZoom): void {
  // Le cadrage par RÉFÉRENCE : il change à chaque geste, l'écouteur ne se réabonne pas pour autant.
  const live = useRef(zoom)
  useEffect(() => {
    live.current = zoom
  }, [zoom])

  useEffect(() => {
    const frame = canvasRef.current?.parentElement
    if (!frame) return
    let hovered = false
    const enter = () => {
      hovered = true
    }
    const leave = () => {
      hovered = false
    }
    function onKeyDown(e: KeyboardEvent) {
      if (!frame || e.ctrlKey || e.metaKey || e.altKey || isTypingTarget(e.target)) return
      if (!planIsTargeted(frame, hovered, document.activeElement)) return
      const command = zoomKeyCommand(e.key)
      if (!command) return
      e.preventDefault()
      applyZoomKey(command, live.current)
    }
    frame.addEventListener('pointerenter', enter)
    frame.addEventListener('pointerleave', leave)
    window.addEventListener('keydown', onKeyDown)
    return () => {
      frame.removeEventListener('pointerenter', enter)
      frame.removeEventListener('pointerleave', leave)
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [canvasRef])
}
