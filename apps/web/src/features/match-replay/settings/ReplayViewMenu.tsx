/**
 * ReplayViewMenu — LE BOUTON « VUE » en haut à droite de la carte, et le panneau qu'il ouvre.
 *
 * Il remplace le bouton de réglages de la barre de lecture : la barre y gagne une commande de moins, et la vue — ce que
 * la carte montre — se règle sur la carte elle-même. Le bouton nomme la vue active, suivie de « · modifié » quand un
 * réglage s'en écarte ; `aria-expanded` dit si le panneau est ouvert. Le panneau lui-même est rendu en portail
 * (cf. `ReplayViewPanel`).
 *
 * À côté du bouton, à sa GAUCHE, vivent l'accès rapide (`ReplayQuickAccess`) puis l'emplacement du commutateur 2D / 3D
 * (`ReplayDimensionSwitch`), rendu seulement si le serveur offre la vue 3D. Ils partagent un même conteneur : la
 * rangée suit la largeur du bouton, qui change avec le nom de la vue active, ce qu'un décalage fixe ne ferait pas.
 *
 * À poser DANS le cadre de la toile : son `absolute right-2 top-2` se résout sur le premier ancêtre positionné, qui est
 * le conteneur de la toile.
 */
import { REPLAY_TEXT } from '../i18n/i18n'
import { ReplayDimensionSwitch } from '../ui/ReplayDimensionSwitch'
import { ReplayQuickAccess } from './ReplayQuickAccess'
import { ReplayViewPanel } from './ReplayViewPanel'
import { SlidersIcon } from './SlidersIcon'
import type { ReplayViewMenuState } from './useReplayViewMenu'

export function ReplayViewMenu({ open, toggle, close, buttonRef, label, model, dimension }: ReplayViewMenuState) {
  const t = REPLAY_TEXT[model.locale]
  return (
    <>
      <div className="absolute right-2 top-2 z-20 flex items-center gap-2">
        <ReplayDimensionSwitch {...dimension} locale={model.locale} />
        <ReplayQuickAccess model={model} />
        <button
          ref={buttonRef}
          type="button"
          onClick={toggle}
          aria-expanded={open}
          aria-haspopup="dialog"
          // LE NOM ACCESSIBLE DIT TOUT : deux étiquettes voisines d'une rangée flex ne se lisent pas toujours avec une espace entre elles.
          aria-label={`${t.viewButton} ${label.name}${label.modified ? ` ${t.settings.buttonModifiedAria}` : ''}`}
          title={t.viewButton}
          className={`inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-md border border-border px-2.5 text-xs backdrop-blur-sm transition-colors ${
            open
              ? 'bg-accent text-accent-foreground'
              : 'bg-card/85 text-muted-foreground hover:bg-accent hover:text-accent-foreground'
          }`}
        >
          <SlidersIcon />
          <span>{t.viewButton}</span>
          <span className="font-semibold text-foreground">{label.name}</span>
          {label.modified && <span className="font-normal text-muted-foreground">{`· ${t.settings.modified}`}</span>}
        </button>
      </div>
      {open && <ReplayViewPanel model={model} onClose={close} triggerRef={buttonRef} />}
    </>
  )
}
