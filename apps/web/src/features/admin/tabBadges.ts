/**
 * tabBadges.ts — dérive les pastilles de compteur des onglets admin depuis le
 * seul overview (déjà pollé, zéro I/O DuckDB ; React Query déduplique la query
 * partagée avec la page État). Fonction pure, testable sans React.
 *
 * Architecture DC-8 (A3.7) — la pastille suit la question de l'onglet :
 * Sync porte le moteur (échecs de cycle, tokens morts, jobs actifs), Données
 * porte l'intégrité (invariants, data health), Détections porte le triage
 * (statut `open` UNIQUEMENT — le bruit `muted`/`resolved` ne colore pas la nav).
 * Priorité de sévérité par onglet : un problème critique (destructive) masque
 * un avertissement.
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { AdminMonitoringOverview } from '@/lib/api/types'

export interface TabBadge {
  count: number
  token: SemanticToken
  /** Dot animé (états actifs, ex. jobs en cours). */
  pulse?: boolean
}

export function computeTabBadges(
  overview: AdminMonitoringOverview | undefined,
): Record<string, TabBadge> {
  const out: Record<string, TabBadge> = {}
  if (!overview) return out

  // Sync : échecs du dernier cycle + tokens morts (les tokens vivent dans Sync,
  // A3.3) = critique ; sinon jobs actifs = info pulsée.
  const tokensBad = overview.tokens
    ? overview.tokens.expired + overview.tokens.absent + overview.tokens.reauth
    : 0
  const syncCritical = (overview.scheduler.available ? overview.scheduler.last_failed : 0) + tokensBad
  if (syncCritical > 0) {
    out['/admin/sync'] = { count: syncCritical, token: 'destructive' }
  } else if (overview.jobs.active_count > 0) {
    out['/admin/sync'] = { count: overview.jobs.active_count, token: 'info', pulse: true }
  }

  // Données : invariants FAIL = critique ; sinon warnings (invariants WARN +
  // audit data health).
  const invariantsRan = overview.invariants.runs_total > 0
  const invariantsFail = invariantsRan ? overview.invariants.fail_last : 0
  const dataWarnings =
    (invariantsRan ? overview.invariants.warn_last : 0) +
    (overview.data_health?.warnings_total ?? 0)
  if (invariantsFail > 0) {
    out['/admin/data'] = { count: invariantsFail, token: 'destructive' }
  } else if (dataWarnings > 0) {
    out['/admin/data'] = { count: dataWarnings, token: 'warning' }
  }

  // Détections : seules les détections `open` colorent la nav (A2.5).
  if (overview.open_detections > 0) {
    out['/admin/detections'] = { count: overview.open_detections, token: 'warning' }
  }

  return out
}
