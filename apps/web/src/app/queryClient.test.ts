/**
 * Politique de rejeu des requêtes (plan perf 2026-09-23, D3.3) : une passerelle coupée
 * (502) ou en délai dépassé (504) n'est jamais rejouée — le 2026-09-23, chaque rejeu d'une
 * page Escouade tronquée relançait le même calcul lourd côté serveur. Les erreurs serveur
 * (500) et la base occupée (503, transitoire) gardent leurs deux nouvelles tentatives.
 */
import { describe, expect, it } from 'vitest'
import { queryClient } from './queryClient'

type RetryDecision = (failureCount: number, error: unknown) => boolean

function retryPolicy(): RetryDecision {
  const retry = queryClient.getDefaultOptions().queries?.retry
  expect(typeof retry).toBe('function')
  return retry as RetryDecision
}

describe('queryClient — rejeu des requêtes', () => {
  it('ne rejoue ni un 502 ni un 504 (passerelle)', () => {
    const retry = retryPolicy()
    expect(retry(0, { status: 502, code: 'unknown_error', retryable: true })).toBe(false)
    expect(retry(0, { status: 504, code: 'unknown_error', retryable: true })).toBe(false)
  })

  it('rejoue deux fois un 500 et un 503 (base occupée), pas une troisième', () => {
    const retry = retryPolicy()
    for (const status of [500, 503]) {
      expect(retry(0, { status, retryable: true })).toBe(true)
      expect(retry(1, { status, retryable: true })).toBe(true)
      expect(retry(2, { status, retryable: true })).toBe(false)
    }
  })

  it('ne rejoue jamais une erreur client (4xx)', () => {
    const retry = retryPolicy()
    for (const status of [400, 401, 403, 404, 409, 422]) {
      expect(retry(0, { status, retryable: false })).toBe(false)
    }
  })

  it('rejoue une erreur réseau sans statut (deux fois)', () => {
    const retry = retryPolicy()
    const network = new TypeError('Failed to fetch')
    expect(retry(0, network)).toBe(true)
    expect(retry(1, network)).toBe(true)
    expect(retry(2, network)).toBe(false)
  })
})
