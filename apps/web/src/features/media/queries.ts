/**
 * Queries TanStack Query — Médias (Slice 8).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MediaAssociateRequest,
  MediaAssociateResponse,
  MediaAuthorsResponse,
  MediaAvailableFilters,
  MediaItemRow,
  MediaLikeRequest,
  MediaLikeResponse,
  MediaMatchCandidatesResponse,
  MediaQueryRequest,
  MediaPageResponse,
  MediaUploadResponse,
} from '@/lib/api/types'

const DEFAULT_MEDIA_SORT = 'date_desc'

const EMPTY_MEDIA_FILTERS: MediaAvailableFilters = {
  playlists: [],
  maps: [],
  modes: [],
}

interface LegacyMediaItemRow {
  basename?: string | null
  file_name?: string | null
  file_path: string
  kind?: string | null
  thumbnail_path?: string | null
  match_id?: string | null
  capture_end_utc?: string | null
  match_start_time?: string | null
  section?: string | null
  owner_gamertag?: string | null
  map_name?: string | null
  mode_name?: string | null
  liked?: boolean | null
  like_count?: number | null
}

interface LegacyMediaPageResponse {
  items: LegacyMediaItemRow[]
  total_count: number
  page: number
  page_size: number
  has_more: boolean
}

type MediaPageApiResponse = MediaPageResponse | LegacyMediaPageResponse

function isLegacyMediaItem(item: LegacyMediaItemRow | MediaItemRow): item is LegacyMediaItemRow {
  return 'file_name' in item
}

function isLegacyMediaPageResponse(response: MediaPageApiResponse): response is LegacyMediaPageResponse {
  return Array.isArray(response.items)
}

export function normalizeMediaKind(kind?: string | null) {
  if (kind === 'video') {
    return 'clip'
  }
  if (kind === 'image') {
    return 'screenshot'
  }
  return kind ?? 'screenshot'
}

function basenameFromPath(filePath: string) {
  const parts = filePath.split('/')
  return parts[parts.length - 1] ?? filePath
}

function normalizeMediaItem(item: LegacyMediaItemRow | MediaItemRow): MediaItemRow {
  const liked = item.liked ?? false
  return {
    basename: item.basename ?? (isLegacyMediaItem(item) ? item.file_name : null) ?? basenameFromPath(item.file_path),
    file_path: item.file_path,
    kind: normalizeMediaKind(item.kind),
    thumbnail_path: item.thumbnail_path ?? null,
    match_id: item.match_id ?? null,
    capture_end_utc: item.capture_end_utc ?? null,
    match_start_time: item.match_start_time ?? null,
    section: item.section ?? 'mine',
    owner_gamertag: item.owner_gamertag ?? null,
    map_name: item.map_name ?? null,
    mode_name: item.mode_name ?? null,
    liked,
    like_count: item.like_count ?? (liked ? 1 : 0),
    likers: 'likers' in item ? (item as MediaItemRow).likers : undefined,
    total_likers: 'total_likers' in item ? (item as MediaItemRow).total_likers : undefined,
  }
}

function normalizeMediaPageResponse(response: MediaPageApiResponse): MediaPageResponse {
  if (isLegacyMediaPageResponse(response)) {
    const items = response.items.map(normalizeMediaItem)
    return {
      items: {
        items,
        pagination: {
          total: response.total_count,
          page: response.page,
          page_size: response.page_size,
          has_next: response.has_more,
          has_prev: response.page > 1,
        },
        freshness: null,
      },
      total_mine: response.total_count,
      total_teammates: 0,
      total_unassigned: 0,
      available_filters: EMPTY_MEDIA_FILTERS,
    }
  }

  return {
    total_mine: response.total_mine,
    total_teammates: response.total_teammates,
    total_unassigned: response.total_unassigned,
    available_filters: {
      playlists: response.available_filters?.playlists ?? [],
      maps: response.available_filters?.maps ?? [],
      modes: response.available_filters?.modes ?? [],
    },
    items: {
      ...response.items,
      freshness: response.items.freshness ?? null,
      items: response.items.items.map(normalizeMediaItem),
    },
  }
}

function updateMediaLikeInResponse(
  response: MediaPageResponse | undefined,
  filePath: string,
  liked: boolean,
  likeCount: number,
  likers?: string[],
  totalLikers?: number,
) {
  if (!response) {
    return response
  }

  // Garde-fou : queryKey ['media', slug] matche aussi mediaAuthors (forme {authors})
  // et autres queries qui n'ont PAS la structure { items: { items: [] } }.
  // Sans ce check, .items.items.map throw et casse la mutation entière.
  if (!response.items || !Array.isArray((response.items as { items?: unknown }).items)) {
    return response
  }

  return {
    ...response,
    items: {
      ...response.items,
      items: response.items.items.map((item) =>
        item.file_path === filePath
          ? {
              ...item,
              liked,
              like_count: likeCount,
              ...(likers !== undefined && { likers }),
              ...(totalLikers !== undefined && { total_likers: totalLikers }),
            }
          : item,
      ),
    },
  }
}

export function useMediaPage(
  playerSlug: string,
  request: MediaQueryRequest,
  page: number,
) {
  // La clé inclut tous les filtres actifs pour que chaque combinaison soit indépendante.
  const requestHash = JSON.stringify({
    p: page,
    s: request.sort,
    k: request.kind_filter,
    sec: request.section_filter,
    auth: request.author_slugs,
    map: request.map_filter,
    mod: request.mode_filter,
    g: request.group_by,
    lo: request.liked_only,
    ua: request.unassigned_only,
  })

  return useQuery({
    queryKey: queryKeys.media(playerSlug, requestHash),
    queryFn: () =>
      api.post<MediaPageApiResponse>(
        `/players/${playerSlug}/pages/media`,
        request,
      ).then(normalizeMediaPageResponse),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
    placeholderData: (prev) => prev,
  })
}

export function useRecentMediaRail(playerSlug: string, limit: number, likedOnly = false) {
  const request: MediaQueryRequest = {
    sort: DEFAULT_MEDIA_SORT,
    liked_only: likedOnly || null,
    pagination: { page: 1, page_size: limit },
  }

  return useQuery({
    queryKey: queryKeys.mediaRail(playerSlug, limit, likedOnly),
    queryFn: () =>
      api.post<MediaPageApiResponse>(
        `/players/${playerSlug}/pages/media`,
        request,
      ).then(normalizeMediaPageResponse),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
  })
}

export function useToggleMediaLike(playerSlug: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: MediaLikeRequest) =>
      api.patch<MediaLikeResponse>(`/players/${playerSlug}/media/likes`, request),
    onMutate: async (request) => {
      const previous = queryClient.getQueriesData<MediaPageResponse>({
        queryKey: queryKeys.mediaBase(playerSlug),
      })

      // Calculer le bon likeCount depuis l'item actuel (incrémenter/décrémenter)
      const allData = queryClient.getQueriesData<MediaPageResponse>({
        queryKey: queryKeys.mediaBase(playerSlug),
      })
      let currentLikeCount = 0
      for (const [, data] of allData) {
        const item = data?.items?.items?.find((i) => i.file_path === request.file_path)
        if (item) {
          currentLikeCount = item.like_count
          break
        }
      }
      const newLikeCount = request.liked ? currentLikeCount + 1 : Math.max(0, currentLikeCount - 1)

      queryClient.setQueriesData<MediaPageResponse>(
        { queryKey: queryKeys.mediaBase(playerSlug) },
        (current) => updateMediaLikeInResponse(
          current,
          request.file_path,
          request.liked,
          newLikeCount,
        ),
      )

      return { previous }
    },
    onError: (_error, _request, context) => {
      context?.previous.forEach(([queryKey, data]) => {
        queryClient.setQueryData(queryKey, data)
      })
    },
    onSuccess: (response) => {
      queryClient.setQueriesData<MediaPageResponse>(
        { queryKey: queryKeys.mediaBase(playerSlug) },
        (current) => updateMediaLikeInResponse(
          current,
          response.file_path,
          response.liked,
          response.like_count,
          response.likers,
          response.total_likers,
        ),
      )
    },
    onSettled: () => {
      // refetchType: 'none' = marque stale SANS refetch immédiat.
      // Le cover-flow ouvert ne voit pas son array items réordonné (qui ferait
      // changer la vidéo affichée). Les autres vues (filtre liked_only, retour
      // sur la galerie) refetcheront quand elles redeviendront actives.
      queryClient.invalidateQueries({
        queryKey: queryKeys.mediaBase(playerSlug),
        refetchType: 'none',
      })
    },
  })
}

/** Mutation upload : envoie des fichiers et invalide toutes les pages médias. */
export function useUploadMedia(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (files: File[]) => {
      const form = new FormData()
      // Envoyer file.lastModified (ms → s Unix) pour chaque fichier pour
      // permettre au backend de corriger le mtime serveur (priorité 2).
      const captureTimes = files.map((f) => Math.floor(f.lastModified / 1000))
      files.forEach((f) => form.append('files', f, f.name))
      form.append('capture_times', JSON.stringify(captureTimes))
      return api.postForm<MediaUploadResponse>(
        `/players/${playerSlug}/media/upload`,
        form,
      )
    },
    onSuccess: () => {
      // Invalide toutes les pages media du joueur (préfixe ['media', slug])
      queryClient.invalidateQueries({
        queryKey: queryKeys.mediaBase(playerSlug),
      })
    },
  })
}

