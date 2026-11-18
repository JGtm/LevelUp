/**
 * replayFrame.ts — LES TROIS CADRES DE LA PAGE DU REJEU et leurs transitions. Logique pure, sans
 * React, sans DOM, sans lecture de touche (replayFrame.test.ts).
 *
 * TROIS MANIÈRES DE REGARDER LE MÊME LECTEUR :
 *  - `standard`    : la page à largeur bornée, dans l'application ;
 *  - `cinema`      : le lecteur sur la largeur du corps de l'application, navigation comprise ;
 *  - `plein-cadre` : le lecteur recouvre la fenêtre du navigateur, sans la navigation de
 *    l'application et SANS plein écran système.
 * Le plein écran du système n'est pas un cadre : il s'ajoute à n'importe lequel et vit hors de ce
 * module.
 *
 * UNE MANIÈRE, PAS UN ENDROIT. L'état naît `standard` à chaque montage de la page ; rien n'est
 * mémorisé ni porté par l'URL.
 *
 * DES ÉVÉNEMENTS DÉJÀ NOMMÉS. Le module ne lit aucune touche : l'appelant traduit la frappe ou le
 * clic en `ReplayFrameEvent`. C'est lui, et non ce module, qui décide quelle couche ouverte
 * (fiche, menu, tiroir) reçoit d'abord la sortie.
 *
 * PLEIN CADRE SE SOUVIENT D'OÙ L'ON VIENT. Le quitter (bascule Plein cadre, sortie) rend le cadre
 * d'avant, `standard` ou `cinema`. La bascule Cinéma, elle, mène toujours à `cinema` depuis le
 * Plein cadre. La sortie n'a aucun effet en `cinema` ni en `standard` : on n'en sort que par la
 * bascule.
 *
 * PENDANT UN EXPORT VIDÉO, AUCUNE BASCULE. Le clip se dessine à son propre format : changer de
 * cadre en cours d'export changerait la toile qui le produit. Les deux bascules sont ignorées.
 * La sortie du Plein cadre reste admise : elle ne touche pas le clip, comme aujourd'hui.
 */

/** Les trois cadres de la page. */
export type ReplayFrame = 'standard' | 'cinema' | 'plein-cadre'

/** Un cadre d'où l'on peut entrer en Plein cadre, donc où l'on peut revenir. */
export type ReplayFrameOrigin = Exclude<ReplayFrame, 'plein-cadre'>

/**
 * L'état : le cadre affiché et le cadre où le Plein cadre ramène. Hors Plein cadre, `returnTo` est
 * le cadre lui-même, de sorte qu'un même cadre n'a qu'une seule représentation.
 */
export interface ReplayFrameState {
  readonly frame: ReplayFrame
  readonly returnTo: ReplayFrameOrigin
}

/** L'état de départ de chaque montage de la page. */
export const INITIAL_REPLAY_FRAME: ReplayFrameState = { frame: 'standard', returnTo: 'standard' }

/** Les événements nommés que la page ou les raccourcis envoient. */
export type ReplayFrameEvent =
  /** Touche T ou icône Cinéma : `standard` et `cinema` s'échangent ; du Plein cadre, mène à `cinema`. */
  | { type: 'toggleCinema' }
  /** Touche F ou icône Plein cadre : entre en Plein cadre, ou rend le cadre d'où l'on vient. */
  | { type: 'toggleFullFrame' }
  /** Échap ou « Quitter » : ne fait quelque chose qu'en Plein cadre, où il rend le cadre d'où l'on vient. */
  | { type: 'exit' }

/** Ce que le réducteur doit savoir du dehors. */
export interface ReplayFrameContext {
  /** Un export vidéo tient la toile : les bascules sont ignorées. */
  readonly exportActive: boolean
}

/** L'état d'un cadre hors Plein cadre : `returnTo` est le cadre lui-même. */
function settledAt(frame: ReplayFrameOrigin): ReplayFrameState {
  return frame === 'standard' ? INITIAL_REPLAY_FRAME : { frame, returnTo: frame }
}

/** Le Plein cadre entré depuis un cadre. */
function fullFrameFrom(origin: ReplayFrameOrigin): ReplayFrameState {
  return { frame: 'plein-cadre', returnTo: origin }
}

/**
 * reduceReplayFrame applique un événement à l'état. Un événement sans effet (sortie hors Plein
 * cadre, bascule pendant un export) rend le MÊME objet, pour ne réveiller personne.
 */
export function reduceReplayFrame(
  state: ReplayFrameState,
  event: ReplayFrameEvent,
  context: ReplayFrameContext,
): ReplayFrameState {
  switch (event.type) {
    case 'toggleCinema':
      if (context.exportActive) return state
      return settledAt(state.frame === 'cinema' ? 'standard' : 'cinema')
    case 'toggleFullFrame':
      if (context.exportActive) return state
      return state.frame === 'plein-cadre' ? settledAt(state.returnTo) : fullFrameFrom(state.frame)
    case 'exit':
      return state.frame === 'plein-cadre' ? settledAt(state.returnTo) : state
  }
}

/**
 * Ce cadre montre-t-il la navigation de l'application (en-tête, fil d'Ariane, languettes) ?
 * Seul le Plein cadre la recouvre.
 */
export function showsAppNavigation(frame: ReplayFrame): boolean {
  return frame !== 'plein-cadre'
}

/**
 * Ce cadre utilise-t-il la barre de lecture d'une ligne ? Le Standard garde la zone de lecture
 * d'aujourd'hui ; le Cinéma et le Plein cadre prennent la barre compacte.
 */
export function usesCompactBar(frame: ReplayFrame): boolean {
  return frame !== 'standard'
}

/**
 * Ce cadre peut-il poser la rangée de fiches (un camp, le score, l'autre camp) au-dessus du plan ?
 * Le Cinéma et le Plein cadre, qui partagent la barre compacte ; le Standard a sa colonne
 * d'équipes. Que la rangée TIENNE (deux camps, places, largeur) est une autre question (`rowLayout`).
 */
export function canPlaceCardRow(frame: ReplayFrame): boolean {
  return frame !== 'standard'
}
