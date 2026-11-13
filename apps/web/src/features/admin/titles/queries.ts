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

import type { components } from '@/lib/api/generated'
export type TitleStatus = 'active' | 'coming_soon' | 'archived'

export type AdminTitleSummary = Omit<components['schemas']['AdminTitleSummary'], 'status'> & {
  status: TitleStatus
}

export type AdminTitlesListResponse = Omit<components['schemas']['AdminTitlesListResponse'], 'titles'> & {
  titles: AdminTitleSummary[] | null
}

export type AdminTitleDetail = Omit<components['schemas']['AdminTitleDetail'], 'status'> & {
  status: TitleStatus
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
export type TitleDiagnostic = components['schemas']['TitleDiagnostic']

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

// Réponse text/plain (pas JSON) → api.getText. Le slug est dans le path (route admin
// title-agnostic) ; cookie de session via le client. À copier côté front avec navigator.clipboard.
export function fetchTitleTomlDraft(slug: string): Promise<string> {
  return api.getText(`/admin/titles/${slug}/toml-draft`)
}
