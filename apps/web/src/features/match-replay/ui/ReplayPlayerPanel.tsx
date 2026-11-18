/**
 * ReplayPlayerPanel — LA FICHE DU JOUEUR DANS LA COLONNE DU STANDARD : ce qu'un clic sur une ligne de la colonne
 * ouvre, à la PLACE de toute la colonne (équipes et fil). « Équipes » la referme. Au Cinéma et au Plein cadre, la
 * fiche est une fenêtre (`ReplayPlayerPopover`).
 *
 * Le panneau est le CONTENEUR : la section qui remplit la colonne, son nom accessible, l'en-tête (retour « Équipes »
 * à gauche, capsule « Échap » à droite), le sélecteur de joueurs, le pied « Éliminations » et la sortie (Échap,
 * retour, « Fil complet »). Tout le contenu — tête, trois actions, bloc « À l'instant », listes de totaux de fin
 * de match — est `ReplayPlayerPanelBody`, que la fenêtre monte à l'identique.
 *
 * LE PANNEAU NE DÉTRUIT NI LES ÉQUIPES NI LE FIL : la page les garde montés sous lui (masqués), si bien que le
 * filtre et le défilement sont retrouvés au retour. Il garde son joueur quand le temps avance.
 *
 * ÉCHAP LE FERME, AVANT LE MODE PLEIN ÉCRAN : le panneau est une couche (`data-replay-escape-layer`,
 * cf. `useReplayFullscreen`), et rend le focus à la fiche du joueur, qu'il quitte par Échap ou par « Équipes ». Une
 * couche plus intérieure ouverte (menu de vitesse, de filtre, panneau de la vue) se ferme d'abord : Échap ne ferme
 * qu'une couche à la fois.
 *
 * UNE SEULE ZONE DÉFILE : l'en-tête, le sélecteur et le corps défilent ENSEMBLE, et le pied reste fixé en bas, hors de
 * cette zone. Retour, Échap et « Fil complet » sortent par le même chemin et rendent la colonne, équipes et fil.
 *
 * Aucun littéral de couleur : l'encre du camp et les neutres sont des jetons sémantiques.
 */
import { useCallback, useEffect, useRef, type ReactNode, type RefObject } from 'react'

import { useDismissable } from '@/lib/hooks/useDismissable'

import { REPLAY_ESCAPE_LAYER_ATTR } from '../hooks/useReplayFullscreen'
import { REPLAY_TEXT } from '../i18n/i18n'
import { COTES_COLONNE_STANDARD } from '../model/cotesStandard'
import { COTES_FICHE_STANDARD } from '../model/cotesFicheStandard'
import { cardName } from '../model/playerCardReadings'
import { ReplayPlayerPanelBody, type ReplayPlayerPanelBodyProps } from './ReplayPlayerPanelBody'
import { PRESS_GUARDS } from './replayPressGuards'

/** Ce que la fiche de la colonne ajoute au corps commun : le sélecteur et le pied. */
export interface ReplayPlayerPanelColumn {
  /** Le sélecteur de joueurs, sous l'en-tête : il défile avec le corps. */
  picker: ReactNode
  /** Le pied fixé en bas, hors de la zone qui défile ; il reçoit la sortie du panneau (la même que « Équipes »). */
  footer: (close: () => void) => ReactNode
}

export interface ReplayPlayerPanelProps extends Omit<ReplayPlayerPanelBodyProps, 'headEnd' | 'scroll'> {
  onClose: () => void
  /** Le sélecteur et le pied de la fiche, qui remplit toute la colonne du Standard. */
  column: ReplayPlayerPanelColumn
}

/** La fiche du joueur dont le panneau est ouvert : la tuile-bouton choisie (`aria-pressed`). */
const OPEN_CARD_SELECTOR = '[data-replay-player-card][aria-pressed="true"]'

