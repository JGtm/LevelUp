/**
 * Queries TanStack Query — Groupes/familles (accès mutuel aux données).
 *
 * Endpoints user-facing (RequireAuth + identité Halo) : cf. apps/go-api groups.go.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { Group, InviteCode } from '@/lib/api/types'

export function useMyGroups(enabled = true) {
  return useQuery({
    queryKey: queryKeys.groups,
    queryFn: () => api.get<Group[]>('/groups'),
    enabled,
  })
}

export function useCreateGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.post<Group>('/groups', { name }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.groups }),
  })
}

export function useRenameGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api.patch<Group>(`/groups/${encodeURIComponent(id)}`, { name }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.groups }),
  })
}

export function useDeleteGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/groups/${encodeURIComponent(id)}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.groups }),
  })
}

/** Génère une invitation "rejoindre le groupe" (consommée via le login Xbox SSO). */
export function useCreateGroupInvite() {
  return useMutation({
    mutationFn: ({ id, expiresInDays }: { id: string; expiresInDays?: number }) =>
      api.post<InviteCode>(`/groups/${encodeURIComponent(id)}/invites`, {
        expires_in_days: expiresInDays ?? 7,
      }),
  })
}
