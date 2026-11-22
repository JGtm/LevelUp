/**
 * playerInstantSources.ts — CE QUE LA FICHE LIT DU DOCUMENT, UNE FOIS : les séries datées de chaque module de
 * lecture, rangées par document (le lobby entier) et par joueur. Sans React.
 *
 * DEUX NIVEAUX, parce que deux durées de vie. Ce qui se lit pour tout le lobby — score et rang, tirs, équipement,
 * sprint et sauts — ne dépend que du document et de ses joueurs (`readInstantDocSeries`) : changer de joueur ne le
 * relit pas. Ce qui se lit pour un joueur — compteurs, éliminations, lancers, médailles — en dépend aussi
 * (`readPlayerInstantSources`). Le rendu appelle chacun dans un `useMemo` et interroge ensuite `playerInstantViewAt`
 * à chaque image, sans rien relire du document.
 *
 * L'ARME D'UNE ÉLIMINATION se nomme par sa clé du registre : le nom est celui que le fil porte sur ses frags
 * (`weaponLabel`), à défaut celui du référentiel des pictogrammes d'arme. Une clé sans nom est `null`.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayScoreTimelineReady } from '../../../lib/replay/scoreTimeline'
import type { ReplayLocale } from '../i18n/i18n'
import { readKillBreakdownSeries, type KillBreakdownSeries } from './killBreakdown'
import type { ReplayFeedEntry } from './killFeedLogic'
import type { WeaponIconsByKey } from './killSourceIcons'
import {
  equipmentSeriesOf,
  readEquipmentSeries,
  type EquipmentSeriesIndex,
  type PlayerEquipmentSeries,
} from './playerEquipmentStats'
import { readInstantEstimatesSeries, type InstantEstimatesSeries } from './playerInstantEstimates'
import { readPlayerInstantSeries, type PlayerInstantSeries } from './playerInstantStats'
import {
  movementSeriesOf,
  readMovementSeries,
  type MovementSeriesIndex,
  type PlayerMovementSeries,
} from './playerMovementStats'
import { readPlayerThrowsPickupsSeries, type PlayerThrowsPickupsSeries } from './playerThrowsPickups'

/** Ce qui se lit une fois par document : les séries du lobby entier. */
export interface InstantDocSeries {
  estimates: InstantEstimatesSeries
  equipment: EquipmentSeriesIndex
  movement: MovementSeriesIndex
}

/** readInstantDocSeries lit, pour le lobby entier, ce qui ne dépend pas du joueur ouvert. */
export function readInstantDocSeries(doc: ReplayDocumentReady, players: readonly ReplayPlayer[]): InstantDocSeries {
  return {
    estimates: readInstantEstimatesSeries({ doc, players }),
    equipment: readEquipmentSeries(doc, players),
    movement: readMovementSeries(doc, players),
  }
}

/** Ce que la lecture d'un joueur demande. */
export interface PlayerInstantSourcesInput {
  doc: ReplayDocumentReady
  player: ReplayPlayer
  /** Le lobby (`ReplayModel.players`) : l'occupant d'un slot et le rang se lisent parmi eux. */
  players: readonly ReplayPlayer[]
  /** Le fil du rejeu (`ReplayModel.feed`) : éliminations, tirs à la tête, médailles datées. */
  feed: readonly ReplayFeedEntry[]
  scoreTimeline: ReplayScoreTimelineReady | undefined
  /** Première image du match (`ReplayWindowBounds.startFrame`), ou `null`. */
  matchStartFrame: number | null
  weaponIcons: WeaponIconsByKey
  locale: ReplayLocale
  docSeries: InstantDocSeries
}

/** Les séries d'un joueur, prêtes à être interrogées image par image. */
export interface PlayerInstantSources {
  doc: ReplayDocumentReady
  player: ReplayPlayer
  locale: ReplayLocale
  stats: PlayerInstantSeries
  kills: KillBreakdownSeries | null
  throws: PlayerThrowsPickupsSeries
  equipment: PlayerEquipmentSeries | null
  movement: PlayerMovementSeries | null
  estimates: InstantEstimatesSeries
  /** Le nom d'une arme par sa clé du registre, ou `null` quand aucun nom n'est connu. */
  weaponName: (weaponKey: string) => string | null
}

/** Les noms d'arme que le fil porte sur ses frags (`weaponLabel`, vide pour une source sans nom propre), par clé. */
function weaponNamesOf(feed: readonly ReplayFeedEntry[]): Map<string, string> {
  const names = new Map<string, string>()
  for (const e of feed) {
    const k = e.kill
    if (k && k.weaponKey && k.weaponLabel && !names.has(k.weaponKey)) names.set(k.weaponKey, k.weaponLabel)
  }
  return names
}

/** readPlayerInstantSources lit, une fois, tout ce qui est daté pour ce joueur. */
export function readPlayerInstantSources(input: PlayerInstantSourcesInput): PlayerInstantSources {
  const { doc, player, players, feed, scoreTimeline, matchStartFrame, weaponIcons, locale, docSeries } = input
  const fromFeed = weaponNamesOf(feed)
  return {
    doc,
    player,
    locale,
    stats: readPlayerInstantSeries({ doc, player, scoreTimeline, feed, matchStartFrame }),
    kills: readKillBreakdownSeries({ doc, player, feed }),
    throws: readPlayerThrowsPickupsSeries({ doc, players, player }),
    equipment: equipmentSeriesOf(docSeries.equipment, player),
    movement: movementSeriesOf(docSeries.movement, player),
    estimates: docSeries.estimates,
    weaponName: (key) => fromFeed.get(key) || weaponIcons.get(key)?.label || null,
  }
}
