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
 * Échap rend le focus au déclencheur. Échap et le clic dehors sont ceux de `useDismissable` (au
 * `pointerdown`, comme le menu de vitesse). La couche se signale au mode plein écran
 * (`data-replay-escape-layer`) : Échap ferme le menu AVANT de quitter le mode. Menu ouvert, AUCUNE
 * touche ne traverse vers le clavier du lecteur (`useReplayShortcuts`) : ni les flèches gauche et
 * droite, ni J, L, F, M, R. La navigation au clavier est dans `useFeedFilterMenuKeys`.
 */
import { useId, useRef, useState, type KeyboardEvent } from 'react'

import { useDismissable } from '@/lib/hooks/useDismissable'

import { useFeedFilterMenuKeys } from '../hooks/useFeedFilterMenuKeys'
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
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  // Échap et clic dehors : le menu part au geste (`pointerdown`), pas au relâché ; Échap rend le
  // focus au déclencheur, le clic dehors le laisse où l'on a cliqué.
  const wrapRef = useDismissable(
    open,
    (reason) => {
      setOpen(false)
      if (reason === 'escape') triggerRef.current?.focus()
    },
    { outside: 'pointerdown' },
  )

  const { onMenuKeyDown, onTriggerKeyDown } = useFeedFilterMenuKeys({ open, setOpen, wrapRef })

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
        <FeedFilterList
          id={menuId}
          options={options}
          active={active}
          onChoose={choose}
          onKeyDown={onMenuKeyDown}
          locale={locale}
        />
      )}
    </div>
  )
}

interface FeedFilterListProps {
  id: string
  options: readonly FeedFilterOption[]
  active: FeedFilterId
  onChoose: (filter: FeedFilterId) => void
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void
  locale: ReplayLocale
}

/** La liste des filtres : une entrée par filtre, le séparateur des « proposés en plus » avant la première. */
function FeedFilterList({ id, options, active, onChoose, onKeyDown, locale }: FeedFilterListProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <div
      id={id}
      role="menu"
      aria-label={t.feedFilterMenuLabel}
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      onKeyDown={onKeyDown}
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
            onClick={() => onChoose(o.id)}
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
  )
}
