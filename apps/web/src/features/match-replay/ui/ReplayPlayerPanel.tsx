/**
 * ReplayPlayerPanel — LE PANNEAU DE JOUEUR DE LA COLONNE : ce qu'un clic sur une fiche ouvre, à la
 * PLACE du fil des événements (la carte n'est jamais couverte). « Retour au fil » le referme.
 *
 * Le panneau est le CONTENEUR : la section bornée en hauteur, son nom accessible, « Retour au fil »
 * posé à droite de la tête, et la sortie (Échap, retour). Tout le contenu — tête, trois actions,
 * bloc « À l'instant », listes de totaux de fin de match — est `ReplayPlayerPanelBody`, que d'autres
 * surfaces montent à l'identique.
 *
 * LE PANNEAU NE DÉTRUIT PAS LE FIL : la page garde le fil monté sous lui (masqué), si bien que le
 * filtre et le défilement sont retrouvés au retour. Il garde son joueur quand le temps avance.
 *
 * ÉCHAP LE FERME, AVANT LE MODE PLEIN ÉCRAN : le panneau est une couche (`data-replay-escape-layer`,
 * cf. `useReplayFullscreen`), et rend le focus à la fiche du joueur, qu'il quitte par Échap ou par
 * « Retour au fil ». Une couche plus intérieure ouverte (menu de vitesse, de filtre, panneau de la
 * vue) se ferme d'abord : Échap ne ferme qu'une couche à la fois.
 *
 * Aucun littéral de couleur : l'encre du camp et les neutres sont des jetons sémantiques.
 */
import { useCallback, useRef, type RefObject } from 'react'

import { Button } from '@/components/ui/button'
import { useDismissable } from '@/lib/hooks/useDismissable'

import { REPLAY_ESCAPE_LAYER_ATTR } from '../hooks/useReplayFullscreen'
import { REPLAY_TEXT } from '../i18n/i18n'
import { cardName } from '../model/playerCardReadings'
import { ReplayPlayerPanelBody, type ReplayPlayerPanelBodyProps } from './ReplayPlayerPanelBody'
import { PRESS_GUARDS } from './replayPressGuards'

export interface ReplayPlayerPanelProps extends Omit<ReplayPlayerPanelBodyProps, 'headEnd'> {
  onClose: () => void
}

/** La fiche du joueur dont le panneau est ouvert : la tuile-bouton choisie (`aria-pressed`). */
const OPEN_CARD_SELECTOR = '[data-replay-player-card][aria-pressed="true"]'

/**
 * usePanelClose — LA SORTIE DU PANNEAU : « Retour au fil » et Échap ferment par le même chemin, qui
 * rend le focus à la fiche (sans quoi il tomberait sur la page avec le panneau retiré). Échap ne
 * ferme que si aucune AUTRE couche d'Échap n'est ouverte : elle se ferme la première, par son
 * propre écouteur.
 */
function usePanelClose(rootRef: RefObject<HTMLElement | null>, onClose: () => void): () => void {
  const close = useCallback(() => {
    const card = document.querySelector<HTMLElement>(OPEN_CARD_SELECTOR)
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

export function ReplayPlayerPanel({ onClose, ...body }: ReplayPlayerPanelProps) {
  const t = REPLAY_TEXT[body.locale]
  const rootRef = useRef<HTMLElement>(null)
  const close = usePanelClose(rootRef, onClose)

  return (
    <section
      ref={rootRef}
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      aria-label={t.playerPanel.labelFmt(cardName(body.player, t))}
      className="flex h-full min-h-0 flex-col gap-2 overflow-hidden rounded-lg border border-border bg-card p-2"
    >
      <ReplayPlayerPanelBody
        {...body}
        headEnd={
          <Button type="button" variant="outline" size="sm" className="shrink-0" {...PRESS_GUARDS} onClick={close}>
            {t.playerPanel.back}
          </Button>
        }
      />
    </section>
  )
}
