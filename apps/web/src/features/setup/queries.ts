/**
 * Queries TanStack Query — Setup (Slice 1).
 *
 * Note : useSettings et useUpdateSettings ont été déplacés dans
 * features/settings/queries.ts (Sprint 51).
 */
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  DeviceFlowStartResponse,
  DeviceFlowStatusResponse,
  CreatePlayerProfileRequest,
  CreatePlayerProfileResponse,
  SmokeTestStartRequest,
  AsyncJobStatus,
  InitialSyncStartRequest,
} from '@/lib/api/types'

// useSetupStatus() supprimé (sprint 29) : GET /setup/status est un artefact mort.
// Utiliser BootstrapResponse.setup_state à la place.

export function useStartDeviceFlow() {
  return useMutation({
    mutationFn: () => api.post<DeviceFlowStartResponse>('/auth/device-flow/start', {}),
  })
}

export function useDeviceFlowStatus(attemptId: string, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.deviceFlow(attemptId),
    queryFn: () => api.get<DeviceFlowStatusResponse>(`/auth/device-flow/${attemptId}`),
    enabled: enabled && !!attemptId,
    refetchInterval: 5_000,
    staleTime: 0,
  })
}

export function useCreatePlayer() {
  return useMutation({
    mutationFn: (req: CreatePlayerProfileRequest) =>
      api.post<CreatePlayerProfileResponse>('/setup/players', req),
  })
}

export function useStartSmokeTest() {
  return useMutation({
    mutationFn: (req: SmokeTestStartRequest) =>
      api.post<AsyncJobStatus>('/setup/smoke-test', req),
  })
}

export function useStartInitialSync() {
  return useMutation({
    mutationFn: (req: InitialSyncStartRequest) =>
      api.post<AsyncJobStatus>('/sync/initial', req),
  })
}

export function useStartDeltaSync() {
  return useMutation({
    mutationFn: (playerSlug: string) =>
      api.post<AsyncJobStatus>(`/players/${playerSlug}/sync`, {}),
  })
}

export function useStartSyncAll() {
  return useMutation({
    mutationFn: () => api.post<AsyncJobStatus>('/sync/all', {}),
  })
}

export function useJobStatus(jobId: string, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.job(jobId),
    queryFn: () => api.get<AsyncJobStatus>(`/jobs/${jobId}`),
    enabled: enabled && !!jobId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (status === 'succeeded' || status === 'failed' || status === 'cancelled' || status === 'interrupted') return false
      return 3_000
    },
    staleTime: 0,
  })
}
