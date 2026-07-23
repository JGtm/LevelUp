import { describe, it, expect } from 'vitest'

import type { AdminActionJournalEntry, SchedulerSnapshot } from '@/lib/api/types'
import { describeActionRun, describeLastCycle, lastCycleLabelKey } from './actionJournalDisplay'

function entry(partial: Partial<AdminActionJournalEntry>): AdminActionJournalEntry {
  return { action: 'sync_cycle', last_run_at: '2026-07-23T10:00:00Z', outcome: 'ok', trigger: 'manual', ...partial }
}

function snapshot(partial: Partial<SchedulerSnapshot>): SchedulerSnapshot {
  return {
    last_cycle_at: '2026-07-23T10:00:00Z',
    interval_minutes: 15,
    pool_size: 1,
    players: [],
    gate: { inflight_watcher: 0, inflight_gate: 0, granted_total: 0, coalesced_total: 0, stale_count: 0 },
    since_boot: true,
    ...partial,
  }
}

describe('describeActionRun (C2)', () => {
  it('entrée absente → null (jamais exécutée)', () => {
    expect(describeActionRun(undefined)).toBeNull()
  })

  it('last_run_at vide → null', () => {
    expect(describeActionRun(entry({ last_run_at: '' }))).toBeNull()
  })

  it('déclencheur automatique (tick) mappé sur trigger_cron', () => {
    const d = describeActionRun(entry({ trigger: 'tick' }))
    expect(d?.triggerKey).toBe('admin.actions.trigger_cron')
    expect(d?.failed).toBe(false)
  })

  it('déclencheur manuel + issue en échec', () => {
    const d = describeActionRun(entry({ trigger: 'manual', outcome: 'error' }))
    expect(d?.triggerKey).toBe('admin.actions.trigger_manual')
    expect(d?.failed).toBe(true)
    expect(d?.at).toBe('2026-07-23T10:00:00Z')
  })
})

describe('describeLastCycle (C1)', () => {
  it('snapshot absent → none (aucune donnée connue)', () => {
    expect(describeLastCycle(undefined)).toEqual({ kind: 'none', at: '' })
  })

  it('last_cycle_at zéro Go (0001-…) → none', () => {
    expect(describeLastCycle(snapshot({ last_cycle_at: '0001-01-01T00:00:00Z' }))).toEqual({ kind: 'none', at: '' })
  })

  it('cycle depuis le boot → live', () => {
    expect(describeLastCycle(snapshot({ since_boot: true }))).toEqual({
      kind: 'live',
      at: '2026-07-23T10:00:00Z',
    })
  })

  it('snapshot réhydraté (aucun cycle depuis le boot) → previous, daté', () => {
    expect(describeLastCycle(snapshot({ since_boot: false }))).toEqual({
      kind: 'previous',
      at: '2026-07-23T10:00:00Z',
    })
  })

  it('libellés distincts live vs previous', () => {
    expect(lastCycleLabelKey('live')).toBe('admin.convergence.last_cycle')
    expect(lastCycleLabelKey('previous')).toBe('admin.convergence.last_cycle_prev_boot')
  })
})
