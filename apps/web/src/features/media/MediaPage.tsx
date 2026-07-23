/**
 * MediaPage — Galerie de médias (Slice 8).
 * Types ref: MediaItemRow, MediaQueryRequest, MediaPageResponse
 */
import { useCallback, useState, useEffect, useMemo, useRef } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import type { LabelValue, MediaItemRow, MediaQueryRequest } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'
import { intlLocale } from '@/lib/formatters'
import { MediaLightbox, MediaThumbnailCard } from './MediaViewer'
import { buildOwnerColorMap } from './mediaOwnerColors'
import { MediaToolbar } from './MediaToolbar'
import { MediaMatchPicker } from './MediaMatchPicker'
import { useMediaPicker } from './useMediaPicker'
import { UploadButton } from './UploadButton'
import { getMediaText, type MediaText } from './i18n'
import { useMediaAuthors, useMediaPage, useToggleMediaLike, useFeedVersion } from './queries'
import { useQueryClient } from '@tanstack/react-query'
import { queryKeys } from '@/lib/query/keys'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'

const PAGE_SIZE = 24

// Une session = run de matchs séparés par ≤ 30 minutes (heuristique alignée
// sur la définition session_id de stats.duckdb).
const SESSION_GAP_MS = 30 * 60 * 1000

interface MediaGroup {
  key: string
  label: string
  items: MediaItemRow[]
}

