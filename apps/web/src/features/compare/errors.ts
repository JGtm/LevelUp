/**
 * Vrai quand la comparaison échoue parce que le joueur est introuvable (404). Décidé sur le
 * statut et le code machine, jamais sur le texte du serveur : celui-ci est un message fixe en
 * anglais, que l'écran ne montre pas.
 */
export function isCompareNotFound(error: unknown): boolean {
  if (typeof error !== 'object' || error === null) return false
  const e = error as { status?: unknown; code?: unknown }
  return e.status === 404 || e.code === 'player_not_found'
}
