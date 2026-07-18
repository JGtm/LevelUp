/**
 * outcomeSequence — modèle de données pur de la bande d'outcomes (RLE) et
 * résolution du match cliqué. Séparé du composant `OutcomeSequenceTape.tsx`
 * (React) pour rester testable isolément et éviter d'exporter des valeurs depuis
 * un fichier de composant (react-refresh).
 */
export type OutcomeValue = 'win' | 'loss' | 'tie' | 'dnf'

export interface OutcomePoint {
  outcome: OutcomeValue
  matchId: string
  map?: string
  mode?: string
  /** Libellé pré-formaté pour le tooltip (ex. « 12 mars · Slayer · Aquarius — 14/9 »). */
  label?: string
}

export interface Run {
  outcome: OutcomeValue
  count: number
  matches: OutcomePoint[]
}

/** toRuns — regroupe les outcomes consécutifs identiques (run-length encoding). */
export function toRuns(arr: OutcomePoint[]): Run[] {
  const runs: Run[] = []
  for (const m of arr) {
    const last = runs[runs.length - 1]
    if (last && last.outcome === m.outcome) {
      last.count += 1
      last.matches.push(m)
    } else {
      runs.push({ outcome: m.outcome, count: 1, matches: [m] })
    }
  }
  return runs
}

/** startOf — index global (0-based) du premier match du run `i`. */
export function startOf(runs: Run[], i: number): number {
  let p = 0
  for (let k = 0; k < i; k++) p += runs[k].count
  return p
}

/**
 * matchIndexAtX — résout le match cliqué à partir d'une valeur X continue de
 * l'axe (0..xMax). Helper PUR : `xValue` est ramené à un index global entier
 * borné [0, total-1], puis on localise le run et l'offset interne pour renvoyer
 * le `OutcomePoint` correspondant (ou `null` si aucun match).
 */
export function matchIndexAtX(runs: Run[], xValue: number): OutcomePoint | null {
  const total = runs.reduce((s, r) => s + r.count, 0)
  if (total === 0) return null
  let idx = Math.floor(xValue)
  if (idx < 0) idx = 0
  if (idx > total - 1) idx = total - 1
  let acc = 0
  for (const r of runs) {
    if (idx < acc + r.count) return r.matches[idx - acc] ?? null
    acc += r.count
  }
  return null
}
