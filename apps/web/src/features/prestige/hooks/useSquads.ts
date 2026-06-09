/**
 * Hooks React Query — Escouade (roster CRUD + défis), Phase C.
 *
 * Consomment les endpoints squad du backend (cf. lib/prestige.ts). Le roster est
 * clé xuid ; le créateur/acteur est identifié par son player_slug (requested_by).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { prestigeApi, type SquadMemberInput } from '@/lib/prestige'

export const squadKeys = {
  mine: (userId: string) => ['prestige', 'squads', userId] as const,
  challenges: (squadId: string) => ['prestige', 'squad-challenges', squadId] as const,
}

/** Escouades dont `userId` (player_slug) est membre-user, roster embarqué. */
export function useMySquads(userId: string) {
  return useQuery({
    queryKey: squadKeys.mine(userId),
    queryFn: () => prestigeApi.listMySquads(userId),
    retry: false,
    enabled: !!userId,
  })
}

/** Défis d'une escouade. */
export function useSquadChallenges(squadId: string) {
  return useQuery({
    queryKey: squadKeys.challenges(squadId),
    queryFn: () => prestigeApi.listSquadChallenges(squadId),
    retry: false,
    enabled: !!squadId,
  })
}

/** Crée une escouade (le créateur = `created_by`, membres initiaux par xuid). */
export function useCreateSquad(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; created_by: string; members?: SquadMemberInput[] }) =>
      prestigeApi.createSquad(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: squadKeys.mine(userId) })
    },
  })
}

/** Ajoute un membre (gardé membre-user côté backend via `requested_by`). */
export function useAddSquadMember(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      squadId,
      ...body
    }: {
      squadId: string
      xuid: string
      gamertag?: string
      requested_by: string
    }) => prestigeApi.addSquadMember(squadId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: squadKeys.mine(userId) })
    },
  })
}

/** Retire un membre (par xuid). */
export function useRemoveSquadMember(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      squadId,
      xuid,
      requestedBy,
    }: {
      squadId: string
      xuid: string
      requestedBy: string
    }) => prestigeApi.removeSquadMember(squadId, xuid, requestedBy),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: squadKeys.mine(userId) })
    },
  })
}

/** Recalcule et persiste la progression d'un défi d'escouade. */
export function useEvaluateSquadChallenge(squadId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, requestedBy }: { id: string; requestedBy: string }) =>
      prestigeApi.evaluateSquadChallenge(id, requestedBy),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: squadKeys.challenges(squadId) })
    },
  })
}
