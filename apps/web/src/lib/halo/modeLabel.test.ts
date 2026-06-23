import { describe, it, expect } from 'vitest'
import { normalizeModeLabel } from './modeLabel'

describe('normalizeModeLabel', () => {
  it('extrait le sous-mode depuis le format technique "Prefix:Mode"', () => {
    expect(normalizeModeLabel('Arena:CTF on Forbidden', 'Forbidden')).toBe('CTF')
  })

  it('strip le suffixe " sur <carte>" (FR) quand mapLabel fourni', () => {
    expect(normalizeModeLabel('Capture du drapeau sur Forbidden', 'Forbidden')).toBe(
      'Capture du drapeau',
    )
  })

  it('strip le suffixe " on <carte>" (EN) quand mapLabel fourni', () => {
    expect(normalizeModeLabel('Slayer on Live Fire', 'Live Fire')).toBe('Slayer')
  })

  it('strip le suffixe map même si le mode porte le nom EN et mapUI le FR (régression "Slayer on Forest sur Forêt")', () => {
    // Le label mode contient le nom de carte ANGLAIS ("Forest") collé, alors
    // que mapLabel est la traduction FR ("Forêt"). Le strip map-spécifique ne
    // matche pas ; le filet générique doit rattraper → "Slayer".
    expect(normalizeModeLabel('Slayer on Forest', 'Forêt')).toBe('Slayer')
  })

  it('strip le suffixe " - Forge" résiduel', () => {
    expect(normalizeModeLabel('Assassin - Forge', null)).toBe('Assassin')
  })

  it('préserve le label déjà traduit sans carte connue', () => {
    expect(normalizeModeLabel('Capture du drapeau', null)).toBe('Capture du drapeau')
  })

  it('retourne null pour un mode vide', () => {
    expect(normalizeModeLabel('', 'Forbidden')).toBeNull()
  })

  it('retourne null pour undefined', () => {
    expect(normalizeModeLabel(undefined, 'Forbidden')).toBeNull()
  })

  it('retourne null pour null', () => {
    expect(normalizeModeLabel(null, null)).toBeNull()
  })
})