/**
 * Polling léger : récupère la version du flux médias toutes les 10 s.
 * Quand la version change, invalide le cache médias du joueur.
 */
export function useFeedVersion(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.feedVersion,
    queryFn: () => api.get<{ version: number }>('/media/feed-version'),
    refetchInterval: 10_000,
    enabled: !!playerSlug,
    select: (data) => data.version,
    notifyOnChangeProps: ['data'],
    staleTime: 0,
  })
}

/** Candidats de matchs pour réassocier un média (fenêtre ±windowMinutes). */
export function useMediaMatchCandidates(
  playerSlug: string,
  filePath: string | null,
  windowMinutes: number,
  enabled = true,
) {
  return useQuery({
    queryKey: queryKeys.mediaMatchCandidates(playerSlug, filePath, windowMinutes),
    queryFn: () =>
      api.get<MediaMatchCandidatesResponse>(
        `/players/${playerSlug}/media/match-candidates?file_path=${encodeURIComponent(filePath ?? '')}&window_minutes=${windowMinutes}`,
      ),
    enabled: enabled && !!playerSlug && !!filePath,
    staleTime: 60_000,
  })
}

/** Mutation pour forcer l'association d'un média à un match précis. */
export function useAssociateMediaToMatch(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: MediaAssociateRequest) =>
      api.post<MediaAssociateResponse>(`/players/${playerSlug}/media/associate`, req),
    onSuccess: () => {
      // Invalide tout le cache média : la map/mode du média a changé
      queryClient.invalidateQueries({ queryKey: queryKeys.mediaBase(playerSlug) })
    },
  })
}

/** Liste des auteurs (db_profiles ∩ ≥1 média sur disque) pour le filtre Auteur. */
export function useMediaAuthors(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.mediaAuthors(playerSlug),
    queryFn: () => api.get<MediaAuthorsResponse>(`/players/${playerSlug}/media/authors`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

// Effet de bord : invalide le cache médias quand la version change.
export function useInvalidateOnFeedVersion(playerSlug: string) {
  const queryClient = useQueryClient()
  const { data: version } = useFeedVersion(playerSlug)
  // Non réactif ici, à utiliser avec useEffect dans un composant parent.
  return (lastVersion: number | undefined) => {
    if (version !== undefined && lastVersion !== undefined && version !== lastVersion) {
      queryClient.invalidateQueries({ queryKey: queryKeys.mediaBase(playerSlug) })
    }
    return version
  }
}
