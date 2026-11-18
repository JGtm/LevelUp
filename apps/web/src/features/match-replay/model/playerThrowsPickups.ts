/**
 * playerThrowsPickups.ts — CE QU'UN JOUEUR A LANCÉ ET RAMASSÉ (ARMES, GRENADES) DU DÉBUT DU MATCH
 * JUSQU'À L'IMAGE LUE, sans React.
 *
 * DEUX TEMPS, comme `playerInstantStats.ts`. `readPlayerThrowsPickupsSeries` lit une fois par joueur et
 * par document ce qui est daté, le range par type en séries triées ; `playerThrowsPickupsAt` répond pour
 * une image par dichotomie, sans rien relire du document.
 *
 * LES SOURCES, TELLES QUE LE DOCUMENT LES PUBLIE :
 *  - LANCERS : `doc.grenades`. Un lancer porte l'index de joueur DU FILM (`i`), jamais le slot : la
 *    jointure passe par `doc.roster` (`filmIndex`, puis `rosterEntryKey`), comme `equipmentUsageLogic`.
 *    Le type est le RANG du catalogue `grenadeLabels`.
 *  - RAMASSAGES : `doc.pickups`, l'événement natif daté à l'image et attribué à son ramasseur. Le genre
 *    (`kind`) donne la nature : `weapon` pour une arme, `grenade` pour une grenade (classe 2, schéma 31).
 *    Un ramassage porte un `xuid` quand le document le nomme ; sinon il est rattaché au joueur qui
 *    OCCUPAIT son slot à l'instant (`buildSlotOwnership`), ce qui couvre les bots.
 *
 * UN DÉCOMPTE EST UNE BORNE INFÉRIEURE pour les ramassages : le balayage natif ne voit que les événements
 * en tête de leur liste (`coverage.pickups.multiEvent`). Les lancers, eux, sont comptés tels que publiés.
 *
 * LA RÈGLE DES FICHES VAUT ICI : une grandeur que le document ne publie pas pour cet artefact est ABSENTE
 * (`null`), jamais un zéro. Le sens de chaque absence est écrit sur le champ de `PlayerThrowsPickups`.
 *
 * Les comptes ne reculent jamais d'une image à l'autre : chaque série est cumulative. Un joueur mort garde
 * ce qu'il a lancé et ramassé. Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { lastIndexAt } from '../../../lib/replay/replayLogic'
import { buildSlotOwnership, rosterEntryKey, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { weaponLabelKeyOf } from '../layers/useReplayWeaponPads'

/** Premier artefact dont `doc.pickups` porte les ramassages natifs (décodeur : « schéma 30 »). */
export const PICKUPS_SCHEMA = 30

/**
 * Premier artefact dont la classe 2 se publie `kind: 'grenade'` (décodeur : « depuis le schéma 31 les
 * classes 2 et 3 ont leur valeur propre » ; avant, une grenade se lit `item`, indistinguable d'autre chose).
 */
export const GRENADE_PICKUP_KIND_SCHEMA = 31

/** Lancers d'un type de grenade : `rank` est l'index de `doc.grenadeLabels` (l'identifiant du document). */
export interface GrenadeThrowCount {
  rank: number
  count: number
}

/** Ramassages d'une arme : `key` est la clé de `doc.weaponLabels` (`weaponLabelKeyOf`). */
export interface WeaponPickupCount {
  key: string
  count: number
}

/**
 * Ramassages d'une grenade : `family` est le slug d'objet que le document publie (`grenade_frag`...), et
 * `null` quand aucun catalogue du titre ne le nomme (le document ne le publie alors pas).
 */
export interface GrenadePickupCount {
  family: string | null
  count: number
}

/** Ce que la lecture rend pour une image : tout est cumulé depuis le début du match. */
export interface PlayerThrowsPickups {
  /**
   * Lancers par type, rangs croissants, types sans lancer omis. `[]` = aucun lancer jusqu'à l'image : le
   * document publie chaque lancer, donc ce n'est pas une absence.
   */
  grenadesThrown: GrenadeThrowCount[]
  /**
   * Ramassages d'armes. `null` = artefact antérieur au schéma 30 (aucun ramassage natif publié). `[]` = aucun
   * ramassage d'arme jusqu'à l'image, sur un artefact qui les publie.
   */
  weaponPickups: WeaponPickupCount[] | null
  /**
   * Ramassages de grenades. `null` = artefact antérieur au schéma 31 : la classe 2 n'y porte pas son nom, la
   * grenade se lit `item` et ne se distingue pas d'un autre objet. `[]` = aucun ramassage jusqu'à l'image.
   */
  grenadePickups: GrenadePickupCount[] | null
}

/** Le document, réduit aux champs lus ici. */
export type ThrowsPickupsDoc = Pick<ReplayDocumentReady, 'roster' | 'grenades' | 'pickups' | 'schemaVersion'>

export interface PlayerThrowsPickupsInput {
  doc: ThrowsPickupsDoc
  /** Tous les joueurs du match (`buildPlayers`) : l'occupant d'un slot se lit parmi eux. */
  players: readonly ReplayPlayer[]
  /** Le joueur dont on lit les lancers et les ramassages (un bot compris, clé `bot:<nom>`). */
  player: ReplayPlayer
}

