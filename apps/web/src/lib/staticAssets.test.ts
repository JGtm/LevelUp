import { describe, it, expect } from 'vitest'
import { staticAssetURL, csrRankImageURL, unrankedBadgeURL } from './staticAssets'

// Note : ces tests valident le comportement avec le flag à false (default Vite
// quand VITE_STATIC_PATHS_TITLE_SCOPED est unset). Pour tester le mode
// title-scoped, il faudrait toggle import.meta.env via vitest config — couvert
// indirectement par les tests de bout en bout au flip Phase 6.5.

describe('staticAssetURL (flag off)', () => {
  it('compose URL flat pour map png', () => {
    expect(staticAssetURL('map', 'Aquarius', '.png')).toBe('/static/maps/Aquarius.png')
  })

  it('compose URL flat pour medal numeric', () => {
    expect(staticAssetURL('medal', '12345', '.png')).toBe('/static/medals/icons/12345.png')
  })

  it('compose URL flat pour csr-rank', () => {
    expect(staticAssetURL('csr-rank', '120px-HINF-CSR_Gold3', '.png')).toBe(
      '/static/ranks/120px-HINF-CSR_Gold3.png',
    )
  })

  it('compose URL flat pour weapon', () => {
    expect(staticAssetURL('weapon', 'br75', '.png')).toBe('/static/weapons-assets/br75.png')
  })

  it('compose URL flat pour commendation', () => {
    expect(staticAssetURL('commendation', 'achilles', '.png')).toBe(
      '/static/commendations/achilles.png',
    )
  })

  it('retourne empty string si id vide', () => {
    expect(staticAssetURL('map', '', '.png')).toBe('')
  })
})

describe('csrRankImageURL (flag off)', () => {
  it('badge Gold 3', () => {
    expect(csrRankImageURL('Gold', 3)).toBe('/static/ranks/120px-HINF-CSR_Gold3.png')
  })

  it('badge Platinum 5', () => {
    expect(csrRankImageURL('Platinum', 5)).toBe('/static/ranks/120px-HINF-CSR_Platinum5.png')
  })

  it('cas spécial Onyx (sans subTier)', () => {
    expect(csrRankImageURL('Onyx', 0)).toBe('/static/ranks/120px-HINF-CSR_Onyx.png')
  })

  it('Onyx avec subTier > 0 reste Onyx (cas spécial)', () => {
    expect(csrRankImageURL('Onyx', 3)).toBe('/static/ranks/120px-HINF-CSR_Onyx.png')
  })

  it('subTier <= 0 (hors Onyx) → fallback Onyx', () => {
    expect(csrRankImageURL('Diamond', 0)).toBe('/static/ranks/120px-HINF-CSR_Onyx.png')
  })

  it('tier vide → empty string', () => {
    expect(csrRankImageURL('', 3)).toBe('')
  })
})

describe('unrankedBadgeURL (flag off)', () => {
  it('retourne URL flat du badge Unranked', () => {
    expect(unrankedBadgeURL()).toBe('/static/ranks/Unranked.png')
  })
})
