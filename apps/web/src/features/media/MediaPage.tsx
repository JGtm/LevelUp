/**
 * MediaPage — Galerie de médias (Slice 8).
 * Types ref: MediaItemRow, MediaQueryRequest, MediaPageResponse
 *
 * Features :
 * - Grille responsive avec thumbnails
 * - Lightbox (navigation ◀▶, en-tête carte/mode/date, autoclose sur Escape)
 * - Likes persistés en localStorage (pas en DB)
 * - Toolbar : type, section, tri, ordre
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams } from '@tanstack/react-router'
import { useMediaPage } from './queries'
import { UploadButton } from './UploadButton'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { MediaItemRow, MediaQueryRequest } from '@/lib/api/types'

// ─── Constantes ───────────────────────────────────────────────────────────────

const LIKES_KEY = 'levelup_media_likes_v1'
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

// ─── Hook likes localStorage ──────────────────────────────────────────────────

function loadLikes(): Set<string> {
  try {
    const raw = localStorage.getItem(LIKES_KEY)
    if (!raw) return new Set()
    return new Set(JSON.parse(raw) as string[])
  } catch {
    return new Set()
  }
}

function useLikes() {
  const [likes, setLikes] = useState<Set<string>>(() => loadLikes())

  const toggle = useCallback((filePath: string) => {
    setLikes((prev) => {
      const next = new Set(prev)
      if (next.has(filePath)) next.delete(filePath)
      else next.add(filePath)
      try {
        localStorage.setItem(LIKES_KEY, JSON.stringify([...next]))
      } catch { /* quota */ }
      return next
    })
  }, [])

  return { likes, toggle }
}

// ─── Lightbox ─────────────────────────────────────────────────────────────────

interface LightboxProps {
  items: MediaItemRow[]
  startIndex: number
  onClose: () => void
}

function Lightbox({ items, startIndex, onClose }: LightboxProps) {
  const [idx, setIdx] = useState(startIndex)
  const videoRef = useRef<HTMLVideoElement>(null)
  const item = items[idx]
  if (!item) return null

  const prev = () => setIdx((i) => Math.max(0, i - 1))
  const next = () => setIdx((i) => Math.min(items.length - 1, i + 1))

  // Keyboard nav
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'ArrowLeft') prev()
      else if (e.key === 'ArrowRight') next()
      else if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose]) // eslint-disable-line react-hooks/exhaustive-deps

  const dateStr = item.match_start_time || item.capture_end_utc
    ? new Date((item.match_start_time ?? item.capture_end_utc)!).toLocaleDateString('fr-FR', {
        day: 'numeric', month: 'long', year: 'numeric',
      })
    : null

  const caption = [item.map_name, dateStr]
    .filter(Boolean)
    .join(' · ')
  const heading = `${caption}  ${idx + 1}/${items.length}`

  const isClip = item.kind === 'clip'

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90"
      onClick={onClose}
    >
      {/* Panneau central */}
      <div
        className="relative flex max-h-screen max-w-5xl w-full flex-col mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* En-tête */}
        <div className="flex items-center justify-between bg-black/60 px-4 py-2 text-white">
          <span className="text-sm opacity-80 truncate">{heading}</span>
          <button
            onClick={onClose}
            className="ml-4 text-white/70 hover:text-white text-xl leading-none"
            aria-label="Fermer"
          >
            ✕
          </button>
        </div>

        {/* Contenu */}
        <div className="relative flex items-center justify-center bg-black overflow-hidden"
             style={{ maxHeight: '80vh' }}>
          {isClip ? (
            <video
              ref={videoRef}
              src={item.file_path}
              controls
              autoPlay
              onEnded={next}
              className="max-h-full max-w-full"
            />
          ) : (
            <img
              src={item.file_path}
              alt={item.basename}
              className="max-h-full max-w-full object-contain"
            />
          )}

          {/* Flèches */}
          {idx > 0 && (
            <button
              onClick={prev}
              className="absolute left-2 top-1/2 -translate-y-1/2 bg-black/50 hover:bg-black/80 text-white text-2xl rounded-full w-10 h-10 flex items-center justify-center"
              aria-label="Précédent"
            >
              ◀
            </button>
          )}
          {idx < items.length - 1 && (
            <button
              onClick={next}
              className="absolute right-2 top-1/2 -translate-y-1/2 bg-black/50 hover:bg-black/80 text-white text-2xl rounded-full w-10 h-10 flex items-center justify-center"
              aria-label="Suivant"
            >
              ▶
            </button>
          )}
        </div>

        {/* Pied */}
        <div className="bg-black/60 px-4 py-2 flex items-center gap-2">
          <Badge variant="secondary" className="text-xs">{item.kind}</Badge>
          <span className="text-xs text-white/60 truncate">{item.basename}</span>
          {item.match_id && (
            <a
              href={`/players/${item.owner_gamertag ?? 'me'}/match/${item.match_id}`}
              className="ml-auto text-xs text-purple-400 hover:text-purple-300 underline whitespace-nowrap"
              onClick={(e) => e.stopPropagation()}
            >
              Voir le match →
            </a>
          )}
        </div>
      </div>
    </div>
  )
}

