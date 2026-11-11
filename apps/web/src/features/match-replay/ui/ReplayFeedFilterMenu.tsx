/**
 * ReplayFeedFilterMenu — LE TITRE DU FIL DEVIENT UN MENU : le bandeau HUD en tête du fil est le
 * bouton qui ouvre la liste des filtres (Tout, Éliminations, Objectifs, puis les familles
 * proposées en plus), avec le pictogramme et le nom du filtre actif. Chaque entrée dit son
 * nombre de lignes dans tout le match.
 *
 * CLAVIER ET ARIA : le déclencheur annonce `aria-haspopup="menu"` et `aria-expanded` ; la liste
 * est un `role="menu"` d'entrées `menuitemradio` (`aria-checked` sur le filtre actif). Flèche
 * bas ouvre depuis le déclencheur ; à l'ouverture le focus va au filtre actif ; flèches haut et
 * bas, Début et Fin déplacent le focus (en boucle) ; Entrée choisit ; Échap et Tab ferment, et
 * Échap rend le focus au déclencheur. Un clic dehors ferme (au `pointerdown`, comme le menu de
 * vitesse). La couche se signale au mode plein écran (`data-replay-escape-layer`) : Échap ferme
 * le menu AVANT de quitter le mode.
 */
import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { FeedFilterId, FeedFilterOption } from '../model/feedFilterLogic'
import { FEED_FILTER_GLYPH } from '../model/feedGlyphs'
import { HUD_BAND_CLASS, hudBandStyle } from '../model/hudBand'
import { EventGlyph } from './ReplayEventGlyph'

/** La première entrée « proposée en plus » : le séparateur du menu s'écrit juste avant elle. */
const FIRST_EXTRA_FILTER: FeedFilterId = 'pickups'

interface Props {
  options: readonly FeedFilterOption[]
  active: FeedFilterId
  onChange: (filter: FeedFilterId) => void
  locale: ReplayLocale
}

export function ReplayFeedFilterMenu({ options, active, onChange, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const [open, setOpen] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  // Clic dehors : le menu part au geste (`pointerdown`), pas au relâché.
  useEffect(() => {
    if (!open) return
    function onPointerDown(e: PointerEvent) {
      const target = e.target as Node | null
      if (target && wrapRef.current?.contains(target)) return
      setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  // À l'ouverture, le focus va au filtre actif : le clavier part de là où l'on est.
  useEffect(() => {
    if (!open) return
    const items = wrapRef.current?.querySelectorAll<HTMLElement>('[role="menuitemradio"]')
    const current = wrapRef.current?.querySelector<HTMLElement>('[role="menuitemradio"][aria-checked="true"]')
    ;(current ?? items?.[0])?.focus()
  }, [open])

  function moveFocus(to: 'next' | 'prev' | 'first' | 'last') {
    const items = Array.from(
      wrapRef.current?.querySelectorAll<HTMLElement>('[role="menuitemradio"]') ?? [],
    )
    if (items.length === 0) return
    const at = items.indexOf(document.activeElement as HTMLElement)
    const next =
      to === 'first' ? 0
      : to === 'last' ? items.length - 1
      : to === 'next' ? (at + 1) % items.length
      : (at - 1 + items.length) % items.length
    items[next]?.focus()
  }

  function onMenuKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const to = { ArrowDown: 'next', ArrowUp: 'prev', Home: 'first', End: 'last' }[e.key] as
      | 'next' | 'prev' | 'first' | 'last' | undefined
    if (to) {
      e.preventDefault()
      moveFocus(to)
    } else if (e.key === 'Escape') {
      e.stopPropagation()
      setOpen(false)
      triggerRef.current?.focus()
    } else if (e.key === 'Tab') {
      setOpen(false)
    } else if (e.key === ' ') {
      // Espace choisit l'entrée : il ne doit pas aussi lancer ou arrêter la lecture (touche du lecteur).
      e.stopPropagation()
    }
  }

  function onTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key === 'ArrowDown' && !open) {
      e.preventDefault()
      setOpen(true)
    } else if (e.key === ' ') {
      e.stopPropagation()
    }
  }

  function choose(id: FeedFilterId) {
    onChange(id)
    setOpen(false)
    triggerRef.current?.focus()
  }

  return (
    <div ref={wrapRef}>
      <div className={`${HUD_BAND_CLASS} text-foreground`} style={hudBandStyle(null)}>
        <button
          ref={triggerRef}
          type="button"
          aria-haspopup="menu"
          aria-expanded={open}
          aria-controls={open ? menuId : undefined}
          aria-label={`${t.feedFilterMenuLabel} : ${t.feedFilterLabel[active]}`}
          onClick={() => setOpen((v) => !v)}
          onKeyDown={onTriggerKeyDown}
          className="flex cursor-pointer items-center gap-1.5 uppercase tracking-[.16em]"
        >
          <EventGlyph glyph={FEED_FILTER_GLYPH[active]} size={12} />
          {t.feedFilterLabel[active]}
          <svg width="9" height="6" viewBox="0 0 9 6" aria-hidden="true" className="shrink-0">
            <path d="M1 1l3.5 4L8 1" stroke="currentColor" strokeWidth="1.5" fill="none" />
          </svg>
        </button>
      </div>
      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label={t.feedFilterMenuLabel}
          // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
          data-replay-escape-layer=""
          onKeyDown={onMenuKeyDown}
          className="absolute left-1.5 top-7 z-20 w-60 rounded-lg border border-border bg-popover p-1 shadow-xl"
        >
          {options.map((o) => (
            <div key={o.id}>
              {o.id === FIRST_EXTRA_FILTER && (
                <div className="px-2 pb-0.5 pt-1.5 text-3xs uppercase tracking-wider text-muted-foreground">
                  {t.feedFilterMoreHeading}
                </div>
              )}
              <button
                type="button"
                role="menuitemradio"
                aria-checked={o.id === active}
                onClick={() => choose(o.id)}
                className={`flex h-7 w-full cursor-pointer items-center gap-2 rounded px-2 text-left text-xs transition-colors ${
                  o.id === active ? 'bg-accent text-accent-foreground' : 'hover:bg-accent'
                }`}
              >
                <EventGlyph glyph={FEED_FILTER_GLYPH[o.id]} />
                {t.feedFilterLabel[o.id]}
                <span className="ml-auto tabular-nums text-muted-foreground">{o.count}</span>
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
