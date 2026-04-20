/**
 * Queries TanStack Query — Médias (Slice 8).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MediaItemRow,
  MediaLikeRequest,
  MediaLikeResponse,
  MediaQueryRequest,
  MediaPageResponse,
  MediaUploadResponse,
} from '@/lib/api/types'

const DEFAULT_MEDIA_SORT = 'date_desc'

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

function normalizeMediaKind(kind?: string | null) {
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
    liked,
    like_count: item.like_count ?? (liked ? 1 : 0),
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
    }
  }

  return {
    total_mine: response.total_mine,
    total_teammates: response.total_teammates,
    total_unassigned: response.total_unassigned,
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
    map: request.map_filter,
    mod: request.mode_filter,
    g: request.group_by,
    lo: request.liked_only,
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
  })
}

export function useRecentMediaRail(playerSlug: string, limit: number) {
  const request: MediaQueryRequest = {
    sort: DEFAULT_MEDIA_SORT,
    pagination: { page: 1, page_size: limit },
  }

  return useQuery({
    queryKey: queryKeys.mediaRail(playerSlug, limit),
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

      queryClient.setQueriesData<MediaPageResponse>(
        { queryKey: queryKeys.mediaBase(playerSlug) },
        (current) => updateMediaLikeInResponse(
          current,
          request.file_path,
          request.liked,
          request.liked ? 1 : 0,
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
      queryClient.invalidateQueries({
        queryKey: queryKeys.mediaBase(playerSlug),
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
      files.forEach((f) => form.append('files', f, f.name))
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
  const queryClient = useQueryClient()
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
