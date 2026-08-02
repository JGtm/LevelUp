/**
 * outcomeSequence — modèle de données pur de la bande d'outcomes (RLE) et
 * résolution du match cliqué. Séparé du composant `OutcomeSequenceTape.tsx`
 * (React) pour rester testable isolément et éviter d'exporter des valeurs depuis
 * un fichier de composant (react-refresh).
 */
export type OutcomeValue = 'win' | 'loss' | 'tie' | 'dnf'

/**
 * Drapeau de dominance d'un match (canonical.DominanceFlag côté Go) :
 * 1=domination, 2=humiliation, 3=remontada, 4=débandade, 5=contre-remontada.
 * `0`/absent = match ordinaire OU titre sans timeline de score (Halo 5) →
 * AUCUN marqueur : l'absence de drapeau ne raconte rien de faux.
 */
export type DominanceValue = 1 | 2 | 3 | 4 | 5

export interface OutcomePoint {
  outcome: OutcomeValue
  matchId: string
  map?: string
  mode?: string
  /** Libellé pré-formaté pour le tooltip (ex. « 12 mars · Slayer · Aquarius — 14/9 »). */
  label?: string
  /**
   * Drapeau de dominance du match. Optionnel : les consommateurs de la bande
   * qui ne le fournissent pas gardent un rendu STRICTEMENT identique.
   */
  dominance?: DominanceValue
}

/**
 * asDominance — normalise un `dominance_flag` d'API (0..5, éventuellement absent)
 * vers `DominanceValue | undefined`. Toute valeur hors 1..5 (dont 0 et les codes
 * inconnus d'un futur titre) devient `undefined` : pas de marqueur inventé.
 */
export function asDominance(flag: number | null | undefined): DominanceValue | undefined {
  if (flag == null) return undefined
  return flag >= 1 && flag <= 5 ? (flag as DominanceValue) : undefined
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

// ─── Encoches de dominance (marqueur traversant de la bande) ────────────────
//
// Le marqueur de dominance est une ENCOCHE VERTICALE qui traverse la bande et la
// dépasse de quelques pixels (décision d'artefact « A » du lot v7.3-2), et non plus
// un losange logé DANS la bande. Conséquence : il n'y a plus de seuil de densité
// sous lequel on n'affiche rien — l'encoche vit à toute densité, quitte à fusionner
// avec ses voisines quand la bande devient trop serrée pour les distinguer.
//
// Contrainte de contraste (audit 2026-08-02) : aucun token de dominance n'atteint
// 3:1 contre la couleur d'issue qu'il surmonte. C'est la GOUTTIÈRE (fond de tooltip)
// de part et d'autre du cœur coloré qui rend l'encoche lisible, pas sa couleur.

/** Densité (px/match) au-dessous ou égale à laquelle les encoches voisines
 *  fusionnent : sous ~2 px/match, deux cœurs de 1,8 px se toucheraient. */
export const NOTCH_MERGE_MAX_PER_MATCH_PX = 2

/**
 * notchCoreWidth — largeur (px) du cœur coloré d'une encoche : la moitié de la
 * largeur d'un match, bornée à [1,8 ; 5] px. Le plancher garantit un trait visible
 * sur un historique dense, le plafond évite un pavé sur une bande de 12 matchs.
 */
export function notchCoreWidth(perMatchW: number): number {
  return Math.min(5, Math.max(1.8, perMatchW * 0.5))
}

/**
 * notchGutterWidth — largeur (px) de la gouttière de CHAQUE côté du cœur (couleur
 * de fond de tooltip). Amincie sous 4 px/match pour ne pas manger le cœur.
 */
export function notchGutterWidth(perMatchW: number): number {
  return perMatchW < 4 ? 0.75 : 1
}

/** Encoche prête à dessiner (une par match, ou une par groupe fusionné). */
export interface DominanceNotch {
  /** Centre de l'encoche, en unités de match depuis le début du run. */
  center: number
  /** Drapeau du groupe ; `null` = drapeaux MÊLÉS (couleur neutre). */
  flag: DominanceValue | null
  /** Nombre de matchs porteurs d'un drapeau fusionnés dans cette encoche. */
  size: number
}

/**
 * clusterDominanceNotches — HELPER PUR (hors React, testable isolément) : projette
 * les matchs porteurs d'un drapeau d'un run en encoches à dessiner.
 *
 * Au-dessus de {@link NOTCH_MERGE_MAX_PER_MATCH_PX} px/match : une encoche par
 * match, drapeau conservé. En dessous : les encoches dont les empreintes peintes
 * (cœur + 2 gouttières) se chevaucheraient fusionnent en une seule, centrée sur le
 * groupe. Un groupe homogène GARDE sa couleur (l'information n'est pas perdue) ;
 * un groupe de drapeaux mêlés perd la sienne (`flag: null`) plutôt que d'élire
 * arbitrairement un drapeau — le tooltip du run porte le détail.
 */
export function clusterDominanceNotches(
  matches: OutcomePoint[],
  perMatchW: number,
): DominanceNotch[] {
  const flagged: Array<{ index: number; flag: DominanceValue }> = []
  matches.forEach((m, index) => {
    if (m.dominance) flagged.push({ index, flag: m.dominance })
  })
  if (flagged.length === 0) return []
  // NaN / densité confortable → aucune fusion (une encoche par match).
  if (!(perMatchW <= NOTCH_MERGE_MAX_PER_MATCH_PX)) {
    return flagged.map(({ index, flag }) => ({ center: index + 0.5, flag, size: 1 }))
  }

  const footprint = notchCoreWidth(perMatchW) + 2 * notchGutterWidth(perMatchW)
  const out: DominanceNotch[] = []
  let start = flagged[0].index
  let end = start
  let flags: DominanceValue[] = [flagged[0].flag]
  const flush = () => {
    const uniform = flags.every((f) => f === flags[0])
    out.push({ center: (start + end) / 2 + 0.5, flag: uniform ? flags[0] : null, size: flags.length })
  }
  for (let i = 1; i < flagged.length; i++) {
    const { index, flag } = flagged[i]
    if ((index - end) * perMatchW < footprint) {
      end = index
      flags.push(flag)
      continue
    }
    flush()
    start = index
    end = index
    flags = [flag]
  }
  flush()
  return out
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
