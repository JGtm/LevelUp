/**
 * Field mappings (Phase D du plan multi-titres).
 *
 * Charge depuis le backend les libellés/format/group de chaque FieldKey
 * canonique et les expose via TanStack Query + un hook useFieldLabel.
 *
 * - L'endpoint backend est /api/v1/titles/{slug}/field-mappings?locale=...
 *   et n'est exposé que si MULTI_TITLE_API_ENABLED=true côté serveur.
 *   Quand le flag est off, le hook tombe en fallback gracieusement (key as label).
 *
 * - Une seule requête par couple (slug, locale) au boot, cache infini :
 *   la couche sémantique est versionnée Git, pas de hot-reload prod.
 *
 * - Voir AUDIT_I18N_REACT_2026-04-25.md pour la stratégie de migration des
 *   strings hardcodées.
 */

import { useQuery, type UseQueryOptions } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

/** Locale supportée — alignée avec appShellStore. */
export type FieldMappingLocale = 'fr' | 'en'

/** Forme d'un FieldMapping retourné par l'endpoint backend. */
export interface FieldMappingDTO {
  label: string
  description?: string
  storage_unit: string
  display_unit: string
  format: string
  display_order: number
  group: string
  icon?: string
}

/** Réponse de l'endpoint /field-mappings. */
export interface FieldMappingsResponse {
  title_slug: string
  schema_version: number
  locale: string
  fields: Record<string, FieldMappingDTO>
}

/**
 * Clé TanStack Query.
 * staleTime infini car la couche sémantique ne change pas en prod sans
 * redéploiement (cf. PLAN §7.3).
 */
export function fieldMappingsQueryKey(slug: string, locale: FieldMappingLocale) {
  return ['field-mappings', slug, locale] as const
}

/**
 * Fetch direct (utile pour les tests + le prefetch).
 *
 * Si le backend retourne 404 (titre absent du registry, ou flag off →
 * route absente du routeur), on retourne un payload vide. Le caller fera
 * fallback sur la key.
 */
export async function fetchFieldMappings(
  slug: string,
  locale: FieldMappingLocale,
): Promise<FieldMappingsResponse> {
  try {
    return await api.get<FieldMappingsResponse>(
      `/titles/${encodeURIComponent(slug)}/field-mappings?locale=${encodeURIComponent(locale)}`,
    )
  } catch (err) {
    const status = (err as { status?: number })?.status
    if (status === 404) {
      return { title_slug: slug, schema_version: 0, locale, fields: {} }
    }
    throw err
  }
}

/**
 * Hook React qui retourne le FieldMappingsResponse pour le titre + locale courants.
 *
 * Usage :
 *   const mappings = useFieldMappings()
 *   const label = mappings.fields['kills']?.label ?? 'Kills'
 */
export function useFieldMappings(
  options?: Omit<UseQueryOptions<FieldMappingsResponse>, 'queryKey' | 'queryFn'>,
) {
  const slug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)

  const query = useQuery<FieldMappingsResponse>({
    queryKey: fieldMappingsQueryKey(slug, locale),
    queryFn: () => fetchFieldMappings(slug, locale),
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
    ...options,
  })

  return query
}

/**
 * Hook compact pour récupérer le libellé d'un FieldKey.
 *
 * Comportement :
 *   - mappings non chargés → retourne la key (string)
 *   - key absente du mapping → retourne la key (fallback)
 *   - sinon → retourne le label localisé
 *
 * Pour des cas avancés (description, format, icon), utiliser useFieldMappings()
 * et accéder directement à mappings.fields[key].
 */
export function useFieldLabel(key: string): string {
  const { data } = useFieldMappings()
  if (!data) return key
  return data.fields[key]?.label ?? key
}

/**
 * Hook pour récupérer le DTO complet (label + format + group + icon).
 * Retourne undefined si les mappings ne sont pas (encore) chargés ou si la
 * key est absente.
 */
export function useFieldMapping(key: string): FieldMappingDTO | undefined {
  const { data } = useFieldMappings()
  return data?.fields[key]
}
