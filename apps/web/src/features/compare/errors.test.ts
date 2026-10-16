import { describe, it, expect } from 'vitest'
import { isCompareNotFound } from './errors'

describe('isCompareNotFound', () => {
  it.each([
    ['404 avec corps JSON', { status: 404, code: 'player_not_found', message: 'player not found' }],
    ['404 sans corps', { status: 404, code: 'unknown_error', message: 'Erreur HTTP 404' }],
    ['code seul', { code: 'player_not_found' }],
  ])('%s : introuvable', (_label, err) => {
    expect(isCompareNotFound(err)).toBe(true)
  })

  it.each([
    ['erreur serveur', { status: 500, code: 'compare_error', message: 'internal error' }],
    ['validation', { status: 400, code: 'invalid_body', message: 'invalid body' }],
    ['texte contenant 404', { status: 500, code: 'x', message: 'ligne 404 du serveur' }],
    ['erreur réseau', new Error('Failed to fetch')],
    ['valeur nulle', null],
  ])('%s : pas un joueur introuvable', (_label, err) => {
    expect(isCompareNotFound(err)).toBe(false)
  })
})
