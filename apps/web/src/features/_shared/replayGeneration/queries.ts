/**
 * TanStack Query — génération à la demande du rejeu d'un match qui n'en a pas (placement local : application
 * de bureau et poste de développement, capacité `can_generate_replays`) : état du match
 * (GET /players/{slug}/matches/{id}/replay/generation) et demande de génération
 * (POST .../replay/generate).
 *
 * Types écrits à la main : `openapi-gen` décrit le routeur du profil démo, qui ne monte aucune route sous
 * capacité ; ils entreront dans `generated.ts` avec le routeur de contrat (liste MANUAL_CAPABILITY_TYPES de
 * lib/api/response-types.guard.test.ts, qui le rappelle dès qu'ils y entrent).
 *
 * L'état se relit toutes les 3 s tant qu'une demande est en file ou en cours, jamais sinon. Au passage à `ready`,
 * le rejeu, la vue du match et l'accueil se relisent : le rejeu s'affiche, `replay_available` devient vrai.
 */
import { useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

import { generationInFlight } from './logic'

/** États servis par le serveur : des codes, les textes sont écrits ici (`text.ts`). */
export type ReplayGenerationState =
  | 'generatable'
  | 'queued'
  | 'running'
  | 'failed'
  | 'film_unavailable'
  | 'not_supported'
  | 'ready'

/** Raisons d'un échec (`failed`) ou d'un refus (`not_supported`). */
export type ReplayGenerationReason =
  | 'film_not_ready'
  | 'fetch_failed'
  | 'token_unavailable'
  | 'build_failed'
  | 'map_not_in_catalog'
  | 'unknown_film_key'
  | 'mode_unmanaged'
  | 'match_unknown'
  | 'no_token'

/** Étape d'une demande en cours : le film se télécharge, puis le rejeu se construit. */
export type ReplayGenerationPhase = 'downloading' | 'building'

export interface ReplayGenerationStatus {
  state: ReplayGenerationState
  reason?: ReplayGenerationReason
  /** Seulement en `running`. */
  phase?: ReplayGenerationPhase
  /** Seulement en `queued` : 1 = le prochain. */
  position?: number
  /** La file attend la fin de la partie (mode jeu). */
  waiting_for_game?: boolean
  /** La file est en pause à la demande de l'utilisateur. */
  paused?: boolean
  /** Seulement en `failed` : un nouvel essai a une chance d'aboutir. */
  retryable?: boolean
}

/** Corps de la réponse de POST .../replay/generate : `created` est faux quand la demande existait déjà. */
export interface ReplayGenerationOutcome {
  created: boolean
  status: ReplayGenerationStatus
}

/** Période de relecture de l'état pendant l'attente, en ms. */
export const REPLAY_GENERATION_POLL_MS = 3_000

/** Refus qui disent que l'état du serveur a bougé (match qui ne se génère plus, file pleine) : l'état se relit. */
const REREAD_STATUSES: readonly number[] = [409, 429]

function replayPath(playerSlug: string, matchId: string, leaf: 'generation' | 'generate'): string {
  return `/players/${playerSlug}/matches/${matchId}/replay/${leaf}`
}

/**
 * État de la génération du rejeu d'un match. `enabled` : la capacité `can_generate_replays` est vraie et le rejeu
 * manque ; sans quoi aucune requête ne part.
 */
export function useReplayGenerationStatus(playerSlug: string, matchId: string, enabled: boolean) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const qc = useQueryClient()
  const query = useQuery({
    queryKey: queryKeys.matchReplayGeneration(playerSlug, titleSlug, matchId),
    queryFn: () => api.get<ReplayGenerationStatus>(replayPath(playerSlug, matchId, 'generation')),
    enabled: enabled && !!playerSlug && !!matchId,
    staleTime: REPLAY_GENERATION_POLL_MS,
    refetchInterval: (q) => (generationInFlight(q.state.data) ? REPLAY_GENERATION_POLL_MS : false),
  })
  const ready = query.data?.state === 'ready'
  useEffect(() => {
    if (!ready) return
    void qc.invalidateQueries({ queryKey: queryKeys.matchReplay(playerSlug, titleSlug, matchId) })
    void qc.invalidateQueries({ queryKey: queryKeys.matchViewAllLocales(playerSlug, titleSlug, matchId) })
    void qc.invalidateQueries({ queryKey: queryKeys.homeAll(playerSlug) })
  }, [ready, qc, playerSlug, titleSlug, matchId])
  return query
}

/**
 * Demande la génération du rejeu. La réponse porte déjà le nouvel état, posé dans le cache ; un 409 ou un 429
 * relit l'état réel plutôt que de garder l'état supposé.
 */
export function useRequestReplayGeneration(playerSlug: string, matchId: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const qc = useQueryClient()
  const key = queryKeys.matchReplayGeneration(playerSlug, titleSlug, matchId)
  return useMutation({
    mutationFn: () => api.post<ReplayGenerationOutcome>(replayPath(playerSlug, matchId, 'generate'), {}),
    onSuccess: (outcome) => qc.setQueryData(key, outcome.status),
    onError: (err) => {
      const status = (err as { status?: unknown } | null)?.status
      if (typeof status === 'number' && REREAD_STATUSES.includes(status)) void qc.invalidateQueries({ queryKey: key })
    },
  })
}
