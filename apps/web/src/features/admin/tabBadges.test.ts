import { describe, expect, it } from 'vitest'

import type { AdminMonitoringOverview } from '@/lib/api/types'
import { computeTabBadges } from './tabBadges'

// Overview minimal : tout sain, surchargé par test.
function baseOverview(): AdminMonitoringOverview {
  return {
    title_slug: 'halo_infinite',
    generated_at: '2026-06-12T12:00:00Z',
    server: { uptime_s: 100, started_at: '2026-06-12T11:58:00Z', version: 'test' },
    scheduler: {
      available: true,
      last_failed: 0,
      last_total: 3,
      in_flight_claims: 0,
      zero_insert_alerts: 0,
    } as AdminMonitoringOverview['scheduler'],
    jobs: { active_count: 0, recent: [] } as AdminMonitoringOverview['jobs'],
    invariants: { runs_total: 1, fail_last: 0, warn_last: 0 } as AdminMonitoringOverview['invariants'],
    snapshot: {
      cut_failures: 0, cut_noop: 0, cuts_produced: 0, partial_total: 0,
      pending_oldest_age_seconds: 0, pending_total: 0, reads_fallback: 0,
      reads_served: 0, ready_match_count: 0, version: 1,
    } as AdminMonitoringOverview['snapshot'],
    open_detections: 0,
    freshness_critical: 0,
    lusr_interior_gaps: 0,
    http: { status_2xx: 0, status_3xx: 0, status_4xx: 0, status_5xx: 0 },
  }
}

describe('computeTabBadges', () => {
  it('overview absent → aucune pastille', () => {
    expect(computeTabBadges(undefined)).toEqual({})
  })

  it('tout sain → aucune pastille', () => {
    expect(computeTabBadges(baseOverview())).toEqual({})
  })

  it('échecs de cycle → pastille destructive sur /admin/sync', () => {
    const o = baseOverview()
    o.scheduler.last_failed = 2
    expect(computeTabBadges(o)['/admin/sync']).toEqual({ count: 2, token: 'destructive' })
  })

  it('jobs actifs sans échec → pastille info pulsée sur /admin/sync', () => {
    const o = baseOverview()
    o.jobs.active_count = 1
    expect(computeTabBadges(o)['/admin/sync']).toEqual({ count: 1, token: 'info', pulse: true })
  })

  it('échec masque les jobs actifs (sévérité)', () => {
    const o = baseOverview()
    o.scheduler.last_failed = 1
    o.jobs.active_count = 3
    expect(computeTabBadges(o)['/admin/sync']).toEqual({ count: 1, token: 'destructive' })
  })

  it('tokens morts → pastille destructive sur /admin/sync (A3.3 : les tokens vivent dans Sync)', () => {
    const o = baseOverview()
    o.tokens = { expired: 1, absent: 1, reauth: 0 } as AdminMonitoringOverview['tokens']
    expect(computeTabBadges(o)['/admin/sync']).toEqual({ count: 2, token: 'destructive' })
  })

  it('data health warnings → pastille warning sur /admin/data (A3.2 : Données)', () => {
    const o = baseOverview()
    o.data_health = { warnings_total: 4 } as AdminMonitoringOverview['data_health']
    expect(computeTabBadges(o)['/admin/data']).toEqual({ count: 4, token: 'warning' })
  })

  it('invariants FAIL → pastille destructive sur /admin/data (masque les warnings)', () => {
    const o = baseOverview()
    o.invariants.fail_last = 1
    o.invariants.warn_last = 2
    expect(computeTabBadges(o)['/admin/data']).toEqual({ count: 1, token: 'destructive' })
  })

  it('invariants WARN seul (pas de critique) → pastille warning sur /admin/data', () => {
    const o = baseOverview()
    o.invariants.warn_last = 2
    expect(computeTabBadges(o)['/admin/data']).toEqual({ count: 2, token: 'warning' })
  })

  it('trous LUSR → pastille warning sur /admin/data (cumulés aux warnings)', () => {
    const o = baseOverview()
    o.lusr_interior_gaps = 3
    expect(computeTabBadges(o)['/admin/data']).toEqual({ count: 3, token: 'warning' })
  })

  it('invariants FAIL masque les trous LUSR (sévérité)', () => {
    const o = baseOverview()
    o.invariants.fail_last = 1
    o.lusr_interior_gaps = 5
    expect(computeTabBadges(o)['/admin/data']).toEqual({ count: 1, token: 'destructive' })
  })

  it('invariants jamais lancés (runs_total=0) → fail_last ignoré', () => {
    const o = baseOverview()
    o.invariants.runs_total = 0
    o.invariants.fail_last = 5 // valeur résiduelle, non significative
    expect(computeTabBadges(o)['/admin/data']).toBeUndefined()
  })

  it('détections open → pastille warning sur /admin/detections (open seul, A2.5)', () => {
    const o = baseOverview()
    o.open_detections = 3
    expect(computeTabBadges(o)['/admin/detections']).toEqual({ count: 3, token: 'warning' })
  })

  it('fraîcheur critical → pastille destructive sur /admin (État, A4.3)', () => {
    const o = baseOverview()
    o.freshness_critical = 2
    expect(computeTabBadges(o)['/admin']).toEqual({ count: 2, token: 'destructive' })
  })
})
