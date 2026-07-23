import { describe, expect, it } from 'vitest'

import type {
  AdminMonitoringOverview,
  AdminPerfStats,
  AdminSchedulerStatusResponse,
  SchedulerCycleRecord,
} from '@/lib/api/types'
import { evaluateDiagnostics } from './diagnostics'

function baseOverview(): AdminMonitoringOverview {
  return {
    title_slug: 'halo_infinite',
    generated_at: '2026-06-12T10:00:00Z',
    server: { uptime_s: 3600, started_at: '2026-06-12T09:00:00Z', version: 'dev' },
    scheduler: {
      available: true,
      last_total: 3,
      last_synced: 3,
      last_skipped: 0,
      last_failed: 0,
      last_duration_ms: 60_000,
      zero_insert_alerts: 0,
      in_flight_claims: 0,
      interval_minutes: 15,
      last_cycle_at: new Date().toISOString(),
    },
    jobs: { active_count: 0, recent: [] },
    invariants: { runs_total: 1, fail_last: 0, warn_last: 0 },
    tokens: { players: 3, ok: 3, expiring: 0, expired: 0, absent: 0, reauth: 0, with_auth_error: 0 },
    snapshot: {
      cut_failures: 0, cut_noop: 0, cuts_produced: 0, partial_total: 0,
      pending_oldest_age_seconds: 0, pending_total: 0, reads_fallback: 0,
      reads_served: 0, ready_match_count: 0, version: 1,
    },
    open_detections: 0,
    freshness_critical: 0,
    lusr_interior_gaps: 0,
    http: { status_2xx: 0, status_3xx: 0, status_4xx: 0, status_5xx: 0 },
  }
}

function cycle(partial: Partial<SchedulerCycleRecord>): SchedulerCycleRecord {
  return {
    at: new Date().toISOString(),
    trigger: 'tick',
    total: 3,
    synced: 3,
    skipped: 0,
    failed: 0,
    duration_ms: 120_000,
    blocked_ms: 0,
    swap_count: 0,
    reads_rejected: 0,
    api_ms: 0,
    persist_write_ms: 0,
    ...partial,
  }
}

function schedulerWith(history: SchedulerCycleRecord[]): AdminSchedulerStatusResponse {
  return {
    available: true,
    history,
    history_since_boot: true,
    zero_insert_warn_threshold: 6,
    snapshot: {
      last_cycle_at: new Date().toISOString(),
      interval_minutes: 15,
      pool_size: 3,
      players: [],
      gate: { inflight_watcher: 0, inflight_gate: 0, granted_total: 0, coalesced_total: 0, stale_count: 0 },
      since_boot: true,
    },
  }
}

describe('evaluateDiagnostics', () => {
  it('état sain → aucun verdict', () => {
    expect(evaluateDiagnostics({ overview: baseOverview(), scheduler: schedulerWith([cycle({})]) })).toEqual([])
  })

  it('invariants FAIL et tokens morts → crit, triés avant les warn', () => {
    const o = baseOverview()
    o.invariants.fail_last = 2
    o.tokens = { ...o.tokens!, expired: 1, ok: 2 }
    o.scheduler.last_failed = 1
    const verdicts = evaluateDiagnostics({ overview: o })
    expect(verdicts[0].level).toBe('crit')
    expect(verdicts.filter((v) => v.level === 'crit')).toHaveLength(2)
    expect(verdicts.some((v) => v.titleKey === 'admin.diag.sync_failures')).toBe(true)
  })

  it('fenêtre d\'indispo dominée par du travail non-DB → verdict api_bound', () => {
    // 40 s bloquées sur un cycle de 120 s (33 %), écritures = 4 s (10 %).
    const verdicts = evaluateDiagnostics({
      scheduler: schedulerWith([
        cycle({ blocked_ms: 40_000, persist_write_ms: 4_000, api_ms: 90_000 }),
      ]),
    })
    const blocked = verdicts.find((v) => v.titleKey === 'admin.diag.blocked_api_bound')
    expect(blocked).toBeDefined()
    expect(blocked!.evidence).toContain('33%')
  })

  it('fenêtre d\'indispo expliquée par les écritures → verdict write_bound', () => {
    const verdicts = evaluateDiagnostics({
      scheduler: schedulerWith([cycle({ blocked_ms: 30_000, persist_write_ms: 25_000 })]),
    })
    expect(verdicts.some((v) => v.titleKey === 'admin.diag.blocked_write_bound')).toBe(true)
  })

  it('petite fenêtre d\'indispo (sous plancher ou sous pourcentage) → silence', () => {
    expect(
      evaluateDiagnostics({ scheduler: schedulerWith([cycle({ blocked_ms: 3_000 })]) }),
    ).toEqual([])
    expect(
      evaluateDiagnostics({
        scheduler: schedulerWith([cycle({ blocked_ms: 6_000, duration_ms: 600_000 })]),
      }),
    ).toEqual([])
  })

  it('503 servis pendant le cycle → verdict reads_rejected', () => {
    const verdicts = evaluateDiagnostics({
      scheduler: schedulerWith([cycle({ reads_rejected: 4 })]),
    })
    expect(verdicts.some((v) => v.titleKey === 'admin.diag.reads_rejected')).toBe(true)
  })

  it('429 API et goulot d\'étape → warn + info', () => {
    const perf: AdminPerfStats = {
      generated_at: '',
      api_calls: [],
      api_buckets: { rate_limited_429: 7, auth: 0, server_5xx: 0, network: 0, other: 0 },
      persist_phases: [],
      postsync_steps: [],
      postsync_total: { name: 'postsync_total', count: 1, sum_ms: 0, avg_ms: 0, max_ms: 0 },
      blocked_window: { name: 'blocked_window', count: 0, sum_ms: 0, avg_ms: 0, max_ms: 0 },
      api_by_player: [],
    }
    const sched = schedulerWith([cycle({})])
    sched.snapshot!.players = [
      {
        gamertag: 'JGtm',
        xuid: 'x1',
        outcome: 'ok',
        reason: '',
        attempted_at: '',
        duration_ms: 0,
        post_sync: {
          perf_scores_computed: 0,
          lusr_updated: 0,
          career_synced: false,
          views_refreshed: 0,
          achievements_synced: false,
          matches_promoted_friends: 0,
          engagement_scores_computed: 0,
          engagement_coefs_updated: 0,
          sessions_assigned: 0,
          weapon_kills_processed: 12,
          weapon_kills_no_film: 0,
          citations_computed: 0,
          dominance_flags_computed: 0,
          converged_events: 0,
          converged_psa: 0,
          duration_ms: 65_000,
          step_timings: [
            { step: 'weapon_kills', duration_ms: 50_000, items: 12 },
            { step: 'scoring', duration_ms: 10_000, items: 3 },
          ],
        },
      },
    ]
    const verdicts = evaluateDiagnostics({ scheduler: sched, perf })
    expect(verdicts.some((v) => v.titleKey === 'admin.diag.api_429' && v.level === 'warn')).toBe(true)
    const dom = verdicts.find((v) => v.titleKey === 'admin.diag.dominant_step')
    expect(dom?.level).toBe('info')
    expect(dom?.evidence).toContain('weapon_kills')
  })
})
