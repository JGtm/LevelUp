/**
 * playerTileActions — CE QUE LA PAGE DONNE AUX TUILES ET AUX LIGNES pour agir sur UN joueur : Suivre (la caméra) et
 * l'œil (Masquer / Afficher), plus « Tout afficher » en tête de la colonne des équipes.
 *
 * `PlayerTileActions` est ce que la page tient (`usePlayerPanel`) et passe à la rangée et à la colonne ; chaque
 * tuile, chaque ligne n'en reçoit que la part de son joueur (`TileActions`, par `tileActionsFor`). Sans
 * `PlayerTileActions`, une tuile n'a aucun bouton de plus : la page décide qui peut agir.
 */

/** Ce que la page tient pour toutes les tuiles et lignes : qui est suivi, qui est masqué, et les trois gestes. */
export interface PlayerTileActions {
  /** Clé de roster du joueur que la caméra suit ; `null` = personne. */
  followedXuid: string | null
  /** Les clés de roster des joueurs masqués. */
  hidden: ReadonlySet<string>
  /** Suivre ce joueur (clé de roster) ; `null` = ne plus suivre. */
  onFollow: (xuid: string | null) => void
  /** Masquer ce joueur, ou l'afficher s'il est masqué. */
  onToggleHidden: (xuid: string) => void
  /** Réafficher tous les joueurs masqués. */
  onShowAll: () => void
}

/** Ce qu'UNE tuile ou UNE ligne reçoit : l'état de son joueur et ses deux gestes. */
export interface TileActions {
  followed: boolean
  hidden: boolean
  /** Suivre ce joueur, ou arrêter de le suivre s'il l'est. */
  onToggleFollow: () => void
  /** Masquer ce joueur, ou l'afficher s'il est masqué. */
  onToggleHidden: () => void
}

/** La part d'un joueur dans les actions de la page ; `undefined` quand la page n'en donne pas. */
export function tileActionsFor(actions: PlayerTileActions | undefined, xuid: string): TileActions | undefined {
  if (!actions) return undefined
  const followed = actions.followedXuid === xuid
  return {
    followed,
    hidden: actions.hidden.has(xuid),
    onToggleFollow: () => actions.onFollow(followed ? null : xuid),
    onToggleHidden: () => actions.onToggleHidden(xuid),
  }
}

/** Vrai quand la page donne des actions ET qu'au moins un joueur est masqué : « Tout afficher » se montre. */
export function showAllVisible(actions: PlayerTileActions | undefined): actions is PlayerTileActions {
  return actions !== undefined && actions.hidden.size > 0
}
