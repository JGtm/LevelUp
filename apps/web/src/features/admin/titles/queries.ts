/**
 * Queries de la section admin « Titres » (PMT-14 volet A).
 *
 * Consomme GET /admin/titles (liste) et GET /admin/titles/{slug} (détail).
 * Read-only, admin-gated côté Go. Les types sont définis localement (petite
 * surface, pas de dépendance au codegen openapi).
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

export type TitleStatus = 'active' | 'coming_soon' | 'archived'

export interface AdminTitleSummary {
  slug: string
  name: string
  provider: string
  icon_url: string
  status: TitleStatus
  capabilities: string[]
  is_default: boolean
  xbox_title_id: string
  steam_app_id: string
  has_mappings: boolean
}

export interface AdminTitlesListResponse {
  titles: AdminTitleSummary[]
  count: number
}

export interface AdminTitleDetail extends AdminTitleSummary {
  schema_version?: number
  /** capabilities.toml : capabilityKey → supported|degraded|not_exposed */
  declared_capabilities?: Record<string, string>
  /** cascade calculée : featureKey → available|degraded|unavailable */
  feature_matrix?: Record<string, string>
}

export function useAdminTitles() {
  return useQuery({
    queryKey: queryKeys.adminTitles,
    queryFn: () => api.get<AdminTitlesListResponse>('/admin/titles'),
    // Registre statique au boot : pas de polling, cache tranquille.
    staleTime: 60_000,
    retry: false,
  })
}

export function useAdminTitleDetail(slug: string | null) {
  return useQuery({
    queryKey: queryKeys.adminTitleDetail(slug ?? ''),
    queryFn: () => api.get<AdminTitleDetail>(`/admin/titles/${slug}`),
    enabled: !!slug,
    staleTime: 60_000,
    retry: false,
  })
}
