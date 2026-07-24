/**
 * phaseProfile — profil d'activité par phase de match (parts de frags), partagé
 * par les charts « Intensité » : Escouade (multi-panneaux), Timeseries et
 * Sessions (mono-panneau).
 *
 * Helper PUR. Chaque manche expose un vecteur `phases` (densité de frags par
 * tranche de 10 % de la durée, déjà normalisé par match côté serveur). La PART
 * d'une manche pour la phase i est `phases[i] / Σ phases` (la normalisation
 * per-match préserve les ratios, cf. plan). Une manche sans frag (Σ = 0) ne
 * contribue pas au profil.
 *
 * Le profil agrège les parts sur les manches exploitables : MÉDIANE par phase
 * (trait central) + quantiles P25 / P75 (enveloppe interquartile qui rend
 * visible l'irrégularité). L'enveloppe n'est significative qu'à partir de
 * `MIN_MATCHES_FOR_ENVELOPE` manches (en-deçà : médiane seule) — le décompte
 * `nMatches` est exposé pour que le builder tranche.
 */

/** Nombre de phases (tranches de 10 % de la durée du match). */
export const PHASE_COUNT = 10

/**
 * Seuil de manches exploitables sous lequel l'enveloppe P25–P75 dégénère
 * (trop peu d'échantillons pour un interquartile fiable) → médiane seule.
 */
export const MIN_MATCHES_FOR_ENVELOPE = 4

/** Profil agrégé : médiane + quantiles par phase, sur `nMatches` manches exploitables. */
export interface PhaseProfileResult {
  /** Médiane des parts par phase (length = PHASE_COUNT). */
  median: number[]
  /** 1er quartile des parts par phase (length = PHASE_COUNT). */
  p25: number[]
  /** 3e quartile des parts par phase (length = PHASE_COUNT). */
  p75: number[]
  /** Nombre de manches exploitables (Σ phases > 0) ayant contribué. */
  nMatches: number
}

/**
 * Parts de frags par phase d'une manche : `phases[i] / Σ phases`, projeté sur un
 * vecteur de longueur PHASE_COUNT (phases manquantes → 0). Retourne `null` si le
 * vecteur est absent ou si Σ phases ≤ 0 (aucun frag : la manche ne contribue pas).
 */
export function phaseShares(phases: number[] | null | undefined): number[] | null {
  if (!phases || phases.length === 0) return null
  let sum = 0
  for (let i = 0; i < phases.length; i += 1) {
    const v = phases[i]
    if (Number.isFinite(v) && v > 0) sum += v
  }
  if (sum <= 0) return null
  const out = new Array<number>(PHASE_COUNT).fill(0)
  for (let i = 0; i < PHASE_COUNT; i += 1) {
    const v = phases[i]
    out[i] = Number.isFinite(v) && v > 0 ? v / sum : 0
  }
  return out
}

/**
 * Quantile linéaire (méthode R-7 / interpolation continue) sur un tableau DÉJÀ
 * TRIÉ ascendant. `q` dans [0,1]. Renvoie 0 sur tableau vide.
 */
function quantileSorted(sorted: number[], q: number): number {
  const n = sorted.length
  if (n === 0) return 0
  if (n === 1) return sorted[0]
  const pos = (n - 1) * q
  const lo = Math.floor(pos)
  const hi = Math.ceil(pos)
  if (lo === hi) return sorted[lo]
  return sorted[lo] + (sorted[hi] - sorted[lo]) * (pos - lo)
}

/**
 * Profil médian + enveloppe P25–P75 des parts par phase sur les manches
 * exploitables (Σ phases > 0). `nMatches` = nombre de manches retenues ; les
 * vecteurs médiane/p25/p75 valent 0 partout si aucune manche n'est exploitable.
 */
export function phaseProfile(rows: Array<{ phases: number[] | null }>): PhaseProfileResult {
  const shares: number[][] = []
  for (const r of rows) {
    const s = phaseShares(r.phases)
    if (s) shares.push(s)
  }
  const nMatches = shares.length
  const median = new Array<number>(PHASE_COUNT).fill(0)
  const p25 = new Array<number>(PHASE_COUNT).fill(0)
  const p75 = new Array<number>(PHASE_COUNT).fill(0)
  if (nMatches === 0) return { median, p25, p75, nMatches }

  for (let phase = 0; phase < PHASE_COUNT; phase += 1) {
    const col = shares.map((s) => s[phase]).sort((a, b) => a - b)
    median[phase] = quantileSorted(col, 0.5)
    p25[phase] = quantileSorted(col, 0.25)
    p75[phase] = quantileSorted(col, 0.75)
  }
  return { median, p25, p75, nMatches }
}
