/**
 * withoutNullRows — retire d'un enregistrement par joueur les entrées `null`.
 *
 * Le contrat sert une série par joueur sous la forme `{ [joueur]: lignes | null }` (une liste nulle
 * côté Go sérialise `null`) ; les graphes de l'Escouade lisent `lignes[joueur] ?? []`, donc une
 * entrée nulle et une entrée absente se valent. Rend `{}` sans enregistrement.
 */
export function withoutNullRows<T>(
  byPlayer: Readonly<Record<string, readonly T[] | null>> | null | undefined,
): Record<string, T[]> {
  const out: Record<string, T[]> = {}
  for (const [player, rows] of Object.entries(byPlayer ?? {})) {
    if (rows != null) out[player] = [...rows]
  }
  return out
}
