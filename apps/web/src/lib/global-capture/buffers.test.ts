import { describe, expect, it, beforeEach } from 'vitest'
import {
  extractStackFromArgs,
  getRecentConsoleEntries,
  getRecentFailedRequests,
  recordConsole,
  recordFailedRequest,
  resetCaptureBuffersForTests,
  stringifyConsoleArgs,
} from './buffers'

beforeEach(() => {
  resetCaptureBuffersForTests()
})

describe('console buffer', () => {
  it('borne le buffer à 20 entrées (FIFO)', () => {
    for (let i = 0; i < 25; i++) {
      recordConsole({ level: 'error', message: `msg-${i}`, timestamp: i })
    }
    const entries = getRecentConsoleEntries()
    expect(entries).toHaveLength(20)
    expect(entries[0]?.message).toBe('msg-5')
    expect(entries[19]?.message).toBe('msg-24')
  })

  it('renvoie un snapshot indépendant du buffer interne', () => {
    recordConsole({ level: 'error', message: 'a', timestamp: 1 })
    const a = getRecentConsoleEntries()
    recordConsole({ level: 'warn', message: 'b', timestamp: 2 })
    expect(a).toHaveLength(1)
    expect(getRecentConsoleEntries()).toHaveLength(2)
  })
})

describe('failed-request buffer', () => {
  it('borne à 5 et FIFO', () => {
    for (let i = 0; i < 8; i++) {
      recordFailedRequest({
        url: `/api/v1/p${i}`,
        method: 'GET',
        status: 500,
        timestamp: i,
      })
    }
    const reqs = getRecentFailedRequests()
    expect(reqs).toHaveLength(5)
    expect(reqs[0]?.url).toBe('/api/v1/p3')
    expect(reqs[4]?.url).toBe('/api/v1/p7')
  })

  it('strippe systématiquement la query string (anti-leak PII)', () => {
    recordFailedRequest({
      url: '/api/v1/players/Foo?token=secret&period=7d',
      method: 'GET',
      status: 500,
      timestamp: 1,
    })
    const [first] = getRecentFailedRequests()
    expect(first?.url).toBe('/api/v1/players/Foo')
    expect(first?.url).not.toContain('token')
    expect(first?.url).not.toContain('secret')
  })

  it('laisse les URLs sans query string intactes', () => {
    recordFailedRequest({
      url: 'https://api.github.com/search/issues',
      method: 'GET',
      status: 403,
      timestamp: 1,
    })
    expect(getRecentFailedRequests()[0]?.url).toBe('https://api.github.com/search/issues')
  })
})

describe('extractStackFromArgs', () => {
  it("renvoie .stack si l'argument est une Error", () => {
    const err = new Error('boom')
    expect(extractStackFromArgs([err])).toBe(err.stack)
  })

  it("renvoie undefined si aucun argument n'est une Error", () => {
    expect(extractStackFromArgs(['foo', 42, { stack: 'fake' }])).toBeUndefined()
  })

  it('prend la première Error rencontrée', () => {
    const e1 = new Error('first')
    const e2 = new Error('second')
    expect(extractStackFromArgs(['ctx', e1, e2])).toBe(e1.stack)
  })
})

describe('stringifyConsoleArgs', () => {
  it('joint les strings', () => {
    expect(stringifyConsoleArgs(['a', 'b', 'c'])).toBe('a b c')
  })

  it('extrait .message des Error', () => {
    expect(stringifyConsoleArgs(['ctx:', new Error('boom')])).toBe('ctx: boom')
  })

  it('JSON.stringify les objets et tombe sur String() en cas d\'échec', () => {
    expect(stringifyConsoleArgs([{ a: 1 }])).toBe('{"a":1}')
    const cyclic: Record<string, unknown> = {}
    cyclic.self = cyclic
    expect(stringifyConsoleArgs([cyclic])).toContain('object')
  })
})

describe('resetCaptureBuffersForTests', () => {
  it('vide les deux buffers', () => {
    recordConsole({ level: 'error', message: 'x', timestamp: 1 })
    recordFailedRequest({ url: '/a', method: 'GET', status: 500, timestamp: 1 })
    resetCaptureBuffersForTests()
    expect(getRecentConsoleEntries()).toHaveLength(0)
    expect(getRecentFailedRequests()).toHaveLength(0)
  })
})
