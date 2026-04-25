/**
 * MediaPage — Galerie de médias (Slice 8).
 * Types ref: MediaItemRow, MediaQueryRequest, MediaPageResponse
 */
import { useState, useEffect, useRef } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import type { LabelValue, MediaItemRow, MediaQueryRequest } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { MediaLightbox, MediaThumbnailCard } from './MediaViewer'
import { MediaToolbar } from './MediaToolbar'
import { UploadButton } from './UploadButton'
import { getMediaText } from './i18n'
import { useMediaAuthors, useMediaPage, useToggleMediaLike, useFeedVersion } from './queries'
import { useQueryClient } from '@tanstack/react-query'
import { queryKeys } from '@/lib/query/keys'

const PAGE_SIZE = 24

function buildFallbackOptions(items: MediaItemRow[], key: 'map_name' | 'mode_name'): LabelValue[] {
  const labels = Array.from(new Set(
    items
      .map((item) => item[key]?.trim())
      .filter((value): value is string => Boolean(value)),
  )).sort((left, right) => left.localeCompare(right))

  return labels.map((label) => ({ label, value: label }))
}

function extractErrorMessage(error: unknown): string {
  if (!error) return ''
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  if (typeof error === 'object') {
    const maybe = error as { message?: unknown; error?: unknown; detail?: unknown }
    if (typeof maybe.message === 'string') return maybe.message
    if (typeof maybe.error === 'string') return maybe.error
    if (typeof maybe.detail === 'string') return maybe.detail
    try {
      return JSON.stringify(error)
    } catch {
      return String(error)
    }
  }
  return String(error)
}

export function MediaPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/media' })
  const locale = useAppShellStore((state) => state.locale)
  const text = getMediaText(locale)
  const [page, setPage] = useState(1)
  const [kindFilter, setKindFilter] = useState('')
  const [authorSlugs, setAuthorSlugs] = useState<string[]>([])
  const [mapFilter, setMapFilter] = useState('')
  const [modeFilter, setModeFilter] = useState('')
  const [groupBy, setGroupBy] = useState('')
  const [sortKey, setSortKey] = useState('date_desc')
  const [likedOnly, setLikedOnly] = useState(false)
  const [lightboxIdx, setLightboxIdx] = useState<number | null>(null)

  const request: MediaQueryRequest = {
    sort: sortKey,
    kind_filter: kindFilter || null,
    author_slugs: authorSlugs.length > 0 ? authorSlugs : null,
    map_filter: mapFilter || null,
    mode_filter: modeFilter || null,
    group_by: groupBy || null,
    liked_only: likedOnly || null,
    pagination: { page, page_size: PAGE_SIZE },
  }

  const { data, isLoading, isError, error, isFetching } = useMediaPage(playerSlug, request, page)
  const { data: authorsData } = useMediaAuthors(playerSlug)
  const authors = authorsData?.authors ?? []
  const toggleMediaLike = useToggleMediaLike(playerSlug)
  const mediaItems: MediaItemRow[] = data?.items?.items ?? []
  const mapOptions = data?.available_filters.maps?.length
    ? data.available_filters.maps
    : buildFallbackOptions(mediaItems, 'map_name')
  const modeOptions = data?.available_filters.modes?.length
    ? data.available_filters.modes
    : buildFallbackOptions(mediaItems, 'mode_name')

  // Polling feed-version : invalide le cache si un autre joueur a uploadé/liké
  const queryClient = useQueryClient()
  const { data: feedVersion } = useFeedVersion(playerSlug)
  const lastFeedVersion = useRef<number | undefined>(undefined)
  useEffect(() => {
    if (feedVersion !== undefined && lastFeedVersion.current !== undefined && feedVersion !== lastFeedVersion.current) {
      queryClient.invalidateQueries({ queryKey: queryKeys.mediaBase(playerSlug) })
    }
    lastFeedVersion.current = feedVersion
  }, [feedVersion, playerSlug, queryClient])
  const pagination = data?.items?.pagination
  const totalPages = pagination ? Math.ceil(pagination.total / PAGE_SIZE) : 1

  function handleKindChange(value: string) {
    setKindFilter(value)
    setPage(1)
  }

  function handleAuthorSlugsChange(slugs: string[]) {
    setAuthorSlugs(slugs)
    setPage(1)
  }

  function handleMapChange(value: string) {
    setMapFilter(value)
    setPage(1)
  }

  function handleModeChange(value: string) {
    setModeFilter(value)
    setPage(1)
  }

  function handleSortChange(value: string) {
    setSortKey(value)
    setPage(1)
  }

  function handleGroupByChange(value: string) {
    setGroupBy(value)
    setPage(1)
  }

  function handleLikedOnlyChange(value: boolean) {
    setLikedOnly(value)
    setPage(1)
  }

  return (
    <div className="flex flex-col gap-6">
      {lightboxIdx !== null && (
        <MediaLightbox
          items={mediaItems}
          onToggleLike={(item) => toggleMediaLike.mutate({ file_path: item.file_path, liked: !item.liked })}
          startIndex={lightboxIdx}
          onClose={() => setLightboxIdx(null)}
          likeDisabled={toggleMediaLike.isPending}
          hasNextPage={page < totalPages}
          onLoadNextPage={() => setPage((current) => current + 1)}
        />
      )}

      {/* Zone d'upload pleine largeur (vient en premier pour rapprocher les filtres de la grille) */}
      <UploadButton playerSlug={playerSlug} fullWidth />

      <div className="px-6">
        <MediaToolbar
          text={text}
          kindFilter={kindFilter}
          authorSlugs={authorSlugs}
          authors={authors}
          mapFilter={mapFilter}
          modeFilter={modeFilter}
          groupBy={groupBy}
          sortKey={sortKey}
          likedOnly={likedOnly}
          mapOptions={mapOptions}
          modeOptions={modeOptions}
          onKindChange={handleKindChange}
          onAuthorSlugsChange={handleAuthorSlugsChange}
          onMapChange={handleMapChange}
          onModeChange={handleModeChange}
          onSortChange={handleSortChange}
          onGroupByChange={handleGroupByChange}
          onLikedOnlyChange={handleLikedOnlyChange}
        />
      </div>

      {isLoading ? null : isError ? (
        <div className="p-8 text-center text-destructive">{text.errorPrefix} {extractErrorMessage(error)}</div>
      ) : mediaItems.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center text-muted-foreground">
            {text.emptyState}
          </CardContent>
        </Card>
      ) : (
        <>
          <div className={`grid grid-cols-2 gap-4 transition-opacity sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 ${isFetching ? 'opacity-70' : 'opacity-100'}`}>
            {mediaItems.map((item, index) => (
              <MediaThumbnailCard
                key={`${item.file_path}-${index}`}
                item={item}
                onToggleLike={(currentItem) => {
                  toggleMediaLike.mutate({
                    file_path: currentItem.file_path,
                    liked: !currentItem.liked,
                  })
                }}
                onOpen={() => setLightboxIdx(index)}
                likeDisabled={toggleMediaLike.isPending}
              />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="mt-2 flex items-center justify-center gap-2">
              <button
                className="rounded border px-3 py-1 text-sm disabled:opacity-40"
                disabled={page === 1}
                onClick={() => setPage((current) => current - 1)}
              >
                {text.previousPage}
              </button>
              <span className="text-sm text-muted-foreground">{text.pageLabel(page, totalPages)}</span>
              <button
                className="rounded border px-3 py-1 text-sm disabled:opacity-40"
                disabled={page >= totalPages}
                onClick={() => setPage((current) => current + 1)}
              >
                {text.nextPage}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
