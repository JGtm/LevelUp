/**
 * MediaPage — Galerie de médias (Slice 8).
 * Types ref: MediaItemRow, MediaQueryRequest, MediaPageResponse
 *
 * Features :
 * - Grille responsive avec thumbnails
 * - Lightbox (navigation ◀▶, vidéo auto-play, fermeture Escape)
 * - Likes persistés côté backend (player DB)
 * - Toolbar : type, section, tri, ordre
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import type { MediaItemRow, MediaQueryRequest } from '@/lib/api/types'
import { MediaLightbox, MediaThumbnailCard } from './MediaViewer'
import { UploadButton } from './UploadButton'
import { useMediaPage, useToggleMediaLike } from './queries'

const KIND_OPTIONS = [
  { value: '', label: 'Tous types' },
  { value: 'screenshot', label: 'Screenshots' },
  { value: 'clip', label: 'Clips' },
  { value: 'thumbnail', label: 'Vignettes' },
]
const SECTION_OPTIONS = [
  { value: '', label: 'Toutes sections' },
  { value: 'career', label: 'Carrière' },
  { value: 'match', label: 'Match' },
  { value: 'highlight', label: 'Highlight' },
]
const SORT_OPTIONS = [
  { value: 'date_desc', label: 'Date ↓' },
  { value: 'date_asc', label: 'Date ↑' },
  { value: 'name_asc', label: 'Nom A→Z' },
]
const GROUP_OPTIONS = [
  { value: '', label: 'Sans groupement' },
  { value: 'map', label: 'Par carte' },
  { value: 'mode', label: 'Par mode' },
  { value: 'week', label: 'Par semaine' },
  { value: 'section', label: 'Par section' },
  { value: 'owner', label: 'Par auteur' },
]
const PAGE_SIZE = 24

export function MediaPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/media' })
  const [page, setPage] = useState(1)
  const [kindFilter, setKindFilter] = useState('')
  const [sectionFilter, setSectionFilter] = useState('')
  const [mapFilter, setMapFilter] = useState('')
  const [modeFilter, setModeFilter] = useState('')
  const [groupBy, setGroupBy] = useState('')
  const [sortKey, setSortKey] = useState('date_desc')
  const [lightboxIdx, setLightboxIdx] = useState<number | null>(null)

  const request: MediaQueryRequest = {
    sort: sortKey,
    kind_filter: kindFilter || null,
    section_filter: sectionFilter || null,
    map_filter: mapFilter || null,
    mode_filter: modeFilter || null,
    group_by: groupBy || null,
    pagination: { page, page_size: PAGE_SIZE },
  }

  const { data, isLoading, isError, error, isFetching } = useMediaPage(playerSlug, request, page)
  const toggleMediaLike = useToggleMediaLike(playerSlug)
  const mediaItems: MediaItemRow[] = data?.items?.items ?? []
  const pagination = data?.items?.pagination
  const totalPages = pagination ? Math.ceil(pagination.total / PAGE_SIZE) : 1
  const totalLabel = data
    ? `${data.total_mine} perso · ${data.total_teammates} équipe · ${data.total_unassigned} non assigné`
    : 'Galerie de captures et clips'

  return (
    <div className="flex flex-col gap-6">
      {lightboxIdx !== null && (
        <MediaLightbox
          items={mediaItems}
          onToggleLike={(item) => toggleMediaLike.mutate({ file_path: item.file_path, liked: !item.liked })}
          startIndex={lightboxIdx}
          onClose={() => setLightboxIdx(null)}
          likeDisabled={toggleMediaLike.isPending}
        />
      )}

      <PageHeader
        title="Médias"
        subtitle={totalLabel}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <select
              className="rounded border px-2 py-1 text-sm"
              value={kindFilter}
              onChange={(event) => {
                setKindFilter(event.target.value)
                setPage(1)
              }}
            >
              {KIND_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
            <select
              className="rounded border px-2 py-1 text-sm"
              value={sectionFilter}
              onChange={(event) => {
                setSectionFilter(event.target.value)
                setPage(1)
              }}
            >
              {SECTION_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
            <input
              type="text"
              placeholder="Filtrer carte…"
              value={mapFilter}
              onChange={(event) => {
                setMapFilter(event.target.value)
                setPage(1)
              }}
              className="w-28 rounded border px-2 py-1 text-sm"
            />
            <input
              type="text"
              placeholder="Filtrer mode…"
              value={modeFilter}
              onChange={(event) => {
                setModeFilter(event.target.value)
                setPage(1)
              }}
              className="w-28 rounded border px-2 py-1 text-sm"
            />
            <select
              className="rounded border px-2 py-1 text-sm"
              value={sortKey}
              onChange={(event) => {
                setSortKey(event.target.value)
                setPage(1)
              }}
            >
              {SORT_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
            <select
              className="rounded border px-2 py-1 text-sm"
              value={groupBy}
              onChange={(event) => {
                setGroupBy(event.target.value)
                setPage(1)
              }}
            >
              {GROUP_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
            <div className="ml-auto">
              <UploadButton playerSlug={playerSlug} />
            </div>
          </div>
        }
      />

      {isLoading ? (
        <div className="flex min-h-64 items-center justify-center"><Spinner size="lg" /></div>
      ) : isError ? (
        <div className="p-8 text-center text-destructive">Erreur : {String(error)}</div>
      ) : mediaItems.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center text-muted-foreground">
            Aucun média disponible pour ces filtres.
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
                ← Précédent
              </button>
              <span className="text-sm text-muted-foreground">Page {page} / {totalPages}</span>
              <button
                className="rounded border px-3 py-1 text-sm disabled:opacity-40"
                disabled={page >= totalPages}
                onClick={() => setPage((current) => current + 1)}
              >
                Suivant →
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
