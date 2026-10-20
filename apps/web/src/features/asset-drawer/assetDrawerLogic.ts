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
 * CÔTÉ SERVEUR, DEPUIS LORS : le dépôt `MetadataRepo.ListMapsByTitle` rend UNE ligne
 * par asset (`DISTINCT ON (m.map_asset_id)`, décision D15 du 2026-09-13), et le service
 * `AssetService.ListMaps` rend UNE carte par VISUEL (URL d'image résolue, lot rr/L4 du
 * 2026-09-23 : plusieurs assets — versions republiées, copies Forge — partagent la même
 * image et donnaient des cartes identiques, par exemple « Solution » en trois
 * exemplaires). Le serveur ne rend donc plus deux fois le même `id`. Ce filtre reste
 * le filet de la clé React : il ne dédoublonne PAS par visuel — cette règle (et le
 * choix du représentant) vit dans le service, là où l'image est résolue.
 *
 * Garder le PREMIER exemplaire est déterministe : l'ordre reçu est celui du tri serveur
 * (nom anglais, puis `id`), stable d'un appel à l'autre — le résultat ne dépend
 * d'aucun hasard de réseau ou de pagination.
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
