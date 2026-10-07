/**
 * useHauteurDuPlan — la hauteur que la fenêtre laisse à la boîte du plan : du haut de la boîte au
 * bas de la zone qui défile, moins la marge du bas (`PLAN_MARGE_BAS_PX`), jamais sous
 * `PLAN_HAUTEUR_MIN_PX`. C'est ce qui tient le cockpit dans l'écran, sans défilement, quelle que
 * soit la hauteur de la fenêtre — la largeur de la boîte suit au rapport du fond (`boiteDuPlan`).
 *
 * LA MESURE NE DÉPEND PAS DU DÉFILEMENT : le haut de la boîte est rapporté au CONTENU de la zone qui
 * défile (`scrollTop` ajouté), donc la hauteur ne change pas quand l'utilisateur fait défiler une
 * fenêtre trop basse. Elle se refait quand la fenêtre change de taille et quand ce qui précède la
 * boîte change de hauteur (la barre de filtres qui passe sur deux lignes, une note qui apparaît).
 */
import { useLayoutEffect, useState, type RefObject } from 'react'

import { PLAN_HAUTEUR_MIN_PX, PLAN_MARGE_BAS_PX } from './cockpit.logic'

/** La zone qui défile autour d'un élément : le premier ancêtre à défilement vertical, sinon le document. */
function zoneQuiDefile(el: HTMLElement): HTMLElement {
  for (let p = el.parentElement; p; p = p.parentElement) {
    const { overflowY } = getComputedStyle(p)
    if (overflowY === 'auto' || overflowY === 'scroll') return p
  }
  return (document.scrollingElement as HTMLElement | null) ?? document.documentElement
}

/** hauteurDisponible — la hauteur laissée sous `haut` (px depuis le haut du contenu) dans une vue de `vue` px. */
export function hauteurDisponible(vue: number, haut: number): number {
  return Math.max(PLAN_HAUTEUR_MIN_PX, Math.floor(vue - haut - PLAN_MARGE_BAS_PX))
}

export function useHauteurDuPlan(ref: RefObject<HTMLElement | null>): number {
  const [hauteur, setHauteur] = useState(() => hauteurDisponible(typeof window === 'undefined' ? 0 : window.innerHeight, 0))
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const zone = zoneQuiDefile(el)
    const estDocument = zone === document.scrollingElement || zone === document.documentElement
    const mesurer = () => {
      const hautZone = estDocument ? 0 : zone.getBoundingClientRect().top
      const vue = estDocument ? window.innerHeight : zone.clientHeight
      const haut = el.getBoundingClientRect().top - hautZone + zone.scrollTop
      setHauteur(hauteurDisponible(vue, haut))
    }
    mesurer()
    window.addEventListener('resize', mesurer)
    // Le patron du dépôt : l'observateur est optionnel (jsdom ne le fournit pas).
    const ro = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(mesurer)
    const contenu = zone.firstElementChild
    if (ro && contenu) ro.observe(contenu)
    return () => {
      window.removeEventListener('resize', mesurer)
      ro?.disconnect()
    }
  }, [ref])
  return hauteur
}
