/**
 * Tests fragRoleDisplayLabel — les deux natures de rôle du niveau 2 (V73-3.2) :
 * clé canonique traduite côté web, ou nom d'ENGIN servi par l'API (véhicules et
 * tourelles, dont les libellés vivent dans le TOML du titre).
 */
import { describe, it, expect } from 'vitest'
import { fragRoleDisplayLabel } from './fragRoleLabel'

/** Résolveur i18n factice : préfixe reconnaissable pour distinguer la source. */
const roleLabel = (r: string) => `i18n(${r})`

describe('fragRoleDisplayLabel', () => {
  it('rôle canonique sans libellé servi → traduction de la clé', () => {
    expect(fragRoleDisplayLabel({ role: 'precision', kills: 12 }, roleLabel)).toBe('i18n(precision)')
  })

  it('engin (weapon_key du titre) → libellé servi par l’API, jamais la clé brute', () => {
    const label = fragRoleDisplayLabel({ role: 'h5_vehicle_warthog', kills: 6, label: 'Warthog' }, roleLabel)
    expect(label).toBe('Warthog')
    expect(label).not.toContain('h5_vehicle_warthog')
  })

  it('deux engins distincts gardent des libellés distincts (exigence du sous-niveau)', () => {
    const ghost = fragRoleDisplayLabel({ role: 'h5_vehicle_ghost', kills: 4, label: 'Ghost' }, roleLabel)
    const banshee = fragRoleDisplayLabel({ role: 'h5_vehicle_banshee', kills: 4, label: 'Banshee' }, roleLabel)
    expect(ghost).toBe('Ghost')
    expect(banshee).toBe('Banshee')
    expect(ghost).not.toBe(banshee)
  })

  it('libellé servi vide ou blanc → repli sur la traduction (jamais un arc sans nom)', () => {
    expect(fragRoleDisplayLabel({ role: 'sniper', kills: 3, label: '' }, roleLabel)).toBe('i18n(sniper)')
    expect(fragRoleDisplayLabel({ role: 'sniper', kills: 3, label: '   ' }, roleLabel)).toBe('i18n(sniper)')
  })
})
