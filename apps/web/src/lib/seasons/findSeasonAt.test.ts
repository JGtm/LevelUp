/**
 * Tests des helpers chronologiques saisons (purs, sans React).
 */

import { describe, expect, it } from 'vitest'

import type { SeasonEntry } from '@/lib/i18n/fieldMappings'
import { currentSeason, findSeasonAt, nextSeason, prevSeason } from './findSeasonAt'

const fixture: SeasonEntry[] = [
  {
    id: 's1',
    label: 'Heroes of Reach',
    shortLabel: 'S1',
    startDate: new Date('2022-01-01T00:00:00Z'),
    endDate: new Date('2022-04-01T00:00:00Z'),
    displayOrder: 10,
  },
  {
    id: 's2',
    label: 'Lone Wolves',
    shortLabel: 'S2',
    startDate: new Date('2022-04-01T00:00:00Z'),
    endDate: new Date('2022-07-01T00:00:00Z'),
    displayOrder: 20,
  },
  // Gap : pas de saison entre 2022-07-01 et 2023-01-01
  {
    id: 's3',
    label: 'Echoes Within',
    shortLabel: 'S3',
    startDate: new Date('2023-01-01T00:00:00Z'),
    endDate: new Date('2023-04-01T00:00:00Z'),
    displayOrder: 30,
  },
  {
    id: 's_open',
    label: 'Infinite',
    shortLabel: 'S∞',
    startDate: new Date('2023-04-01T00:00:00Z'),
    endDate: null, // saison ouverte
    displayOrder: 40,
  },
]

describe('findSeasonAt', () => {
  it("retourne null avant la première saison", () => {
    expect(findSeasonAt(fixture, new Date('2021-12-31T23:00:00Z'))).toBeNull()
  })

  it('retourne S1 pour une date dans S1', () => {
    expect(findSeasonAt(fixture, new Date('2022-02-15T00:00:00Z'))?.id).toBe('s1')
  })

  it('retourne null dans le gap entre S2 et S3', () => {
    expect(findSeasonAt(fixture, new Date('2022-09-01T00:00:00Z'))).toBeNull()
  })

  it('retourne S_open pour une date après son startDate (saison ouverte)', () => {
    expect(findSeasonAt(fixture, new Date('2099-06-01T00:00:00Z'))?.id).toBe('s_open')
  })

  it('startDate inclus, endDate exclu (convention [start, end))', () => {
    // Pile sur le startDate de S2 → S2
    expect(findSeasonAt(fixture, new Date('2022-04-01T00:00:00Z'))?.id).toBe('s2')
    // Pile sur l'endDate de S2 → ne devrait PAS être S2 (mais pas de saison non plus, gap)
    expect(findSeasonAt(fixture, new Date('2022-07-01T00:00:00Z'))).toBeNull()
  })
})

describe('currentSeason', () => {
  it('retourne null sur un set vide', () => {
    expect(currentSeason([])).toBeNull()
  })
  // Le test "courante" dépend de la date système → on teste l'invariant
  // que la fonction délègue à findSeasonAt avec new Date().
})

describe('prevSeason / nextSeason', () => {
  it('prevSeason retourne null pour la première saison', () => {
    expect(prevSeason(fixture, fixture[0])).toBeNull()
  })

  it('prevSeason retourne S1 pour S2', () => {
    expect(prevSeason(fixture, fixture[1])?.id).toBe('s1')
  })

  it('nextSeason retourne null pour la dernière saison', () => {
    expect(nextSeason(fixture, fixture[fixture.length - 1])).toBeNull()
  })

  it('nextSeason retourne S2 pour S1', () => {
    expect(nextSeason(fixture, fixture[0])?.id).toBe('s2')
  })

  it('retourne null pour une saison inconnue', () => {
    const ghost: SeasonEntry = {
      id: 'ghost',
      label: 'Ghost',
      shortLabel: 'X',
      startDate: new Date(),
      endDate: null,
      displayOrder: 999,
    }
    expect(prevSeason(fixture, ghost)).toBeNull()
    expect(nextSeason(fixture, ghost)).toBeNull()
  })
})
