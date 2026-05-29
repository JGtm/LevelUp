/**
 * Tests — skillTierBands.frameToTier : cadrage de l'axe Y sur la/les bande(s)
 * de palier LUSR (évite l'axe qui repart de 0 et le zoom trop éloigné).
 */
import { describe, it, expect } from 'vitest'
import { frameToTier } from './skillTierBands'

describe('frameToTier', () => {
  it('cadre sur un seul palier LUSR contenant les données (Argent)', () => {
    expect(frameToTier(1250, 1350)).toEqual({ min: 1200, max: 1400 })
  })

  it('cadre sur deux paliers quand les données franchissent une frontière', () => {
    expect(frameToTier(1380, 1450)).toEqual({ min: 1200, max: 1600 })
  })

  it('hors plage des paliers (CSR bas) : arrondit à un pas propre de 100', () => {
    expect(frameToTier(820, 880)).toEqual({ min: 800, max: 900 })
  })

  it('garde-fou anti-dégénérescence : données plates pile sur un multiple', () => {
    expect(frameToTier(800, 800)).toEqual({ min: 700, max: 900 })
  })

  it('palier Onyx ouvert : ne plafonne jamais à 9999', () => {
    const r = frameToTier(2100, 2200)
    expect(r.min).toBe(2000)
    expect(r.max).toBe(2200)
    expect(r.max).toBeLessThan(9000)
  })

  it('ne descend jamais sous 0', () => {
    expect(frameToTier(0, 0)).toEqual({ min: 0, max: 100 })
  })
})
