/**
 * hillFeedLogic.ts — LA LIGNE « PREND LA COLLINE » DU FIL : une par CHANGEMENT DE TENANT de la
 * colline, lu dans les intervalles de propriété que l'artefact publie (`zoneStates[].spans`).
 *
 * UNE LIGNE = UN CAMP QUI PREND LA COLLINE À UN AUTRE. Le tenant d'une colline est le propriétaire
 * de son intervalle ; `owner` nul dit « personne ne la tient » (neutre ou disputée : le document ne
 * distingue pas les deux). Passer à ce neutre ne fait pas de ligne, et le même camp qui reprend la
 * colline après une dispute n'en fait pas non plus : le tenant n'a pas changé. Seule la première
 * prise d'une colline, et le passage d'un camp à l'autre, s'écrivent.
 *
 * LE TENANT PRÉCÉDENT NE SURVIT PAS À LA COLLINE. Une colline tourne : quand un nouvel intervalle ne
 * suit pas le précédent (`t0` au-delà de l'image qui suit son `t1`), c'est une autre activation, et
 * son premier tenant est une prise même si c'est le camp qui tenait la précédente.
 *
 * UNE COLLINE SE RECONNAÎT À SES INTERVALLES `active` (`isHillState`, la lecture du calque des
 * zones) : le producteur ne pose `active` que sur la colline du moment. Les bases des modes à zones
 * n'en portent aucun et ne passent pas ici ; elles gardent leurs lignes `zone_captures`.
 *
 * LA TENUE N'EST PAS LE SCORE : les tics de garde qui rapprochent du point ne font pas de ligne.
 *
 * LA GARDE D'HORLOGE EST CELLE DU CALQUE DE SCORE (`filmClockTrusted`) : sans origine sûre, les
 * intervalles sont décalés d'un écart inconnu, donc aucune ligne.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { filmClockTrusted } from '../../../lib/replay/scoreTimeline'

import { isHillState } from '../layers/zoneStatesLayer'
import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'

type DocumentZoneState = ReplayDocumentReady['zoneStates'][number]

/** Une prise : la colline (index de zone), le camp (index du registre) et l'image où il la tient. */
interface HillTake {
  zoneRef: number
  teamId: number
  frame: number
}

/** Les prises d'une colline : chaque intervalle dont le propriétaire diffère du tenant précédent. */
function hillTakes(state: DocumentZoneState): HillTake[] {
  const takes: HillTake[] = []
  let tenant: number | null = null
  let previousEnd: number | null = null
  for (const span of state.spans) {
    // Les bornes sont incluses : un intervalle qui prolonge le précédent commence à l'image d'après.
    if (previousEnd === null || span.t0 > previousEnd + 1) tenant = null
    previousEnd = span.t1
    if (span.owner == null || span.owner === tenant) continue
    tenant = span.owner
    takes.push({ zoneRef: state.zoneRef, teamId: span.owner, frame: span.t0 })
  }
  return takes
}

/** Les prises de colline du document, à leur instant sur l'axe du rejeu, triées. */
export function hillFeedEntries(doc: ReplayDocumentReady): ReplayFeedEntry[] {
  if (!filmClockTrusted(doc)) return []
  return doc.zoneStates
    .filter(isHillState)
    .flatMap(hillTakes)
    .sort((a, b) => a.frame - b.frame)
    .map((take) =>
      eventFeedEntry(`h-${take.zoneRef}-${take.frame}`, frameToMs(take.frame, doc), {
        family: 'objectives',
        kind: 'hill',
        teamId: take.teamId,
      }),
    )
}
