import { describe, it, expect } from 'vitest'
import { buildMatchHeadingStr } from './format'

// Les tests unitaires de normalizeModeLabel vivent désormais avec le helper
// partagé : src/lib/halo/modeLabel.test.ts.

describe('buildMatchHeadingStr', () => {
  it('fr : assemble "Mode sur Carte" quand les deux sont présents', () => {
    expect(buildMatchHeadingStr('Forbidden', 'Capture du drapeau', 'fr')).toBe(
      'Capture du drapeau sur Forbidden',
    )
  })

  it('en : assemble "Mode on Carte"', () => {
    expect(buildMatchHeadingStr('Forbidden', 'CTF', 'en')).toBe('CTF on Forbidden')
  })

  it('normalise le mode avant assemblage (strip "Arena:")', () => {
    expect(buildMatchHeadingStr('Forbidden', 'Arena:CTF on Forbidden', 'fr')).toBe(
      'CTF sur Forbidden',
    )
  })

  it('strip le nom de carte EN collé même si mapUI est FR (régression "Slayer on Forest sur Forêt")', () => {
    expect(buildMatchHeadingStr('Forêt', 'Slayer on Forest', 'fr')).toBe('Slayer sur Forêt')
  })

  it('retourne juste la carte si le mode est null — cas dégradé "Forbidden"', () => {
    expect(buildMatchHeadingStr('Forbidden', null, 'fr')).toBe('Forbidden')
    expect(buildMatchHeadingStr('Forbidden', undefined, 'fr')).toBe('Forbidden')
    expect(buildMatchHeadingStr('Forbidden', '', 'fr')).toBe('Forbidden')
  })

  it('retourne juste le mode si la carte est null', () => {
    expect(buildMatchHeadingStr(null, 'Assassin', 'fr')).toBe('Assassin')
  })

  it('retourne chaîne vide si les deux sont null', () => {
    expect(buildMatchHeadingStr(null, null, 'fr')).toBe('')
  })

  it('mode EN non traduit + carte → "CTF sur Forbidden" (EN leak documenté)', () => {
    expect(buildMatchHeadingStr('Forbidden', 'CTF', 'fr')).toBe('CTF sur Forbidden')
  })
})
