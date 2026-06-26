/**
 * relationsFilter — filtrage CLIENT (chips) du tableau du hub Relations.
 *
 * Aucun appel réseau : on filtre la liste `relations[]` déjà servie par le
 * backend. Le « noyau dur » lit le flag serveur `is_core` (source unique :
 * analysis/relations.IsCore) — aucun seuil dupliqué côté front. « Vus récemment »
 * = dernière rencontre dans les 30 derniers jours (notion purement client).
 */
import type { RelationInsight } from '@/lib/api/types'

export type RelationFilter = 'all' | 'core' | 'allies' | 'rivals' | 'recent'

const RECENT_DAYS = 30

function isCore(r: RelationInsight): boolean {
  return r.is_core
}

function isRecent(r: RelationInsight): boolean {
  if (!r.last_seen_at) return false
  const ts = new Date(r.last_seen_at).getTime()
  if (Number.isNaN(ts)) return false
  return Date.now() - ts <= RECENT_DAYS * 86_400_000
}

export function filterRelations(
  relations: RelationInsight[],
  filter: RelationFilter,
): RelationInsight[] {
  switch (filter) {
    case 'core':
      return relations.filter(isCore)
    case 'allies':
      return relations.filter((r) => r.teammate_matches > 0)
    case 'rivals':
      return relations.filter((r) => r.enemy_matches > 0)
    case 'recent':
      return relations.filter(isRecent)
    case 'all':
    default:
      return relations
  }
}

export function coreRelations(relations: RelationInsight[]): RelationInsight[] {
  return relations.filter(isCore)
}
