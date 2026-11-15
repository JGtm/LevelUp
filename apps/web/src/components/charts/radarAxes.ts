/**
 * radarAxes — axes communs d'un radar multi-séries.
 *
 * Un radar ECharts n'a qu'UN jeu d'indicateurs pour toutes ses séries, et rend une valeur manquante
 * par un sommet au centre (comme un zéro). Les indicateurs sont donc les axes présents dans TOUTES
 * les séries, dans l'ordre de la première, et les valeurs de chaque série se lisent par clé d'axe :
 * une série qui n'a pas un axe (joueur non mesuré) le retire pour toutes, jamais un point à 0 ni un
 * décalage d'index.
 */
export interface KeyedAxis {
  axis: string
}

/** Clés des axes présents dans toutes les séries, dans l'ordre de la première. */
export function commonAxisKeys(axesBySeries: ReadonlyArray<ReadonlyArray<KeyedAxis> | null | undefined>): string[] {
  const [first, ...rest] = axesBySeries
  if (!first) return []
  const others = rest.map((axes) => new Set((axes ?? []).map((a) => a.axis)))
  return first.map((a) => a.axis).filter((key) => others.every((keys) => keys.has(key)))
}

/** Axes d'une série réordonnés sur `keys` (clés présentes dans cette série, garanties par commonAxisKeys). */
export function axesByKeys<T extends KeyedAxis>(axes: ReadonlyArray<T> | null | undefined, keys: string[]): T[] {
  const byKey = new Map((axes ?? []).map((a) => [a.axis, a]))
  return keys.map((key) => byKey.get(key)).filter((a): a is T => a !== undefined)
}