// ─── Vignette média ───────────────────────────────────────────────────────────

interface MediaCardProps {
  item: MediaItemRow
  isLiked: boolean
  onLike: (e: React.MouseEvent) => void
  onClick: () => void
}

function MediaCard({ item, isLiked, onLike, onClick }: MediaCardProps) {
  const dateStr = item.capture_end_utc
    ? new Date(item.capture_end_utc).toLocaleDateString('fr-FR', {
        day: '2-digit', month: '2-digit', year: '2-digit',
      })
    : null

  return (
    <div
      className="group relative rounded-lg border overflow-hidden bg-gray-50 hover:shadow-md transition-shadow cursor-pointer"
      onClick={onClick}
    >
      {/* Thumbnail */}
      <div className="aspect-video bg-gray-200 flex items-center justify-center overflow-hidden">
        {item.thumbnail_path ? (
          <img
            src={item.thumbnail_path}
            alt={item.basename}
            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
          />
        ) : (
          <span className="text-2xl text-gray-400">{item.kind === 'clip' ? '▶' : '🖼'}</span>
        )}
      </div>

      {/* Like button */}
      <button
        onClick={onLike}
        className={`absolute top-1.5 right-1.5 rounded-full w-7 h-7 flex items-center justify-center text-base transition-colors
          ${isLiked ? 'bg-red-500 text-white' : 'bg-black/40 text-white/60 hover:bg-red-500/60 hover:text-white'}`}
        aria-label={isLiked ? 'Retirer le like' : 'Liker'}
      >
        ♥
      </button>

      {/* Infos */}
      <div className="p-2 flex flex-col gap-0.5">
        <div className="flex items-center gap-1">
          <Badge variant={item.kind === 'clip' ? 'default' : 'secondary'} className="text-xs">{item.kind}</Badge>
          {item.map_name && <span className="text-xs text-gray-400 truncate">{item.map_name}</span>}
        </div>
        <p className="text-xs text-gray-600 truncate" title={item.basename}>{item.basename}</p>
        {dateStr && <p className="text-xs text-gray-400">{dateStr}</p>}
      </div>
    </div>
  )
}

// ─── Page principale ─────────────────────────────────────────────────────────

