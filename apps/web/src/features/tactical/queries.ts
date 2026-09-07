/**
 * queries.ts — TanStack Query hooks pour la feature Tactique.
 */
import { useQuery } from '@tanstack/react-query'
import { queryKeys } from '@/lib/query/keys'

/**
 * Réponse du serveur pour GET /players/{slug}/tactical/{mapId}/raster
 */
export interface TacticalRasterResponse {
  cells: Array<{ col: number; row: number; valeur: number }>
  gridSize: { cell: number; nx: number; ny: number; minX: number; minY: number }
  scale: { lo: number; hi: number }
  filled: number
  matchs_filtres: number
  matchs_retenus: number
  matchs_en_attente: number
  matchs_non_cuisables: number
  echange: number // taux d'échange pour cette carte
  grappes: Array<{
    id: string
    nom_fr: string
    nom_en: string
    x: number
    y: number
    matchs: number
  }>
  // Pour isole : isolation metrics
  isolation?: {
    [playerXuid: string]: { taux: number; count: number }
  }
}

/**
 * Réponse du serveur pour GET /players/{slug}/tactical/maps
 */
export interface TacticalMapsPlayedResponse {
  maps: Array<{
    id: string
    name_fr: string
    name_en: string
    matches_total: number
    matches_retained: number
    wins: number
    outcome_distribution: {
      wins: number
      losses: number
    }
  }>
}

/**
 * useTacticalMapsPlayed — charge la liste des cartes jouées.
 */
export function useTacticalMapsPlayed(
  playerSlug: string,
  titleSlug: string,
  filterHash: string,
  enabled: boolean = true,
) {
  return useQuery<TacticalMapsPlayedResponse>({
    queryKey: queryKeys.tacticalMapsPlayed(playerSlug, titleSlug, filterHash),
    queryFn: async () => {
      const resp = await fetch(
        `/api/players/${playerSlug}/tactical/maps?filters=${encodeURIComponent(filterHash)}`,
        { headers: { 'X-LevelUp-Title': titleSlug } },
      )
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
      return resp.json()
    },
    enabled,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}

/**
 * useTacticalRaster — charge le raster pour une carte, question, et configuration.
 */
export function useTacticalRaster(
  playerSlug: string,
  titleSlug: string,
  mapId: string,
  filterHash: string,
  question: string,
  who: string,
  spawn: string,
  enabled: boolean = true,
) {
  return useQuery<TacticalRasterResponse>({
    queryKey: queryKeys.tacticalRaster(playerSlug, titleSlug, mapId, filterHash, question, who, spawn),
    queryFn: async () => {
      const params = new URLSearchParams({
        filters: filterHash,
        question,
        who,
        spawn,
      })
      const resp = await fetch(
        `/api/players/${playerSlug}/tactical/${mapId}/raster?${params}`,
        { headers: { 'X-LevelUp-Title': titleSlug } },
      )
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
      return resp.json()
    },
    enabled,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}

/**
 * useTacticalBackgroundImage — charge l'image de fond pour une carte.
 */
export function useTacticalBackgroundImage(
  playerSlug: string,
  titleSlug: string,
  mapId: string,
  enabled: boolean = true,
) {
  return useQuery<Blob>({
    queryKey: ['tactical', playerSlug, titleSlug, 'background', mapId] as const,
    queryFn: async () => {
      const resp = await fetch(
        `/api/tactical/${mapId}/background.png`,
        { headers: { 'X-LevelUp-Title': titleSlug } },
      )
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
      return resp.blob()
    },
    enabled,
    staleTime: 24 * 60 * 60 * 1000, // 24 hours (static asset)
    gcTime: 48 * 60 * 60 * 1000, // 48 hours
  })
}