/** Un événement daté sur l'axe des images, consultable par `lastIndexAt` (série triée). */
interface Dated {
  t: number
}

/** Ce que `readPlayerThrowsPickupsSeries` range pour `playerThrowsPickupsAt`. */
export interface PlayerThrowsPickupsSeries {
  readonly throws: ReadonlyMap<number, readonly Dated[]>
  readonly weapon: ReadonlyMap<string, readonly Dated[]> | null
  readonly grenade: ReadonlyMap<string | null, readonly Dated[]> | null
}

/** Les index de joueur du film qui désignent ce joueur dans le roster (`rosterEntryKey`). */
function filmIndexesOf(doc: ThrowsPickupsDoc, player: ReplayPlayer): Set<number> {
  const indexes = new Set<number>()
  for (const entry of doc.roster) if (rosterEntryKey(entry) === player.xuid) indexes.add(entry.filmIndex)
  return indexes
}

/** Les clés sous lesquelles un ramassage nomme un joueur : son xuid et celui de sa ligne de feuille. */
function identityKeysOf(player: ReplayPlayer): string[] {
  const keys = [player.xuid]
  const board = player.board?.xuid
  if (board && board !== player.xuid) keys.push(board)
  return keys.filter((k) => k !== '')
}

/** Ajoute un événement daté à la série de sa clé ; les séries sont triées à la fin de la lecture. */
function push<K>(map: Map<K, Dated[]>, key: K, t: number): void {
  const series = map.get(key)
  if (series) series.push({ t })
  else map.set(key, [{ t }])
}

/** Trie chaque série par instant : `lastIndexAt` exige une série croissante. */
function sortedSeries<K>(map: Map<K, Dated[]>): Map<K, Dated[]> {
  for (const series of map.values()) series.sort((a, b) => a.t - b.t)
  return map
}

/**
 * readPlayerThrowsPickupsSeries lit, une fois, ce qui est daté pour ce joueur. Les lancers sont rangés par
 * type ; les ramassages le sont par arme et par grenade quand le document les publie pour cet artefact.
 */
export function readPlayerThrowsPickupsSeries(input: PlayerThrowsPickupsInput): PlayerThrowsPickupsSeries {
  const { doc, players, player } = input
  const films = filmIndexesOf(doc, player)
  const keys = identityKeysOf(player)
  const ownership = buildSlotOwnership(players)
  const owns = (p: { slot: number; t: number; xuid?: string }): boolean =>
    p.xuid ? keys.includes(p.xuid) : ownership.ownerAtFrame(p.slot, p.t)?.xuid === player.xuid

  const throws = new Map<number, Dated[]>()
  for (const g of doc.grenades) if (films.has(g.i)) push(throws, g.rank, g.t)

  const hasWeapons = doc.schemaVersion >= PICKUPS_SCHEMA
  const hasGrenades = doc.schemaVersion >= GRENADE_PICKUP_KIND_SCHEMA
  const weapon = new Map<string, Dated[]>()
  const grenade = new Map<string | null, Dated[]>()
  for (const p of doc.pickups) {
    if (p.kind === 'weapon' && hasWeapons && owns(p)) push(weapon, weaponLabelKeyOf(p.w), p.t)
    if (p.kind === 'grenade' && hasGrenades && owns(p)) push(grenade, p.family ?? null, p.t)
  }
  return {
    throws: sortedSeries(throws),
    weapon: hasWeapons ? sortedSeries(weapon) : null,
    grenade: hasGrenades ? sortedSeries(grenade) : null,
  }
}

/** Les séries d'une grandeur par clé, arrêtées à l'image : une clé sans événement jusque-là est omise. */
function countsUpTo<K>(series: ReadonlyMap<K, readonly Dated[]>, frame: number): [K, number][] {
  const out: [K, number][] = []
  for (const [key, events] of series) {
    const n = lastIndexAt(events, frame) + 1
    if (n > 0) out.push([key, n])
  }
  return out
}

/** Ordre croissant de deux identifiants textuels ; l'absence (`null`) se range après eux. */
function compareKeys(a: string | null, b: string | null): number {
  if (a === null || b === null) return a === b ? 0 : a === null ? 1 : -1
  return a < b ? -1 : a > b ? 1 : 0
}

/**
 * playerThrowsPickupsAt rend ce que le joueur a lancé et ramassé du début du match jusqu'à `frame` (compris).
 * Un événement daté exactement à `frame` compte.
 */
export function playerThrowsPickupsAt(s: PlayerThrowsPickupsSeries, frame: number): PlayerThrowsPickups {
  return {
    grenadesThrown: countsUpTo(s.throws, frame)
      .map(([rank, count]) => ({ rank, count }))
      .sort((a, b) => a.rank - b.rank),
    weaponPickups: s.weapon
      ? countsUpTo(s.weapon, frame)
          .map(([key, count]) => ({ key, count }))
          .sort((a, b) => compareKeys(a.key, b.key))
      : null,
    grenadePickups: s.grenade
      ? countsUpTo(s.grenade, frame)
          .map(([family, count]) => ({ family, count }))
          .sort((a, b) => compareKeys(a.family, b.family))
      : null,
  }
}
