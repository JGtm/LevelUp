/**
 * Tests — placeOwnerResolver : la place que le film écrit (`roster[].filmIndex`) désigne UN joueur, par la seule table
 * du roster ; plusieurs identités sur un index se départagent par la présence, sinon personne.
 */
import { describe, expect, it } from 'vitest'

import { placeOwnerResolver, type PlaceRosterEntry } from './placeOwner'

const entry = (filmIndex: number, over: Partial<PlaceRosterEntry> = {}): PlaceRosterEntry => ({
  filmIndex,
  xuid: `x${filmIndex}`,
  name: `Joueur${filmIndex}`,
  presence: [],
  ...over,
})

describe('placeOwnerResolver', () => {
  it('rend le xuid de l’unique identité de la place', () => {
    const owner = placeOwnerResolver([entry(0), entry(1)])
    expect(owner(0, 10)).toBe('x0')
    expect(owner(1, 10)).toBe('x1')
  })

  it('désigne un bot par sa clé de roster `bot:<nom>`, sans xuid', () => {
    const owner = placeOwnerResolver([entry(0), entry(2, { xuid: '', bot: true, name: 'Marc' })])
    expect(owner(2, 5)).toBe('bot:Marc')
  })

  it('une place que le roster ne porte pas ne revient à personne', () => {
    expect(placeOwnerResolver([entry(0)])(7, 10)).toBeNull()
  })

  it('deux identités sur une place : celle dont la présence couvre l’image, bornes comprises', () => {
    const owner = placeOwnerResolver([
      entry(0, { xuid: 'a1', presence: [{ from: 0, to: 49 }] }),
      entry(0, { xuid: 'c1', presence: [{ from: 50, to: 200 }] }),
    ])
    expect(owner(0, 49)).toBe('a1')
    expect(owner(0, 50)).toBe('c1')
  })

  it('deux identités sur une place et aucune présente (ou deux) : personne, jamais les deux', () => {
    const seuls = placeOwnerResolver([
      entry(0, { xuid: 'a1', presence: [{ from: 0, to: 10 }] }),
      entry(0, { xuid: 'c1', presence: [{ from: 50, to: 60 }] }),
    ])
    expect(seuls(0, 30)).toBeNull()
    const deux = placeOwnerResolver([
      entry(0, { xuid: 'a1', presence: [{ from: 0, to: 100 }] }),
      entry(0, { xuid: 'c1', presence: [{ from: 0, to: 100 }] }),
    ])
    expect(deux(0, 30)).toBeNull()
  })

  it('la borne haute de la présence est `toMax` quand elle existe', () => {
    const owner = placeOwnerResolver([
      entry(0, { xuid: 'a1', presence: [{ from: 0, to: 10, toMax: 30 }] }),
      entry(0, { xuid: 'c1', presence: [{ from: 31, to: 100 }] }),
    ])
    expect(owner(0, 25)).toBe('a1')
  })
})
