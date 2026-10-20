/**
 * timeline.ts — logique pure de la timeline du pipeline post-sync (testable
 * sans React) : segments proportionnels à la durée, couleur stable par étape
 * (tokens chart-series), étapes dominantes.
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { PostSyncStepTiming } from '@/lib/api/types'

export interface TimelineSegment {
  step: string
  durationMs: number
  items: number
  /** Part de la durée totale (0..100, arrondie à 0.1). */
  pct: number
  token: SemanticToken
}

/** Couleur stable par étape (cycle sur les 8 tokens série). */
const STEP_TOKENS: SemanticToken[] = [
  'chart-series-1',
  'chart-series-2',
  'chart-series-3',
  'chart-series-4',
  'chart-series-5',
  'chart-series-6',
  'chart-series-7',
  'chart-series-8',
]

/** Ordre canonique du pipeline (mapping couleur stable inter-joueurs). */
const STEP_ORDER = [
  'enrichment_rows',
  'scoring',
  'convergence_events',
  'convergence_psa',
  'citations',
  'dominance',
  'skill_rating',
  'csr_snapshots',
  'friends',
  'aggregates',
  'media_scan',
  'achievements',
]

export function stepToken(step: string): SemanticToken {
  const idx = STEP_ORDER.indexOf(step)
  return STEP_TOKENS[(idx >= 0 ? idx : STEP_ORDER.length) % STEP_TOKENS.length]
}

/**
 * Construit les segments de timeline à partir des timings d'étapes.
 * Les étapes à 0 ms sont écartées (un pipeline sain a beaucoup d'étapes
 * no-op) ; les proportions sont calculées sur la somme des durées non
 * nulles. Retourne [] si rien de significatif.
 */
export function buildTimelineSegments(timings: PostSyncStepTiming[] | undefined): TimelineSegment[] {
  if (!timings?.length) return []
  const significant = timings.filter((t) => t.duration_ms > 0)
  const total = significant.reduce((acc, t) => acc + t.duration_ms, 0)
  if (total <= 0) return []
  return significant.map((t) => ({
    step: t.step,
    durationMs: t.duration_ms,
    items: t.items,
    pct: Math.round((t.duration_ms / total) * 1000) / 10,
    token: stepToken(t.step),
  }))
}

/** Les n étapes les plus lentes (pour le résumé texte sous la timeline). */
export function slowestSteps(segments: TimelineSegment[], n = 3): TimelineSegment[] {
  return [...segments].sort((a, b) => b.durationMs - a.durationMs).slice(0, n)
}

/**
 * Étape dominante d'un pipeline : retournée seulement si elle dépasse
 * thresholdPct ET que le pipeline a duré au moins minTotalMs (un pipeline de
 * 200 ms dominé à 90 % n'est pas un goulot).
 */
export function dominantStep(
  timings: PostSyncStepTiming[] | undefined,
  thresholdPct = 60,
  minTotalMs = 30_000,
): TimelineSegment | undefined {
  const segments = buildTimelineSegments(timings)
  if (!segments.length) return undefined
  const total = segments.reduce((acc, s) => acc + s.durationMs, 0)
  if (total < minTotalMs) return undefined
  const top = slowestSteps(segments, 1)[0]
  return top.pct >= thresholdPct ? top : undefined
}
