/**
 * usageMetricKinds.test.ts — l'encre des rôles d'objectif (D9, plan Emprise, 2026-09-26).
 */
import { describe, expect, it } from 'vitest'

import { roleToken } from './usageMetricKinds'

describe('encres des rôles d’objectif', () => {
  it('les trois rôles prennent la famille objective-role-*', () => {
    expect(roleToken('take')).toBe('objective-role-take')
    expect(roleToken('defend')).toBe('objective-role-defend')
    expect(roleToken('hold')).toBe('objective-role-hold')
  })

  it('un rôle non catalogué prend le repli neutre', () => {
    expect(roleToken('autre')).toBe('chart-series-4')
  })
})
