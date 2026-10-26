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
    const labels = names(buildSkillTierMarkArea({ locale: 'fr', yMin: 1400, yMax: 1600, grid: LUSR_TIER_GRID, theme: tc, plotHeightPx: 230 }))
    expect(labels).toContain('Or III')
    expect(labels).toContain('Or I')
  })

  it('CSR : labels en chiffres arabes selon la locale EN (Diamond 3)', () => {
    const labels = names(buildSkillTierMarkArea({ locale: 'en', yMin: 1200, yMax: 1500, grid: CSR_TIER_GRID, theme: tc, plotHeightPx: 230 }))
    expect(labels).toContain('Diamond 3')
  })

  it('alterne le shading entre bandes voisines', () => {
    const ma = buildSkillTierMarkArea({ locale: 'fr', yMin: 1400, yMax: 1600, grid: LUSR_TIER_GRID, theme: tc, plotHeightPx: 230 })
    const colors = ma.data.map(([a]) => (a as { itemStyle: { color: string } }).itemStyle.color)
    expect(colors[0]).not.toBe(colors[1])
  })

  it('Onyx : une bande sans numéro de sous-palier', () => {
    const labels = names(buildSkillTierMarkArea({ locale: 'fr', yMin: 1900, yMax: 2200, grid: LUSR_TIER_GRID, theme: tc, plotHeightPx: 230 }))
    expect(labels).toContain('Onyx')
  })

  type Entry = { name: string; yAxis: number; label: { show: boolean; position: string } }
  const entries = (ma: { data: Array<[object, object]> }) => ma.data.map(([a]) => a as Entry)

  it('label centré verticalement dans sa bande (pas collé à la borne du rang suivant)', () => {
    const ma = buildSkillTierMarkArea({ locale: 'en', yMin: 1400, yMax: 1500, grid: CSR_TIER_GRID, theme: tc, plotHeightPx: 230 })
    expect(entries(ma).every(e => e.label.position === 'insideLeft')).toBe(true)
    // 1475 tombe dans la bande Diamond 6 [1450, 1500[.
    const d6 = entries(ma).find(e => e.name === 'Diamond 6')
    expect(d6?.yAxis).toBe(1450)
  })

  it('axe très étiré : un label par palier entier, jamais de sous-paliers empilés', () => {
    // Cas de la carrière : LUSR de 1000 à 2200 sur ~230 px → sous-palier Bronze ≈ 6 px.
    const labels = names(buildSkillTierMarkArea({ locale: 'fr', yMin: 1000, yMax: 2200, grid: LUSR_TIER_GRID, theme: tc, plotHeightPx: 230 }))
    expect(labels).toEqual(['Bronze', 'Argent', 'Or', 'Platine', 'Diamant', 'Onyx'])
  })

  it('bande trop fine pour son label (clippée au bord du cadre) : label masqué', () => {
    // Diamant [1800, 2000[ clippé à [1990, 2000[ : 10 pts ≈ 8 px.
    const ma = buildSkillTierMarkArea({ locale: 'fr', yMin: 1990, yMax: 2200, grid: LUSR_TIER_GRID, theme: tc, plotHeightPx: 170 })
    const clipped = entries(ma).find(e => e.yAxis === 1990)
    expect(clipped?.label.show).toBe(false)
    expect(entries(ma).find(e => e.name === 'Onyx')?.label.show).toBe(true)
  })
})
