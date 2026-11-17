/**
 * playerEquipmentStats.ts — L'ÉQUIPEMENT D'UN JOUEUR DU DÉBUT DU MATCH JUSQU'À L'IMAGE LUE, par famille :
 * pris, utilisé, lâché en mourant. Sans React.
 *
 * DEUX TEMPS, comme `playerInstantStats`. `readEquipmentSeries` lit une fois par document tout ce qui est
 * daté (prises, consommations, épisodes, poses, lâchers), l'attribue au joueur qui occupe le slot À L'IMAGE
 * du geste et range, par joueur, un instantané par geste ; `playerEquipmentStatsAt` répond pour une image
 * par dichotomie, sans rien relire du document.
 *
 * AUCUNE RÈGLE N'EST RÉÉCRITE : ce sont les lectures du total du match (`equipmentUsageLogic`), arrêtées à
 * une image. La date de chaque geste vient de `readEquipmentGestures` (épisodes, poses, lâchers) et de
 * `readEquipmentChanges` (prises, consommations) ; le côté « utilisé » de chaque famille vient de
 * `usageUsedOf` (épisodes pour les deux bonus, poses déployées pour le mur, consommations de charge pour
 * tout autre déployable) ; le « lâché » de la pile `tally.dropped` pontée par `droppedFamilyOf` ; le gardé
 * de `keptOf`. À la dernière image du match, chaque famille égale donc sa colonne de la grille du match.
 *
 * CE QUE « PRIS » VEUT DIRE ICI : le total de la pile de la grille, « N objets pris » = utilisé + gardé +
 * lâché, soit `max(prises datées, utilisé + lâché)`. Un objet de réapparition, jamais « pris » par
 * construction, mais utilisé ou lâché, compte ainsi comme pris : la grille du match le compte.
 *
 * LA LISTE DES FAMILLES est celle que la grille du match affiche, dans son ordre (`equipmentFamilyOrder`) :
 * le grappin, le propulseur et le répulseur n'y ont pas de ligne (périmètre du bilan, `KEPT_FAMILIES`), donc
 * AUCUNE entrée ici — jamais un zéro. À l'image lue, une famille dont aucun objet n'est pris jusque-là est
 * absente : la lecture ne dévoile pas ce qui se passera.
 *
 * Les BOTS comptent comme tout joueur : un geste revient au joueur qui occupe le slot, dont la clé de roster
 * est `bot:<nom>` ; l'index se lit par cette clé puis par le xuid de la ligne de feuille (`identityKeys`).
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildSlotOwnership, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import {
  applyEquipmentGesture,
  readEquipmentGestures,
  type EquipmentGesture,
  type GestureAt,
} from './equipmentGestures'
import {
  applyEquipmentChange,
  keptOf,
  readEquipmentChanges,
  usageUsedOf,
  type DatedEquipmentChange,
} from './equipmentKeptLogic'
import {
  emptyTally,
  equipmentFamilyOrder,
  equipmentRowPlayers,
  type EquipmentUsageTally,
} from './equipmentUsageLogic'
import { droppedFamilyOf } from './gameChangers'
import { identityKeys } from './playerInstantStats'

/** Les trois comptes d'une famille jusqu'à l'image, cumulés depuis le début du match. */
export interface EquipmentFamilyStats {
  /** Identifiant de famille du bilan (`KEPT_FAMILIES` : `wall`, `sensor`, `camo`, `overshield`...). Jamais un libellé. */
  family: string
  /**
   * Objets pris : utilisé + gardé + lâché, le total de la pile de la grille du match. Toujours au moins 1 :
   * une famille sans objet pris n'est pas dans la liste.
   */
  taken: number
  /** Objets utilisés, au sens de `usageUsedOf` : activés (bonus), posés (mur) ou consommés (autres déployables). */
  used: number
  /** Objets lâchés en mourant (`equipmentPlacements` d'origine `dropped`). */
  dropped: number
}

/** Ce que la lecture rend pour une image. */
export interface PlayerEquipmentStats {
  /**
   * Les familles dont au moins un objet est pris à l'image, dans l'ordre de la grille du match. Liste vide =
   * aucun objet pris jusque-là (un zéro mesuré). Les familles hors du périmètre du bilan (grappin,
   * propulseur, répulseur) n'y figurent jamais : le bilan ne les mesure pas, ce n'est pas « aucune ».
   */
  families: readonly EquipmentFamilyStats[]
}

/** Un geste daté, avant l'attribution : un geste du total du match ou un changement d'équipement. */
type DatedEvent = { t: number; gesture: EquipmentGesture } | { t: number; change: DatedEquipmentChange }

/** Les comptes de toutes les familles juste après le geste daté `t` ; l'objet est rendu tel quel à chaque image. */
interface EquipmentStep {
  t: number
  stats: PlayerEquipmentStats
}

