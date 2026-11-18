/**
 * ReplayPlayerPopover — LA FICHE DU JOUEUR EN FENÊTRE, au Cinéma et au Plein cadre : ce qu'un clic sur une fiche de la
 * rangée (ou sur une ligne de la colonne de repli) ouvre. Une fenêtre de 330 px ancrée SOUS la fiche cliquée, bord
 * gauche aligné sur le sien (par la droite si elle sortirait de la page), qui porte `ReplayPlayerPanelBody`.
 *
 * NON BLOQUANTE. Aucun voile : la lecture continue, le plan reste zoomable et déplaçable, le fil et la barre de
 * lecture restent utilisables, et un clic ailleurs ne la ferme PAS — ce n'est pas un menu. Elle se ferme par sa
 * croix, par Échap, ou en recliquant la fiche du même joueur ; cliquer une AUTRE fiche change de joueur (la fenêtre,
 * remontée par la clé du cadre, se ré-ancre sous la nouvelle fiche).
 *
 * ÉCHAP : la fenêtre est une couche (`data-replay-escape-layer`) fermée par `useDismissable` (`useWindowClose`,
 * Échap seul : `outside: null`). Une frappe ferme une couche : la fenêtre avant le Plein cadre, une couche plus
 * intérieure (menu de vitesse, panneau de la vue) avant la fenêtre. Le focus revient à la fiche.
 *
 * POSITION : un portail sur `body` en position fixe (`useAnchoredPanel`) — le plan porte `overflow-hidden`. Le
 * déclencheur est la fiche ouverte (`aria-pressed`), relue à chaque placement : une fiche remontée garde son ancre.
 * La hauteur est bornée par le BAS DU PLAN (la toile) : au-delà, le corps défile en interne, la tête et les actions
 * restent en place.
 *
 * Aucun littéral de couleur : surface, bordures et neutres sont des jetons sémantiques.
 */
import { useCallback, useEffect, useRef, type RefObject } from 'react'
import { createPortal } from 'react-dom'

import { useDismissable } from '@/lib/hooks/useDismissable'

import { REPLAY_ESCAPE_LAYER_ATTR } from '../hooks/useReplayFullscreen'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COTES_FENETRE_FICHE } from '../model/cotesFenetreFiche'
import { cardName } from '../model/playerCardReadings'
import { useAnchoredPanel } from '../settings/useAnchoredPanel'
import { ReplayPlayerPanelBody, type ReplayPlayerPanelBodyProps } from './ReplayPlayerPanelBody'
import { PRESS_GUARDS } from './replayPressGuards'

export interface ReplayPlayerPopoverProps extends Omit<ReplayPlayerPanelBodyProps, 'headEnd' | 'scroll'> {
  onClose: () => void
}

/** Le plan : la toile de la grille du cadre ; son bord bas borne la hauteur de la fenêtre. */
const PLAN_SELECTOR = '[data-replay-grid] canvas'

/** La fiche (ou la ligne) dont la fenêtre est ouverte : la tuile-bouton choisie (`aria-pressed`). */
const OPEN_CARD_SELECTOR = '[data-replay-player-card][aria-pressed="true"]'

/** Le déclencheur, relu à chaque placement : la fiche (ou la ligne) dont la fenêtre est ouverte. */
const ANCRE_FICHE: RefObject<HTMLElement | null> = {
  get current() {
    return document.querySelector<HTMLElement>(OPEN_CARD_SELECTOR)
  },
}

/** L'élément borne, relu à chaque placement : la toile du plan. */
const BORNE_PLAN: RefObject<HTMLElement | null> = {
  get current() {
    return document.querySelector<HTMLElement>(PLAN_SELECTOR)
  },
}

/**
 * LA SORTIE DE LA FENÊTRE : la croix et Échap ferment par le même chemin, qui rend le focus à la fiche (sans quoi il
 * tomberait sur la page). Échap, et lui seul (`outside: null`) : un clic ailleurs ne ferme rien. Il ne ferme que si
 * aucune AUTRE couche d'Échap n'est ouverte : elle se ferme la première, par son propre écouteur.
 */
function useWindowClose(rootRef: RefObject<HTMLElement | null>, onClose: () => void): () => void {
  const returnTo = useRef<HTMLElement | null>(null)
  useEffect(() => () => returnTo.current?.focus({ preventScroll: true }), [])
  const close = useCallback(() => {
    returnTo.current = document.querySelector<HTMLElement>(OPEN_CARD_SELECTOR)
    onClose()
  }, [onClose])
  useDismissable(true, close, { ref: rootRef, outside: null, innerLayerSelector: `[${REPLAY_ESCAPE_LAYER_ATTR}]` })
  return close
}

/** La capsule « Échap » puis la croix de fermeture, à droite de la tête de la fiche. */
function HeadEnd({ close, locale }: { close: () => void; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  const K = COTES_FENETRE_FICHE.capsule
  const X = COTES_FENETRE_FICHE.croix
  return (
    <div className="flex shrink-0 items-center gap-2">
      <kbd
        className="inline-grid place-items-center rounded border border-border font-sans text-muted-foreground"
        style={{ minWidth: K.largeurMinPx, height: K.hauteurPx, paddingInline: K.margeIntPx, fontSize: K.taillePx }}
      >
        {t.frames.escapeKey}
      </kbd>
      <button
        type="button"
        aria-label={t.playerPanel.close}
        title={t.playerPanel.close}
        className="inline-grid place-items-center text-muted-foreground hover:bg-muted hover:text-foreground"
        style={{ width: X.cotePx, height: X.cotePx, borderRadius: X.rayonPx }}
        {...PRESS_GUARDS}
        onClick={close}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={X.traitPx}
          strokeLinecap="round"
          aria-hidden="true"
          style={{ width: X.tracePx, height: X.tracePx }}
        >
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
    </div>
  )
}

export function ReplayPlayerPopover({ onClose, ...body }: ReplayPlayerPopoverProps) {
  const t = REPLAY_TEXT[body.locale]
  const C = COTES_FENETRE_FICHE
  const rootRef = useRef<HTMLElement>(null)
  const close = useWindowClose(rootRef, onClose)
  const pos = useAnchoredPanel(ANCRE_FICHE, C.largeurPx, { align: 'start', boundRef: BORNE_PLAN })
  return createPortal(
    <section
      ref={rootRef}
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      data-replay-fiche-popover=""
      aria-label={t.playerPanel.labelFmt(cardName(body.player, t))}
      className="fixed z-50 flex max-w-[calc(100vw-1rem)] flex-col overflow-hidden border border-border bg-popover text-popover-foreground shadow-xl outline-none"
      style={{
        left: pos.left,
        top: pos.top,
        width: C.largeurPx,
        maxHeight: pos.maxHeight,
        borderRadius: C.rayonPx,
        padding: `${C.margeVerticalePx}px ${C.margeLateralePx}px`,
        gap: C.ecartBlocsPx,
      }}
    >
      <ReplayPlayerPanelBody {...body} scroll="internal" headEnd={<HeadEnd close={close} locale={body.locale} />} />
    </section>,
    document.body,
  )
}
