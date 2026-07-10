/**
 * diagnostics.ts — moteur de verdicts pur du panneau Diagnostic (testable
 * sans React). Interprète les métriques avec des seuils EXPLICITES et rend
 * des verdicts triés par sévérité : le dashboard désigne le point faible au
 * lieu de laisser l'admin le déduire.
 *
 * Entrées : données des queries déjà chargées par l'overview (zéro requête
 * supplémentaire) — overview, scheduler (+historique avec charge par cycle),
 * contention, perf.
 */
import type {
  AdminMonitoringOverview,
  AdminPerfStats,
  AdminSchedulerStatusResponse,
  DBContentionResponse,
} from '@/lib/api/types'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import { dominantStep } from '../convergence/timeline'

export type VerdictLevel = 'crit' | 'warn' | 'info'

export interface Verdict {
  level: VerdictLevel
  titleKey: AdminManifestKey
  /** Évidence chiffrée formatée par le caller (valeurs brutes ici). */
  evidence: string
  /** Onglet de drill-down. */
  to: string
}

// Seuils explicites (commentés = justifiés).
export const DIAG_THRESHOLDS = {
  /** Cycle en retard si plus vieux que 2× l'intervalle configuré. */
  staleCycleFactor: 2,
  /** Fenêtre d'indispo « notable » : >= 20 % de la durée du cycle ET >= 5 s. */
  blockedPctWarn: 20,
  blockedMsFloor: 5_000,
  /** Blocage ponctuel déjà alerté par la carte contention. */
  maxBlockedMsWarn: 1_000,
  /** Étape dominante du post-sync (cf. timeline.dominantStep). */
  dominantStepPct: 60,
  dominantStepMinTotalMs: 30_000,
} as const

export interface DiagnosticsInput {
  overview?: AdminMonitoringOverview
  scheduler?: AdminSchedulerStatusResponse
  contention?: DBContentionResponse
  perf?: AdminPerfStats
}

// evaluateDiagnostics croise les sources et retourne les verdicts triés
// (crit → warn → info). Fonction volontairement linéaire : une règle = un
// bloc, lisible et testable.
export function evaluateDiagnostics(input: DiagnosticsInput): Verdict[] {
  const out: Verdict[] = []
  const { overview, scheduler, contention, perf } = input

  if (overview) {
    evaluateOverviewRules(overview, out)
  }
  if (scheduler?.available && scheduler.snapshot) {
    evaluateSchedulerRules(scheduler, overview, out)
  }
  if (contention) {
    if (contention.max_blocked_ms >= DIAG_THRESHOLDS.maxBlockedMsWarn || contention.swap_failures > 0) {
      out.push({
        level: 'warn',
        titleKey: 'admin.diag.contention',
        evidence: `max ${contention.max_blocked_ms} ms · échecs swap ${contention.swap_failures}`,
        to: '/admin/system',
      })
    }
  }
  if (perf) {
    evaluatePerfRules(perf, out)
  }

  const rank: Record<VerdictLevel, number> = { crit: 0, warn: 1, info: 2 }
  return out.sort((a, b) => rank[a.level] - rank[b.level])
}

function evaluateOverviewRules(o: AdminMonitoringOverview, out: Verdict[]): void {
  if (o.invariants.runs_total > 0 && o.invariants.fail_last > 0) {
    out.push({
      level: 'crit',
      titleKey: 'admin.diag.invariants_fail',
      evidence: `${o.invariants.fail_last} FAIL`,
      to: '/admin/data',
    })
  }
  const tokensBad = o.tokens ? o.tokens.expired + o.tokens.reauth + o.tokens.absent : 0
  if (tokensBad > 0) {
    out.push({
      level: 'crit',
      titleKey: 'admin.diag.tokens',
      evidence: `${tokensBad}`,
      to: '/admin/sync',
    })
  }
  if (o.scheduler.available && o.scheduler.last_failed > 0) {
    out.push({
      level: 'warn',
      titleKey: 'admin.diag.sync_failures',
      evidence: `${o.scheduler.last_failed}/${o.scheduler.last_total}`,
      to: '/admin/sync',
    })
  }
  if (o.scheduler.zero_insert_alerts > 0) {
    out.push({
      level: 'warn',
      titleKey: 'admin.diag.zero_insert',
      evidence: `${o.scheduler.zero_insert_alerts}`,
      to: '/admin/sync',
    })
  }
  if (o.data_health && o.data_health.warnings_total > 0) {
    out.push({
      level: 'warn',
      titleKey: 'admin.diag.data_health',
      evidence: `${o.data_health.warnings_total}`,
      to: '/admin/data',
    })
  }
  const failedJobs = (o.jobs.recent ?? []).filter((j) => j.status === 'failed').length
  if (failedJobs > 0) {
    out.push({
      level: 'warn',
      titleKey: 'admin.diag.jobs_failed',
      evidence: `${failedJobs}`,
      to: '/admin/sync',
    })
  }
}

