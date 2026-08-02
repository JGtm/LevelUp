/**
 * Field mappings (Phase D du plan multi-titres).
 *
 * Charge depuis le backend les libellés/format/group de chaque FieldKey
 * canonique et les expose via TanStack Query + un hook useFieldLabel.
 *
 * - L'endpoint backend est /api/v1/titles/{slug}/field-mappings?locale=...
 *   Il est monté SANS CONDITION depuis le 2026-08-02 (v7.3 lot 2, item 3.3 : le
 *   flag de rollout MULTI_TITLE_API_ENABLED a été supprimé). Ces libellés sont
 *   désormais la source unique — il n'existe plus de dictionnaire de repli côté
 *   TS. Si la réponse manque (réseau, titre sans la clé), le hook rend la clé
 *   elle-même ; useMetricLabel l'humanise alors (lib/i18n/metricLabel.ts).
 *
 * - Une seule requête par couple (slug, locale) au boot, cache infini :
 *   la couche sémantique est versionnée Git, pas de hot-reload prod.
 *
 * - Voir AUDIT_I18N_REACT_2026-04-25.md pour la stratégie de migration des
 *   strings hardcodées.
 */

import { useMemo } from 'react'
import { useQuery, type UseQueryOptions } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import type { Locale } from '@/lib/i18n/locale'

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

/** DTO d'un asset (mode, tier, cadence, statut, saison, …). Plan finition multi-titres §3.1.
 *
 * Champs `start_date` / `end_date` / `extra` ne sont peuplés que pour les kinds
 * time-bounded (saisons, opérations, événements). Pour les autres kinds ils
 * restent absents grâce à omitempty côté backend.
 */
export interface AssetMappingDTO {
  label: string
  color_token?: string
  icon?: string
  display_order: number
  start_date?: string // ISO 8601 (RFC 3339)
  end_date?: string // ISO 8601 (RFC 3339), absent = saison ouverte
  extra?: Record<string, string>
}

/** DTO d'un outcome (win/loss/tie/dnf). */
export interface OutcomeMappingDTO {
  label: string
  color_token: string
}

/** Réponse de l'endpoint /field-mappings. */
export interface FieldMappingsResponse {
  title_slug: string
  schema_version: number
  locale: string
  fields: Record<string, FieldMappingDTO>
  /** Indexé par kind (mode, challenge_tier, etc.) puis par id. Optionnel. */
  assets?: Record<string, Record<string, AssetMappingDTO>>
  /** Indexé par outcome key (win/loss/tie/dnf). Optionnel. */
  outcomes?: Record<string, OutcomeMappingDTO>
}

/**
 * Clé TanStack Query.
 * staleTime infini car la couche sémantique ne change pas en prod sans
 * redéploiement (cf. PLAN §7.3).
 */
