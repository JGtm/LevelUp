/**
 * Rafraîchissement tant qu'une analyse de film est en cours.
 *
 * Le serveur pose `analysis_pending` sur un match synchronisé dont le film n'est pas encore
 * analysé (règle `analysis.IsAnalysisPending`, bornée à 24 h après la synchro). Tant qu'au
 * moins un match AFFICHÉ porte ce signal, les requêtes qui le portent se rechargent toutes
 * les 60 s ; elles s'arrêtent d'elles-mêmes quand plus rien n'est en cours.
 *
 * Le serveur sert aussi les CAUSES (`analysis_pending_parts`, mêmes conditions que le booléen,
 * invariant `analysis_pending == (parts non vide)`) : chaque emplacement en attente regarde
 * SA cause (`hasPendingPart`), jamais le booléen seul. La pastille, l'infobulle, le bandeau et
 * le rechargement restent pilotés par le booléen.
 *
 * `analysis_state` (posé seulement quand `analysis_pending` est vrai) dit si un décodage de film
 * porte sur CE match (`in_progress`) ou si le match attend son tour (`queued`) : « en cours »
 * ne s'affiche que dans le premier cas (`isAnalysisInProgress`).
 */
import type { HomePageResponse, MatchViewResponse } from '@/lib/api/types'

/** Période de rechargement tant qu'une analyse est en cours. */
export const ANALYSIS_PENDING_REFETCH_MS = 60_000

/** Période du poll de première synchro de l'accueil (joueur sans aucun match). */
export const FIRST_SYNC_REFETCH_MS = 30_000

/** Les causes servies par le serveur (`domain.AnalysisPart*`). */
export const ANALYSIS_PART = {
  killSource: 'kill_source',
  citations: 'citations',
  replay: 'replay',
} as const

export type AnalysisPendingPart = (typeof ANALYSIS_PART)[keyof typeof ANALYSIS_PART]

/** Les états servis par le serveur (`domain.AnalysisState*`). */
export const ANALYSIS_STATE = {
  inProgress: 'in_progress',
  queued: 'queued',
} as const

/** Tout objet qui porte l'état de l'analyse (en-tête de la vue match). */
export type AnalysisStateCarrier = { analysis_state?: string | null }

/** Un décodage porte sur CE match. Absent ou autre valeur : en attente, jamais « en cours ». */
export function isAnalysisInProgress(item: AnalysisStateCarrier | null | undefined): boolean {
  return item?.analysis_state === ANALYSIS_STATE.inProgress
}

/** Tout objet qui porte les causes (tuile de l'accueil, en-tête de la vue match). */
export type AnalysisPendingCarrier = { analysis_pending_parts?: ReadonlyArray<string> | null }

/** L'élément attend CETTE cause. Absent, vide ou null : aucune cause. */
export function hasPendingPart(item: AnalysisPendingCarrier | null | undefined, part: AnalysisPendingPart): boolean {
  return item?.analysis_pending_parts?.includes(part) === true
}

type PendingFlag = { analysis_pending?: boolean | null }

/** Au moins un des éléments porte le signal. */
export function anyAnalysisPending(items: ReadonlyArray<PendingFlag | null | undefined> | null | undefined): boolean {
  return (items ?? []).some((item) => item?.analysis_pending === true)
}

type HomeData = Pick<HomePageResponse, 'recent_matches' | 'favorite_matches'> & {
  hero?: { kpis?: { total_matches?: number | null } | null } | null
}

/** Une tuile affichée de l'accueil (récents ou favoris) a son analyse en cours. */
export function homeHasAnalysisPending(data: Pick<HomePageResponse, 'recent_matches' | 'favorite_matches'> | undefined): boolean {
  return anyAnalysisPending(data?.recent_matches) || anyAnalysisPending(data?.favorite_matches)
}

/**
 * `refetchInterval` de l'accueil. La première synchro (aucun match encore) garde son poll de
 * 30 s ; sinon 60 s tant qu'une tuile est en cours d'analyse ; sinon plus de poll.
 */
export function homeRefetchInterval(data: HomeData | undefined): number | false {
  if ((data?.hero?.kpis?.total_matches ?? 0) <= 0) return FIRST_SYNC_REFETCH_MS
  return homeHasAnalysisPending(data) ? ANALYSIS_PENDING_REFETCH_MS : false
}

/** Fraîcheur de l'accueil : courte tant qu'une analyse est en cours (retour sur la page). */
export function homeStaleTime(data: Pick<HomePageResponse, 'recent_matches' | 'favorite_matches'> | undefined, normalMs: number): number {
  return homeHasAnalysisPending(data) ? ANALYSIS_PENDING_REFETCH_MS : normalMs
}

/** La vue match affichée a son analyse en cours. */
export function matchViewHasAnalysisPending(data: Partial<Pick<MatchViewResponse, 'header'>> | undefined): boolean {
  return data?.header?.analysis_pending === true
}
