/**
 * perMinute — cadence par minute jouée côté web : `total / (secondes / 60)`.
 *
 * Miroir de `analysis.PerMinute` (Go) : la division se fait par les minutes, jamais par
 * `total * 60 / secondes` (l'arrondi diffère). `null` quand la durée est inconnue ou nulle
 * (le match n'a pas de cadence, il n'est pas compté à zéro).
 */
export function perMinute(total: number, seconds: number | null | undefined): number | null {
  if (seconds == null || !Number.isFinite(seconds) || seconds <= 0) return null
  return total / (seconds / 60)
}

/**
 * pooledPerMinute — cadence d'un ensemble de matchs : SOMME des comptes / SOMME des minutes
 * jouées (jamais la moyenne des rapports par match, qui donnerait autant de poids à un match
 * de trois minutes qu'à un match de quinze). Un match sans durée ou sans compte (`null`)
 * sort du numérateur ET du dénominateur ; sans aucun match retenu, `null` (absent, pas zéro).
 */
export function pooledPerMinute<T>(
  items: readonly T[],
  count: (item: T) => number | null | undefined,
  seconds: (item: T) => number | null | undefined,
): number | null {
  let total = 0
  let secs = 0
  for (const item of items) {
    const c = count(item)
    const s = seconds(item)
    if (c == null || !Number.isFinite(c) || s == null || !Number.isFinite(s) || s <= 0) continue
    total += c
    secs += s
  }
  return perMinute(total, secs)
}
