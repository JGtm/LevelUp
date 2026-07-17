/**
 * whatsNew — sélection CLIENT du volet « Quoi de neuf » du hub Relations.
 *
 * Aucun appel réseau : tout vient de `relations[]` déjà servi. Deux groupes :
 *  - Nouvelles têtes : `first_seen_at` dans les NEW_FACE_WINDOW_DAYS derniers
 *    jours (notion purement client — pas de SQL).
 *  - Retrouvailles : flag serveur `is_revived` (source unique
 *    analysis/relations.IsRevived — fenêtres 30 j / 90 j côté Go, pas de seuil
 *    dupliqué ici).
 *
 * Chaque groupe est plafonné à WHATS_NEW_MAX_PER_GROUP gamertags ; le reste est
 * exposé en `overflow` (compteur « +N »).
 */
import type { RelationInsight } from '@/lib/api/types'

// NEW_FACE_WINDOW_DAYS : miroir client de analysis/relations.NewFaceWindowDays.
export const NEW_FACE_WINDOW_DAYS = 30
// WHATS_NEW_MAX_PER_GROUP : gamertags affichés avant le compteur « +N ».
export const WHATS_NEW_MAX_PER_GROUP = 5

const DAY_MS = 86_400_000

export interface WhatsNewGroup {
  players: RelationInsight[] // plafonné à WHATS_NEW_MAX_PER_GROUP
  overflow: number // nombre au-delà du plafond (>= 0)
  total: number
}

export interface WhatsNew {
  newFaces: WhatsNewGroup
  reunions: WhatsNewGroup
}

/** isNewFace — première rencontre dans la fenêtre récente (client). */
export function isNewFace(r: RelationInsight, now: number): boolean {
  if (!r.first_seen_at) return false
  const ts = new Date(r.first_seen_at).getTime()
  if (Number.isNaN(ts)) return false
  return now - ts <= NEW_FACE_WINDOW_DAYS * DAY_MS
}

/** isReunion — retrouvailles (flag serveur is_revived). */
export function isReunion(r: RelationInsight): boolean {
  return r.is_revived
}

function epoch(iso: string | null | undefined): number {
  if (!iso) return 0
  const ts = new Date(iso).getTime()
  return Number.isNaN(ts) ? 0 : ts
}

function buildGroup(rows: RelationInsight[], key: (r: RelationInsight) => number): WhatsNewGroup {
  const sorted = [...rows].sort((a, b) => key(b) - key(a))
  return {
    players: sorted.slice(0, WHATS_NEW_MAX_PER_GROUP),
    overflow: Math.max(0, sorted.length - WHATS_NEW_MAX_PER_GROUP),
    total: sorted.length,
  }
}

/**
 * computeWhatsNew — construit les deux groupes triés (plus récent d'abord) et
 * plafonnés. `now` injectable pour les tests (défaut : Date.now()).
 */
export function computeWhatsNew(relations: RelationInsight[], now: number = Date.now()): WhatsNew {
  return {
    newFaces: buildGroup(
      relations.filter((r) => isNewFace(r, now)),
      (r) => epoch(r.first_seen_at),
    ),
    reunions: buildGroup(relations.filter(isReunion), (r) => epoch(r.last_seen_at)),
  }
}

/** hasWhatsNew — au moins un groupe non vide (pilote le rendu conditionnel). */
export function hasWhatsNew(w: WhatsNew): boolean {
  return w.newFaces.total > 0 || w.reunions.total > 0
}
