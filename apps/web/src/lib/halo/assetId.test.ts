import { describe, it, expect } from 'vitest'
import { looksLikeAssetId } from './assetId'

describe('looksLikeAssetId', () => {
  it('détecte un asset_id (UUID) brut', () => {
    // map Cliffhanger non résolue, playlist Quick Play non résolue.
    expect(looksLikeAssetId('5324364b-39a8-4f93-96a6-b80a1f18ce8a')).toBe(true)
    expect(looksLikeAssetId('1B1691DC-D8B9-4B1F-825D-CB1C065184C1')).toBe(true)
  })

  it('accepte les noms de map lisibles', () => {
    expect(looksLikeAssetId('Domicile')).toBe(false)
    expect(looksLikeAssetId('Dévissage')).toBe(false)
    expect(looksLikeAssetId('Cliffhanger')).toBe(false)
  })

  it('gère null / undefined / vide / non-UUID', () => {
    expect(looksLikeAssetId(null)).toBe(false)
    expect(looksLikeAssetId(undefined)).toBe(false)
    expect(looksLikeAssetId('')).toBe(false)
    expect(looksLikeAssetId('5324364b')).toBe(false) // fragment, pas un UUID complet
  })
})
