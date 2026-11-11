/**
 * feedEventLines.ts — LES LIGNES DU FIL QUI NE SONT PAS DES ÉLIMINATIONS : objectifs, prises
 * d'armes et de bonus, montées en véhicule. Types seulement, plus l'assemblage.
 *
 * ELLES NE VIVENT PAS DANS `model.feed`. Le fil de la page (`buildFeedEntries` + présences) est
 * lu par la frise et par les effets de mort ; ces lignes ne servent QUE la liste affichée, que
 * `ReplayKillFeed` assemble par `withEventLines`. Une ligne d'événement est une `ReplayFeedEntry`
 * dont `kill`, `medal` et `death` sont nuls et dont `event` est posé : elle se trie et se coupe
 * à l'instant courant (`feedAt`) comme toutes les autres.
 */
import type { ReplayFeedEntry } from './killFeedLogic'
import type { ObjectiveFeedStat } from './objectiveFeedLogic'

/** Une action d'objectif nommée (liste fermée : cf. `objectiveFeedLogic`). */
export interface ObjectiveEventLine {
  family: 'objectives'
  xuid: string
  stat: ObjectiveFeedStat
}

/** Une prise d'arme ou de bonus sur un socle : le NOM est résolu par l'appelant. */
export interface PickupEventLine {
  family: 'pickups'
  xuid: string
  name: string
}

/** Une montée à bord d'un véhicule : le NOM de la famille est celui du document. */
export interface VehicleEventLine {
  family: 'vehicles'
  xuid: string
  name: string
}

export type FeedEventLine = ObjectiveEventLine | PickupEventLine | VehicleEventLine

/** Une ligne d'événement, au format commun du fil. */
export function eventFeedEntry(
  key: string,
  replayMs: number,
  event: FeedEventLine,
): ReplayFeedEntry {
  return { key, replayMs, kill: null, medal: null, death: null, event }
}

/**
 * withEventLines — le fil complet PLUS les lignes d'événement, trié. Rend une liste NEUVE : le
 * fil reçu n'est ni modifié ni recopié en place (D-1 du plan : `model.feed` reste entier).
 * Le tri est stable : à instant égal, la ligne du fil passe avant la ligne d'événement.
 */
export function withEventLines(
  feed: readonly ReplayFeedEntry[],
  events: readonly ReplayFeedEntry[],
): ReplayFeedEntry[] {
  if (events.length === 0) return [...feed]
  return [...feed, ...events].sort((a, b) => a.replayMs - b.replayMs)
}
