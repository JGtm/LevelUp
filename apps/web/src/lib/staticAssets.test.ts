import { describe, it, expect } from 'vitest'
import { staticAssetURL, csrRankImageURL, unrankedBadgeURL } from './staticAssets'

// Default Phase 6.5+ : title-scoped activé (VITE_STATIC_PATHS_TITLE_SCOPED unset
// ou != "false"). Pour tester le mode flat (rollback), il faudrait toggle
// import.meta.env à "false" via vitest config — non couvert ici.

describe('staticAssetURL (default title-scoped)', () => {
  it('compose URL pour map png', () => {
    expect(staticAssetURL('map', 'Aquarius', '.png')).toBe('/static/maps/halo_infinite/Aquarius.png')
  })

  it('compose URL pour medal numeric', () => {
    expect(staticAssetURL('medal', '12345', '.png')).toBe('/static/medals/halo_infinite/12345.png')
  })

  it('compose URL pour csr-rank', () => {
    expect(staticAssetURL('csr-rank', '120px-HINF-CSR_Gold3', '.png')).toBe(
      '/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png',
    )
  })

  it('compose URL pour weapon', () => {
    expect(staticAssetURL('weapon', 'br75', '.png')).toBe('/static/weapons-assets/halo_infinite/br75.png')
  })

  it('compose URL pour sound (wav du rejeu 2D)', () => {
    expect(staticAssetURL('sound', 'hinf_br75', '.wav')).toBe(
      '/static/sounds/halo_infinite/hinf_br75.wav',
    )
  })

  it('compose URL pour commendation (titleSlug explicite)', () => {
    expect(staticAssetURL('commendation', 'achilles', '.png', 'halo_5_guardians')).toBe(
      '/static/commendations/halo_5_guardians/achilles.png',
    )
  })

  it('retourne empty string si id vide', () => {
    expect(staticAssetURL('map', '', '.png')).toBe('')
  })
})

describe('csrRankImageURL (default title-scoped)', () => {
  it('badge Gold 3', () => {
    expect(csrRankImageURL('Gold', 3)).toBe('/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png')
  })

  it('badge Platinum 5', () => {
    expect(csrRankImageURL('Platinum', 5)).toBe('/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png')
  })

  it('cas spécial Onyx (sans subTier)', () => {
    expect(csrRankImageURL('Onyx', 0)).toBe('/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png')
  })

  it('Onyx avec subTier > 0 reste Onyx (cas spécial)', () => {
    expect(csrRankImageURL('Onyx', 3)).toBe('/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png')
  })

  it('subTier <= 0 (hors Onyx) → fallback Onyx', () => {
    expect(csrRankImageURL('Diamond', 0)).toBe('/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png')
  })

  it('tier vide → empty string', () => {
    expect(csrRankImageURL('', 3)).toBe('')
  })
})

describe('unrankedBadgeURL (default title-scoped)', () => {
  it('retourne URL title-scopée du badge unranked_0', () => {
    expect(unrankedBadgeURL()).toBe('/static/ranks/halo_infinite/unranked_0.png')
  })
})
