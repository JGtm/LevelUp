import { describe, it, expect } from 'vitest'
import { normalizeModeLabel, buildMatchHeadingStr } from './format'

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

  it('strip le suffixe " - Forge" résiduel', () => {
    // Go normalise déjà pair_name avant d'envoyer mode_ui — ce cas couvre un
    // label déjà partiellement normalisé avec suffixe Forge encore présent.
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

  it('retourne juste la carte si le mode est null — cas dégradé "Forbidden"', () => {
    // Régression : si mode_ui est null/vide depuis l'API, le titre ne doit pas
    // être vide mais afficher au moins la carte.
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

  it('mode FR déjà traduit + carte → titre complet correct', () => {
    // Cas nominal après fix mode_name_tr dans GetMatchMeta.
    expect(buildMatchHeadingStr('Forbidden', 'Capture du drapeau', 'fr')).toBe(
      'Capture du drapeau sur Forbidden',
    )
  })

  it('mode EN non traduit + carte → "CTF sur Forbidden" (EN leak documenté)', () => {
    // Cas de dégradation : mode_name_tr absent → EN affiché, toléré.
    expect(buildMatchHeadingStr('Forbidden', 'CTF', 'fr')).toBe('CTF sur Forbidden')
  })
})
