/**
 * Tests fragRoleDisplayLabel — les deux natures de rôle du niveau 2 (V73-3.2), le libellé servi
 * par l'API dans la langue de la requête (ADR 0040 D2) et le repli générique (D3) : jamais une
 * clé i18n brute à l'écran.
 */
import { describe, it, expect } from 'vitest'
import { fragRoleDisplayLabel } from './fragRoleLabel'

/** Résolveur i18n factice : imite formatMessage (contrat réel — renvoie la clé BRUTE
 *  quand elle est absente du manifeste), sauf pour generic_object (toujours résolu). */
const roleLabel = (r: string) => (r === 'generic_object' ? 'Objet' : `i18n(${r})`)

/** Résolveur qui imite une clé manifeste manquante (formatMessage renvoie la clé brute),
 *  MAIS sait résoudre generic_object — comme le ferait le vrai fragsManifest une fois la
 *  clé de repli ajoutée. */
const roleLabelUnknown = (r: string) => (r === 'generic_object' ? 'Objet' : `frags.role.${r}`)

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

  it('libellé servi vide ou blanc, clé canonique traduite → repli sur la traduction', () => {
    expect(fragRoleDisplayLabel({ role: 'sniper', kills: 3, label: '' }, roleLabel)).toBe('i18n(sniper)')
    expect(fragRoleDisplayLabel({ role: 'sniper', kills: 3, label: '   ' }, roleLabel)).toBe('i18n(sniper)')
  })

  // ── Repli générique (D3, 2026-08-29) — EXIGENCE ABSOLUE : jamais la clé brute ──
  describe('repli générique quand rien n’est servi ni traduit', () => {
    it('aucun libellé servi, AUCUNE clé canonique → libellé générique, jamais la clé brute', () => {
      const r = { role: 'hinf_new_object_not_yet_seeded', kills: 2 }
      const label = fragRoleDisplayLabel(r, roleLabelUnknown)
      expect(label).toBe('Objet')
      expect(label).not.toContain('frags.role.')
    })

    it('libellé servi blanc (espaces) → repli générique', () => {
      const r = { role: 'hinf_x', kills: 1, label: '  ' }
      expect(fragRoleDisplayLabel(r, roleLabelUnknown)).toBe('Objet')
    })
  })
})
