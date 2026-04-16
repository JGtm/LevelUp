/**
 * Queries TanStack Query — Setup & Settings (Slice 1).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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
  SettingsResponse,
  UpdateSettingsRequest,
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

export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: () => api.get<SettingsResponse>('/settings'),
    staleTime: 5 * 60 * 1000,
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: UpdateSettingsRequest) =>
      api.patch<SettingsResponse>('/settings', req),
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.settings, data)
    },
  })
}
