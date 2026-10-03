import { describe, expect, it } from 'vitest'

import { parseVehicleManifest, vehicleBodyPx, vehicleManifestAssetId } from './vehicleSpriteManifest'

describe('parseVehicleManifest', () => {
  it('retient échelle, bordure et marge des familles dimensionnées', () => {
    const map = parseVehicleManifest([
      { famille: 'warthog', scale_mm_per_px: 10, outline: 'warthog_outline.png', pad: 4 },
      { famille: 'ghost', scale_mm_per_px: 10 },
      { famille: 'tourelle_auto_bannie', statut: 'sans_asset' },
    ])
    expect(map.get('warthog')).toEqual({ mmPerPx: 10, outline: 'warthog_outline.png', pad: 4 })
    expect(map.get('ghost')).toEqual({ mmPerPx: 10, outline: null, pad: 0 })
    expect(map.has('tourelle_auto_bannie')).toBe(false)
  })

  it('rend une table vide sur un manifeste qui n est pas un tableau', () => {
    expect(parseVehicleManifest({ famille: 'warthog' }).size).toBe(0)
    expect(parseVehicleManifest(null).size).toBe(0)
  })

  it('ignore une marge négative ou non numérique', () => {
    const map = parseVehicleManifest([
      { famille: 'a', scale_mm_per_px: 10, pad: -3 },
      { famille: 'b', scale_mm_per_px: 10, pad: '4' },
    ])
    expect(map.get('a')?.pad).toBe(0)
    expect(map.get('b')?.pad).toBe(0)
  })
})

describe('vehicleBodyPx', () => {
  it('retire la marge des deux côtés : la boîte redessinée égale le sprite historique', () => {
    // warthog : 222 px avant la bordure, 230 px avec pad = 4.
    expect(vehicleBodyPx(230, 4)).toBe(222)
    expect(vehicleBodyPx(230, 0)).toBe(230)
  })

  it('garde la dimension naturelle quand la marge mangerait toute l image', () => {
    expect(vehicleBodyPx(8, 4)).toBe(8)
  })
})

describe('vehicleManifestAssetId', () => {
  it('compose l identifiant d asset sans extension', () => {
    expect(vehicleManifestAssetId('warthog_outline.png')).toBe('replay/warthog_outline')
  })
})
