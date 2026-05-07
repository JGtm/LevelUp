/**
 * Tests Vitest pour teamNames — résolution team_id → nom officiel Halo.
 * Aligné sur le port 1:1 du TEAM_MAP Python (src/config.py:380-390).
 */
import { describe, it, expect } from 'vitest'

import { parseTeamSideID, resolveTeamName } from './teamNames'

describe('parseTeamSideID', () => {
  it('extrait l\'entier depuis "t{N}" pour 0..9', () => {
    expect(parseTeamSideID('t0')).toBe(0)
    expect(parseTeamSideID('t1')).toBe(1)
    expect(parseTeamSideID('t8')).toBe(8)
    expect(parseTeamSideID('t10')).toBe(10)
  })

  it('retourne null pour les valeurs invalides', () => {
    expect(parseTeamSideID(null)).toBeNull()
    expect(parseTeamSideID(undefined)).toBeNull()
    expect(parseTeamSideID('')).toBeNull()
    expect(parseTeamSideID('invalid')).toBeNull()
    expect(parseTeamSideID('eagle')).toBeNull()
    expect(parseTeamSideID('t')).toBeNull()
    // Format proche mais pas exact : préfixe T majuscule rejeté.
    expect(parseTeamSideID('T0')).toBeNull()
    // Suffixes non numériques rejetés.
    expect(parseTeamSideID('t0a')).toBeNull()
  })
})

describe('resolveTeamName', () => {
  it('mappe les 9 team_ids Halo Infinite vers leurs noms officiels', () => {
    expect(resolveTeamName('t0')).toBe('Eagle')
    expect(resolveTeamName('t1')).toBe('Cobra')
    expect(resolveTeamName('t2')).toBe('Hades')
    expect(resolveTeamName('t3')).toBe('Valkyrie')
    expect(resolveTeamName('t4')).toBe('Rampart')
    expect(resolveTeamName('t5')).toBe('Cutlass')
    expect(resolveTeamName('t6')).toBe('Valor')
    expect(resolveTeamName('t7')).toBe('Hazard')
    expect(resolveTeamName('t8')).toBe('Observer')
  })

  it('retourne null pour un team_id valide mais hors map', () => {
    // Le caller fait fallback "Équipe N" pour le rendu.
    expect(resolveTeamName('t9')).toBeNull()
    expect(resolveTeamName('t100')).toBeNull()
  })

  it('retourne null pour un team_side malformé', () => {
    expect(resolveTeamName(null)).toBeNull()
    expect(resolveTeamName(undefined)).toBeNull()
    expect(resolveTeamName('')).toBeNull()
    expect(resolveTeamName('invalid')).toBeNull()
  })
})
