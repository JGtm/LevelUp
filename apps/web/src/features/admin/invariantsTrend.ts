/**
 * invariantsTrend — logique pure de la tendance du dashboard
 * « Intégrité des données » (extraite d'AdminPage pour testabilité).
 *
 * Principe : snapshot des counts par (scope, invariant) persisté en
 * localStorage. La baseline de comparaison est ROULANTE : au premier run de la
 * session elle vient du localStorage (comparaison inter-sessions), puis chaque
 * nouveau run (generated_at différent) compare au run précédent — pas au
 * snapshot figé au mount (bug corrigé 2026-06-10 : un refetch intra-session
 * comparait à l'état pré-mount et pouvait masquer une régression).
 */
import type { AdminInvariantsResponse } from '@/lib/api/types'

export const INVARIANTS_SNAPSHOT_KEY = 'admin-invariants-snapshot'

/** Scope des invariants globaux dans les clés de snapshot. */
export const SHARED_SCOPE_KEY = '__shared__'

/** "scope|invariant_key" -> count. Scope = player_slug ou SHARED_SCOPE_KEY. */
export type InvariantsSnapshot = Record<string, number>

// read/write : supprimes (A8.2) — la persistance localStorage passe par le
// hook canonique useCounterSnapshot (countersTrend read/write).

export function buildInvariantsSnapshot(data: AdminInvariantsResponse): InvariantsSnapshot {
  const snap: InvariantsSnapshot = {}
  for (const v of data.shared_violations ?? []) {
    snap[`${SHARED_SCOPE_KEY}|${v.key}`] = v.count
  }
  for (const r of data.reports ?? []) {
    for (const v of r.violations ?? []) {
      snap[`${r.player_slug}|${v.key}`] = v.count
    }
  }
  return snap
}

/**
 * Delta d'un invariant vs la baseline. undefined = pas de référence (première
 * apparition de la clé) ou count inchangé.
 */
export function invariantDelta(
  previous: InvariantsSnapshot,
  scope: string,
  invariantKey: string,
  count: number,
): number | undefined {
  const prev = previous[`${scope}|${invariantKey}`]
  if (prev === undefined || prev === count) return undefined
  return count - prev
}
