import { describe, expect, it } from 'vitest'

import { displayPlayerName, isXuidLike, maskedPlayerLabel, stripBotSuffix } from './displayName'

describe('isXuidLike', () => {
  it('détecte le format xuid(...)', () => {
    expect(isXuidLike('xuid(2533274800000001)')).toBe(true)
    expect(isXuidLike('XUID(123)')).toBe(true)
  })
  it('détecte un identifiant numérique long (>= 15 chiffres)', () => {
    expect(isXuidLike('2533274800000001')).toBe(true) // 16 chiffres
    expect(isXuidLike('123456789012345')).toBe(true) // 15 chiffres
  })
  it('ne masque PAS un gamertag légitime', () => {
    expect(isXuidLike('JGtm')).toBe(false)
    expect(isXuidLike('Madina97294')).toBe(false)
    expect(isXuidLike('Player 123')).toBe(false)
    expect(isXuidLike('1234')).toBe(false) // court → gamertag possible
  })
  it('ne vise pas les bots bid(N.0) (résolus côté serveur)', () => {
    expect(isXuidLike('bid(3.0)')).toBe(false)
  })
})

describe('maskedPlayerLabel', () => {
  it('utilise les 4 derniers chars', () => {
    expect(maskedPlayerLabel('2533274800000001')).toBe('Joueur 0001')
  })
  it('retombe sur la valeur entière si trop courte', () => {
    expect(maskedPlayerLabel('xy')).toBe('Joueur xy')
  })
})

describe('displayPlayerName', () => {
  it('retourne le gamertag quand il est valide', () => {
    expect(displayPlayerName('JGtm', '2533274800000001')).toBe('JGtm')
  })
  it('masque quand le gamertag est vide', () => {
    expect(displayPlayerName('', '2533274800000001')).toBe('Joueur 0001')
    expect(displayPlayerName(null, '2533274800000002')).toBe('Joueur 0002')
    expect(displayPlayerName(undefined, '2533274800000003')).toBe('Joueur 0003')
  })
  it('masque quand un xuid brut a fuité dans le champ gamertag (donnée corrompue)', () => {
    expect(displayPlayerName('2533274800000004', '2533274800000004')).toBe(
      'Joueur 0004',
    )
    expect(displayPlayerName('xuid(2533274800000005)', '2533274800000005')).toBe(
      'Joueur 0005',
    )
  })
  it('retourne "Joueur inconnu" sans gamertag ni xuid', () => {
    expect(displayPlayerName(null, null)).toBe('Joueur inconnu')
    expect(displayPlayerName('', '')).toBe('Joueur inconnu')
  })
  it('INVARIANT : ne retourne JAMAIS une valeur au format xuid brut', () => {
    const cases: Array<[string | null, string | null]> = [
      ['2533274800000001', '2533274800000001'],
      ['xuid(123456789012345)', 'xuid(123456789012345)'],
      [null, '2533274800000009'],
      ['', '999999999999999'],
      ['ValidGamertag', '2533274800000001'],
    ]
    for (const [gt, xu] of cases) {
      expect(isXuidLike(displayPlayerName(gt, xu))).toBe(false)
    }
  })

  describe('suffixe " [bot]" (marqueur de donnée killsource / schéma 36)', () => {
    it('retire le suffixe final', () => {
      expect(displayPlayerName('343 Razzle [bot]', 'bid(3.0)')).toBe('343 Razzle')
    })
    it('laisse un gamertag SANS suffixe inchangé', () => {
      expect(displayPlayerName('JGtm', '2533274800000001')).toBe('JGtm')
    })
    it('ne touche pas un suffixe en PLEIN MILIEU du nom', () => {
      expect(displayPlayerName('343 [bot] Razzle', '2533274800000001')).toBe('343 [bot] Razzle')
    })
    it('ne retire que la casse EXACTE " [bot]" (différente = inchangée)', () => {
      expect(displayPlayerName('343 Razzle [BOT]', '2533274800000001')).toBe('343 Razzle [BOT]')
      expect(displayPlayerName('343 Razzle [Bot]', '2533274800000001')).toBe('343 Razzle [Bot]')
    })
  })
})

describe('stripBotSuffix', () => {
  it('retire le suffixe final ` [bot]`', () => {
    expect(stripBotSuffix('343 Razzle [bot]')).toBe('343 Razzle')
  })
  it('laisse une chaîne sans suffixe inchangée', () => {
    expect(stripBotSuffix('343 Razzle')).toBe('343 Razzle')
  })
  it('ne touche pas une occurrence en plein milieu', () => {
    expect(stripBotSuffix('343 [bot] Razzle')).toBe('343 [bot] Razzle')
  })
})
