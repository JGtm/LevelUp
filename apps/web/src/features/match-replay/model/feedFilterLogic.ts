/**
 * feedFilterLogic.ts — LES FILTRES DU FIL des éliminations. Logique pure, aucune chaîne
 * d'interface (les libellés vivent dans `i18n/`).
 *
 * LE FILTRE NE PORTE QUE SUR LA LISTE AFFICHÉE. `filterFeedEntries` rend une liste neuve et ne
 * touche jamais `model.feed` : la frise et les effets de mort lisent ce fil-là, entier.
 *
 * SÉMANTIQUE (celle de la maquette, `matchFilter`) :
 *  - Tout : chaque forme de ligne ;
 *  - Éliminations : les frags ;
 *  - Objectifs, Armes et bonus, Véhicules : les lignes d'événement de leur famille ;
 *  - Médailles : les médailles seules ET les frags qui en portent ;
 *  - Assistances : les frags dont l'assistance est NOMMÉE.
 * Assistances et médailles restent sur la ligne de leur élimination dans « Tout » : leurs
 * entrées de menu servent à les isoler, pas à les dédoubler.
 *
 * LE MENU. Toutes les entrées sont proposées, sauf Objectifs : un match sans aucune ligne
 * d'objectif (Assassin, ou horloge non sûre) n'a pas de commande à proposer. Le choix mémorisé
 * qui n'existe pas pour le match ouvert retombe sur Tout (`effectiveFeedFilter`) sans que la
 * préférence soit réécrite : elle revient dès qu'un match qui l'a est ouvert.
 */
import type { ReplayFeedEntry } from './killFeedLogic'

/** Les filtres, dans l'ordre du menu : trois d'abord, puis « proposées en plus ». */
export const FEED_FILTER_IDS = [
  'tout',
  'kills',
  'objectives',
  'pickups',
  'vehicles',
  'medals',
  'assists',
] as const

export type FeedFilterId = (typeof FEED_FILTER_IDS)[number]

export const DEFAULT_FEED_FILTER: FeedFilterId = 'tout'

/** Le filtre qui n'est proposé que si le match a au moins une ligne de sa famille. */
const CONDITIONAL_FILTER: FeedFilterId = 'objectives'

/** Une entrée du menu : le filtre et son nombre de lignes dans TOUT le match. */
export interface FeedFilterOption {
  id: FeedFilterId
  count: number
}

/** Vrai quand la ligne passe le filtre. */
export function matchesFeedFilter(entry: ReplayFeedEntry, filter: FeedFilterId): boolean {
  switch (filter) {
    case 'tout':
      return true
    case 'kills':
      return entry.kill !== null
    case 'objectives':
    case 'pickups':
    case 'vehicles':
      return entry.event?.family === filter
    case 'medals':
      return entry.medal !== null || (entry.kill?.medals.length ?? 0) > 0
    case 'assists':
      return entry.kill?.assistState === 'named'
  }
}

/** Le nombre de lignes de chaque filtre dans la liste complète du match. */
export function countByFeedFilter(entries: readonly ReplayFeedEntry[]): Record<FeedFilterId, number> {
  const counts = Object.fromEntries(FEED_FILTER_IDS.map((id) => [id, 0])) as Record<FeedFilterId, number>
  for (const e of entries) {
    for (const id of FEED_FILTER_IDS) if (matchesFeedFilter(e, id)) counts[id]++
  }
  return counts
}

/** Les entrées du menu pour ce match : toutes, sauf Objectifs quand le match n'a aucune ligne d'objectif. */
export function availableFeedFilters(entries: readonly ReplayFeedEntry[]): FeedFilterOption[] {
  const counts = countByFeedFilter(entries)
  return FEED_FILTER_IDS.filter((id) => id !== CONDITIONAL_FILTER || counts[id] > 0).map((id) => ({
    id,
    count: counts[id],
  }))
}

/** Le filtre APPLIQUÉ : le mémorisé s'il est proposé pour ce match, Tout sinon. */
export function effectiveFeedFilter(
  stored: FeedFilterId,
  available: readonly FeedFilterOption[],
): FeedFilterId {
  return available.some((o) => o.id === stored) ? stored : DEFAULT_FEED_FILTER
}

/** La liste affichée : les lignes du fil qui passent le filtre. Liste neuve, fil intact. */
export function filterFeedEntries(
  entries: readonly ReplayFeedEntry[],
  filter: FeedFilterId,
): ReplayFeedEntry[] {
  return filter === 'tout' ? [...entries] : entries.filter((e) => matchesFeedFilter(e, filter))
}
