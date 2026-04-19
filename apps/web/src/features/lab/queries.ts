import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type {
  LabContractsResponse,
  LabDiagnosticsResponse,
  LabResourcesResponse,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

export interface LabResourcesParams {
  snapshotKey?: string
  assetID?: string
  assetSearch?: string
  medalID?: number | null
  medalSearch?: string
  limit?: number
}

function buildLabResourcesPath(params: LabResourcesParams) {
  const search = new URLSearchParams()
  if (params.snapshotKey) {
    search.set('snapshot_key', params.snapshotKey)
  }
  if (params.assetID) {
    search.set('asset_id', params.assetID)
  }
  if (params.assetSearch) {
    search.set('asset_search', params.assetSearch)
  }
  if (params.medalID != null) {
    search.set('medal_id', String(params.medalID))
  }
  if (params.medalSearch) {
    search.set('medal_search', params.medalSearch)
  }
  if (params.limit) {
    search.set('limit', String(params.limit))
  }
  const qs = search.toString()
  return qs ? `/lab/resources?${qs}` : '/lab/resources'
}

function buildLabResourcesHash(params: LabResourcesParams) {
  return JSON.stringify(params)
}

export function useLabResources(params: LabResourcesParams, enabled = true) {
  const requestHash = buildLabResourcesHash(params)

  return useQuery({
    queryKey: queryKeys.labResources(requestHash),
    queryFn: () => api.get<LabResourcesResponse>(buildLabResourcesPath(params)),
    enabled,
    staleTime: 30 * 1000,
  })
}

export function useLabContracts(enabled = true) {
  return useQuery({
    queryKey: queryKeys.labContracts,
    queryFn: () => api.get<LabContractsResponse>('/lab/contracts'),
    enabled,
    staleTime: 60 * 1000,
  })
}

export function useLabDiagnostics(enabled = true) {
  return useQuery({
    queryKey: queryKeys.labDiagnostics,
    queryFn: () => api.get<LabDiagnosticsResponse>('/lab/diagnostics'),
    enabled,
    staleTime: 60 * 1000,
  })
}
