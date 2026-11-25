/**
 * chatterFeedLogic.ts — LES LIGNES DU FIL QUI SONT DES RÉPLIQUES : une ligne par réplique affichable, au format commun du fil
 * (`eventFeedEntry`). Elles n'entrent que dans la liste AFFICHÉE (`withEventLines`), comme les objectifs et les prises :
 * `model.feed` n'en porte aucune, la frise et les effets de mort ne les voient pas.
 *
 * L'instant est celui de la réplique sur l'axe du film (`frameToMs`), le même que les tirs et les prises ; la ligne se coupe à
 * l'instant lu (`feedAt`) comme toutes les autres.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

import type { ChatterLine } from './chatterLines'
import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'

/** Les lignes du fil des répliques données (déjà filtrées par les familles, la fréquence et le réglage du fil). */
export function chatterFeedEntries(
  lines: readonly ChatterLine[],
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
): ReplayFeedEntry[] {
  return lines.map((l) =>
    eventFeedEntry(l.key, frameToMs(l.frame, doc), {
      family: 'replies',
      xuid: l.xuid,
      text: l.text,
      voice: l.voice !== null,
      chatterFamily: l.family,
    }),
  )
}
