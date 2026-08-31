/**
 * Tests fragRoleDisplayLabel — les deux natures de rôle du niveau 2 (V73-3.2), le choix
 * de LOCALE entre label/label_en (D2, 2026-08-29) et le repli générique (D3) : jamais une
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
  it('rôle canonique sans libellé servi → traduction de la clé (FR)', () => {
    expect(fragRoleDisplayLabel({ role: 'precision', kills: 12 }, 'fr', roleLabel)).toBe('i18n(precision)')
  })

  it('rôle canonique sans libellé servi → traduction de la clé (EN)', () => {
    expect(fragRoleDisplayLabel({ role: 'precision', kills: 12 }, 'en', roleLabel)).toBe('i18n(precision)')
  })

  it('engin (weapon_key du titre) → libellé servi par l’API, jamais la clé brute', () => {
    const label = fragRoleDisplayLabel({ role: 'h5_vehicle_warthog', kills: 6, label: 'Warthog' }, 'fr', roleLabel)
    expect(label).toBe('Warthog')
    expect(label).not.toContain('h5_vehicle_warthog')
  })

  it('deux engins distincts gardent des libellés distincts (exigence du sous-niveau)', () => {
    const ghost = fragRoleDisplayLabel({ role: 'h5_vehicle_ghost', kills: 4, label: 'Ghost' }, 'fr', roleLabel)
    const banshee = fragRoleDisplayLabel({ role: 'h5_vehicle_banshee', kills: 4, label: 'Banshee' }, 'fr', roleLabel)
    expect(ghost).toBe('Ghost')
    expect(banshee).toBe('Banshee')
    expect(ghost).not.toBe(banshee)
  })

  it('libellé servi vide ou blanc, clé canonique traduite → repli sur la traduction', () => {
    expect(fragRoleDisplayLabel({ role: 'sniper', kills: 3, label: '' }, 'fr', roleLabel)).toBe('i18n(sniper)')
    expect(fragRoleDisplayLabel({ role: 'sniper', kills: 3, label: '   ' }, 'fr', roleLabel)).toBe('i18n(sniper)')
  })

  // ── Choix de locale (D2, 2026-08-29) ──────────────────────────────────────────
  describe('choix entre label (FR) et label_en (EN)', () => {
    const role = { role: 'hinf_coil_kinetic', kills: 3, label: 'Bobine à fusion UNSC', label_en: 'UNSC Fusion Coil' }

    it('locale EN → label_en', () => {
      expect(fragRoleDisplayLabel(role, 'en', roleLabel)).toBe('UNSC Fusion Coil')
    })

    it('locale FR → label', () => {
      expect(fragRoleDisplayLabel(role, 'fr', roleLabel)).toBe('Bobine à fusion UNSC')
    })

    it('locale EN sans label_en → repli croisé sur label (FR)', () => {
      const r = { role: 'hinf_repulsor', kills: 1, label: 'Répulseur' } // pas de label_en
      expect(fragRoleDisplayLabel(r, 'en', roleLabel)).toBe('Répulseur')
    })

    it('locale FR sans label → repli croisé sur label_en', () => {
      const r = { role: 'hinf_repulsor', kills: 1, label_en: 'Repulsor' } // pas de label
      expect(fragRoleDisplayLabel(r, 'fr', roleLabel)).toBe('Repulsor')
    })

    it('label_en blanc (espaces) → traité comme vide, repli croisé sur label', () => {
      const r = { role: 'hinf_repulsor', kills: 1, label: 'Répulseur', label_en: '   ' }
      expect(fragRoleDisplayLabel(r, 'en', roleLabel)).toBe('Répulseur')
    })
  })

  // ── Repli générique (D3, 2026-08-29) — EXIGENCE ABSOLUE : jamais la clé brute ──
  describe('repli générique quand rien n’est servi ni traduit', () => {
    it('label et label_en vides, AUCUNE clé canonique → libellé générique, jamais la clé brute', () => {
      const r = { role: 'hinf_new_object_not_yet_seeded', kills: 2 }
      const label = fragRoleDisplayLabel(r, 'fr', roleLabelUnknown)
      expect(label).toBe('Objet')
      expect(label).not.toContain('frags.role.')
    })

    it('même cas en EN → repli générique, jamais la clé brute', () => {
      const r = { role: 'hinf_new_object_not_yet_seeded', kills: 2 }
      const label = fragRoleDisplayLabel(r, 'en', roleLabelUnknown)
      expect(label).toBe('Objet') // le résolveur factice rend le même texte quelle que soit la locale
      expect(label).not.toContain('frags.role.')
    })

    it('label/label_en vides mais tous deux blancs (espaces) → repli générique', () => {
      const r = { role: 'hinf_x', kills: 1, label: '  ', label_en: '  ' }
      expect(fragRoleDisplayLabel(r, 'fr', roleLabelUnknown)).toBe('Objet')
    })
  })
})
