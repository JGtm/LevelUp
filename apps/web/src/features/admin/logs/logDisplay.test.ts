import { describe, expect, it } from 'vitest'

import type { AdminLogEntry } from '@/lib/api/types'
import { flattenLogFields, logEntryDetail, logEntryText, logLevelStatus } from './logDisplay'

describe('logLevelStatus', () => {
  it('mappe error/warn et neutralise le reste', () => {
    expect(logLevelStatus('error')).toBe('error')
    expect(logLevelStatus('WARN')).toBe('warning')
    expect(logLevelStatus('warning')).toBe('warning')
    expect(logLevelStatus('info')).toBe('idle')
    expect(logLevelStatus('debug')).toBe('idle')
    expect(logLevelStatus('unknown')).toBe('idle')
  })
})

describe('flattenLogFields', () => {
  it('produit des chips clé=valeur triées, objets sérialisés, valeurs tronquées', () => {
    const chips = flattenLogFields({
      zebra: 1,
      alpha: 'x',
      nested: { a: 1 },
      long: 'y'.repeat(100),
    })
    expect(chips[0]).toBe('alpha=x')
    expect(chips).toContain('nested={"a":1}')
    expect(chips).toContain('zebra=1')
    const long = chips.find((c) => c.startsWith('long='))
    expect(long).toBeDefined()
    expect(long!.length).toBeLessThanOrEqual('long='.length + 48)
    expect(long!.endsWith('…')).toBe(true)
  })

  it('fields absent → vide ; null → "null"', () => {
    expect(flattenLogFields(undefined)).toEqual([])
    expect(flattenLogFields({ k: null })).toEqual(['k=null'])
  })
})

describe('logEntryText / logEntryDetail', () => {
  it('texte principal : msg > raw > err', () => {
    expect(logEntryText({ level: 'info', msg: 'hello' } as AdminLogEntry)).toBe('hello')
    expect(logEntryText({ level: 'unknown', raw: 'panic!' } as AdminLogEntry)).toBe('panic!')
    expect(logEntryText({ level: 'error', err: 'boom' } as AdminLogEntry)).toBe('boom')
  })

  it('détail : raw brut si présent, sinon JSON indenté (jamais de crash)', () => {
    expect(logEntryDetail({ level: 'unknown', raw: 'ligne brute' } as AdminLogEntry)).toBe('ligne brute')
    const detail = logEntryDetail({ level: 'info', msg: 'm', fields: { a: 1 } } as AdminLogEntry)
    expect(detail).toContain('"msg": "m"')
  })
})
