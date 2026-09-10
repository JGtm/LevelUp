/**
 * assetDrawerLogic — logique PURE de l'Asset Drawer (lot 4.2, nettoyage console rejeu).
 *
 * Aucun composant, aucune I/O : le dédoublonnage se teste seul, sans monter React.
 */
import type { AssetMeta } from '@/lib/api/types'

/**
 * dedupeAssetsById — supprime les doublons d'un catalogue d'assets, IDENTIQUES PAR
 * `id`, en gardant la PREMIÈRE occurrence rencontrée.
 *
 * POURQUOI CE GARDE-FOU EXISTE (constat 2026-09-10, revue navigateur de la page de
 * rejeu) : `AssetGrid` rend ce catalogue avec `key={asset.id}` — un `id` dupliqué
 * produit l'avertissement React « Encountered two children with the same key », et
 * `AssetDrawer` reste monté en permanence dans `AppShell` (translaté hors écran quand
 * fermé, jamais démonté) : le doublon apparaît donc sur TOUTE page, rejeu compris.
 *
 * ROOT CAUSE, CÔTÉ SERVEUR (Go, hors périmètre — cf. `.ai/thought_log.md` 2026-09-10) :
 * `MetadataRepo.ListMapsByTitle` dédoublonne par `SELECT DISTINCT ON (m.name_canonical)`,
 * PAS par `map_asset_id`. Si le catalogue porte deux libellés distincts pour le MÊME
 * `map_asset_id`, la requête rend deux lignes de même `id` sous deux noms — ce n'est
 * donc PAS une clé composite qui masquerait un doublon de données : c'est un doublon
 * RÉEL du même asset, qu'il faut réellement retirer, pas seulement re-clé.
 *
 * Garder le PREMIER exemplaire est déterministe : l'ordre reçu est celui du tri serveur
 * (`ORDER BY name_canonical, name_en`), stable d'un appel à l'autre — le résultat ne
 * dépend d'aucun hasard de réseau ou de pagination.
 */
export function dedupeAssetsById(items: readonly AssetMeta[]): AssetMeta[] {
  const seen = new Set<string>()
  const out: AssetMeta[] = []
  for (const item of items) {
    if (seen.has(item.id)) continue
    seen.add(item.id)
    out.push(item)
  }
  return out
}
