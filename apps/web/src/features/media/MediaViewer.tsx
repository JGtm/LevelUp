import { useEffect, useState } from 'react'
import { Badge } from '@/components/ui/badge'
import type { MediaItemRow } from '@/lib/api/types'

function formatMediaDate(value: string | null) {
  if (!value) {
    return null
  }
  return new Date(value).toLocaleDateString('fr-FR', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
  })
}

function formatLightboxHeading(item: MediaItemRow, index: number, total: number) {
  const dateStr = item.match_start_time || item.capture_end_utc
    ? new Date((item.match_start_time ?? item.capture_end_utc)!).toLocaleDateString('fr-FR', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : null
  return [item.map_name, dateStr, `${index + 1}/${total}`].filter(Boolean).join(' · ')
}

interface MediaLikeButtonProps {
  isLiked: boolean
  likeCount: number
  onToggle: () => void
  compact?: boolean
  disabled?: boolean
}

function MediaLikeButton({
  isLiked,
  likeCount,
  onToggle,
  compact = false,
  disabled = false,
}: MediaLikeButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={(event) => {
        event.stopPropagation()
        onToggle()
      }}
      className={compact
        ? `absolute right-1.5 top-1.5 flex h-7 min-w-7 items-center justify-center gap-1 rounded-full px-2 text-[11px] font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isLiked ? 'bg-destructive text-white' : 'bg-black/45 text-white/70 hover:bg-destructive/70 hover:text-white'}`
        : `inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isLiked ? 'border-destructive bg-destructive text-white' : 'border-white/20 bg-black/35 text-white/85 hover:border-destructive hover:bg-destructive/70'}`}
      aria-label={isLiked ? 'Retirer le like' : 'Liker'}
    >
      <span aria-hidden="true">♥</span>
      {compact
        ? likeCount > 0 && <span>{likeCount}</span>
        : <span>{isLiked ? `Aimé · ${likeCount}` : `Liker · ${likeCount}`}</span>}
    </button>
  )
}

interface MediaThumbnailCardProps {
  item: MediaItemRow
  onToggleLike: (item: MediaItemRow) => void
  onOpen: () => void
  likeDisabled?: boolean
}

export function MediaThumbnailCard({ item, onToggleLike, onOpen, likeDisabled = false }: MediaThumbnailCardProps) {
  const [isPreviewing, setIsPreviewing] = useState(false)
  const showPreview = item.kind === 'clip' && isPreviewing
  const dateStr = formatMediaDate(item.capture_end_utc)

  return (
    <article
      className="group relative overflow-hidden rounded-lg border bg-muted transition-shadow hover:shadow-md"
      onClick={onOpen}
      onMouseEnter={() => setIsPreviewing(true)}
      onMouseLeave={() => setIsPreviewing(false)}
      onFocus={() => setIsPreviewing(true)}
      onBlur={() => setIsPreviewing(false)}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onOpen()
        }
      }}
      role="button"
      tabIndex={0}
    >
      <div className="relative aspect-video overflow-hidden bg-muted">
        {showPreview ? (
          <video
            src={item.file_path}
            poster={item.thumbnail_path ?? undefined}
            autoPlay
            loop
            muted
            playsInline
            preload="metadata"
            className="h-full w-full object-cover"
          />
        ) : item.thumbnail_path ? (
          <img
            src={item.thumbnail_path}
            alt={item.basename}
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-2xl text-muted-foreground">
            {item.kind === 'clip' ? '▶' : '🖼'}
          </div>
        )}
        {item.kind === 'clip' && !showPreview && (
          <span className="absolute bottom-2 left-2 rounded-full bg-black/55 px-2 py-0.5 text-[11px] font-medium text-white">
            Aperçu au survol
          </span>
        )}
        <MediaLikeButton
          compact
          isLiked={item.liked}
          likeCount={item.like_count}
          onToggle={() => onToggleLike(item)}
          disabled={likeDisabled}
        />
      </div>

      <div className="flex flex-col gap-1 p-2">
        <div className="flex items-center gap-1">
          <Badge variant={item.kind === 'clip' ? 'default' : 'secondary'} className="text-xs">
            {item.kind}
          </Badge>
          {item.map_name && <span className="truncate text-xs text-muted-foreground">{item.map_name}</span>}
        </div>
        <p className="truncate text-xs text-muted-foreground" title={item.basename}>{item.basename}</p>
        {dateStr && <p className="text-xs text-muted-foreground">{dateStr}</p>}
      </div>
    </article>
  )
}

interface MediaLightboxProps {
  items: MediaItemRow[]
  onToggleLike: (item: MediaItemRow) => void
  startIndex: number
  onClose: () => void
  likeDisabled?: boolean
}

export function MediaLightbox({ items, onToggleLike, startIndex, onClose, likeDisabled = false }: MediaLightboxProps) {
  const [index, setIndex] = useState(startIndex)

  useEffect(() => {
    setIndex(startIndex)
  }, [startIndex])

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'ArrowLeft') {
        setIndex((current) => Math.max(0, current - 1))
      }
      if (event.key === 'ArrowRight') {
        setIndex((current) => Math.min(items.length - 1, current + 1))
      }
      if (event.key === 'Escape') {
        onClose()
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [items.length, onClose])

  const item = items[index]
  if (!item) {
    return null
  }

  const heading = formatLightboxHeading(item, index, items.length)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90" onClick={onClose}>
      <div className="relative mx-4 flex max-h-screen w-full max-w-5xl flex-col" onClick={(event) => event.stopPropagation()}>
        <div className="flex items-center justify-between bg-black/60 px-4 py-2 text-white">
          <span className="truncate text-sm opacity-80">{heading}</span>
          <button
            type="button"
            onClick={onClose}
            className="ml-4 text-xl leading-none text-white/70 hover:text-white"
            aria-label="Fermer"
          >
            ✕
          </button>
        </div>

        <div className="relative flex items-center justify-center overflow-hidden bg-black" style={{ maxHeight: '80vh' }}>
          {item.kind === 'clip' ? (
            <video
              src={item.file_path}
              controls
              autoPlay
              playsInline
              preload="metadata"
              className="max-h-full max-w-full"
            />
          ) : (
            <img src={item.file_path} alt={item.basename} className="max-h-full max-w-full object-contain" />
          )}

          {index > 0 && (
            <button
              type="button"
              onClick={() => setIndex((current) => Math.max(0, current - 1))}
              className="absolute left-2 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 text-2xl text-white hover:bg-black/80"
              aria-label="Précédent"
            >
              ◀
            </button>
          )}
          {index < items.length - 1 && (
            <button
              type="button"
              onClick={() => setIndex((current) => Math.min(items.length - 1, current + 1))}
              className="absolute right-2 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 text-2xl text-white hover:bg-black/80"
              aria-label="Suivant"
            >
              ▶
            </button>
          )}
        </div>

        <div className="flex items-center gap-2 bg-black/60 px-4 py-2">
          <Badge variant="secondary" className="text-xs">{item.kind}</Badge>
          <span className="truncate text-xs text-white/60">{item.basename}</span>
          <MediaLikeButton
            isLiked={item.liked}
            likeCount={item.like_count}
            onToggle={() => onToggleLike(item)}
            disabled={likeDisabled}
          />
          {item.match_id && (
            <a
              href={`/players/${item.owner_gamertag ?? 'me'}/match/${item.match_id}`}
              className="ml-auto whitespace-nowrap text-xs text-primary underline hover:text-primary/80"
              onClick={(event) => event.stopPropagation()}
            >
              Voir le match →
            </a>
          )}
        </div>
      </div>
    </div>
  )
}
