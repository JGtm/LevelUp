import { isKeyboardFocus } from '@/lib/dom/focusVisible'
import {
  claimTitle,
  findTitleSource,
  isCoveredByTooltip,
  refreshClaim,
  releaseTitle,
  STASH_ATTRIBUTE,
  type TitleSource,
} from './titleTooltip.logic'

/** Délai entre l'entrée du pointeur sur un `title` et l'affichage : évite le clignotement en traversant une barre d'outils. */
export const SHOW_DELAY_MS = 150

export interface TitleTooltipHooks {
  /** Affiche l'infobulle de `owner`, ou la met à jour (texte neuf, repositionnement après un défilement). */
  show(owner: Element, text: string): void
  hide(): void
}

type Mode = 'hover' | 'focus'
type ListenerEntry = [EventTarget, string, EventListener, boolean]

/**
 * Branche la couche des `title` sur `doc` : UN jeu d'écouteurs délégués (aucun observateur permanent).
 *
 * Deux prises en charge indépendantes : l'élément sous le POINTEUR et l'élément qui a le FOCUS CLAVIER. Un
 * élément garde son `title` vidé tant que l'une des deux le tient : Tab vers A pendant que le pointeur reste
 * sur B ne rend pas son titre à B (l'infobulle native de B reviendrait sous le pointeur).
 *
 * Cycle : pris en charge à l'entrée -> infobulle affichée après `SHOW_DELAY_MS` (tout de suite au clavier) ->
 * masquée au clic, à Échap, au redimensionnement, et au défilement SAUF pour une infobulle ouverte au focus
 * (elle se repositionne, ou se masque si l'élément sort de la fenêtre : le défilement d'un Tab vers un
 * contrôle hors écran ne doit pas la tuer). Masquer ne rend pas le `title` : il revient quand plus rien ne
 * tient l'élément, quand la fenêtre perd le focus ou devient cachée, et au détachement.
 *
 * Pendant toute la prise en charge, un observateur ciblé sur les éléments tenus suit les réécritures et
 * retraits du `title` par l'application (rendu React) : une réécriture met l'infobulle à jour, un retrait
 * l'oublie sans rien restituer. Un élément sous une infobulle JS (`isCoveredByTooltip`) est pris en charge
 * sans rien afficher. Un second observateur, actif pendant l'affichage seulement, masque l'infobulle d'un
 * élément retiré du DOM.
 */
class TitleTooltipController {
  private hover: Element | null = null
  private focus: Element | null = null
  private visual: { owner: Element; mode: Mode } | null = null
  private visible = false
  private timer: number | null = null
  private readonly titleObserver: MutationObserver
  private removalObserver: MutationObserver | null = null
  private readonly listeners: ListenerEntry[]

  private readonly doc: Document
  private readonly win: Window
  private readonly hooks: TitleTooltipHooks
  private readonly delayMs: number

  constructor(doc: Document, win: Window, hooks: TitleTooltipHooks, delayMs: number) {
    this.doc = doc
    this.win = win
    this.hooks = hooks
    this.delayMs = delayMs
    this.titleObserver = new MutationObserver(() => this.onTitleMutation())
    this.listeners = [
      [doc, 'pointerover', this.onPointerOver, false],
      [doc, 'pointerout', this.onPointerOut, false],
      [doc, 'pointerdown', this.hideVisual, true],
      [doc, 'click', this.hideVisual, true],
      [doc, 'focusin', this.onFocusIn, false],
      [doc, 'focusout', this.onFocusOut, false],
      [doc, 'keydown', this.onKeyDown, true],
      [doc, 'visibilitychange', this.onVisibility, false],
      [win, 'scroll', this.onScroll, true],
      [win, 'resize', this.hideVisual, false],
      [win, 'blur', this.releaseAll, false],
    ]
    for (const [target, type, fn, capture] of this.listeners) target.addEventListener(type, fn, { capture, passive: true })
  }

  detach(): void {
    for (const [target, type, fn, capture] of this.listeners) target.removeEventListener(type, fn, { capture })
    this.releaseAll()
    this.titleObserver.disconnect()
  }

  // --- prises en charge -------------------------------------------------------------------------------------

  private holds(owner: Element): boolean {
    return owner === this.hover || owner === this.focus
  }

  /** Reprend l'observation des seuls éléments encore tenus (un observateur ne désinscrit pas une cible isolée). */
  private observeHeld(): void {
    this.titleObserver.disconnect()
    for (const owner of new Set([this.hover, this.focus])) {
      if (!owner) continue
      if (owner instanceof SVGElement) this.titleObserver.observe(owner, { childList: true, characterData: true, subtree: true })
      else this.titleObserver.observe(owner, { attributes: true, attributeFilter: ['title'] })
    }
  }

  /** Rend le titre de `owner` si plus rien ne le tient. */
  private drop(owner: Element | null): void {
    if (!owner || this.holds(owner)) return
    releaseTitle(owner)
    this.observeHeld()
    if (this.visual?.owner === owner) this.hideVisual()
  }

  /** `owner` n'est plus utilisable (retiré du DOM, titre supprimé par l'application) : plus aucune prise. */
  private dissociate(owner: Element): void {
    if (this.hover === owner) this.hover = null
    if (this.focus === owner) this.focus = null
    this.drop(owner)
  }

