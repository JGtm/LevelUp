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

  it('data health warnings → pastille warning sur /admin/data-quality', () => {
    const o = baseOverview()
    o.data_health = { warnings_total: 4 } as AdminMonitoringOverview['data_health']
    expect(computeTabBadges(o)['/admin/data-quality']).toEqual({ count: 4, token: 'warning' })
  })

  it('invariants FAIL + tokens en erreur → pastille destructive cumulée sur /admin/system', () => {
    const o = baseOverview()
    o.invariants.fail_last = 1
    o.tokens = { expired: 1, absent: 1, reauth: 0 } as AdminMonitoringOverview['tokens']
    expect(computeTabBadges(o)['/admin/system']).toEqual({ count: 3, token: 'destructive' })
  })

  it('invariants WARN seul (pas de critique) → pastille warning sur /admin/system', () => {
    const o = baseOverview()
    o.invariants.warn_last = 2
    expect(computeTabBadges(o)['/admin/system']).toEqual({ count: 2, token: 'warning' })
  })

  it('invariants jamais lancés (runs_total=0) → fail_last ignoré', () => {
    const o = baseOverview()
    o.invariants.runs_total = 0
    o.invariants.fail_last = 5 // valeur résiduelle, non significative
    expect(computeTabBadges(o)['/admin/system']).toBeUndefined()
  })
})
