/**
 * skullFeedLogic.ts — LES LIGNES DU CRÂNE D'ODDBALL : « prend le crâne » au début de chaque portage,
 * « perd le crâne » à la fin d'un portage FERMÉ seulement.
 *
 * UN PORTAGE OUVERT N'A PAS DE LIGNE DE FIN (`closed === false`, le `carried_open` de la couche des
 * drapeaux et du crâne) : rien ne ferme le portage, sa fin n'est qu'une borne haute de l'axe, pas un
 * instant. Écrire « perd le crâne » à cet instant daterait un événement que le film n'a pas vu.
 *
 * LA GARDE D'HORLOGE EST CELLE DE TOUT LE REJEU (`filmClockTrusted`, comme `objectiveFeedLogic`) : sans
 * origine sûre, les portages sont décalés d'un écart inconnu — pas de ligne, muet vaut mieux que faux.
 * Ni le score qui s'égrène (points du crâne) ni la réapparition du crâne ne font de ligne.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { filmClockTrusted } from '../../../lib/replay/scoreTimeline'

import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'

/** Les lignes du crâne du document, triées par instant (à instant égal, la perte précède la prise). */
export function skullFeedEntries(doc: ReplayDocumentReady): ReplayFeedEntry[] {
  if (!filmClockTrusted(doc)) return []
  const entries: ReplayFeedEntry[] = []
  doc.skullCarries.forEach((carry, i) => {
    entries.push(
      eventFeedEntry(`sk-${i}-taken`, frameToMs(carry.t0, doc), {
        family: 'objectives',
        kind: 'skull',
        xuid: carry.xuid,
        change: 'taken',
      }),
    )
    if (!carry.closed) return
    entries.push(
      eventFeedEntry(`sk-${i}-lost`, frameToMs(carry.t1, doc), {
        family: 'objectives',
        kind: 'skull',
        xuid: carry.xuid,
        change: 'lost',
      }),
    )
  })
  return entries.sort((a, b) => a.replayMs - b.replayMs || lostFirst(a) - lostFirst(b))
}

/** 0 pour une perte, 1 pour le reste : à instant égal, le crâne change de main dans le bon sens. */
function lostFirst(entry: ReplayFeedEntry): number {
  const line = entry.event
  return line && 'change' in line && line.change === 'lost' ? 0 : 1
}
