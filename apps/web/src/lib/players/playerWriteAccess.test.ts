import { describe, expect, it } from 'vitest'
import { canWritePlayer } from './playerWriteAccess'

describe('canWritePlayer', () => {
  it('liste inconnue : rien n est masqué', () => {
    expect(canWritePlayer(null, 'bob')).toBe(true)
  })
  it('profil possédé : inscriptible', () => {
    expect(canWritePlayer(['alice'], 'alice')).toBe(true)
  })
  it('profil d un ami (lisible, absent de la liste) : lecture seule', () => {
    expect(canWritePlayer(['alice'], 'bob')).toBe(false)
  })
  it('slug absent : lecture seule quand la liste est connue', () => {
    expect(canWritePlayer(['alice'], undefined)).toBe(false)
  })
})
