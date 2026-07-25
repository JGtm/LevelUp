import { describe, expect, it } from 'vitest'

import { matchAvgLifeSeconds } from './avgLife'

describe('matchAvgLifeSeconds', () => {
  it('préfère la valeur réelle de l’API au proxy', () => {
    // proxy = 600 / (9 + 1) = 60 s ; valeur réelle = 12 s → 12 gagne.
    expect(matchAvgLifeSeconds({ avg_life_seconds: 12, time_played_seconds: 600, deaths: 9 })).toBe(12)
  })

  it('retombe sur temps joué / (morts + 1) quand la valeur réelle manque', () => {
    expect(matchAvgLifeSeconds({ time_played_seconds: 600, deaths: 9 })).toBe(60)
    expect(matchAvgLifeSeconds({ avg_life_seconds: null, time_played_seconds: 600, deaths: 9 })).toBe(60)
  })

  it('traite une valeur réelle nulle ou négative comme absente', () => {
    expect(matchAvgLifeSeconds({ avg_life_seconds: 0, time_played_seconds: 300, deaths: 4 })).toBe(60)
  })

  it('retourne null sans aucune source exploitable', () => {
    expect(matchAvgLifeSeconds({ deaths: 3 })).toBeNull()
    expect(matchAvgLifeSeconds({ time_played_seconds: 0, deaths: 3 })).toBeNull()
    expect(matchAvgLifeSeconds({ time_played_seconds: null, deaths: 3 })).toBeNull()
  })

  it('gère zéro mort (le match compte pour une seule vie)', () => {
    expect(matchAvgLifeSeconds({ time_played_seconds: 120, deaths: 0 })).toBe(120)
  })
})
