/**
 * assetDrawerLogic.test.ts — dédoublonnage du catalogue d'assets (lot 4.2).
 *
 * Constat (revue navigateur 2026-09-10) : neuf avertissements React « Encountered two
 * children with the same key » sur la page de rejeu, clés = des `map_id` (GUID). Le
 * composant fautif est `AssetGrid` (`key={asset.id}`), monté en permanence par
 * `AssetDrawer` dans `AppShell` (le tiroir reste dans le DOM, translaté hors écran,
 * même fermé) — donc visible sur TOUTE page, rejeu compris.
 *
 * Root cause CÔTÉ SERVEUR (Go, hors périmètre de ce lot — cf. thought_log) :
 * `MetadataRepo.ListMapsByTitle` (apps/go-api/internal/platform/duckdb/
 * metadata_repo_assets_list.go) dédoublonne par `SELECT DISTINCT ON (m.name_canonical)`,
 * PAS par `map_asset_id`. Si `maps_catalog` porte deux `name_canonical` distincts pour
 * le MÊME `map_asset_id` (rename de carte, doublon de catalogue), la requête rend deux
 * lignes avec le même `id` et des libellés différents — exactement la forme du bug
 * observé. Ce test prouve le dédoublonnage CÔTÉ WEB, en dernier rempart avant le rendu.
 */
import { describe, it, expect } from 'vitest'
import type { AssetMeta } from '@/lib/api/types'
import { dedupeAssetsById } from './assetDrawerLogic'

function map(id: string, nameEn: string): AssetMeta {
  return { id, name_en: nameEn, name_fr: '', image_url: '' } as AssetMeta
}

describe('dedupeAssetsById', () => {
  it('supprime les entrées de même id, en gardant la PREMIÈRE (ordre serveur déterministe)', () => {
    const doublon = [
      map('9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99', 'Aquarius'),
      map('b302eb62-0000-0000-0000-000000000000', 'Bazaar'),
      // Même id que la première ligne, sous un AUTRE libellé — le doublon observé.
      map('9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99', 'Aquarius (Forge)'),
    ]

    const dedup = dedupeAssetsById(doublon)

    expect(dedup).toHaveLength(2)
    expect(dedup.map((a) => a.id)).toEqual([
      '9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99',
      'b302eb62-0000-0000-0000-000000000000',
    ])
    // La première occurrence (« Aquarius ») gagne — pas la seconde.
    expect(dedup[0].name_en).toBe('Aquarius')
  })

  it('ne modifie rien quand tous les id sont déjà uniques', () => {
    const uniques = [map('a', 'Aquarius'), map('b', 'Bazaar'), map('c', 'Catalyst')]
    expect(dedupeAssetsById(uniques)).toEqual(uniques)
  })

  it('liste vide → liste vide', () => {
    expect(dedupeAssetsById([])).toEqual([])
  })
})