export function fieldMappingsQueryKey(slug: string, locale: Locale) {
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
  locale: Locale,
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
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const callerEnabled = options?.enabled ?? true

  // Gate `isBootstrapped` (G8) : avant l'hydratation du bootstrap, currentTitleSlug
  // et locale valent leurs défauts store ('halo_infinite' / 'fr' — appShellStore.ts),
  // pas la session réelle. __root.tsx rend l'Outlet NU pendant cette fenêtre (une
  // passe de rendu avant que hydrateFromBootstrap commit isBootstrapped=true), donc
  // ce hook peut monter avec des valeurs encore par défaut puis, une fois hydraté,
  // avec les vraies valeurs — deux clés distinctes si locale/titre diffèrent des
  // défauts (ex. session/démo en 'en') = 2 requêtes pour la même donnée. On attend
  // la locale/le titre résolus avant d'activer la query (même patron que
  // isBootstrapped ailleurs : players/$.tsx, LoginPage, SetupPage).
  const query = useQuery<FieldMappingsResponse>({
    queryKey: fieldMappingsQueryKey(slug, locale),
    queryFn: () => fetchFieldMappings(slug, locale),
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
    ...options,
    enabled: isBootstrapped && callerEnabled,
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

/**
 * Hook compact pour le libellé d'un asset (`mode`, `challenge_tier`, etc.).
 *
 * Comportement :
 *   - mappings non chargés ou kind absent → retourne `id`
 *   - id absent du kind → retourne `id`
 *   - sinon → retourne le label localisé
 */
export function useAssetLabel(kind: string, id: string): string {
  const { data } = useFieldMappings()
  if (!data?.assets) return id
  return data.assets[kind]?.[id]?.label ?? id
}

/**
 * Hook pour récupérer le DTO complet d'un asset (label + color_token + icon).
 * Utile pour appliquer un color_token via tokenCssVar dans les composants
 * qui rendent un badge/tier coloré.
 */
export function useAssetMapping(kind: string, id: string): AssetMappingDTO | undefined {
  const { data } = useFieldMappings()
  return data?.assets?.[kind]?.[id]
}

/**
 * Hook compact pour le libellé d'un outcome (`win`, `loss`, `tie`, `dnf`).
 *
 * Comportement :
 *   - mappings non chargés ou outcomes absents → retourne `key`
 *   - key absente → retourne `key`
 *   - sinon → retourne le label localisé
 */
export function useOutcomeLabel(key: string): string {
  const { data } = useFieldMappings()
  if (!data?.outcomes) return key
  return data.outcomes[key]?.label ?? key
}

/**
 * Hook pour récupérer le DTO complet d'un outcome (label + color_token).
 */
export function useOutcomeMapping(key: string): OutcomeMappingDTO | undefined {
  const { data } = useFieldMappings()
  return data?.outcomes?.[key]
}

// ─── Saisons : sélecteur dérivé sur le kind "season" du field-mappings ─────

/**
 * Saison projetée depuis le kind "season" du field-mappings.
 *
 * Les champs typés (startDate / endDate / shortLabel / csrSeasonId) sont
 * dérivés des chaînes ISO et de extra côté backend.
 */
export interface SeasonEntry {
  id: string
  label: string
  shortLabel: string
  startDate: Date
  endDate: Date | null // null = saison ouverte (en cours)
  displayOrder: number
  csrSeasonId?: string
}

function toSeasonEntry(id: string, dto: AssetMappingDTO): SeasonEntry | null {
  if (!dto.start_date) return null
  const start = new Date(dto.start_date)
  if (Number.isNaN(start.getTime())) return null
  let end: Date | null = null
  if (dto.end_date) {
    const e = new Date(dto.end_date)
    if (!Number.isNaN(e.getTime())) end = e
  }
  return {
    id,
    label: dto.label,
    shortLabel: dto.extra?.short_label ?? id,
    startDate: start,
    endDate: end,
    displayOrder: dto.display_order,
    csrSeasonId: dto.extra?.csr_season_id,
  }
}

/**
 * Comparateur d'affichage des saisons : la plus RÉCENTE d'abord (DESC).
 *
 * Clé de tri = `startDate` (récence réelle), départage par `displayOrder` DESC.
 * Keyer sur la date réelle — et NON sur `displayOrder` — place correctement les
 * saisons « DB-only » (connues de Waypoint mais pas encore du TOML) : le backend
 * leur attribue un `displayOrder` synthétique élevé (`mergeSeasonSources`,
 * seasons_catalog.go) qui, trié DESC brut, les ferait sauter en tête à tort ;
 * leur `startDate` réelle les remet à leur juste place chronologique. Ordre
 * « récent-en-haut » cohérent avec les autres sélecteurs de saison (Carrière CSR
 * `sortCSRSeasonsDesc`). Réf. GH5-1.
 */
export function compareSeasonsRecentFirst(a: SeasonEntry, b: SeasonEntry): number {
  const byDate = b.startDate.getTime() - a.startDate.getTime()
  if (byDate !== 0) return byDate
  return b.displayOrder - a.displayOrder
}

/**
 * Hook qui retourne la liste des saisons du titre courant, triées la plus
 * RÉCENTE d'abord (startDate DESC, cf. compareSeasonsRecentFirst). Sélecteur
 * dérivé sur useFieldMappings — pas de queryKey additionnelle, le cache est
 * partagé. Consommé par la SaisonPill (Omnibar), le sélecteur Explorer et la
 * dropdown Carrière ; les consommateurs qui ont besoin de l'ordre chronologique
 * (PeriodSessionRail prev/next) le recalculent en interne (findSeasonAt).
 *
 * Comportement :
 *   - Mappings non chargés ou kind "season" absent → tableau vide
 *   - Entrées sans start_date valide → ignorées (filtrage défensif)
 */
export function useSeasons(): SeasonEntry[] {
  const { data } = useFieldMappings()
  return useMemo(() => {
    const raw = data?.assets?.season ?? {}
    const entries: SeasonEntry[] = []
    for (const [id, dto] of Object.entries(raw)) {
      const e = toSeasonEntry(id, dto)
      if (e) entries.push(e)
    }
    entries.sort(compareSeasonsRecentFirst)
    return entries
  }, [data])
}
