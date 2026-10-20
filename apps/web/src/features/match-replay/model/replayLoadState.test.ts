import { describe, expect, it } from 'vitest'

import { replayLoadState } from './replayLoadState'

describe('replayLoadState', () => {
  it('chargement en cours', () => {
    expect(replayLoadState({ isLoading: true, hasTracks: false, error: null })).toBe('loading')
  })
  it('artefact avec pistes : prêt', () => {
    expect(replayLoadState({ isLoading: false, hasTracks: true, error: null })).toBe('ready')
  })
  it('404 : le match n’a pas de rejeu', () => {
    expect(replayLoadState({ isLoading: false, hasTracks: false, error: { status: 404 } })).toBe('empty')
  })
  it('artefact sans piste : vide', () => {
    expect(replayLoadState({ isLoading: false, hasTracks: false, error: null })).toBe('empty')
  })
  it('503 ou réseau : panne, pas absence', () => {
    expect(replayLoadState({ isLoading: false, hasTracks: false, error: { status: 503 } })).toBe('error')
    expect(replayLoadState({ isLoading: false, hasTracks: false, error: new TypeError('fetch') })).toBe('error')
  })
})
