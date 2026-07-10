/**
 * tabBadges.ts — dérive les pastilles de compteur des onglets admin depuis le
 * seul overview (déjà pollé, zéro I/O DuckDB ; React Query déduplique la query
 * partagée avec la page Vue d'ensemble). Fonction pure, testable sans React.
 *
 * Priorité de sévérité par onglet : un problème critique (destructive) masque
 * un avertissement. La Convergence n'a pas de pastille : son backlog vit dans
 * un endpoint séparé coûteux (résout toutes les player DBs) — pas chargé ici.
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

  // Sync & Jobs : échecs du dernier cycle prioritaires, sinon jobs actifs.
  if (overview.scheduler.available && overview.scheduler.last_failed > 0) {
    out['/admin/sync'] = { count: overview.scheduler.last_failed, token: 'destructive' }
  } else if (overview.jobs.active_count > 0) {
    out['/admin/sync'] = { count: overview.jobs.active_count, token: 'info', pulse: true }
  }

  // Qualité données : warnings du dernier audit data health.
  if (overview.data_health && overview.data_health.warnings_total > 0) {
    out['/admin/data-quality'] = { count: overview.data_health.warnings_total, token: 'warning' }
  }

  // Détections ouvertes (persistées) — porté par l'onglet Logs tant que
  // « Détections » n'est pas un onglet à part (A3). Seul le statut `open`
  // colore la nav : le bruit `muted`/`resolved` n'apparaît pas (A2.5).
  if (overview.open_detections > 0) {
    out['/admin/logs'] = { count: overview.open_detections, token: 'warning' }
  }

  // Système : invariants FAIL + tokens à problème = critique ; sinon WARN.
  const tokensBad = overview.tokens
    ? overview.tokens.expired + overview.tokens.absent + overview.tokens.reauth
    : 0
  const invariantsRan = overview.invariants.runs_total > 0
  const critical = (invariantsRan ? overview.invariants.fail_last : 0) + tokensBad
  if (critical > 0) {
    out['/admin/system'] = { count: critical, token: 'destructive' }
  } else if (invariantsRan && overview.invariants.warn_last > 0) {
    out['/admin/system'] = { count: overview.invariants.warn_last, token: 'warning' }
  }

  return out
}
