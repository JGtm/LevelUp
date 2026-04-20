/**
 * Queries TanStack Query — Auth locale (login/register/logout).
 */
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
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
  return useMutation({
    mutationFn: (req: LoginRequest) =>
      api.post<LoginResponse>('/auth/login', req),
  })
}

export function useRegister() {
  return useMutation({
    mutationFn: (req: RegisterRequest) =>
      api.post<RegisterResponse>('/auth/register', req),
  })
}

export function useLogout() {
  return useMutation({
    mutationFn: () => api.post<void>('/auth/logout'),
  })
}

// ---------------------------------------------------------------------------
// Admin : utilisateurs
// ---------------------------------------------------------------------------

export function useAdminUsers() {
  return useQuery({
    queryKey: ['admin', 'users'] as const,
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
// Admin : invitations
// ---------------------------------------------------------------------------

export function useAdminInvites() {
  return useQuery({
    queryKey: ['admin', 'invites'] as const,
    queryFn: () => api.get<AdminInviteSummary[]>('/admin/invites'),
  })
}

export function useGenerateInvite() {
  return useMutation({
    mutationFn: (expiresInDays?: number) =>
      api.post<InviteCode>('/admin/invites', { expires_in_days: expiresInDays ?? 7 }),
  })
}

export function useRevokeInvite() {
  return useMutation({
    mutationFn: (code: string) =>
      api.delete<void>(`/admin/invites/${encodeURIComponent(code)}`),
  })
}
