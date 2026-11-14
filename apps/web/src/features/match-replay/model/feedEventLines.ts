/**
 * feedEventLines.ts — LES LIGNES DU FIL QUI NE SONT PAS DES ÉLIMINATIONS : objectifs, prises
 * d'armes et de bonus (socle ou sol), montées et destructions de véhicule. Types seulement, plus
 * l'assemblage.
 *
 * ELLES NE VIVENT PAS DANS `model.feed`. Le fil de la page (`buildFeedEntries` + présences) est
 * lu par la frise et par les effets de mort ; ces lignes ne servent QUE la liste affichée, que
 * `ReplayKillFeed` assemble par `withEventLines`. Une ligne d'événement est une `ReplayFeedEntry`
 * dont `kill`, `medal` et `death` sont nuls et dont `event` est posé : elle se trie et se coupe
 * à l'instant courant (`feedAt`) comme toutes les autres.
 */
import type { ReplayFeedEntry } from './killFeedLogic'
import type { ObjectiveFeedStat } from './objectiveFeedLogic'
import type { PadFamily } from './weaponPadFamilies'

/** Une action d'objectif nommée (liste fermée : cf. `objectiveFeedLogic`). */
export interface ObjectiveEventLine {
  family: 'objectives'
  xuid: string
  stat: ObjectiveFeedStat
}

/**
 * Une prise d'arme ou de bonus sur un socle, ou (`ground`) une arme ramassée au sol : le NOM est
 * résolu par l'appelant.
 */
export interface PickupEventLine {
  family: 'pickups'
  xuid: string
  name: string
  /**
   * La nature de ce qui est pris, celle qui teinte les socles de la carte (`padFamilyOf`) : arme de
   * râtelier, arme spéciale ou équipement. Elle choisit le pictogramme de la ligne.
   */
  nature: PadFamily
  /** Vrai pour une arme ramassée au sol ; absent pour une prise sur socle. */
  ground?: true
}

/** Une montée à bord d'un véhicule : le NOM de la famille est celui du document. */
export interface VehicleEventLine {
  family: 'vehicles'
  xuid: string
  name: string
}

/** Le pictogramme d'une arme, tel que le fil des éliminations le sert (masque à teindre ou image). */
export interface FeedWeaponIcon {
  imageUrl: string
  tinted: boolean
  /** Le nom propre de l'arme ; vide quand le titre ne le nomme pas. */
  label: string
}

/**
 * Une DESTRUCTION de véhicule. L'auteur et les joueurs tués avec le véhicule sont facultatifs : un destructeur
 * humain porte son xuid (`xuid`), un destructeur bot son nom de roster (`botName`), et sans l'un ni
 * l'autre la ligne dit seulement que le véhicule est détruit ; sans joueur tué elle n'en nomme
 * aucun. Le NOM est celui de la famille, comme pour une montée. `weapon` est le pictogramme de
 * l'arme qui a détruit le véhicule ; absent quand la cause n'est pas résolue ou que le référentiel
 * du titre n'a aucun dessin pour elle : la ligne ne porte alors aucun pictogramme.
 */
export interface VehicleDestroyedEventLine {
  family: 'vehicles'
  destroyed: true
  xuid: string | null
  botName: string | null
  killedXuids: readonly string[]
  name: string
  weapon?: FeedWeaponIcon
}

export type FeedEventLine =
  | ObjectiveEventLine
  | PickupEventLine
  | VehicleEventLine
  | VehicleDestroyedEventLine

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
