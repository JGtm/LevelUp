import { describe, expect, it } from 'vitest'

import { replayLoadState } from './replayLoadState'

const base = { isLoading: false, hasTracks: false, error: null, identityPending: false }

describe('replayLoadState', () => {
  it('chargement en cours', () => {
    expect(replayLoadState({ ...base, isLoading: true })).toBe('loading')
  })
  it('artefact avec pistes et vue du match tranchée : prêt', () => {
    expect(replayLoadState({ ...base, hasTracks: true })).toBe('ready')
  })
  it('artefact servi mais vue du match en attente : toujours en chargement', () => {
    expect(replayLoadState({ ...base, hasTracks: true, identityPending: true })).toBe('loading')
  })
  it('404 : le match n’a pas de rejeu, sans attendre la vue du match', () => {
    expect(replayLoadState({ ...base, error: { status: 404 }, identityPending: true })).toBe('empty')
  })
  it('artefact sans piste : vide', () => {
    expect(replayLoadState(base)).toBe('empty')
  })
  it('503 ou réseau : panne, pas absence', () => {
    expect(replayLoadState({ ...base, error: { status: 503 } })).toBe('error')
    expect(replayLoadState({ ...base, error: new TypeError('fetch') })).toBe('error')
  })
})
