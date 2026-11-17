/**
 * replayFrameSlots — CE QUE CHAQUE CADRE POSE DANS LES DEUX EMPLACEMENTS DE LA TOILE : sa barre de lecture
 * (`transport`) et les superpositions de son bloc (`blockOverlays`). Les fonctions sont définies ICI, au
 * niveau du module : leur identité ne change pas d'un rendu à l'autre, ni d'une bascule de cadre à l'autre
 * pour un même contenu.
 *
 * La table est fermée : un cadre ajouté doit dire ce qu'il pose. Aujourd'hui les trois cadres posent la
 * barre de lecture complète et la commande de zoom flottante, dans le coin bas-droit du bloc de la carte.
 */
import type { ReplayFrame } from '../../model/replayFrame'
import { ReplayTransport } from '../ReplayTransport'
import { ReplayZoomControl } from '../ReplayZoomControl'
import type { ReplayCanvasSlots } from '../replayTransportCommands'

const FULL_BAR_SLOTS: ReplayCanvasSlots = {
  transport: (commands) => <ReplayTransport {...commands} />,
  blockOverlays: ({ zoom, locale }) => <ReplayZoomControl zoom={zoom} locale={locale} />,
}

const SLOTS_BY_FRAME: Record<ReplayFrame, ReplayCanvasSlots> = {
  standard: FULL_BAR_SLOTS,
  cinema: FULL_BAR_SLOTS,
  'plein-cadre': FULL_BAR_SLOTS,
}

export function canvasSlotsOf(frame: ReplayFrame): ReplayCanvasSlots {
  return SLOTS_BY_FRAME[frame]
}
