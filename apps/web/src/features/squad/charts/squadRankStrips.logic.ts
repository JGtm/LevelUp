/**
 * squadRankStrips.logic — les décisions PURES du sous-graphe « Rang » de la page Contributions
 * (Escouade › Performances) : un bandeau par joueur, une courbe du rang dans sa seule couleur.
 *
 * L'ÉCHELLE EST CE QUI FAIT LA LECTURE. Chaque bandeau est centré sur les valeurs de SON joueur
 * (deux joueurs à 1 200 et à 1 500 points se lisent côte à côte), mais TOUS les bandeaux ont la
 * MÊME amplitude : le plus grand écart (max − min) parmi les joueurs, plus une marge. Les pentes
 * se comparent d'un bandeau à l'autre, et des rangs proches ne se recouvrent pas. Le MMR d'équipe,
 * dans un dernier bandeau, a sa propre échelle (il ne se compare pas à un rang individuel).
 *
 * Aucune dépendance React/ECharts : `squadRankStripsChart.ts` ne fait que peindre ce que ce
 * module décide.
 */
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

/** Marge de l'échelle commune : part du plus grand écart, de chaque côté. */
export const RANK_MARGIN_SHARE = 0.15
/** Marge plancher (points de rang) : un joueur au rang plat garde un bandeau lisible. */
export const RANK_MARGIN_MIN = 4

/** Nature du rang : CSR (classé) ou LUSR (note maison), exclusifs par match. */
export type RankKind = 'csr' | 'lusr'

/** Un point d'un bandeau : le rang d'un joueur à l'issue d'un match de la session. */
export interface RankPoint {
  /** Position du match dans la session (axe X partagé). */
  order: number
  rating: number
  /** Variation du rang sur ce match, signée ; `null` quand elle n'est pas servie. */
  delta: number | null
  kind: RankKind
}

/** Le bandeau d'un joueur. */
export interface RankStrip {
  gamertag: string
  points: RankPoint[]
  /** Nature la plus fréquente de la session (à égalité : la plus récente) : celle de l'étiquette. */
  kind: RankKind
  /** Rang du dernier match de cette nature. */
  endRating: number
  /** Variation nette sur la session, première variation comprise : fin − (début − variation du début). */
  sessionDelta: number
  /** Centre du bandeau : milieu de l'étendue des valeurs du joueur, entier. */
  center: number
}

/** Bornes d'un bandeau. */
export interface StripBounds {
  min: number
  max: number
}

function kindOf(p: SquadPerformanceSeriesPoint): RankKind {
  return p.skill_rating_type === 'csr' ? 'csr' : 'lusr'
}

/**
 * rankStrips construit les bandeaux des joueurs (dans l'ordre donné) qui ont au moins un rang
 * sur la session. `rows` est la série par joueur servie par la page ; un joueur sans rang n'a pas
 * de bandeau.
 */
export function rankStrips(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  players: readonly string[],
): RankStrip[] {
  const out: RankStrip[] = []
  for (const gamertag of players) {
    const points: RankPoint[] = (rows[gamertag] ?? [])
      .filter((p) => p.skill_rating != null)
      .map((p) => ({
        order: p.match_order,
        rating: p.skill_rating as number,
        delta: p.skill_delta ?? null,
        kind: kindOf(p),
      }))
      .sort((a, b) => a.order - b.order)
    if (points.length === 0) continue
    const csr = points.filter((p) => p.kind === 'csr').length
    const lusr = points.length - csr
    const last = points[points.length - 1]
    const kind: RankKind = csr === lusr ? last.kind : csr > lusr ? 'csr' : 'lusr'
    const ofKind = points.filter((p) => p.kind === kind)
    const first = ofKind[0]
    const end = ofKind[ofKind.length - 1]
    const values = points.map((p) => p.rating)
    out.push({
      gamertag,
      points,
      kind,
      endRating: end.rating,
      sessionDelta: end.rating - (first.rating - (first.delta ?? 0)),
      center: Math.round((Math.min(...values) + Math.max(...values)) / 2),
    })
  }
  return out
}

/** L'étendue (max − min) des valeurs d'un bandeau. */
export function stripRange(strip: RankStrip): number {
  const values = strip.points.map((p) => p.rating)
  return Math.max(...values) - Math.min(...values)
}

/**
 * halfAmplitude — la DEMI-amplitude commune à tous les bandeaux : la moitié du plus grand écart
 * de la session, plus la marge (`RANK_MARGIN_SHARE` de cet écart, au moins `RANK_MARGIN_MIN`),
 * arrondie à l'entier supérieur. 0 sans bandeau.
 */
export function halfAmplitude(strips: readonly RankStrip[]): number {
  if (strips.length === 0) return 0
  const widest = Math.max(...strips.map(stripRange))
  const margin = Math.max(widest * RANK_MARGIN_SHARE, RANK_MARGIN_MIN)
  return Math.ceil(widest / 2 + margin)
}

/** Les bornes d'un bandeau : son centre, plus et moins la demi-amplitude commune. */
export function stripBounds(strip: RankStrip, half: number): StripBounds {
  return { min: strip.center - half, max: strip.center + half }
}

/** MMR d'équipe : une valeur par match (la première servie), `null` sans valeur. */
export function teamMmrByMatch(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  players: readonly string[],
  matches: number,
): (number | null)[] {
  const out = new Array<number | null>(matches).fill(null)
  for (const gamertag of players) {
    for (const p of rows[gamertag] ?? []) {
      if (p.team_mmr != null && out[p.match_order] === null) out[p.match_order] = p.team_mmr
    }
  }
  return out
}

/** Bornes du bandeau du MMR d'équipe : ses valeurs, avec la même règle de marge, entières. */
export function mmrBounds(values: readonly (number | null)[]): StripBounds | null {
  const present = values.filter((v): v is number => v !== null)
  if (present.length === 0) return null
  const lo = Math.min(...present)
  const hi = Math.max(...present)
  const margin = Math.max((hi - lo) * RANK_MARGIN_SHARE, RANK_MARGIN_MIN)
  return { min: Math.floor(lo - margin), max: Math.ceil(hi + margin) }
}