function evaluateSchedulerRules(
  s: AdminSchedulerStatusResponse,
  overview: AdminMonitoringOverview | undefined,
  out: Verdict[],
): void {
  const snap = s.snapshot
  if (!snap) return

  // Cycle en retard vs intervalle configuré.
  if (snap.last_cycle_at && !snap.last_cycle_at.startsWith('0001-') && snap.interval_minutes > 0) {
    const ageMin = (Date.now() - new Date(snap.last_cycle_at).getTime()) / 60_000
    if (ageMin > snap.interval_minutes * DIAG_THRESHOLDS.staleCycleFactor) {
      out.push({
        level: 'warn',
        titleKey: 'admin.diag.stale_cycle',
        evidence: `${Math.round(ageMin)} min > ${snap.interval_minutes * DIAG_THRESHOLDS.staleCycleFactor} min`,
        to: '/admin/sync',
      })
    }
  }

  // Corrélation indispo lectures : dernier cycle de l'historique.
  const last = (s.history ?? [])[0]
  if (last && last.duration_ms > 0) {
    const blockedPct = Math.round((last.blocked_ms / last.duration_ms) * 100)
    if (last.blocked_ms >= DIAG_THRESHOLDS.blockedMsFloor && blockedPct >= DIAG_THRESHOLDS.blockedPctWarn) {
      // Décomposition : si les écritures n'expliquent qu'une fraction de la
      // fenêtre, le writer est tenu pendant du travail non-DB (API/films) —
      // c'est l'argument « resserrer la fenêtre RW » (Option B du plan B-swap).
      const writeShare = last.blocked_ms > 0 ? Math.round((last.persist_write_ms / last.blocked_ms) * 100) : 0
      const apiBound = writeShare < 50
      out.push({
        level: 'warn',
        titleKey: apiBound ? 'admin.diag.blocked_api_bound' : 'admin.diag.blocked_write_bound',
        evidence: `${Math.round(last.blocked_ms / 1000)} s (${blockedPct}% du cycle) · écritures ${writeShare}%`,
        to: '/admin/sync',
      })
    }
    if (last.reads_rejected > 0) {
      out.push({
        level: 'warn',
        titleKey: 'admin.diag.reads_rejected',
        evidence: `${last.reads_rejected}`,
        to: '/admin/system',
      })
    }
  }

  // Goulot d'étape post-sync (joueur le plus lent du dernier cycle).
  for (const p of snap.players ?? []) {
    const dom = dominantStep(
      p.post_sync?.step_timings ?? undefined,
      DIAG_THRESHOLDS.dominantStepPct,
      DIAG_THRESHOLDS.dominantStepMinTotalMs,
    )
    if (dom) {
      out.push({
        level: 'info',
        titleKey: 'admin.diag.dominant_step',
        evidence: `${p.gamertag} : ${dom.step} ${Math.round(dom.durationMs / 1000)} s (${dom.pct}%)`,
        to: '/admin/data',
      })
      break // un seul verdict de goulot (le premier joueur concerné)
    }
  }
  void overview
}

function evaluatePerfRules(perf: AdminPerfStats, out: Verdict[]): void {
  if (perf.api_buckets.rate_limited_429 > 0) {
    out.push({
      level: 'warn',
      titleKey: 'admin.diag.api_429',
      evidence: `${perf.api_buckets.rate_limited_429}`,
      to: '/admin/sync',
    })
  }
  if (perf.api_buckets.auth > 0) {
    out.push({
      level: 'warn',
      titleKey: 'admin.diag.api_auth',
      // Erreurs auth API = santé tokens → onglet Sync (les tokens y vivent, A3.3).
      evidence: `${perf.api_buckets.auth}`,
      to: '/admin/sync',
    })
  }
  if (perf.api_buckets.server_5xx > 0) {
    out.push({
      level: 'info',
      titleKey: 'admin.diag.api_5xx',
      evidence: `${perf.api_buckets.server_5xx}`,
      to: '/admin/sync',
    })
  }
}