export function MediaPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/media' })
  const { likes, toggle: toggleLike } = useLikes()

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

  const mediaItems: MediaItemRow[] = data?.items?.items ?? []
  const pagination = data?.items?.pagination
  const totalPages = pagination ? Math.ceil(pagination.total / PAGE_SIZE) : 1
  const totalLabel = data
    ? `${data.total_mine} perso · ${data.total_teammates} équipe · ${data.total_unassigned} non assigné`
    : 'Galerie de captures et clips'

  return (
    <div className="flex flex-col gap-6">
      {lightboxIdx !== null && (
        <Lightbox
          items={mediaItems}
          startIndex={lightboxIdx}
          onClose={() => setLightboxIdx(null)}
        />
      )}

      <PageHeader
        title="Médias"
        subtitle={totalLabel}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <select className="rounded border px-2 py-1 text-sm" value={kindFilter}
              onChange={(e) => { setKindFilter(e.target.value); setPage(1) }}>
              {KIND_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
            <select className="rounded border px-2 py-1 text-sm" value={sectionFilter}
              onChange={(e) => { setSectionFilter(e.target.value); setPage(1) }}>
              {SECTION_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
            <input
              type="text"
              placeholder="Filtrer carte…"
              value={mapFilter}
              onChange={(e) => { setMapFilter(e.target.value); setPage(1) }}
              className="rounded border px-2 py-1 text-sm w-28"
            />
            <input
              type="text"
              placeholder="Filtrer mode…"
              value={modeFilter}
              onChange={(e) => { setModeFilter(e.target.value); setPage(1) }}
              className="rounded border px-2 py-1 text-sm w-28"
            />
            <select className="rounded border px-2 py-1 text-sm" value={sortKey}
              onChange={(e) => { setSortKey(e.target.value); setPage(1) }}>
              {SORT_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
            <select className="rounded border px-2 py-1 text-sm" value={groupBy}
              onChange={(e) => { setGroupBy(e.target.value); setPage(1) }}>
              {GROUP_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
            <div className="ml-auto">
              <UploadButton playerSlug={playerSlug} />
            </div>
          </div>
        }
      />

      {isLoading ? (
        <div className="flex items-center justify-center min-h-64"><Spinner size="lg" /></div>
      ) : isError ? (
        <div className="p-8 text-center text-red-600">Erreur : {String(error)}</div>
      ) : mediaItems.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center text-gray-500">
            Aucun média disponible pour ces filtres.
          </CardContent>
        </Card>
      ) : (
        <>
          <div className={`grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 transition-opacity ${isFetching ? 'opacity-70' : 'opacity-100'}`}>
            {mediaItems.map((item, i) => (
              <MediaCard
                key={`${item.file_path}-${i}`}
                item={item}
                isLiked={likes.has(item.file_path)}
                onLike={(e) => { e.stopPropagation(); toggleLike(item.file_path) }}
                onClick={() => setLightboxIdx(i)}
              />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-2">
              <button className="px-3 py-1 rounded border text-sm disabled:opacity-40"
                disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
                ← Précédent
              </button>
              <span className="text-sm text-gray-500">Page {page} / {totalPages}</span>
              <button className="px-3 py-1 rounded border text-sm disabled:opacity-40"
                disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
                Suivant →
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useMediaPage } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { MediaItemRow, MediaQueryRequest } from '@/lib/api/types'

// ─── Constantes ───────────────────────────────────────────────────────────────

const KIND_OPTIONS = [
  { value: '', label: 'Tous' },
  { value: 'screenshot', label: 'Screenshots' },
  { value: 'clip', label: 'Clips' },
  { value: 'thumbnail', label: 'Vignettes' },
]

const SECTION_OPTIONS = [
  { value: '', label: 'Toutes' },
  { value: 'career', label: 'Carrière' },
  { value: 'match', label: 'Match' },
  { value: 'highlight', label: 'Highlight' },
]

const PAGE_SIZE = 24

// ─── Sous-composant : vignette média ─────────────────────────────────────────

interface MediaCardProps {
  item: MediaItemRow
}

function MediaCard({ item }: MediaCardProps) {
  const dateStr = item.capture_end_utc
    ? new Date(item.capture_end_utc).toLocaleDateString('fr-FR', {
        day: '2-digit',
        month: '2-digit',
        year: '2-digit',
      })
    : null

  return (
    <div className="group relative rounded-lg border overflow-hidden bg-gray-50 hover:shadow-md transition-shadow">
      {/* Thumbnail */}
      <div className="aspect-video bg-gray-200 flex items-center justify-center overflow-hidden">
        {item.thumbnail_path ? (
          <img
            src={item.thumbnail_path}
            alt={item.basename}
            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
          />
        ) : (
          <span className="text-2xl text-gray-400">{item.kind === 'clip' ? '[clip]' : '[img]'}</span>
        )}
      </div>

      {/* Infos */}
      <div className="p-2 flex flex-col gap-1">
        <div className="flex items-center gap-1">
          <Badge variant={item.kind === 'clip' ? 'default' : 'secondary'} className="text-xs">
            {item.kind}
          </Badge>
          <Badge variant="outline" className="text-xs">{item.section}</Badge>
        </div>
        <p className="text-xs text-gray-600 truncate" title={item.basename}>
          {item.basename}
        </p>
        {item.map_name && (
          <p className="text-xs text-gray-400">{item.map_name}</p>
        )}
        {dateStr && <p className="text-xs text-gray-400">{dateStr}</p>}
      </div>

      {/* Overlay lien */}
      <a
        href={item.file_path}
        target="_blank"
        rel="noopener noreferrer"
        className="absolute inset-0"
        aria-label={`Ouvrir ${item.basename}`}
      />
    </div>
  )
}

// ─── Page principale ─────────────────────────────────────────────────────────

export function MediaPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/media' })

  const [page, setPage] = useState(1)
  const [kindFilter, setKindFilter] = useState('')
  const [sectionFilter, setSectionFilter] = useState('')

  const request: MediaQueryRequest = {
    kind_filter: kindFilter || null,
    section_filter: sectionFilter || null,
    pagination: { page, page_size: PAGE_SIZE },
  }

  const { data, isLoading, isError, error, isFetching } = useMediaPage(
    playerSlug,
    request,
    page,
  )

  // data.items est de type PaginatedResponse<MediaItemRow>
  const paginatedItems = data?.items
  const mediaItems: MediaItemRow[] = paginatedItems?.items ?? []
  const pagination = paginatedItems?.pagination
  const totalPages = pagination ? Math.ceil(pagination.total / PAGE_SIZE) : 1

  const totalLabel = data
    ? `${data.total_mine} perso · ${data.total_teammates} équipe · ${data.total_unassigned} non assigné`
    : 'Galerie de captures et clips'

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Médias"
        subtitle={totalLabel}
        actions={
          <div className="flex gap-2">
            <select
              className="rounded border px-2 py-1 text-sm"
              value={kindFilter}
              onChange={(e) => { setKindFilter(e.target.value); setPage(1) }}
            >
              {KIND_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
            <select
              className="rounded border px-2 py-1 text-sm"
              value={sectionFilter}
              onChange={(e) => { setSectionFilter(e.target.value); setPage(1) }}
            >
              {SECTION_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>
        }
      />

      {isLoading ? (
        <div className="flex items-center justify-center min-h-64">
          <Spinner size="lg" />
        </div>
      ) : isError ? (
        <div className="p-8 text-center text-red-600">
          Erreur : {String(error)}
        </div>
      ) : mediaItems.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center text-gray-500">
            Aucun média disponible pour ces filtres.
          </CardContent>
        </Card>
      ) : (
        <>
          <div
            className={`grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 transition-opacity ${
              isFetching ? 'opacity-70' : 'opacity-100'
            }`}
          >
            {mediaItems.map((item, idx) => (
              <MediaCard key={`${item.file_path}-${idx}`} item={item} />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-2">
              <button
                className="px-3 py-1 rounded border text-sm disabled:opacity-40"
                disabled={page === 1}
                onClick={() => setPage((p) => p - 1)}
              >
                ← Précédent
              </button>
              <span className="text-sm text-gray-500">
                Page {page} / {totalPages}
              </span>
              <button
                className="px-3 py-1 rounded border text-sm disabled:opacity-40"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
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
