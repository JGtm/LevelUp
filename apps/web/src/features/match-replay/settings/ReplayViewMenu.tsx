/**
 * ReplayViewMenu — LE BOUTON « VUE » en haut à droite de la carte, et le panneau qu'il ouvre.
 *
 * Il remplace le bouton de réglages de la barre de lecture : la barre y gagne une commande de
 * moins, et la vue — ce que la carte montre — se règle sur la carte elle-même. Le bouton nomme
 * la vue active (un préréglage, ou « Personnalisé ») ; `aria-expanded` dit si le panneau est
 * ouvert. Le panneau lui-même est rendu en portail (cf. `ReplayViewPanel`).
 *
 * À poser DANS le cadre de la toile : son `absolute right-2 top-2` se résout sur le premier
 * ancêtre positionné, qui est le conteneur de la toile.
 */
import { SlidersIcon } from './SlidersIcon'
import { ReplayViewPanel } from './ReplayViewPanel'
import type { ReplayViewMenuState } from './useReplayViewMenu'
import { REPLAY_TEXT } from '../i18n/i18n'

export function ReplayViewMenu({ open, toggle, buttonRef, active, panel }: ReplayViewMenuState) {
  const t = REPLAY_TEXT[panel.locale]
  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={toggle}
        aria-expanded={open}
        // LE NOM ACCESSIBLE DIT LES DEUX : deux étiquettes voisines d'une rangée flex ne se lisent
        // pas toujours avec une espace entre elles.
        aria-label={`${t.viewButton} ${t.viewPreset[active]}`}
        title={t.viewButton}
        className={`absolute right-2 top-2 z-20 inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-md border border-border px-2.5 text-xs backdrop-blur-sm transition-colors ${
          open
            ? 'bg-accent text-accent-foreground'
            : 'bg-card/85 text-muted-foreground hover:bg-accent hover:text-accent-foreground'
        }`}
      >
        <SlidersIcon />
        <span>{t.viewButton}</span>
        <span className="font-semibold text-foreground">{t.viewPreset[active]}</span>
      </button>
      {open && <ReplayViewPanel {...panel} triggerRef={buttonRef} />}
    </>
  )
}