  private setHover(source: TitleSource | null): void {
    const previous = this.hover
    if (source && source.owner === previous) return
    if (source) claimTitle(source)
    this.hover = source?.owner ?? null
    this.observeHeld()
    this.drop(previous)
    if (previous && this.visual?.owner === previous && this.visual.mode === 'hover') this.hideVisual()
    if (source) this.present(source.owner, 'hover')
  }

  private setFocus(source: TitleSource | null): void {
    const previous = this.focus
    if (source && source.owner === previous) return
    if (source) claimTitle(source)
    this.focus = source?.owner ?? null
    this.observeHeld()
    this.drop(previous)
    if (previous && this.visual?.owner === previous && this.visual.mode === 'focus') this.hideVisual()
    if (source) this.present(source.owner, 'focus')
  }

  private releaseAll = (): void => {
    this.hideVisual()
    const held = [this.hover, this.focus]
    this.hover = null
    this.focus = null
    for (const owner of held) this.drop(owner)
  }

  // --- affichage --------------------------------------------------------------------------------------------

  private present(owner: Element, mode: Mode): void {
    this.hideVisual()
    if (isCoveredByTooltip(owner)) return
    this.visual = { owner, mode }
    if (mode === 'focus') this.open()
    else this.timer = this.win.setTimeout(this.open, this.delayMs)
  }

  private open = (): void => {
    this.timer = null
    const visual = this.visual
    if (!visual) return
    if (!visual.owner.isConnected) {
      this.dissociate(visual.owner)
      return
    }
    if (!this.render(visual.owner)) return
    this.visible = true
    this.removalObserver = new MutationObserver(() => {
      if (this.visual && !this.visual.owner.isConnected) this.dissociate(this.visual.owner)
    })
    this.removalObserver.observe(this.doc.body, { childList: true, subtree: true })
  }

  /** (Re)dessine l'infobulle de `owner` d'après le texte retenu ; `false` si plus de texte (prise oubliée). */
  private render(owner: Element): boolean {
    const text = owner.getAttribute(STASH_ATTRIBUTE)
    if (text === null) {
      this.dissociate(owner)
      return false
    }
    this.hooks.show(owner, text.trim())
    return true
  }

  private hideVisual = (): void => {
    if (this.timer !== null) this.win.clearTimeout(this.timer)
    this.timer = null
    this.removalObserver?.disconnect()
    this.removalObserver = null
    this.visual = null
    if (!this.visible) return
    this.visible = false
    this.hooks.hide()
  }

  // --- événements -------------------------------------------------------------------------------------------

  private onTitleMutation(): void {
    for (const owner of new Set([this.hover, this.focus])) {
      if (!owner) continue
      const change = refreshClaim(owner)
      if (change.kind === 'gone') this.dissociate(owner)
      else if (change.kind === 'text' && this.visible && this.visual?.owner === owner) this.hooks.show(owner, change.text.trim())
    }
  }

  private onPointerOver = (e: Event): void => {
    if ((e as PointerEvent).pointerType === 'touch' || !(e.target instanceof Element)) return
    this.setHover(findTitleSource(e.target))
  }

  private onPointerOut = (e: Event): void => {
    if ((e as PointerEvent).relatedTarget === null) this.setHover(null)
  }

  private onFocusIn = (e: Event): void => {
    if (!(e.target instanceof Element) || !isKeyboardFocus(e.target)) return
    this.setFocus(findTitleSource(e.target))
  }

  private onFocusOut = (e: Event): void => {
    if (this.focus && e.target instanceof Node && this.focus.contains(e.target)) this.setFocus(null)
  }

  private onKeyDown = (e: Event): void => {
    if ((e as KeyboardEvent).key === 'Escape') this.hideVisual()
  }

  private onVisibility = (): void => {
    if (this.doc.visibilityState === 'hidden') this.releaseAll()
  }

  /**
   * Un défilement masque l'infobulle d'un survol (le contenu a bougé sous le pointeur) mais PAS celle d'un
   * focus clavier : Tab vers un contrôle hors écran fait défiler la page après l'ouverture, et l'infobulle
   * doit suivre son contrôle, sauf s'il sort de la fenêtre.
   */
  private onScroll = (): void => {
    const visual = this.visual
    if (!visual) return
    if (visual.mode === 'hover' || !this.visible) {
      this.hideVisual()
      return
    }
    const r = visual.owner.getBoundingClientRect()
    const outside = r.bottom < 0 || r.top > this.win.innerHeight || r.right < 0 || r.left > this.win.innerWidth
    if (outside) this.hideVisual()
    else this.render(visual.owner)
  }
}

/** Branche la couche des `title` sur `doc` ; retourne la fonction de détachement (rend les titres, retire les écouteurs). */
export function attachTitleTooltips(doc: Document, hooks: TitleTooltipHooks, delayMs = SHOW_DELAY_MS): () => void {
  const win = doc.defaultView
  if (!win) return () => {}
  const controller = new TitleTooltipController(doc, win, hooks, delayMs)
  return () => controller.detach()
}