/**
 * Ce que `readEquipmentSeries` range pour `playerEquipmentStatsAt` : les instantanés d'un joueur, triés par
 * image. Le champ ne se lit pas ailleurs : il est l'état interne d'une lecture, exposé pour le typage seulement.
 */
export interface PlayerEquipmentSeries {
  readonly steps: readonly EquipmentStep[]
}

/** Les séries de tous les joueurs d'une ligne, sous leur clé de roster et sous le xuid de leur ligne de feuille. */
export type EquipmentSeriesIndex = ReadonlyMap<string, PlayerEquipmentSeries>

/** Aucun objet pris : la réponse avant le premier geste, partagée (elle est immuable). */
const NO_EQUIPMENT: PlayerEquipmentStats = { families: [] }

/** Ce que la lecture a besoin du document. */
export type EquipmentSeriesSource = Pick<
  ReplayDocumentReady,
  'equipmentEpisodes' | 'equipmentPlacements' | 'equipmentChanges' | 'abilityLabels' | 'frameIntervalMs'
>

/** Les trois comptes de chaque famille du bilan dont un objet est pris, d'après le compteur et les prises du joueur. */
function familiesOf(
  order: readonly string[],
  tally: EquipmentUsageTally,
  taken: Record<string, number>,
): EquipmentFamilyStats[] {
  const out: EquipmentFamilyStats[] = []
  for (const family of order) {
    const used = usageUsedOf(tally, family)
    const dropped = tally.dropped[droppedFamilyOf(family)] ?? 0
    const total = used + keptOf(taken[family] ?? 0, used, dropped) + dropped
    if (total > 0) out.push({ family, taken: total, used, dropped })
  }
  return out
}

/** Verse les gestes d'un joueur dans l'ordre de leur date et range un instantané après chacun. */
function stepsOf(events: readonly DatedEvent[], order: readonly string[]): EquipmentStep[] {
  const tally = emptyTally()
  const taken: Record<string, number> = {}
  return [...events]
    .sort((a, b) => a.t - b.t)
    .map((e) => {
      if ('gesture' in e) applyEquipmentGesture(tally, e.gesture)
      else applyEquipmentChange(tally, taken, e.change)
      return { t: e.t, stats: { families: familiesOf(order, tally, taken) } }
    })
}

/**
 * readEquipmentSeries lit, une fois par document, tout ce qui est daté pour chaque joueur d'une ligne
 * (`equipmentRowPlayers` : les mêmes que la grille du match). `players` est le roster du film (`buildPlayers`).
 * Un geste sans poseur mesuré, ou dont le slot n'a pas de propriétaire à l'image du geste, ne revient à personne.
 */
export function readEquipmentSeries(
  doc: EquipmentSeriesSource,
  players: readonly ReplayPlayer[],
): EquipmentSeriesIndex {
  const rows = equipmentRowPlayers(players)
  const ownership = buildSlotOwnership(rows)
  const eventsOf = new Map<string, DatedEvent[]>()
  const give = (at: GestureAt, event: DatedEvent): void => {
    const owner = ownership.ownerAtFrame(at.slot, at.t)
    if (!owner) return
    const own = eventsOf.get(owner.xuid)
    if (own) own.push(event)
    else eventsOf.set(owner.xuid, [event])
  }
  for (const gesture of readEquipmentGestures(doc)) {
    if (gesture.at) give(gesture.at, { t: gesture.at.t, gesture })
  }
  for (const change of readEquipmentChanges(doc).changes) {
    give({ slot: change.slot, t: change.t }, { t: change.t, change })
  }

  const order = equipmentFamilyOrder()
  const index = new Map<string, PlayerEquipmentSeries>()
  for (const player of rows) {
    const series: PlayerEquipmentSeries = { steps: stepsOf(eventsOf.get(player.xuid) ?? [], order) }
    for (const key of identityKeys(player)) index.set(key, series)
  }
  return index
}

/**
 * equipmentSeriesOf — la série d'un joueur : sa clé de roster, puis le xuid de sa ligne de feuille. `null` = le
 * joueur n'a pas de ligne dans la grille du match (le film ne l'a pas vu vivre, ou n'écrit pas son équipe) :
 * son équipement n'est pas établi.
 */
export function equipmentSeriesOf(index: EquipmentSeriesIndex, player: ReplayPlayer): PlayerEquipmentSeries | null {
  for (const key of identityKeys(player)) {
    const series = index.get(key)
    if (series) return series
  }
  return null
}

/**
 * playerEquipmentStatsAt rend l'équipement du joueur du début du match jusqu'à `frame` (compris). `null` quand
 * la série l'est : le joueur n'a pas de ligne, son équipement n'est pas établi (jamais « aucun objet pris »).
 */
export function playerEquipmentStatsAt(series: PlayerEquipmentSeries | null, frame: number): PlayerEquipmentStats | null {
  if (!series) return null
  const i = lastIndexAt(series.steps, frame)
  return i >= 0 ? series.steps[i].stats : NO_EQUIPMENT
}
