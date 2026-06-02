/**
 * Tests — skillTierBands : cadrage de l'axe Y sur la magnitude (frameToData) et
 * bandes de sous-palier (buildSkillTierMarkArea), pour LUSR et CSR.
 */
import { describe, it, expect } from 'vitest'
import { frameToData, buildSkillTierMarkArea } from './skillTierBands'
import { LUSR_TIER_GRID, CSR_TIER_GRID } from '@/lib/skillTiers'

describe('frameToData', () => {
  it('LUSR : cadre autour des données, les contient, sans dézoomer au tier entier', () => {
    const r = frameToData(1466, 1466, LUSR_TIER_GRID) // Or, session plate
    expect(r.min).toBeLessThanOrEqual(1466)
    expect(r.max).toBeGreaterThanOrEqual(1466)
    expect(r.min).toBeGreaterThan(1300)
    expect(r.max).toBeLessThan(1700)
    expect(r.max - r.min).toBeGreaterThan(0)
  })

  it('CSR : snappe sur les sous-rangs de 50 (Diamant)', () => {
    expect(frameToData(1330, 1370, CSR_TIER_GRID)).toEqual({ min: 1250, max: 1450 })
  })

  it('CSR bas : reste sur la grille CSR (Or 600-900), pas la grille LUSR', () => {
    const r = frameToData(820, 880, CSR_TIER_GRID)
    expect(r.min).toBeGreaterThanOrEqual(600)
    expect(r.max).toBeLessThanOrEqual(1000)
    expect(r.min).toBeLessThanOrEqual(820)
    expect(r.max).toBeGreaterThanOrEqual(880)
  })

  it('plancher : session plate → au moins MIN_BANDS sous-paliers visibles', () => {
    const r = frameToData(1325, 1325, CSR_TIER_GRID) // Diamant ∈ [1300,1350], sous-rangs de 50
    expect(r.max - r.min).toBeGreaterThanOrEqual(150)
  })

  it('palier ouvert Onyx (LUSR) : ne plafonne jamais à 9999', () => {
    const r = frameToData(2100, 2150, LUSR_TIER_GRID)
    expect(r.min).toBe(2000)
    expect(r.max).toBe(2200)
    expect(r.max).toBeLessThan(9000)
  })

  it('palier ouvert Onyx (CSR) : arrondi au STEP', () => {
    expect(frameToData(1600, 1650, CSR_TIER_GRID)).toEqual({ min: 1500, max: 1700 })
  })

  it('ne descend jamais sous 0 (CSR Bronze bas)', () => {
    const r = frameToData(0, 0, CSR_TIER_GRID)
    expect(r.min).toBe(0)
    expect(r.max).toBeGreaterThan(0)
  })
})

describe('buildSkillTierMarkArea', () => {
  const tc = { splitAreaA: 'rgba(1,1,1,.1)', splitAreaB: 'rgba(1,1,1,.3)', axisLabel: '#999' }
  const names = (ma: { data: Array<[object, object]> }) =>
    ma.data.map(([a]) => (a as { name: string }).name)

  it('LUSR : labels en chiffres romains (Or III) dans le cadre', () => {
    const labels = names(buildSkillTierMarkArea('fr', 1400, 1600, LUSR_TIER_GRID, tc))
    expect(labels).toContain('Or III')
    expect(labels).toContain('Or I')
  })

  it('CSR : labels en chiffres arabes selon la locale EN (Diamond 3)', () => {
    const labels = names(buildSkillTierMarkArea('en', 1200, 1500, CSR_TIER_GRID, tc))
    expect(labels).toContain('Diamond 3')
  })

  it('alterne le shading entre bandes voisines', () => {
    const ma = buildSkillTierMarkArea('fr', 1400, 1600, LUSR_TIER_GRID, tc)
    const colors = ma.data.map(([a]) => (a as { itemStyle: { color: string } }).itemStyle.color)
    expect(colors[0]).not.toBe(colors[1])
  })

  it('Onyx : une bande sans numéro de sous-palier', () => {
    const labels = names(buildSkillTierMarkArea('fr', 1900, 2200, LUSR_TIER_GRID, tc))
    expect(labels).toContain('Onyx')
  })
})