/**
 * usePanelClose — LA SORTIE DU PANNEAU : « Équipes » et Échap ferment par le même chemin, qui
 * rend le focus à la fiche (sans quoi il tomberait sur la page avec le panneau retiré). Échap ne
 * ferme que si aucune AUTRE couche d'Échap n'est ouverte : elle se ferme la première, par son
 * propre écouteur.
 */
function usePanelClose(rootRef: RefObject<HTMLElement | null>, onClose: () => void): () => void {
  // Dans la colonne du Standard, la fiche recouvre les équipes (masquées) : un élément masqué ne prend pas le
  // focus, la fiche est donc rendue une seconde fois quand le panneau est retiré et les équipes redevenues visibles.
  const returnTo = useRef<HTMLElement | null>(null)
  useEffect(() => () => returnTo.current?.focus({ preventScroll: true }), [])
  const close = useCallback(() => {
    const card = document.querySelector<HTMLElement>(OPEN_CARD_SELECTOR)
    returnTo.current = card
    onClose()
    card?.focus({ preventScroll: true })
  }, [onClose])
  useDismissable(true, close, {
    ref: rootRef,
    outside: null,
    innerLayerSelector: `[${REPLAY_ESCAPE_LAYER_ATTR}]`,
  })
  return close
}

/** Le retour « Équipes » (chevron à gauche) et la capsule « Échap » à droite, tête de la zone qui défile. */
function ColumnHead({ close, locale }: { close: () => void; locale: ReplayPlayerPanelProps['locale'] }) {
  const t = REPLAY_TEXT[locale]
  const R = COTES_FICHE_STANDARD.retour
  const K = COTES_FICHE_STANDARD.capsule
  return (
    <div className="flex items-center" style={{ gap: R.ecartPx }}>
      <button
        type="button"
        className="inline-flex items-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
        style={{
          height: R.hauteurPx,
          gap: R.ecartPx,
          paddingLeft: R.margeGauchePx,
          paddingRight: R.margeDroitePx,
          fontSize: R.taillePx,
        }}
        {...PRESS_GUARDS}
        onClick={close}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          aria-hidden="true"
          style={{ width: R.chevronPx, height: R.chevronPx }}
        >
          <path d="M15 18l-6-6 6-6" />
        </svg>
        {t.playerPanel.backToTeams}
      </button>
      <kbd
        className="ml-auto inline-grid place-items-center rounded border border-border font-sans text-muted-foreground"
        style={{ minWidth: K.largeurMinPx, height: K.hauteurPx, paddingInline: K.margeIntPx, fontSize: K.taillePx }}
      >
        {t.frames.escapeKey}
      </kbd>
    </div>
  )
}

export function ReplayPlayerPanel({ onClose, column, ...body }: ReplayPlayerPanelProps) {
  const t = REPLAY_TEXT[body.locale]
  const rootRef = useRef<HTMLElement>(null)
  const close = usePanelClose(rootRef, onClose)
  return (
    <section
      ref={rootRef}
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      data-replay-fiche-column=""
      aria-label={t.playerPanel.labelFmt(cardName(body.player, t))}
      className="flex h-full min-h-0 flex-col"
      style={{
        padding: `${COTES_COLONNE_STANDARD.margeVerticalePx}px ${COTES_COLONNE_STANDARD.margeLateralePx}px`,
        gap: COTES_COLONNE_STANDARD.ecartBlocsPx,
      }}
    >
      {/* UNE SEULE ZONE QUI DÉFILE : retour, sélecteur, tête, actions et sections ; le pied n'y est pas. */}
      <div
        data-replay-fiche-scroll=""
        className="flex min-h-0 flex-1 flex-col overflow-y-auto"
        style={{ gap: COTES_FICHE_STANDARD.ecartBlocsPx, paddingRight: COTES_FICHE_STANDARD.margeBarrePx }}
      >
        <ColumnHead close={close} locale={body.locale} />
        {column.picker}
        <ReplayPlayerPanelBody {...body} scroll="outer" />
      </div>
      {column.footer(close)}
    </section>
  )
}
