/**
 * Détection d'un asset_id Halo brut (UUID) — nom de map/mode/playlist NON résolu.
 *
 * Un match non enrichi stocke l'asset_id de l'asset dans match_registry au lieu du
 * nom lisible ; ce GUID ne doit JAMAIS s'afficher tel quel. Garde-fou front, miroir
 * de la garde backend `looksLikeAssetID` (media_repo_translations.go). Le backend
 * résout normalement le nom via maps_catalog ou renvoie null — ceci est la défense
 * en profondeur côté UI.
 */
const ASSET_ID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** Retourne true si la valeur a la forme d'un asset_id (UUID) brut. */
export function looksLikeAssetId(value: string | null | undefined): boolean {
  if (!value) return false
  return ASSET_ID_RE.test(value.trim())
}
