import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Badge } from '@/components/ui/badge'
import { GifHoverThumbnail } from '@/components/ui/gif-hover-thumbnail'
import type { MediaItemRow } from '@/lib/api/types'

function formatMediaDate(value: string | null | undefined) {
  if (!value) {
    return null
  }
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) {
    return null
  }
  const datePart = d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: '2-digit' })
  const timePart = d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
  return `${timePart} ${datePart}`
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

// LikersLine — affiche "Alice, Bob et 3 autres ♥" sous le bouton like
function LikersLine({ likers, totalLikers }: { likers?: string[]; totalLikers?: number }) {
  if (!totalLikers || totalLikers === 0) return null
  const names = likers ?? []
  const rest = totalLikers - names.length
  let label: string
  if (names.length === 0) {
    label = `${totalLikers} ♥`
  } else if (rest <= 0) {
    label = `${names.join(', ')} ♥`
  } else {
    label = `${names.join(', ')} et ${rest} autre${rest > 1 ? 's' : ''} ♥`
  }
  return <p className="text-[11px] text-rose-400 leading-tight">{label}</p>
}

function HeartIcon({ filled, className }: { filled: boolean; className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill={filled ? 'currentColor' : 'none'}
      stroke="currentColor"
      strokeWidth={2}
      className={className}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z"
      />
    </svg>
  )
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
  const [isAnimating, setIsAnimating] = useState(false)

  function handleClick(event: React.MouseEvent) {
    event.stopPropagation()
    onToggle()
    setIsAnimating(true)
    setTimeout(() => setIsAnimating(false), 250)
  }

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={handleClick}
      className={compact
        ? `absolute right-1.5 top-1.5 flex h-7 min-w-7 items-center justify-center gap-1 rounded-full px-2 text-[11px] font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isLiked ? 'bg-black/55 text-rose-400' : 'bg-black/45 text-white/50 hover:text-rose-300'}`
        : `inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isLiked ? 'border-rose-500/40 bg-rose-500/10 text-rose-400' : 'border-white/20 bg-black/35 text-white/85 hover:border-rose-400/40 hover:text-rose-300'}`}
      aria-label={isLiked ? 'Retirer le like' : 'Liker'}
    >
      <HeartIcon
        filled={isLiked}
        className={`transition-transform duration-200 ${compact ? 'h-4 w-4' : 'h-3.5 w-3.5'} ${isAnimating ? 'scale-125' : 'scale-100'}`}
      />
      {compact
        ? likeCount > 0 && <span>{likeCount}</span>
        : <span>{likeCount}</span>}
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
  const [isHovering, setIsHovering] = useState(false)
  const dateStr = formatMediaDate(item.capture_end_utc ?? item.match_start_time)

  return (
    <article
      className="group relative overflow-hidden rounded-lg border bg-muted transition-shadow hover:shadow-md"
      onClick={onOpen}
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
      onFocus={() => setIsHovering(true)}
      onBlur={() => setIsHovering(false)}
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
        {item.kind === 'clip' && item.thumbnail_path ? (
          <GifHoverThumbnail
            src={item.thumbnail_path}
            isActive={isHovering}
            className="absolute inset-0 h-full w-full"
          />
        ) : item.thumbnail_path ? (
          <img
            src={item.thumbnail_path}
            alt={item.basename}
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
          />
        ) : item.kind === 'screenshot' ? (
          <img
            src={item.file_path}
            alt={item.basename}
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-2xl text-muted-foreground">
            ▶
          </div>
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
        <div className="flex items-center gap-1 min-w-0">
          <Badge variant={item.kind === 'clip' ? 'default' : 'secondary'} className="text-xs shrink-0">
            {item.kind}
          </Badge>
          {item.map_name && <span className="truncate text-xs text-muted-foreground min-w-0">{item.map_name}</span>}
          {dateStr && <span className="ml-auto shrink-0 text-xs text-muted-foreground">{dateStr}</span>}
        </div>
        <LikersLine likers={item.likers} totalLikers={item.total_likers} />
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
  hasNextPage?: boolean
  onLoadNextPage?: () => void
  /** Index global du 1er item de la page courante (0-indexed). Utilisé pour le compteur X/Y. */
  globalIndexOffset?: number
  /** Nombre total de médias toutes pages confondues. Si absent, fallback sur items.length. */
  globalTotal?: number
  /** Si fourni, active le bouton "Réassocier" qui ouvre MediaMatchPicker. */
  onReassociate?: (item: MediaItemRow) => void
}

const IMAGE_AUTOCHAIN_DELAY_MS = 7000
const FADE_TRANSITION_MS = 250

export function MediaLightbox({
  items,
  onToggleLike,
  startIndex,
  onClose,
  likeDisabled = false,
  hasNextPage = false,
  onLoadNextPage,
  globalIndexOffset = 0,
  globalTotal,
  onReassociate,
}: MediaLightboxProps) {
  const [index, setIndex] = useState(startIndex)
  const [autoChain, setAutoChain] = useState(false)
  const [pendingPageAdvance, setPendingPageAdvance] = useState(false)
  const [contentVisible, setContentVisible] = useState(true)

  useEffect(() => {
    setIndex(startIndex)
  }, [startIndex])

  // Quand une nouvelle page arrive après une demande d'avance, saute à l'item 0.
  useEffect(() => {
    if (pendingPageAdvance && items.length > 0) {
      setIndex(0)
      setPendingPageAdvance(false)
    }
  }, [items, pendingPageAdvance])

  const item = items[index]
  const isLast = index >= items.length - 1
  const isClip = item?.kind === 'clip'
  const canAdvanceFurther = !isLast || hasNextPage

  // Brève transition fade-in à chaque changement d'item (atténue la coupure
  // quand l'auto-chain enchaîne deux médias).
  useEffect(() => {
    if (!item) return
    setContentVisible(false)
    const timer = window.setTimeout(() => setContentVisible(true), 30)
    return () => window.clearTimeout(timer)
  }, [item?.file_path])

  const goNext = useCallback(() => {
    setIndex((current) => {
      if (current < items.length - 1) return current + 1
      if (hasNextPage && onLoadNextPage && !pendingPageAdvance) {
        setPendingPageAdvance(true)
        onLoadNextPage()
      }
      return current
    })
  }, [items.length, hasNextPage, onLoadNextPage, pendingPageAdvance])

  const goPrev = useCallback(() => {
    setIndex((current) => Math.max(0, current - 1))
  }, [])

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'ArrowLeft') goPrev()
      if (event.key === 'ArrowRight') goNext()
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [goNext, goPrev, onClose])

  // Image auto-advance — clips s'enchaînent via onEnded sur le <video>.
  useEffect(() => {
    if (!autoChain || !item || isClip || !canAdvanceFurther || pendingPageAdvance) return
    const timeout = setTimeout(goNext, IMAGE_AUTOCHAIN_DELAY_MS)
    return () => clearTimeout(timeout)
  }, [autoChain, item, isClip, canAdvanceFurther, pendingPageAdvance, goNext])

  const handleVideoEnded = useCallback(() => {
    if (autoChain && canAdvanceFurther && !pendingPageAdvance) goNext()
  }, [autoChain, canAdvanceFurther, pendingPageAdvance, goNext])

  if (!item) {
    return null
  }

  // Compteur X/Y : position GLOBALE (page courante × pageSize + index local) sur total global,
  // pas l'index local sur la taille de page (sinon "5/24" trompeur quand il y a 47 médias).
  const total = globalTotal ?? items.length
  const globalIndex = globalIndexOffset + index
  const heading = formatLightboxHeading(item, globalIndex, total)
  const autoChainLabel = autoChain ? 'Enchaînement actif' : 'Activer l\'enchaînement'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90" onClick={onClose}>
      <div className="relative mx-4 flex max-h-screen w-full max-w-5xl flex-col" onClick={(event) => event.stopPropagation()}>
        <div className="flex items-center justify-between bg-black/60 px-4 py-2 text-white">
          <span className="truncate text-sm opacity-80">{heading}</span>
          <div className="flex items-center gap-3">
            {onReassociate && (
              <button
                type="button"
                onClick={() => onReassociate(item)}
                className="rounded border border-white/20 px-2 py-0.5 text-xs text-white/80 hover:border-white/50 hover:text-white"
                title="Réassocier ce média à un autre match"
              >
                Réassocier
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="text-xl leading-none text-white/70 hover:text-white"
              aria-label="Fermer"
            >
              ✕
            </button>
          </div>
        </div>

        <div className="relative flex items-center justify-center overflow-hidden bg-black" style={{ maxHeight: '80vh' }}>
          <div
            className="flex max-h-full max-w-full items-center justify-center transition-opacity ease-out"
            style={{
              opacity: contentVisible ? 1 : 0,
              transitionDuration: `${FADE_TRANSITION_MS}ms`,
            }}
          >
            {isClip ? (
              <video
                key={item.file_path}
                src={item.file_path}
                controls
                autoPlay
                playsInline
                preload="metadata"
                onEnded={handleVideoEnded}
                className="max-h-full max-w-full"
              />
            ) : (
              <img src={item.file_path} alt={item.basename} className="max-h-full max-w-full object-contain" />
            )}
          </div>

          <button
            type="button"
            onClick={() => setAutoChain((current) => !current)}
            aria-pressed={autoChain}
            aria-label={autoChainLabel}
            title={autoChainLabel}
            className={`absolute right-2 top-2 inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium backdrop-blur transition-colors ${
              autoChain
                ? 'border-rose-400/60 bg-rose-500/20 text-rose-200 hover:bg-rose-500/30'
                : 'border-white/20 bg-black/60 text-white/85 hover:border-white/40 hover:text-white'
            }`}
          >
            <span aria-hidden="true">⏵</span>
            <span>Enchaîner</span>
          </button>

          {index > 0 && (
            <button
              type="button"
              onClick={goPrev}
              className="absolute left-2 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 text-2xl text-white hover:bg-black/80"
              aria-label="Précédent"
            >
              ◀
            </button>
          )}
          {canAdvanceFurther && (
            <button
              type="button"
              onClick={goNext}
              disabled={pendingPageAdvance}
              className="absolute right-2 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 text-2xl text-white hover:bg-black/80 disabled:cursor-wait disabled:opacity-60"
              aria-label="Suivant"
            >
              ▶
            </button>
          )}
        </div>

        <div className="flex flex-col gap-1 bg-black/60 px-4 py-2">
          <div className="flex items-center gap-2">
          <Badge variant="secondary" className="text-xs">{item.kind}</Badge>
          <MediaLikeButton
            isLiked={item.liked}
            likeCount={item.like_count}
            onToggle={() => onToggleLike(item)}
            disabled={likeDisabled}
          />
          {item.match_id && item.owner_gamertag ? (
            <Link
              to="/players/$playerSlug/matches/$matchId"
              params={{ playerSlug: item.owner_gamertag, matchId: item.match_id }}
              className="ml-auto inline-flex items-center gap-1 whitespace-nowrap rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-white/80 transition-colors hover:border-white/40 hover:text-white"
              onClick={(event) => event.stopPropagation()}
            >
              Voir le match →
            </Link>
          ) : (
            <span className="ml-auto whitespace-nowrap text-xs text-white/35 italic">Aucun match associé</span>
          )}
          </div>
          <LikersLine likers={item.likers} totalLikers={item.total_likers} />
        </div>
      </div>
    </div>
  )
}
