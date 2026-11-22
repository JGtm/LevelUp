/**
 * usePlayerPanel — L'ÉTAT DU PANNEAU DE JOUEUR, côté React.
 *
 * Les règles (joueur choisi, ensemble des joueurs masqués, Seul, règles entre Suivre et Masquer) sont pures et
 * vivent dans `model/playerPanelLogic` ; ce fichier ne porte que l'ÉTAT et ses gestes.
 *
 * LE JOUEUR SUIVI n'est pas un état de ce hook : c'est le joueur regardé par la page, que la page
 * lui passe (`watchedXuid`) et qu'il ne fait que retrouver au roster (`tracked`). Suivre (la
 * caméra) est un état à part, `state.followXuid`, que la fiche ne tient pas. L'ENSEMBLE DES JOUEURS MASQUÉS
 * (`state.hidden`) est un troisième état, indépendant de la fiche et de la caméra.
 *
 * RIEN N'EST MÉMORISÉ (D-4) : ni dans le navigateur, ni d'un match à l'autre. La route garde le même
 * composant quand seul `$matchId` change, donc l'état est rangé AVEC la clé du match (`scopeKey`) et
 * REMIS À ZÉRO dès qu'elle diffère : un autre match ne rouvre jamais le panneau du précédent ni ne garde ses
 * joueurs masqués, même quand le même joueur y figure au roster, et revenir au match d'avant ne les ressuscite pas
 * non plus (lire « vide » sans effacer laisserait l'ancien état revenir avec l'ancienne clé).
 *
 * LES GESTES GARDENT LEUR IDENTITÉ : ils descendent en props jusqu'au canvas, aux fiches et aux tuiles.
 */
import { useCallback, useMemo, useState } from 'react'

import type { CameraFollow } from '../model/cameraMenu'
import {
  followedPlayerOf,
  hiddenPlayersOf,
  NO_PLAYER_PANEL,
  panelFocusOf,
  reducePlayerPanel,
  type HiddenPlayers,
  type PlayerFocus,
  type PlayerPanelEvent,
  type PlayerPanelState,
  trackedPlayerOf,
} from '../model/playerPanelLogic'
import type { PlayerTileActions } from '../model/playerTileActions'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

export interface PlayerPanel {
  state: PlayerPanelState
  /** Le joueur de la fiche ouverte, son « Seul » et son masquage ; `null` = fermée (ou joueur parti du roster). */
  focus: PlayerFocus | null
  /** Les joueurs masqués, retrouvés au roster, pour le canvas ; `null` = personne n'est masqué. */
  hiddenPlayers: HiddenPlayers | null
  /** Un clic sur la fiche d'un joueur (clé de roster). */
  onCard: (xuid: string) => void
  /** Le joueur suivi (le joueur regardé par la page) retrouvé au roster ; `null` = personne, ou absent du roster. */
  tracked: ReplayPlayer | null
  /** « Seul » sur ce joueur (clé de roster) : masque tous les autres, ou réaffiche tout s'il est déjà seul. */
  onSolo: (xuid: string) => void
  /** L'œil d'un joueur (clé de roster) : le masque, ou le réaffiche. */
  onToggleHidden: (xuid: string) => void
  /** « Tout afficher » : plus aucun joueur masqué. */
  onShowAll: () => void
  /** Suivre ce joueur (clé de roster), fiche ouverte ou non ; `null` = ne suivre personne. */
  onFollow: (xuid: string | null) => void
  /** Fermer la fiche (la croix de la fenêtre, « Équipes » de la colonne, Échap). */
  onClose: () => void
  /** Un geste manuel de zoom ou de déplacement sur la carte : Suivre s'arrête. */
  onFollowStop: () => void
  /**
   * La caméra, telle que la toile la lit : le joueur suivi (fiche ouverte ou non) et le geste « Suivre ce
   * joueur ». L'objet garde son identité tant que le joueur suivi et le roster ne changent pas.
   */
  camera: CameraFollow
  /** Ce que les tuiles et les lignes lisent : qui est suivi, qui est masqué, et leurs gestes. */
  tileActions: PlayerTileActions
}

interface Scoped {
  scopeKey: string
  state: PlayerPanelState
}

export function usePlayerPanel(
  scopeKey: string,
  players: readonly ReplayPlayer[],
  watchedXuid: string | null,
): PlayerPanel {
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
  // « Seul » masque les autres joueurs DU ROSTER : le geste rend les clés que la page tient à cet instant.
  const rosterKeys = useMemo(() => players.map((p) => p.xuid), [players])
  const onCard = useCallback((xuid: string) => dispatch({ type: 'card', xuid }), [dispatch])
  const onSolo = useCallback((xuid: string) => dispatch({ type: 'solo', xuid, roster: rosterKeys }), [dispatch, rosterKeys])
  const onToggleHidden = useCallback((xuid: string) => dispatch({ type: 'toggleHidden', xuid }), [dispatch])
  const onShowAll = useCallback(() => dispatch({ type: 'showAll' }), [dispatch])
  const onFollow = useCallback((xuid: string | null) => dispatch({ type: 'follow', xuid }), [dispatch])
  const onClose = useCallback(() => dispatch({ type: 'close' }), [dispatch])
  const onFollowStop = useCallback(() => dispatch({ type: 'stopFollow' }), [dispatch])

  const focus = useMemo(() => panelFocusOf(state, players), [state, players])
  const hiddenPlayers = useMemo(() => hiddenPlayersOf(state.hidden, players), [state.hidden, players])
  const tracked = useMemo(() => trackedPlayerOf(watchedXuid, players), [watchedXuid, players])
  const followed = useMemo(() => followedPlayerOf(state, players), [state, players])
  const camera = useMemo(() => ({ player: followed, onFollow }), [followed, onFollow])
  const tileActions = useMemo(
    () => ({ followedXuid: followed?.xuid ?? null, hidden: state.hidden, onFollow, onToggleHidden, onShowAll }),
    [followed, state.hidden, onFollow, onToggleHidden, onShowAll],
  )
  return {
    state, focus, hiddenPlayers, tracked, onCard, onSolo, onToggleHidden, onShowAll, onFollow, onClose, onFollowStop,
    camera, tileActions,
  }
}
