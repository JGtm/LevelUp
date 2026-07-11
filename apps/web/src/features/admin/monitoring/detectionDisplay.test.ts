import { describe, expect, it } from 'vitest'

import {
  detectionLevelToken,
  detectionStatusMeta,
  filterDetectionsByStatus,
} from './detectionDisplay'

describe('detectionStatusMeta', () => {
  it('mappe chaque statut vers son libellé + token', () => {
    expect(detectionStatusMeta('open')).toEqual({
      labelKey: 'admin.detections.status_open',
      token: 'warning',
    })
    expect(detectionStatusMeta('acked').token).toBe('info')
    expect(detectionStatusMeta('resolved').token).toBe('success')
    // Sourdine : neutre (aucun token).
    expect(detectionStatusMeta('muted').token).toBeUndefined()
  })

  it('retombe sur open pour un statut inconnu', () => {
    expect(detectionStatusMeta('bogus').labelKey).toBe('admin.detections.status_open')
  })
})

describe('detectionLevelToken', () => {
  it('mappe le niveau vers un token de badge', () => {
    expect(detectionLevelToken('ERROR')).toBe('destructive')
    expect(detectionLevelToken('warn')).toBe('warning')
    expect(detectionLevelToken('WARNING')).toBe('warning')
    expect(detectionLevelToken('INFO')).toBe('info')
  })
})

describe('filterDetectionsByStatus', () => {
  const rows = [
    { status: 'open', fingerprint: 'a' },
    { status: 'muted', fingerprint: 'b' },
    { status: 'open', fingerprint: 'c' },
    { status: 'resolved', fingerprint: 'd' },
  ]

  it('all ne filtre rien', () => {
    expect(filterDetectionsByStatus(rows, 'all')).toHaveLength(4)
  })

  it('filtre par statut exact', () => {
    expect(filterDetectionsByStatus(rows, 'open').map((r) => r.fingerprint)).toEqual(['a', 'c'])
    expect(filterDetectionsByStatus(rows, 'muted')).toHaveLength(1)
    expect(filterDetectionsByStatus(rows, 'resolved')).toHaveLength(1)
  })
})
