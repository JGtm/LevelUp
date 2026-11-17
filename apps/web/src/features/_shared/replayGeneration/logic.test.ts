/**
 * Tests — logique pure de la génération d'un rejeu : une ligne par état et par raison, aucune action quand elle
 * est impossible, relecture seulement pendant l'attente.
 */
import { describe, it, expect } from 'vitest'
import { generationInFlight, generationView } from './logic'
import type { ReplayGenerationReason, ReplayGenerationState, ReplayGenerationStatus } from './queries'

const st = (state: ReplayGenerationState, rest: Partial<ReplayGenerationStatus> = {}): ReplayGenerationStatus => ({
  state,
  ...rest,
})

describe('generationView', () => {
  it('sans état lu : rien', () => {
    expect(generationView(undefined)).toEqual({ action: null, pending: null, messageKey: null })
  })

  it('generatable : générer', () => {
    expect(generationView(st('generatable'))).toEqual({ action: 'generate', pending: null, messageKey: 'generatable' })
  })

  it('queued : attente, jamais de bouton ; la fin de la partie et la pause priment sur la position', () => {
    expect(generationView(st('queued', { position: 2 }))).toEqual({ action: null, pending: 'queued', messageKey: 'queued' })
    expect(generationView(st('queued'))).toEqual({ action: null, pending: 'queued', messageKey: 'queuedNoPosition' })
    expect(generationView(st('queued', { position: 1, waiting_for_game: true })).messageKey).toBe('waitingForGame')
    expect(generationView(st('queued', { position: 1, paused: true })).messageKey).toBe('paused')
    expect(generationView(st('queued', { position: 1, paused: true, waiting_for_game: true })).messageKey).toBe(
      'waitingForGame',
    )
  })

  it('running : téléchargement puis construction, jamais de bouton', () => {
    expect(generationView(st('running', { phase: 'downloading' }))).toEqual({
      action: null,
      pending: 'downloading',
      messageKey: 'downloading',
    })
    expect(generationView(st('running', { phase: 'building' }))).toEqual({
      action: null,
      pending: 'building',
      messageKey: 'building',
    })
    expect(generationView(st('running')).pending).toBe('building')
  })

  it('failed : réessayer seulement si le serveur le dit, quelle que soit la raison', () => {
    const retryable: ReplayGenerationReason[] = ['film_not_ready', 'fetch_failed', 'token_unavailable', 'build_failed']
    for (const reason of retryable) {
      expect(generationView(st('failed', { reason, retryable: true }))).toEqual({
        action: 'retry',
        pending: null,
        messageKey: 'failed',
      })
    }
    const final: ReplayGenerationReason[] = ['map_not_in_catalog', 'unknown_film_key']
    for (const reason of final) {
      expect(generationView(st('failed', { reason, retryable: false }))).toEqual({
        action: null,
        pending: null,
        messageKey: 'failedFinal',
      })
    }
    expect(generationView(st('failed')).action).toBeNull()
  })

  it('film_unavailable : une phrase, aucun bouton', () => {
    expect(generationView(st('film_unavailable'))).toEqual({
      action: null,
      pending: null,
      messageKey: 'filmUnavailable',
    })
  })

  it('not_supported : une phrase par raison, aucun bouton', () => {
    expect(generationView(st('not_supported', { reason: 'mode_unmanaged' }))).toEqual({
      action: null,
      pending: null,
      messageKey: 'modeUnmanaged',
    })
    expect(generationView(st('not_supported', { reason: 'match_unknown' })).messageKey).toBe('matchUnknown')
    expect(generationView(st('not_supported', { reason: 'no_token' })).messageKey).toBe('noToken')
    expect(generationView(st('not_supported', { reason: 'no_token' })).action).toBeNull()
    expect(generationView(st('not_supported'))).toEqual({ action: null, pending: null, messageKey: null })
  })

  it('ready : ouvrir le rejeu', () => {
    expect(generationView(st('ready'))).toEqual({ action: 'open', pending: null, messageKey: 'ready' })
  })

  it('un état inconnu de ce client : rien, aucun bouton', () => {
    expect(generationView({ state: 'archived' } as unknown as ReplayGenerationStatus)).toEqual({
      action: null,
      pending: null,
      messageKey: null,
    })
  })

  it('seuls generatable et ready offrent une action quand rien n’est réessayable', () => {
    const states: ReplayGenerationState[] = [
      'generatable', 'queued', 'running', 'failed', 'film_unavailable', 'not_supported', 'ready',
    ]
    const withAction = states.filter((s) => generationView(st(s, { retryable: false })).action !== null)
    expect(withAction).toEqual(['generatable', 'ready'])
  })
})

describe('generationInFlight', () => {
  it('vrai en file et en cours seulement', () => {
    expect(generationInFlight(undefined)).toBe(false)
    expect(generationInFlight(st('queued'))).toBe(true)
    expect(generationInFlight(st('running'))).toBe(true)
    for (const s of ['generatable', 'failed', 'film_unavailable', 'not_supported', 'ready'] as const) {
      expect(generationInFlight(st(s))).toBe(false)
    }
  })
})
