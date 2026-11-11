/**
 * vehicleFeedLogic.ts — « VÉHICULES » : les montées à bord, tirées des épisodes d'occupation
 * `doc.vehicles[].rides[]`. Logique pure.
 *
 * CE QUE LE DOCUMENT PERMET, ET CE QU'IL NE PERMET PAS. Un épisode porte son occupant (`xuid`),
 * un début daté (`t0`, `src` = `event` ou `mixed` : à la milliseconde ; `gap` : borné à la
 * demi-seconde du flux, non daté) et un siège. Il ne porte AUCUNE destruction (`end` vaut
 * toujours `unknown`) : le fil ne dit donc que des montées, jamais « a détruit ». Et l'attribution
 * n'est pas exhaustive — la primitive d'occupation ne nomme que 15,6 à 21,1 % des vies de
 * véhicule (cf. `ReplayVehicleRide` côté Go) : cette liste est FIDÈLE, pas COMPLÈTE.
 *
 * UNE LIGNE = UNE MONTÉE : occupant nommé, début daté, famille résolue. Deux épisodes du même
 * joueur dans le même véhicule qui se chevauchent (conducteur et passager ne se départagent pas
 * toujours par la géométrie — l'ambiguïté est publiée) ne font qu'une ligne. Le décor, les
 * éléments de carte et les vies déclarées décor par le serveur ne sont pas des véhicules.
 *
 * LE NOM EST CELUI DU DOCUMENT : `vehicleLabels[famille]` dans la langue du lecteur quand il
 * existe, sinon la clé de famille, qui EST le nom propre du jeu (Warthog, Ghost). Une famille
 * vide (châssis non résolu) ne fait aucune ligne : aucun nom n'est inventé.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { pickText } from '@/lib/i18n/pickText'
import type { Locale } from '@/lib/i18n/locale'

import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'
import { vehicleIsDecor, vehicleIsMapElement } from './vehiclesLayer'

/** L'épisode d'occupation, tel que le document le publie. */
type Ride = ReplayDocumentReady['vehicles'][number]['rides'][number]

/** La source d'un début NON daté : le seul trou de position, borné à la demi-seconde du flux. */
const UNDATED_SRC = 'gap'

export function vehicleFeedEntries(doc: ReplayDocumentReady, locale: Locale): ReplayFeedEntry[] {
  const entries: ReplayFeedEntry[] = []
  doc.vehicles.forEach((track, ti) => {
    const family = track.family
    if (!family || vehicleIsDecor(family) || track.scenery) return
    const label = doc.vehicleLabels?.[family]
    if (vehicleIsMapElement(label?.kind)) return
    const name = pickText(label, locale) || family
    for (const ride of firstBoardings(track.rides)) {
      entries.push(
        eventFeedEntry(`v-${ride.xuid}-${ti}-${ride.t0}`, frameToMs(ride.t0, doc), {
          family: 'vehicles',
          xuid: ride.xuid as string,
          name,
        }),
      )
    }
  })
  entries.sort((a, b) => a.replayMs - b.replayMs)
  return entries
}

/**
 * firstBoardings — pour UN véhicule, l'épisode de début de chaque montée : nommé, daté, et pas
 * à l'intérieur d'une montée du même joueur déjà comptée (un chevauchement n'est pas une seconde
 * montée).
 */
function firstBoardings(rides: readonly Ride[]): Ride[] {
  const sorted = rides
    .filter((r) => !!r.xuid && r.src !== UNDATED_SRC)
    .sort((a, b) => a.t0 - b.t0)
  const lastEnd = new Map<string, number>()
  const out: Ride[] = []
  for (const r of sorted) {
    const xuid = r.xuid as string
    const end = lastEnd.get(xuid)
    if (end !== undefined && r.t0 <= end) {
      lastEnd.set(xuid, Math.max(end, r.t1))
      continue
    }
    lastEnd.set(xuid, r.t1)
    out.push(r)
  }
  return out
}
