/**
 * MediaPage — Galerie de médias (Slice 8).
 * Types ref: MediaItemRow, MediaQueryRequest, MediaPageResponse
 */
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
