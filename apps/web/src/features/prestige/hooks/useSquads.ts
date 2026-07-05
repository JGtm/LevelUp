/**
 * Hooks React Query — Escouade (roster CRUD + défis), Phase C.
 *
 * Consomment les endpoints squad du backend (cf. lib/prestige.ts). Le roster est
 * clé xuid ; le créateur/acteur est identifié par son player_slug (requested_by).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  prestigeApi,
  type SquadMemberInput,
  type SquadMode,
  type EvalType,
  type WindowType,
} from '@/lib/prestige'
import { queryKeys } from '@/lib/query/keys'

/** Escouades dont `userId` (player_slug) est membre-user, roster embarqué. */
export function useMySquads(userId: string) {
  return useQuery({
    queryKey: queryKeys.squad.mine(userId),
    queryFn: () => prestigeApi.listMySquads(userId),
    retry: false,
    enabled: !!userId,
  })
}

/** Défis d'une escouade. */
export function useSquadChallenges(squadId: string) {
  return useQuery({
    queryKey: queryKeys.squad.challenges(squadId),
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
      qc.invalidateQueries({ queryKey: queryKeys.squad.mine(userId) })
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
      qc.invalidateQueries({ queryKey: queryKeys.squad.mine(userId) })
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
      qc.invalidateQueries({ queryKey: queryKeys.squad.mine(userId) })
    },
  })
}

/** Renomme une escouade (gardé membre-user côté backend via `requested_by`). */
export function useRenameSquad(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ squadId, name }: { squadId: string; name: string }) =>
      prestigeApi.renameSquad(squadId, { name, requested_by: userId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.squad.mine(userId) })
    },
  })
}

/** Supprime une escouade (retrait append-only de tous les membres). */
export function useDeleteSquad(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ squadId }: { squadId: string }) =>
      prestigeApi.deleteSquad(squadId, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.squad.mine(userId) })
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
      qc.invalidateQueries({ queryKey: queryKeys.squad.challenges(squadId) })
    },
  })
}

/** Orientation coach de l'escouade : l'axe focal (le plus faible) à renforcer. */
export function useSquadOrientation(squadId: string, requestedBy: string) {
  return useQuery({
    queryKey: queryKeys.squad.orientation(squadId, requestedBy),
    queryFn: () => prestigeApi.squadOrientation(squadId, requestedBy),
    retry: false,
    enabled: !!squadId && !!requestedBy,
  })
}

/** Génère un pool de défis suggérés (biaisé coach) pour l'escouade. */
export function useRefreshSquadPool() {
  return useMutation({
    mutationFn: ({
      squadId,
      titleSlug,
      requestedBy,
    }: {
      squadId: string
      titleSlug: string
      requestedBy: string
    }) => prestigeApi.refreshSquadPool(squadId, { title_slug: titleSlug, requested_by: requestedBy }),
  })
}

/** Crée un défi d'escouade (depuis un template du pool). */
export function useCreateSquadChallenge(squadId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      template_id: string
      title_slug: string
      mode: SquadMode
      eval_type: EvalType
      window_type: WindowType
      window_value?: string
      target_per_member?: number
      created_by: string
    }) => prestigeApi.createSquadChallenge(squadId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.squad.challenges(squadId) })
    },
  })
}
