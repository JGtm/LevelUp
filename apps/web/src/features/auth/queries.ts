/**
 * Queries TanStack Query — Auth locale (login/register/logout).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  AdminUserSummary,
  AdminInviteSummary,
  InviteCode,
} from '@/lib/api/types'

// ---------------------------------------------------------------------------
// Auth publique
// ---------------------------------------------------------------------------

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: LoginRequest) =>
      api.post<LoginResponse>('/auth/login', req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap }),
  })
}

export function useRegister() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: RegisterRequest) =>
      api.post<RegisterResponse>('/auth/register', req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap }),
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<void>('/auth/logout'),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap }),
  })
}

/** PR-C : définition/changement opt-in du mot de passe de l'utilisateur connecté. */
export function useSetPassword() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (password: string) => api.post<void>('/auth/password', { password }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap }),
  })
}

// ---------------------------------------------------------------------------
// Admin : utilisateurs
// ---------------------------------------------------------------------------

export function useAdminUsers() {
  return useQuery({
    queryKey: queryKeys.adminUsers,
    queryFn: () => api.get<AdminUserSummary[]>('/admin/users'),
  })
}

export function useDeleteUser() {
  return useMutation({
    mutationFn: (username: string) =>
      api.delete<void>(`/admin/users/${encodeURIComponent(username)}`),
  })
}

export function useChangeRole() {
  return useMutation({
    mutationFn: ({ username, role }: { username: string; role: 'admin' | 'user' }) =>
      api.patch<void>(`/admin/users/${encodeURIComponent(username)}/role`, { role }),
  })
}

export function useResetPassword() {
  return useMutation({
    mutationFn: ({ username, newPassword }: { username: string; newPassword: string }) =>
      api.patch<void>(`/admin/users/${encodeURIComponent(username)}/password`, { new_password: newPassword }),
  })
}

// ---------------------------------------------------------------------------
// Admin : invitations SANS groupe
// ---------------------------------------------------------------------------
//
// Deux invitations coexistent, et elles ne font pas la même chose : celle d'un
// propriétaire de groupe (features/groups/queries.ts) fait REJOINDRE un groupe ;
// celle-ci, réservée à l'admin, crée un compte qui ne voit que lui-même — et lui
// donne le droit de créer son profil joueur, même sur instance verrouillée.

export function useAdminInvites() {
  return useQuery({
    queryKey: queryKeys.adminInvites,
    queryFn: () => api.get<AdminInviteSummary[]>('/admin/invites'),
  })
}

export function useGenerateAdminInvite() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (expiresInDays?: number) =>
      api.post<InviteCode>('/admin/invites', { expires_in_days: expiresInDays ?? 7 }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.adminInvites }),
  })
}

export function useRevokeAdminInvite() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => api.delete<void>(`/admin/invites/${encodeURIComponent(code)}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.adminInvites }),
  })
}
