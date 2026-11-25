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
import { foldCarrierKills } from './carrierKillFold'
import type { ReplayFeedEntry } from './killFeedLogic'
import type { ChatterFamily } from './chatterLines'
import type { ObjectiveFeedStat } from './objectiveFeedLogic'
import type { PadFamily } from './weaponPadFamilies'

/**
 * Une action d'objectif nommée (liste fermée : cf. `objectiveFeedLogic`). `assistXuids` : sur une
 * capture, les joueurs crédités d'une assistance que le film date sur cette capture ; absent quand
 * aucune ne s'y rattache.
 */
export interface ObjectiveEventLine {
  family: 'objectives'
  xuid: string
  stat: ObjectiveFeedStat
  assistXuids?: readonly string[]
}

/**
 * La FIN D'UNE MANCHE d'un match à plusieurs manches : le score de manche de chaque camp, ni le
 * cumul ni le meneur du match. `index` est le rang d'affichage (1-based) ; les camps sont ceux du film.
 */
export interface RoundEndEventLine {
  family: 'objectives'
  kind: 'roundEnd'
  index: number
  scores: ReadonlyArray<{ teamId: number; score: number }>
}

/** Un joueur prend le crâne (début d'un portage) ou le perd (fin d'un portage FERMÉ seulement). */
export interface SkullEventLine {
  family: 'objectives'
  kind: 'skull'
  xuid: string
  change: 'taken' | 'lost'
}

/** Un camp (index du registre) ouvre une tenue de la colline. */
export interface HillEventLine {
  family: 'objectives'
  kind: 'hill'
  teamId: number
}

/** Les lignes d'objectif qui ne sont pas une action nommée du joueur : manche, crâne, colline. */
export type ObjectiveStateLine = RoundEndEventLine | SkullEventLine | HillEventLine

/**
 * La nature de ce qui est pris, pour le pictogramme de la ligne : celle des socles de la carte
 * (`PadFamily`, qui ne change pas) et un quatrième cas propre au fil, l'ÉQUIPEMENT QUI CHANGE LA PARTIE
 * (surbouclier, camouflage : les équipements nommés par `POWER_PAD_KEYS`). Un équipement hors de cette liste
 * reste un équipement ordinaire.
 */
export type PickupNature = PadFamily | 'powerupGameChanger'

/**
 * Une prise d'arme ou de bonus sur un socle, ou (`ground`) une arme ramassée au sol : le NOM est
 * résolu par l'appelant.
 */
export interface PickupEventLine {
  family: 'pickups'
  xuid: string
  name: string
  /**
   * La nature de ce qui est pris : arme de râtelier, arme spéciale, équipement, ou équipement qui change la
   * partie. Elle choisit le pictogramme de la ligne.
   */
  nature: PickupNature
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

/**
 * Une RÉPLIQUE d'un Spartan : le joueur qui parle et ce qu'il dit, dans la langue de l'interface. `voice` dit si la réplique a
 * une voix dans cette langue (le pictogramme ne promet pas un son qui n'existe pas) ; `chatterFamily` sert la mise en forme.
 */
export interface ChatterEventLine {
  family: 'replies'
  xuid: string
  text: string
  voice: boolean
  chatterFamily: ChatterFamily
}

export type FeedEventLine =
  | ObjectiveEventLine
  | ObjectiveStateLine
  | PickupEventLine
  | VehicleEventLine
  | VehicleDestroyedEventLine
  | ChatterEventLine

/** Vrai pour une ligne d'objectif de manche, de crâne ou de colline (celles qui ne portent pas `stat`). */
export function isObjectiveStateLine(line: FeedEventLine): line is ObjectiveStateLine {
  return line.family === 'objectives' && 'kind' in line
}

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
 *
 * LES ACTIONS « PORTEUR ÉLIMINÉ » SE REPLIENT SUR LEUR FRAG (`foldCarrierKills`) : la ligne
 * d'événement disparaît et la copie du frag, dans la liste rendue, porte `flagCarrierKill`.
 */
export function withEventLines(
  feed: readonly ReplayFeedEntry[],
  events: readonly ReplayFeedEntry[],
): ReplayFeedEntry[] {
  if (events.length === 0) return [...feed]
  const folded = foldCarrierKills(feed, events)
  return [...folded.feed, ...folded.events].sort((a, b) => a.replayMs - b.replayMs)
}
