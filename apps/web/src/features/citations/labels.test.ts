import { describe, expect, it } from 'vitest'

import { citationsManifest } from '@/lib/i18n/generated/citations'
import { citationCategoryLabel } from './labels'

/**
 * Miroir de canonical.AllCommendationCategories()
 * (apps/go-api/internal/games/canonical/commendation_category.go). Toute clé
 * ajoutée côté Go doit l'être ici ET dans citations.toml — sinon l'UI affiche la
 * clé brute. Ce test est le garde-rail de cette parité (patron
 * halo_5/medal_category_test.go côté Go).
 */
const CANONICAL_CATEGORY_KEYS = [
  'multiplayer',
  'game_mode',
  'weapon',
  'vehicle',
  'enemy',
  'spartan_companies',
  'other',
] as const

describe('citationCategoryLabel', () => {
  it('traduit chaque clé canonique en FR et en EN, sans trou ni doublon', () => {
    const fr = new Set<string>()
    const en = new Set<string>()
    for (const key of CANONICAL_CATEGORY_KEYS) {
      const labelFr = citationCategoryLabel(key, 'fr')
      const labelEn = citationCategoryLabel(key, 'en')
      // Une clé non traduite retomberait sur elle-même (dégradation visible).
      expect(labelFr, `clé ${key} sans libellé FR`).not.toBe(key)
      expect(labelEn, `clé ${key} sans libellé EN`).not.toBe(key)
      expect(labelFr.trim()).not.toBe('')
      expect(labelEn.trim()).not.toBe('')
      fr.add(labelFr)
      en.add(labelEn)
    }
    // Deux catégories distinctes ne doivent pas partager un libellé.
    expect(fr.size).toBe(CANONICAL_CATEGORY_KEYS.length)
    expect(en.size).toBe(CANONICAL_CATEGORY_KEYS.length)
  })

  it('déclare les clés dans le manifeste sous citations.category.<clé>', () => {
    for (const key of CANONICAL_CATEGORY_KEYS) {
      expect(citationsManifest, `citations.category.${key} absente du manifeste`).toHaveProperty(
        `citations.category.${key}`,
      )
    }
  })

  it('rend des libellés FR sans anglicisme pour les catégories traduisibles', () => {
    expect(citationCategoryLabel('weapon', 'fr')).toBe('Armes')
    expect(citationCategoryLabel('vehicle', 'fr')).toBe('Véhicules')
    expect(citationCategoryLabel('game_mode', 'fr')).toBe('Modes de jeu')
    expect(citationCategoryLabel('other', 'fr')).toBe('Autres')
  })

  it('retombe sur la clé brute si elle est inconnue du manifeste', () => {
    expect(citationCategoryLabel('categorie_future_2027', 'fr')).toBe('categorie_future_2027')
    expect(citationCategoryLabel('', 'en')).toBe('')
  })
})
