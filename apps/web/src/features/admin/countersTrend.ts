/**
 * countersTrend — généralisation pure du pattern invariantsTrend : snapshot
 * clé→count en localStorage + delta vs visite précédente. Utilisé par les
 * pages Convergence (backlog par joueur) et Qualité données (compteurs
 * d'inconnus) — tendance lisible sans historique serveur.
 */

export type CountersSnapshot = Record<string, number>

export function readCountersSnapshot(storageKey: string): CountersSnapshot {
  try {
    const raw = localStorage.getItem(storageKey)
    return raw ? (JSON.parse(raw) as CountersSnapshot) : {}
  } catch {
    return {}
  }
}

export function writeCountersSnapshot(storageKey: string, snap: CountersSnapshot): void {
  try {
    localStorage.setItem(storageKey, JSON.stringify(snap))
  } catch {
    /* quota/SSR : tendance simplement absente */
  }
}

/**
 * Delta d'un compteur vs la baseline. undefined = pas de référence (première
 * apparition) ou valeur inchangée — l'UI n'affiche alors rien.
 */
export function counterDelta(
  previous: CountersSnapshot,
  key: string,
  count: number,
): number | undefined {
  const prev = previous[key]
  if (prev === undefined || prev === count) return undefined
  return count - prev
}
