import { describe, expect, it } from 'vitest'
import type { MatchDelay } from '@/lib/api/types'
import { DEBRIEF_COLUMNS, formatOffset, objectiveStatus, stepForColumn } from './debriefDelay.logic'

const base: MatchDelay = { title_slug: 'halo_infinite', match_id: 'm1', citations_players: 0, steps: [] }

describe('debriefDelay.logic', () => {
  it('écrit un écart en +m:ss, avant la fin en −m:ss, sans référence en —', () => {
    expect(formatOffset(0)).toBe('+0:00')
    expect(formatOffset(95.4)).toBe('+1:35')
    expect(formatOffset(-5)).toBe('−0:05')
    expect(formatOffset(undefined)).toBe('—')
    expect(formatOffset(Number.NaN)).toBe('—')
  })

  it('la colonne « rejeu lancé » prend la mise en file ou le début de la construction locale', () => {
    const replay = DEBRIEF_COLUMNS.find((c) => c.key === 'admin.buildqueue.delays_col_replay')!
    const local = { ...base, steps: [{ step: 'replay_build_started' as const, at: 'x', offset_seconds: 30 }] }
    expect(stepForColumn(local, replay)?.offset_seconds).toBe(30)
    expect(stepForColumn(base, replay)).toBeUndefined()
    expect(stepForColumn({ ...base, steps: null }, replay)).toBeUndefined()
  })

  it('objectif : tenu, dépassé, incomplet', () => {
    expect(objectiveStatus({ ...base, objective_met: true })).toBe('met')
    expect(objectiveStatus({ ...base, objective_met: false })).toBe('missed')
    expect(objectiveStatus(base)).toBe('pending')
  })
})
