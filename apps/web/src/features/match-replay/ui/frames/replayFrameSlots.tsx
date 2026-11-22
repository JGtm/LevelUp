/**
 * replayFrameSlots — CE QUE CHAQUE CADRE POSE DANS LES DEUX EMPLACEMENTS DE LA TOILE : sa barre de lecture
 * (`transport`) et les superpositions de son bloc (`blockOverlays`). Les fonctions sont définies ICI, au
 * niveau du module : leur identité ne change pas d'un rendu à l'autre, ni d'une bascule de cadre à l'autre
 * pour un même contenu.
 *
 * DEUX JEUX D'EMPLACEMENTS, et le modèle décide lequel (`usesCompactBar`), jamais un nom de cadre écrit ici :
 *  - la barre complète et la commande de zoom flottante, dans le coin bas-droit du bloc de la carte (Standard) ;
 *  - la barre d'une ligne (Cinéma, Plein cadre), qui porte son propre zoom : aucune superposition de bloc, sauf
 *    le fil des éliminations quand la rangée de fiches est posée (`planFeedView`), et le menu de la caméra à
 *    gauche du son (le Standard n'a pas ce menu : le suivi s'y commande depuis la fiche).
 * Un cadre ajouté dit sa barre dans le modèle, et reçoit le jeu d'emplacements qui va avec.
 *
 * LE FIL DU PLAN EST LE SEUL EMPLACEMENT QUI LIT LE REJEU : sa fonction de rendu est une fermeture sur la vue
 * de la page (le fil, l'instant lu, les noms), refaite à chaque rendu de la page, qui publie l'instant lu. Elle
 * ne porte aucun état et n'appelle aucun hook ; le composant qu'elle rend (`ReplayFeedOverlay`) est le même à
 * chaque rendu, donc jamais remonté. La barre et la commande de zoom, elles, restent définies au niveau du module.
 */
import { COTES_BARRE_COMPACTE as C } from '../../model/cotesBarreCompacte'
import { playerKeys } from '../../model/feedFollowed'
import { usesCompactBar, type ReplayFrame } from '../../model/replayFrame'
import { overlaysTransportBar } from '../../model/transportBarLogic'
import { ReplayCameraMenu } from '../ReplayCameraMenu'
import { ReplayFeedOverlay } from '../ReplayFeedOverlay'
import { ReplayTransport } from '../ReplayTransport'
import { ReplayTransportCompact } from '../ReplayTransportCompact'
import { ReplayZoomControl } from '../ReplayZoomControl'
import type { ReplayCanvasSlots } from '../replayTransportCommands'
import type { ReplayFrameView } from './replayFrameTypes'

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

/**
 * canvasSlotsOf rend les emplacements du cadre. `planFeedView` : la vue que le fil des éliminations du plan lit,
 * ou `null` quand il n'est pas posé (`planFeedPosed`) ; seule la barre d'une ligne le porte, la barre complète
 * du Standard garde sa colonne.
 */
export function canvasSlotsOf(frame: ReplayFrame, planFeedView: ReplayFrameView | null = null): ReplayCanvasSlots {
  if (!usesCompactBar(frame)) return FULL_BAR_SLOTS
  if (!planFeedView) return COMPACT_BAR_SLOTS
  const { model, nowMs, locale, feedZoneOf } = planFeedView
  return {
    ...COMPACT_BAR_SLOTS,
    planFeed: true,
    blockOverlays: ({ planFeedShown, frameControl, thinBar }) =>
      planFeedShown && (
        <ReplayFeedOverlay
          entries={model.feed}
          nowMs={nowMs}
          playWindow={model.window}
          xuidMeta={model.identity}
          allegiance={model.allegiance}
          marks={model.marks}
          locale={locale}
          zoneOf={feedZoneOf}
          followedKeys={playerKeys(planFeedView.playerPanel.camera.player)}
          liftPx={overlaysTransportBar(frameControl.frame, thinBar) ? C.barreHauteurPx : 0}
        />
      ),
  }
}
