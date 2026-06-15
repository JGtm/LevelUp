/**
 * Queries de la section admin « Titres » (PMT-14 volet A).
 *
 * Consomme GET /admin/titles (liste) et GET /admin/titles/{slug} (détail).
 * Read-only, admin-gated côté Go. Les types sont définis localement (petite
 * surface, pas de dépendance au codegen openapi).
 */
import { useQuery } from '@tanstack/react-query'

import { api, API_BASE_URL } from '@/lib/api/client'
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

// --- Diagnostic santé d'un titre (config TOML + réalité DB) ---

// Formes imbriquées inlinées dans TitleDiagnostic (knip : pas d'export mort ;
// elles ne sont consommées que via l'inférence du hook côté page).
export interface TitleDiagnostic {
  title_slug: string
  config_files: { name: string; present: boolean; required: boolean }[]
  databases: {
    name: string
    exists: boolean
    tables?: { name: string; exists: boolean; rows: number }[]
    error?: string
  }[]
  /** Écarts déclaré-vs-réalité (vide/absent = pas de drift). */
  drifts?: { feature: string; kind: string; computed: string; reason: string }[]
}

export function useAdminTitleDiagnostic(slug: string | null) {
  return useQuery({
    queryKey: queryKeys.adminTitleDiagnostic(slug ?? ''),
    queryFn: () => api.get<TitleDiagnostic>(`/admin/titles/${slug}/diagnostic`),
    enabled: !!slug,
    staleTime: 30_000,
    retry: false,
  })
}

// --- Brouillon capabilities.toml (D10 : presse-papier, zéro écriture serveur) ---

// Réponse text/plain (pas JSON) → fetch dédié plutôt que le client api.
// Le slug est dans le path (route admin title-agnostic) ; cookie de session via
// credentials. À copier côté front avec navigator.clipboard.
export async function fetchTitleTomlDraft(slug: string): Promise<string> {
  const res = await fetch(`${API_BASE_URL}/admin/titles/${slug}/toml-draft`, {
    credentials: 'include',
  })
  if (!res.ok) {
    throw new Error(`toml-draft ${res.status}`)
  }
  return res.text()
}
