import { isKeyboardFocus } from '@/lib/dom/focusVisible'
import { claimTitle, findTitleSource, isCoveredByTooltip, releaseTitle, type TitleSource } from './titleTooltip.logic'

/** Délai entre l'entrée du pointeur sur un `title` et l'affichage : évite le clignotement en traversant une barre d'outils. */
export const SHOW_DELAY_MS = 150

export interface TitleTooltipHooks {
  show(owner: Element, text: string): void
  hide(): void
}

/**
 * Branche la couche des `title` sur `doc` : UN jeu d'écouteurs délégués (aucun observateur permanent).
 *
 * Cycle d'un élément : pris en charge à l'entrée (survol, ou focus clavier) -> infobulle affichée après
 * `SHOW_DELAY_MS` (tout de suite au clavier) -> infobulle masquée au clic, au défilement, au
 * redimensionnement et à Échap, mais l'élément RESTE pris en charge tant que le pointeur ou le focus ne le
 * quitte pas : lui rendre son `title` sous un pointeur immobile ferait réapparaître l'infobulle native. Le
 * `title` revient à la sortie, à la perte du focus, quand la fenêtre perd le focus, et au démontage.
 *
 * Un élément sous une infobulle JS (`isCoveredByTooltip`) est pris en charge sans rien afficher.
 * Un observateur de mutations n'existe que pendant l'affichage, pour masquer l'infobulle d'un élément retiré du DOM.
 *
 * Retourne la fonction de détachement (rend les titres, retire les écouteurs).
 */
export function attachTitleTooltips(doc: Document, hooks: TitleTooltipHooks, delayMs = SHOW_DELAY_MS): () => void {
  const win = doc.defaultView
  if (!win) return () => {}
  let current: Element | null = null
  let timer: number | null = null
  let observer: MutationObserver | null = null
  let visible = false

  const hideVisual = () => {
    if (timer !== null) win.clearTimeout(timer)
    timer = null
    observer?.disconnect()
    observer = null
    if (!visible) return
    visible = false
    hooks.hide()
  }

  const release = () => {
    hideVisual()
    if (current) releaseTitle(current)
    current = null
  }

  const open = (owner: Element, text: string) => {
    timer = null
    if (current !== owner) return
    if (!owner.isConnected) {
      release()
      return
    }
    visible = true
    hooks.show(owner, text.trim())
    observer = new MutationObserver(() => {
      if (current && !current.isConnected) release()
    })
    observer.observe(doc.body, { childList: true, subtree: true })
  }

  const engage = (source: TitleSource, immediate: boolean) => {
    if (current === source.owner) return
    release()
    claimTitle(source)
    current = source.owner
    if (isCoveredByTooltip(source.owner)) return
    if (immediate) open(source.owner, source.text)
    else timer = win.setTimeout(() => open(source.owner, source.text), delayMs)
  }

  const onPointerOver = (e: Event) => {
    if ((e as PointerEvent).pointerType === 'touch' || !(e.target instanceof Element)) return
    const source = findTitleSource(e.target)
    if (source) engage(source, false)
    else release()
  }
  const onPointerOut = (e: Event) => {
    if ((e as PointerEvent).relatedTarget === null) release()
  }
  const onFocusIn = (e: Event) => {
    if (!(e.target instanceof Element) || !isKeyboardFocus(e.target)) return
    const source = findTitleSource(e.target)
    if (source) engage(source, true)
    else release()
  }
  const onFocusOut = (e: Event) => {
    if (current && e.target instanceof Node && current.contains(e.target)) release()
  }
  const onKeyDown = (e: Event) => {
    if ((e as KeyboardEvent).key === 'Escape') hideVisual()
  }
  const onHidePage = () => {
    if (doc.visibilityState === 'hidden') release()
  }

  const listeners: Array<[EventTarget, string, EventListener, boolean]> = [
    [doc, 'pointerover', onPointerOver, false],
    [doc, 'pointerout', onPointerOut, false],
    [doc, 'pointerdown', hideVisual, true],
    [doc, 'click', hideVisual, true],
    [doc, 'focusin', onFocusIn, false],
    [doc, 'focusout', onFocusOut, false],
    [doc, 'keydown', onKeyDown, true],
    [doc, 'visibilitychange', onHidePage, false],
    [win, 'scroll', hideVisual, true],
    [win, 'resize', hideVisual, false],
    [win, 'blur', release, false],
  ]
  for (const [target, type, fn, capture] of listeners) target.addEventListener(type, fn, { capture, passive: true })

  return () => {
    for (const [target, type, fn, capture] of listeners) target.removeEventListener(type, fn, { capture })
    release()
  }
}
