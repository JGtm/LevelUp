/**
 * periodSessionNav — helpers purs pour le rail de navigation période/session.
 *
 * Le rail sticky (PeriodSessionRail) affiche le scope temporel actif et
 * propose 2 boutons prev/next pour explorer rapidement. Cas d'usage primaire :
 * landing sur la dernière session, naviguer vers les sessions précédentes en
 * 1 clic.
 *
 * Conventions :
 *  - `all_sessions` est trié latest→oldest (idx 0 = la plus récente).
 *  - "Session précédente" = idx + 1 (plus ancienne).
 *  - "Session suivante"   = idx - 1 (plus récente).
 *  - Pour les périodes (preset ou custom), prev/next = sliding window de la
 *    durée de la fenêtre courante (cap à aujourd'hui pour next).
 */
import { isoDate } from '@/components/shell/_filter_pills/_hooks'
import type { FilterContextInput, PeriodInput, SessionOption } from '@/lib/api/types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type RailMode =
  | { kind: 'session'; session: SessionOption; index: number; total: number }
  | { kind: 'multi-session'; count: number }
  | { kind: 'period'; period: PeriodInput; durationDays: number }
  /** Aucun scope choisi mais ≥1 session existe : rail informatif "Toutes les sessions",
   *  boutons prev/next désactivés (pas de fenêtre à shift). */
  | { kind: 'all-time'; total: number }
  | { kind: 'hidden' }

// ---------------------------------------------------------------------------
// Helpers de mode
// ---------------------------------------------------------------------------

/** Détermine le mode du rail à partir du filterContext et des sessions disponibles. */
export function getRailMode(
  ctx: FilterContextInput,
  allSessions: SessionOption[],
): RailMode {
  const picked = ctx.sessions?.picked_sessions ?? []

  if (picked.length === 1) {
    const id = picked[0]
    const idx = allSessions.findIndex((s) => s.session_id === id)
    if (idx === -1) {
      // Session pickée mais introuvable (cache stale, sync en cours) → hidden.
      return { kind: 'hidden' }
    }
    return { kind: 'session', session: allSessions[idx], index: idx, total: allSessions.length }
  }

  if (picked.length >= 2) {
    return { kind: 'multi-session', count: picked.length }
  }

  // Pas de session pickée → période (preset ou custom)
  const period = ctx.period
  if (period?.start_date && period?.end_date) {
    const days = daysBetween(period.start_date, period.end_date)
    if (days > 0) {
      return { kind: 'period', period, durationDays: days }
    }
  }

  // Aucun scope MAIS au moins une session existe → mode all-time informatif.
  if (allSessions.length > 0) {
    return { kind: 'all-time', total: allSessions.length }
  }

  // Aucune donnée du tout (premier launch, sync pas terminé) → rail caché.
  return { kind: 'hidden' }
}

// ---------------------------------------------------------------------------
// Navigation session
// ---------------------------------------------------------------------------

/** Retourne la session précédente (plus ancienne) ou null si déjà la plus ancienne. */
export function computePrevSession(
  currentId: string,
  all: SessionOption[],
): SessionOption | null {
  const idx = all.findIndex((s) => s.session_id === currentId)
  if (idx === -1) return null
  if (idx >= all.length - 1) return null
  return all[idx + 1]
}

/** Retourne la session suivante (plus récente) ou null si déjà la plus récente. */
export function computeNextSession(
  currentId: string,
  all: SessionOption[],
): SessionOption | null {
  const idx = all.findIndex((s) => s.session_id === currentId)
  if (idx <= 0) return null
  return all[idx - 1]
}

// ---------------------------------------------------------------------------
// Navigation période — sliding window de la durée courante
// ---------------------------------------------------------------------------

const MS_PER_DAY = 86_400_000

/** Nombre de jours entre 2 dates ISO (start ≤ end). Renvoie 0 si invalide. */
export function daysBetween(startISO: string, endISO: string): number {
  const start = parseISO(startISO)
  const end = parseISO(endISO)
  if (!start || !end || end < start) return 0
  return Math.round((end.getTime() - start.getTime()) / MS_PER_DAY)
}

function parseISO(iso: string): Date | null {
  const d = new Date(iso + 'T00:00:00Z')
  return isNaN(d.getTime()) ? null : d
}

/**
 * Slide la fenêtre vers le passé : `[J-2N, J-N]` à partir de `[J-N, J]`.
 * Le "next" devient juste avant le "start" actuel.
 */
export function computePrevWindow(period: PeriodInput): PeriodInput | null {
  if (!period.start_date || !period.end_date) return null
  const start = parseISO(period.start_date)
  const end = parseISO(period.end_date)
  if (!start || !end) return null
  const durationDays = Math.round((end.getTime() - start.getTime()) / MS_PER_DAY)
  if (durationDays < 1) return null

  const newEnd = new Date(start.getTime() - MS_PER_DAY)
  const newStart = new Date(newEnd.getTime() - durationDays * MS_PER_DAY)
  return { start_date: isoDateUTC(newStart), end_date: isoDateUTC(newEnd) }
}

/**
 * Slide la fenêtre vers le futur, capée à aujourd'hui.
 * Renvoie null si on est déjà à la dernière fenêtre possible (end >= today).
 */
export function computeNextWindow(
  period: PeriodInput,
  today: Date = new Date(),
): PeriodInput | null {
  if (!period.start_date || !period.end_date) return null
  const start = parseISO(period.start_date)
  const end = parseISO(period.end_date)
  if (!start || !end) return null
  const durationDays = Math.round((end.getTime() - start.getTime()) / MS_PER_DAY)
  if (durationDays < 1) return null

  const todayUTC = parseISO(isoDate(today)) ?? today
  // Si la fenêtre actuelle se termine aujourd'hui ou plus tard → pas de next.
  if (end.getTime() >= todayUTC.getTime()) return null

  const newStart = new Date(end.getTime() + MS_PER_DAY)
  const newEnd = new Date(newStart.getTime() + durationDays * MS_PER_DAY)

  // Cap à aujourd'hui.
  const cappedEnd = newEnd.getTime() > todayUTC.getTime() ? todayUTC : newEnd
  return { start_date: isoDateUTC(newStart), end_date: isoDateUTC(cappedEnd) }
}

/** Comme isoDate du _hooks mais en UTC pour cohérence avec parseISO. */
function isoDateUTC(d: Date): string {
  const yyyy = d.getUTCFullYear()
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0')
  const dd = String(d.getUTCDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}
