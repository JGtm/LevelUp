/**
 * Hooks TanStack Query pour le coach_advisor (ADR 0020 Phase 10).
 *
 * - useCoachProposals(slug, status) : GET /players/{slug}/coach/proposals
 * - useAcceptCoachProposal(slug)    : POST /accept (mutation + invalidation)
 * - useDismissCoachProposal(slug)   : POST /dismiss (mutation + invalidation)
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

import type {
  AcceptResponse,
  DismissResponse,
  ProposalStatus,
  ProposalsListResponse,
} from './types'

/**
 * Liste les proposals coach pour un joueur, filtré optionnellement par status.
 *
 * Si CoachProactiveMode est désactivé côté settings, l'endpoint retourne une
 * liste vide (le hook ne s'occupe pas de la lecture du toggle — c'est l'UI
 * qui décide d'afficher ou non la carte).
 */
export function useCoachProposals(playerSlug: string, status: ProposalStatus | undefined) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.coachProposals(playerSlug, titleSlug, status),
    queryFn: () => {
      const qs = status ? `?status=${status}` : ''
      return api.get<ProposalsListResponse>(`/players/${playerSlug}/coach/proposals${qs}`)
    },
    enabled: !!playerSlug,
    staleTime: 60_000,
  })
}

/**
 * Accept une proposal (matérialise un challenge ou un arc via Prestige).
 * Invalide la liste après succès pour rafraîchir le statut.
 */
export function useAcceptCoachProposal(playerSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (proposalId: string) =>
      api.post<AcceptResponse>(`/players/${playerSlug}/coach/proposals/${proposalId}/accept`),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: queryKeys.coachAll(playerSlug) })
    },
  })
}

/**
 * Dismiss une proposal. Idempotent côté backend (no-op si déjà résolue).
 */
export function useDismissCoachProposal(playerSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (proposalId: string) =>
      api.post<DismissResponse>(`/players/${playerSlug}/coach/proposals/${proposalId}/dismiss`),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: queryKeys.coachAll(playerSlug) })
    },
  })
}
