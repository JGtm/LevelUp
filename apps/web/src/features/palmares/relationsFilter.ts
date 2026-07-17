/**
 * relationsFilter — filtrage CLIENT (chips) du tableau du hub Relations.
 *
 * Aucun appel réseau : on filtre la liste `relations[]` déjà servie par le
 * backend. Le « noyau dur » lit le flag serveur `is_core` (source unique :
 * analysis/relations.IsCore) — aucun seuil dupliqué côté front. « Vus récemment »
 * = dernière rencontre dans les 30 derniers jours (notion purement client).
 * « Multi-jeux » = relations aussi croisées sur un AUTRE titre : lit le badge
 * cross-jeu déjà servi (source unique backend, seuil serveur ≥ 3 matchs communs) —
 * aucune plomberie cliente supplémentaire.
 */
import type { RelationInsight } from '@/lib/api/types'

export type RelationFilter = 'all' | 'core' | 'allies' | 'rivals' | 'recent' | 'cross'

const RECENT_DAYS = 30

/**
 * CROSS_GAME_BADGE_KEY — clé i18n du badge cross-jeu (source unique côté front).
 * Doit rester alignée avec le backend (relations.CrossGameBadge, label_key
 * "narrative.encounter.cross_game"). Toute lecture du badge cross-jeu passe par
 * cette constante — ne pas ré-écrire le littéral ailleurs.
 */
export const CROSS_GAME_BADGE_KEY = 'narrative.encounter.cross_game'

function isCore(r: RelationInsight): boolean {
  return r.is_core
}

/**
 * isCrossGame — vrai si la relation porte le badge cross-jeu (croisée sur un
 * autre titre au-dessus du seuil serveur). Le badge est best-effort : absent si
 * l'enrichissement cross-titre a échoué ou sous le seuil.
 */
export function isCrossGame(r: RelationInsight): boolean {
  return (r.badges ?? []).some((b) => b.label_key === CROSS_GAME_BADGE_KEY)
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
    case 'cross':
      return relations.filter(isCrossGame)
    case 'all':
    default:
      return relations
  }
}

export function coreRelations(relations: RelationInsight[]): RelationInsight[] {
  return relations.filter(isCore)
}

/**
 * hasCrossGameRelations — au moins une relation croisée sur un autre titre.
 * Pilote l'affichage conditionnel du chip « Multi-jeux » : pas de segment mort
 * pour les profils mono-titre.
 */
export function hasCrossGameRelations(relations: RelationInsight[]): boolean {
  return relations.some(isCrossGame)
}