function buildFallbackOptions(items: MediaItemRow[], key: 'map_name' | 'mode_name'): LabelValue[] {
  const labels = Array.from(new Set(
    items
      .map((item) => item[key]?.trim())
      .filter((value): value is string => Boolean(value)),
  )).sort((left, right) => left.localeCompare(right))

  return labels.map((label) => ({ label, value: label, count: 0 }))
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

function itemTimestamp(item: MediaItemRow): number | null {
  const raw = item.capture_end_utc ?? item.match_start_time
  if (!raw) return null
  const t = new Date(raw).getTime()
  return Number.isNaN(t) ? null : t
}

function buildSessionGroups(items: MediaItemRow[], text: MediaText, locale: ManifestLocale): MediaGroup[] {
  if (items.length === 0) return []
  const formatter = new Intl.DateTimeFormat(intlLocale(locale), {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

  // Le backend trie déjà par DESC (groupBy=session = timeOrderExpr DESC).
  // On regroupe les items contigus dont l'écart de timestamp est ≤ SESSION_GAP_MS.
  const groups: MediaGroup[] = []
  let currentGroup: MediaGroup | null = null
  let lastTs: number | null = null

  for (const item of items) {
    const ts = itemTimestamp(item)
    if (ts === null) {
      // pas de timestamp → bucket "session inconnue", à part
      const orphan = groups.find((g) => g.key === 'session:__none__')
      if (orphan) {
        orphan.items.push(item)
      } else {
        groups.push({ key: 'session:__none__', label: text.groupSection.unknownSession, items: [item] })
      }
      continue
    }
    const tooFar = lastTs === null || Math.abs(lastTs - ts) > SESSION_GAP_MS
    if (!currentGroup || tooFar) {
      currentGroup = {
        key: `session:${ts}`,
        label: `${text.groupSection.sessionOfPrefix} ${formatter.format(new Date(ts))}`,
        items: [],
      }
      groups.push(currentGroup)
    }
    currentGroup.items.push(item)
    lastTs = ts
  }
  return groups
}

function buildSimpleGroups(items: MediaItemRow[], groupBy: string, text: MediaText): MediaGroup[] {
  if (items.length === 0) return []
  const order: string[] = []
  const map = new Map<string, MediaGroup>()
  for (const item of items) {
    let key: string
    let label: string
    switch (groupBy) {
      case 'owner': {
        const owner = item.owner_gamertag?.trim() || ''
        key = owner ? `owner:${owner}` : 'owner:__none__'
        label = owner || text.groupSection.unknownOwner
        break
      }
      case 'map': {
        const mapName = item.map_name?.trim() || ''
        key = mapName ? `map:${mapName}` : 'map:__none__'
        label = mapName || text.groupSection.unknownMap
        break
      }
      case 'mode': {
        const modeName = item.mode_name?.trim() || ''
        key = modeName ? `mode:${modeName}` : 'mode:__none__'
        label = modeName || text.groupSection.unknownMode
        break
      }
      default:
        return [{ key: '__all__', label: '', items }]
    }
    let group = map.get(key)
    if (!group) {
      group = { key, label, items: [] }
      map.set(key, group)
      order.push(key)
    }
    group.items.push(item)
  }
  return order.map((key) => map.get(key) as MediaGroup)
}

function buildGroups(items: MediaItemRow[], groupBy: string, text: MediaText, locale: ManifestLocale): MediaGroup[] {
  if (!groupBy || items.length === 0) {
    return [{ key: '__all__', label: '', items }]
  }
  if (groupBy === 'session') return buildSessionGroups(items, text, locale)
  return buildSimpleGroups(items, groupBy, text)
}

export function MediaPage() {
  const { playerSlug } = useParams({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/media' })
  const locale = useAppShellStore((state) => state.locale)
  const currentPlayerGamertag = useAppShellStore(
    (s) => s.currentPlayer?.gamertag ?? s.availablePlayers.find((p) => p.player_slug === playerSlug)?.gamertag,
  )
  const text = getMediaText(locale)
  const [page, setPage] = useState(1)
  const [kindFilter, setKindFilter] = useState('')
  // Défaut explicite = joueur courant sélectionné. [] = tous les auteurs (pas de filtre player_slug).
  const [authorSlugs, setAuthorSlugs] = useState<string[]>([playerSlug])
  const [playlistFilter, setPlaylistFilter] = useState('')
  const [mapFilter, setMapFilter] = useState('')
  const [modeFilter, setModeFilter] = useState('')
  const [groupBy, setGroupBy] = useState('')
  const [sortKey, setSortKey] = useState('date_desc')
  const [likedOnly, setLikedOnly] = useState(false)
  const [unassignedOnly, setUnassignedOnly] = useState(false)
  const [lightboxIdx, setLightboxIdx] = useState<number | null>(null)
  const picker = useMediaPicker()
  const [autoChain, setAutoChain] = useState(false)

  const request: MediaQueryRequest = {
    sort: sortKey,
    section_filter: null,
    kind_filter: kindFilter || null,
    author_slugs: authorSlugs.length > 0 ? authorSlugs : null,
    playlist_filter: playlistFilter || null,
    map_filter: mapFilter || null,
    mode_filter: modeFilter || null,
    group_by: groupBy || null,
    liked_only: likedOnly || null,
    unassigned_only: unassignedOnly || null,
    pagination: { page, page_size: PAGE_SIZE },
  }

  const { data, isLoading, isError, error, isFetching } = useMediaPage(playerSlug, request, page)
  const { data: authorsData } = useMediaAuthors(playerSlug)
  const authors = authorsData?.authors ?? []
  const toggleMediaLike = useToggleMediaLike(playerSlug)
  // Mémoïsé pour stabiliser les useMemo/useCallback consommateurs (indexByPath,
  // groups, handleOpenMatch) — sinon `[] ?? data` change de référence à chaque render.
  const mediaItems = useMemo<MediaItemRow[]>(() => data?.items?.items ?? [], [data])
  const playlistOptions = data?.available_filters.playlists ?? []
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

  const groups = useMemo(
    () => buildGroups(mediaItems, groupBy, text, locale),
    [mediaItems, groupBy, text, locale],
  )

  const playerColorMap = useMemo(
    () => buildOwnerColorMap(mediaItems, currentPlayerGamertag),
    [mediaItems, currentPlayerGamertag],
  )

  const indexByPath = useMemo(() => {
    const map = new Map<string, number>()
    mediaItems.forEach((item, index) => map.set(item.file_path, index))
    return map
  }, [mediaItems])

  // Navigation contextuelle : quand on ouvre un match depuis la galerie, on
  // capte la liste des match_ids présents dans la page courante pour que les
  // boutons prev/next sur la page Match restent dans le scope « médias ».
  // Limite connue : seuls les ~24 médias de la page courante sont inclus
  // (PAGE_SIZE) — pas l'ensemble des médias filtrés multi-pages.
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const handleOpenMatch = useCallback(
    (matchId: string) => {
      const matchIds = Array.from(
        new Set(
          mediaItems
            .map((item) => item.match_id)
            .filter((id): id is string => Boolean(id)),
        ),
      )
      navigateToMatch(matchId, {
        source: 'media',
        matchIds,
        contextDescriptor: { kind: 'media' },
      })
    },
    [mediaItems, navigateToMatch],
  )

  function handleKindChange(value: string) {
    setKindFilter(value)
    setPage(1)
  }

  function handleAuthorSlugsChange(slugs: string[]) {
    setAuthorSlugs(slugs)
    setPage(1)
  }

  function handlePlaylistChange(value: string) {
    setPlaylistFilter(value)
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

  function handleUnassignedOnlyChange(value: boolean) {
    setUnassignedOnly(value)
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
          globalIndexOffset={(page - 1) * PAGE_SIZE}
          globalTotal={pagination?.total ?? mediaItems.length}
          onReassociate={picker.openFor}
          currentPlayerGamertag={currentPlayerGamertag}
          playerSlug={playerSlug}
          onOpenMatch={handleOpenMatch}
          autoChain={autoChain}
          onToggleAutoChain={() => setAutoChain((prev) => !prev)}
        />
      )}

      {picker.state && (
        <MediaMatchPicker
          playerSlug={playerSlug}
          filePath={picker.state.filePath}
          hasCurrentMatch={picker.state.hasCurrentMatch}
          onClose={picker.close}
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
          playlistFilter={playlistFilter}
          mapFilter={mapFilter}
          modeFilter={modeFilter}
          groupBy={groupBy}
          sortKey={sortKey}
          likedOnly={likedOnly}
          unassignedOnly={unassignedOnly}
          totalUnassigned={data?.total_unassigned ?? 0}
          playlistOptions={playlistOptions}
          mapOptions={mapOptions}
          modeOptions={modeOptions}
          onKindChange={handleKindChange}
          onAuthorSlugsChange={handleAuthorSlugsChange}
          onPlaylistChange={handlePlaylistChange}
          onMapChange={handleMapChange}
          onModeChange={handleModeChange}
          onSortChange={handleSortChange}
          onGroupByChange={handleGroupByChange}
          onLikedOnlyChange={handleLikedOnlyChange}
          onUnassignedOnlyChange={handleUnassignedOnlyChange}
        />
      </div>

      {isLoading ? null : isError ? (
        <div className="p-8 text-center text-destructive">{text.errorPrefix} {extractErrorMessage(error)}</div>
      ) : mediaItems.length === 0 ? (
        <Card>
          <CardContent className="flex min-h-48 items-center justify-center text-muted-foreground">
            {text.emptyState}
          </CardContent>
        </Card>
      ) : (
        <>
          <div className={`flex flex-col gap-6 transition-opacity ${isFetching ? 'opacity-70' : 'opacity-100'}`}>
            {groups.map((group) => (
              <section key={group.key} className="flex flex-col gap-3">
                {group.label && (
                  <h3 className="px-1 text-sm font-semibold text-muted-foreground">{group.label}</h3>
                )}
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {group.items.map((item) => {
                    const flatIndex = indexByPath.get(item.file_path) ?? 0
                    return (
                      <MediaThumbnailCard
                        key={item.file_path}
                        item={item}
                        onToggleLike={(currentItem) => {
                          toggleMediaLike.mutate({
                            file_path: currentItem.file_path,
                            liked: !currentItem.liked,
                          })
                        }}
                        onOpen={() => setLightboxIdx(flatIndex)}
                        likeDisabled={toggleMediaLike.isPending}
                        playerSlug={playerSlug}
                        currentPlayerGamertag={currentPlayerGamertag}
                        playerColorMap={playerColorMap}
                        onAssociate={picker.openFor}
                        onOpenMatch={handleOpenMatch}
                      />
                    )
                  })}
                </div>
              </section>
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
