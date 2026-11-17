/**
 * replayFrameSlots — CE QUE CHAQUE CADRE POSE DANS LES DEUX EMPLACEMENTS DE LA TOILE : sa barre de lecture
 * (`transport`) et les superpositions de son bloc (`blockOverlays`). Les fonctions sont définies ICI, au
 * niveau du module : leur identité ne change pas d'un rendu à l'autre, ni d'une bascule de cadre à l'autre
 * pour un même contenu.
 *
 * DEUX JEUX D'EMPLACEMENTS, et le modèle décide lequel (`usesCompactBar`), jamais un nom de cadre écrit ici :
 *  - la barre complète et la commande de zoom flottante, dans le coin bas-droit du bloc de la carte (Standard) ;
 *  - la barre d'une ligne (Cinéma, Plein cadre), qui porte son propre zoom : aucune superposition de bloc, et le
 *    menu de la caméra à gauche du son (le Standard n'a pas ce menu : le suivi s'y commande depuis la fiche).
 * Un cadre ajouté dit sa barre dans le modèle, et reçoit le jeu d'emplacements qui va avec.
 */
import { usesCompactBar, type ReplayFrame } from '../../model/replayFrame'
import { ReplayCameraMenu } from '../ReplayCameraMenu'
import { ReplayTransport } from '../ReplayTransport'
import { ReplayTransportCompact } from '../ReplayTransportCompact'
import { ReplayZoomControl } from '../ReplayZoomControl'
import type { ReplayCanvasSlots } from '../replayTransportCommands'

const FULL_BAR_SLOTS: ReplayCanvasSlots = {
  transport: (commands) => <ReplayTransport {...commands} />,
  blockOverlays: ({ zoom, locale }) => <ReplayZoomControl zoom={zoom} locale={locale} />,
}

const COMPACT_BAR_SLOTS: ReplayCanvasSlots = {
  transport: (commands) => (
    <ReplayTransportCompact
      {...commands}
      camera={<ReplayCameraMenu timeline={commands.timeline} cameraFollow={commands.cameraFollow} locale={commands.locale} />}
    />
  ),
}

export function canvasSlotsOf(frame: ReplayFrame): ReplayCanvasSlots {
  return usesCompactBar(frame) ? COMPACT_BAR_SLOTS : FULL_BAR_SLOTS
}
