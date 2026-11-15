/**
 * usePlayerPanel — L'ÉTAT DU PANNEAU DE JOUEUR, côté React (lot 4 du plan « maquettes du rejeu »).
 *
 * Les règles (joueur choisi, exclusions de Suivre / Seul / Masquer, remise à zéro) sont pures et
 * vivent dans `model/playerPanelLogic` ; ce fichier ne porte que l'ÉTAT et ses gestes.
 *
 * RIEN N'EST MÉMORISÉ (D-4) : ni dans le navigateur, ni d'un match à l'autre. La route garde le même
 * composant quand seul `$matchId` change, donc l'état est rangé AVEC la clé du match (`scopeKey`) et
 * REMIS À ZÉRO dès qu'elle diffère : un autre match ne rouvre jamais le panneau du précédent, même
 * quand le même joueur y figure au roster, et revenir au match d'avant ne le ressuscite pas non
 * plus (lire « vide » sans effacer laisserait l'ancien état revenir avec l'ancienne clé).
 *
 * LES GESTES GARDENT LEUR IDENTITÉ : ils descendent en props jusqu'au canvas et aux fiches.
 */
import { useCallback, useMemo, useState } from 'react'

import {
  NO_PLAYER_PANEL,
  panelFocusOf,
  reducePlayerPanel,
  type PlayerFocus,
  type PlayerPanelAction,
  type PlayerPanelEvent,
  type PlayerPanelState,
} from '../model/playerPanelLogic'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

export interface PlayerPanel {
  state: PlayerPanelState
  /** Le joueur du panneau ouvert et ses actions, pour le canvas ; `null` = fermé (ou joueur parti du roster). */
  focus: PlayerFocus | null
  /** Un clic sur la fiche d'un joueur (clé de roster). */
  onCard: (xuid: string) => void
  /** Une bascule de Suivre, Seul ou Masquer. */
  onToggle: (action: PlayerPanelAction) => void
  /** « Retour au fil ». */
  onClose: () => void
  /** Un geste manuel de zoom ou de déplacement sur la carte : Suivre s'arrête. */
  onFollowStop: () => void
}

interface Scoped {
  scopeKey: string
  state: PlayerPanelState
}

export function usePlayerPanel(scopeKey: string, players: readonly ReplayPlayer[]): PlayerPanel {
  const [stored, setStored] = useState<Scoped>({ scopeKey, state: NO_PLAYER_PANEL })
  // L'effacement se fait AU RENDU, pas dans un effet : aucune image ne montre l'état d'un autre match.
  if (stored.scopeKey !== scopeKey) setStored({ scopeKey, state: NO_PLAYER_PANEL })
  const state = stored.scopeKey === scopeKey ? stored.state : NO_PLAYER_PANEL

  const dispatch = useCallback(
    (event: PlayerPanelEvent) =>
      setStored((prev) => {
        const current = prev.scopeKey === scopeKey ? prev.state : NO_PLAYER_PANEL
        const next = reducePlayerPanel(current, event)
        return next === prev.state && prev.scopeKey === scopeKey ? prev : { scopeKey, state: next }
      }),
    [scopeKey],
  )
  const onCard = useCallback((xuid: string) => dispatch({ type: 'card', xuid }), [dispatch])
  const onToggle = useCallback((action: PlayerPanelAction) => dispatch({ type: 'toggle', action }), [dispatch])
  const onClose = useCallback(() => dispatch({ type: 'close' }), [dispatch])
  const onFollowStop = useCallback(() => dispatch({ type: 'stopFollow' }), [dispatch])

  const focus = useMemo(() => panelFocusOf(state, players), [state, players])
  return { state, focus, onCard, onToggle, onClose, onFollowStop }
}
