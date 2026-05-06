/**
 * CoverFlowModal — modale plein écran avec carrousel coverflow pour les médias.
 *
 * P8.5 (revue 2026-04-29 gap #14) : promu de `components/ui/` vers
 * `features/media/` car le composant est strictement media-specific
 * (importait MediaLikeButton + getMediaModalsText). La frontière inversée
 * `components/ → features/` est désormais respectée.
 */
'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { MediaItemRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { MediaLikeButton } from './MediaViewer'
import { getMediaModalsText } from './i18n-modals'

interface CoverFlowModalProps {
  items: MediaItemRow[]
  onToggleLike: (item: MediaItemRow) => void
  startIndex: number
  onClose: () => void
  likeDisabled?: boolean
  hasNextPage?: boolean
  onLoadNextPage?: () => void
  globalIndexOffset?: number
  globalTotal?: number
  onReassociate?: (item: MediaItemRow) => void
  autoChain?: boolean
  onToggleAutoChain?: () => void
}

type SlotPos = { x: string; scale: number; opacity: number }
const POSITIONS: Record<number, SlotPos> = {
  [-2]: { x: '-115%', scale: 0.55, opacity: 0 },
  [-1]: { x: '-55%', scale: 0.7, opacity: 0.45 },
  [0]: { x: '0%', scale: 1, opacity: 1 },
  [1]: { x: '55%', scale: 0.7, opacity: 0.45 },
  [2]: { x: '115%', scale: 0.55, opacity: 0 },
}

const ANIM_MS = 500
const ANIM_EASE = 'cubic-bezier(0.32, 0.72, 0, 1)'
const WINDOW_RADIUS = 2
const IMAGE_AUTOCHAIN_DELAY_MS = 7000

function formatHeading(item: MediaItemRow, index: number, total: number) {
  // Format court HH:MM JJ/MM/AA (cohérent avec formatMediaDate des thumbnails).
  const raw = item.capture_end_utc ?? item.match_start_time
  let dateStr: string | null = null
  if (raw) {
    const d = new Date(raw)
    if (!Number.isNaN(d.getTime())) {
      const datePart = d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: '2-digit' })
      const timePart = d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
      dateStr = `${timePart} ${datePart}`
    }
  }
  return [
    `${index + 1}/${total}`,
    item.map_name,
    dateStr,
  ].filter(Boolean).join(' · ')
}

interface ClipPlayerProps {
  filePath: string
  basename: string | null
  isCenter: boolean
  relPos: number
  videoRef: (el: HTMLVideoElement | null) => void
  onEnded: (() => void) | undefined
}

/**
 * Wrapper <video> qui gère les erreurs de chargement (MIME non supporté,
 * fichier inaccessible, codec absent). Affiche un message clair plutôt
 * qu'un cadre noir vide.
 */
function ClipPlayer({ filePath, basename, isCenter, relPos, videoRef, onEnded }: ClipPlayerProps) {
  const [error, setError] = useState<string | null>(null)

  useEffect(() => { setError(null) }, [filePath])

  if (error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center bg-card p-4 text-center text-foreground/80">
        <svg className="mb-3 h-10 w-10 text-muted-foreground/60" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
        <p className="text-sm font-medium">Lecture impossible</p>
        <p className="mt-1 text-xs text-muted-foreground">{error}</p>
        {basename && <p className="mt-2 text-xs text-muted-foreground/70">{basename}</p>}
      </div>
    )
  }

  return (
    <video
      ref={videoRef}
      src={filePath}
      controls={isCenter}
      muted={!isCenter}
      preload={Math.abs(relPos) <= 1 ? 'auto' : 'metadata'}
      onEnded={onEnded}
      onError={(e) => {
        const code = e.currentTarget.error?.code
        const msg =
          code === 4 ? 'Format vidéo non supporté par le navigateur'
          : code === 3 ? 'Erreur de décodage de la vidéo'
          : code === 2 ? 'Erreur réseau lors du chargement'
          : 'Vidéo inaccessible'
        setError(msg)
      }}
      className="h-full w-full bg-card"
    />
  )
}

export function CoverFlowModal({
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
  autoChain = false,
  onToggleAutoChain,
}: CoverFlowModalProps) {
  const locale = useAppShellStore((s) => s.locale)
  const text = getMediaModalsText(locale)
  // L'identité de l'item courant est suivie via son file_path (stable),
  // pas via un index (qui peut changer si l'array items est réordonné).
  const [currentFilePath, setCurrentFilePath] = useState<string | null>(items[startIndex]?.file_path ?? null)
  const [pendingPageAdvance, setPendingPageAdvance] = useState(false)
  const animatingRef = useRef(false)
  const videoRefs = useRef<Map<string, HTMLVideoElement>>(new Map())
  const autoChainTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Garde la dernière position valide pour éviter les sauts visuels si
  // l'item disparaît temporairement (refetch entre 2 vues).
  const lastValidIdxRef = useRef<number>(startIndex)

  // L'index courant est calculé dynamiquement depuis le file_path
  const committedIdx = useMemo(() => {
    if (!currentFilePath) {
      const fallback = Math.min(startIndex, Math.max(0, items.length - 1))
      lastValidIdxRef.current = fallback
      return fallback
    }
    const idx = items.findIndex((item) => item.file_path === currentFilePath)
    if (idx >= 0) {
      lastValidIdxRef.current = idx
      return idx
    }
    return Math.min(lastValidIdxRef.current, Math.max(0, items.length - 1))
  }, [items, currentFilePath, startIndex])

  // Quand startIndex change (nouvelle ouverture de lightbox), reset le file_path
  useEffect(() => {
    if (animatingRef.current) return
    const newFilePath = items[startIndex]?.file_path ?? null
    setCurrentFilePath(newFilePath)
  }, [startIndex])

  useEffect(() => {
    if (pendingPageAdvance && items.length > 0) {
      setCurrentFilePath(items[0]?.file_path ?? null)
      setPendingPageAdvance(false)
    }
  }, [items, pendingPageAdvance])

  const canPrev = committedIdx > 0
  const canNext = committedIdx < items.length - 1
  const canAdvanceFurther = !canNext || hasNextPage
  const currentItem = items[committedIdx] ?? null
  const isCurrentClip = currentItem?.kind === 'clip'

  const total = globalTotal ?? items.length
  const globalIndex = globalIndexOffset + committedIdx
  const heading = currentItem ? formatHeading(currentItem, globalIndex, total) : ''

  function navigate(dir: 'next' | 'prev') {
    if (animatingRef.current) return
    if (dir === 'next' && !canNext) {
      if (hasNextPage && onLoadNextPage && !pendingPageAdvance) {
        setPendingPageAdvance(true)
        onLoadNextPage()
      }
      return
    }
    if (dir === 'prev' && !canPrev) return

    const target = committedIdx + (dir === 'next' ? 1 : -1)
    const targetItem = items[target]
    if (!targetItem) return

    animatingRef.current = true
    setCurrentFilePath(targetItem.file_path)

    window.setTimeout(() => {
      animatingRef.current = false
    }, ANIM_MS)
  }

  useEffect(() => {
    videoRefs.current.forEach((vid, path) => {
      if (path === currentItem?.file_path) {
        vid.muted = false
        if (vid.paused) {
          const p = vid.play()
          if (p && typeof p.catch === 'function') p.catch(() => undefined)
        }
      } else {
        vid.muted = true
        if (!vid.paused) vid.pause()
        if (vid.currentTime > 0) vid.currentTime = 0
      }
    })
  }, [currentItem])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft') navigate('prev')
      if (e.key === 'ArrowRight') navigate('next')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, committedIdx, items.length])

  useEffect(() => {
    if (autoChainTimerRef.current) clearTimeout(autoChainTimerRef.current)
    if (!autoChain || !currentItem || isCurrentClip || !canAdvanceFurther || pendingPageAdvance) {
      return
    }
    autoChainTimerRef.current = window.setTimeout(() => navigate('next'), IMAGE_AUTOCHAIN_DELAY_MS)
    return () => {
      if (autoChainTimerRef.current) clearTimeout(autoChainTimerRef.current)
    }
  }, [autoChain, currentItem, isCurrentClip, canAdvanceFurther, pendingPageAdvance])

  const slots = useMemo(() => {
    const result: { item: MediaItemRow; relPos: number; absIdx: number }[] = []
    for (let off = -WINDOW_RADIUS; off <= WINDOW_RADIUS; off++) {
      const idx = committedIdx + off
      if (idx >= 0 && idx < items.length) {
        result.push({ item: items[idx], relPos: off, absIdx: idx })
      }
    }
    return result
  }, [items, committedIdx])

  function setVideoRef(filePath: string) {
    return (el: HTMLVideoElement | null) => {
      if (el) videoRefs.current.set(filePath, el)
      else videoRefs.current.delete(filePath)
    }
  }

  const handleVideoEnded = useCallback(() => {
    if (autoChain && canAdvanceFurther && !pendingPageAdvance) {
      navigate('next')
    }
  }, [autoChain, canAdvanceFurther, pendingPageAdvance])

  if (!currentItem) {
    return null
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/90"
      onClick={onClose}
    >
      <div
        className="relative mx-4 flex max-h-[90vh] w-full max-w-5xl flex-col rounded-xl overflow-hidden"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-border bg-card/95 px-4 py-2 text-foreground">
          <div className="flex min-w-0 items-center gap-3">
            <span className="truncate text-sm text-foreground/80">{heading}</span>
            {onReassociate && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onReassociate(currentItem)
                }}
                className="shrink-0 rounded border border-border px-2 py-0.5 text-xs text-foreground/80 hover:border-foreground/50 hover:text-foreground whitespace-nowrap"
                title={text.coverFlow.reassociateTitle}
              >
                {text.coverFlow.reassociateButton}
              </button>
            )}
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <div onClick={(e) => e.stopPropagation()}>
              <MediaLikeButton
                isLiked={currentItem.liked}
                likeCount={currentItem.like_count}
                onToggle={() => onToggleLike(currentItem)}
                disabled={likeDisabled}
              />
            </div>
            {onToggleAutoChain && (
              <button
                onClick={onToggleAutoChain}
                title={autoChain ? text.coverFlow.disableChaining : text.coverFlow.enableChaining}
                className={`flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors ${
                  autoChain
                    ? 'bg-success/20 border-success/50 text-success'
                    : 'bg-destructive/20 border-destructive/50 text-destructive'
                }`}
              >
                <svg
                  className="h-3.5 w-3.5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M13 5l7 7-7 7M5 5l7 7-7 7"
                  />
                </svg>
                {text.coverFlow.chainButton}
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="text-xl leading-none text-muted-foreground hover:text-foreground"
              aria-label={text.coverFlow.closeAriaLabel}
            >
              ✕
            </button>
          </div>
        </div>

        <div className="relative w-full flex-1 overflow-visible bg-card">
          <div className="relative aspect-video w-full overflow-visible mx-auto">
            <button
              onClick={(e) => {
                e.stopPropagation()
                navigate('prev')
              }}
              disabled={!canPrev}
              className="absolute left-3 top-1/2 z-20 -translate-y-1/2 rounded-full border border-border bg-card/90 p-2 text-foreground transition-colors hover:bg-card disabled:cursor-not-allowed disabled:opacity-30 md:left-6"
            >
              <svg
                className="h-6 w-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            {slots.map(({ item, relPos }) => {
              const pos = POSITIONS[relPos]
              const isCenter = relPos === 0
              return (
                <div
                  key={item.file_path}
                  className="absolute inset-0 overflow-hidden rounded-xl"
                  style={{
                    transform: `translateX(${pos.x}) scale(${pos.scale})`,
                    opacity: pos.opacity,
                    transition: `transform ${ANIM_MS}ms ${ANIM_EASE}, opacity ${ANIM_MS}ms ${ANIM_EASE}`,
                    pointerEvents: isCenter ? 'auto' : 'none',
                    zIndex: 10 - Math.abs(relPos),
                    willChange: 'transform, opacity',
                  }}
                  onClick={
                    isCenter
                      ? undefined
                      : (e) => {
                          e.stopPropagation()
                          if (relPos < 0) navigate('prev')
                          else navigate('next')
                        }
                  }
                >
                  {item.kind === 'clip' ? (
                    <ClipPlayer
                      filePath={item.file_path}
                      basename={item.basename}
                      isCenter={isCenter}
                      relPos={relPos}
                      videoRef={setVideoRef(item.file_path)}
                      onEnded={isCenter && autoChain && canAdvanceFurther ? handleVideoEnded : undefined}
                    />
                  ) : (
                    <img
                      src={item.file_path}
                      alt={item.basename}
                      className="h-full w-full object-contain bg-card"
                    />
                  )}
                </div>
              )
            })}

            <button
              onClick={(e) => {
                e.stopPropagation()
                navigate('next')
              }}
              disabled={!canNext && !hasNextPage}
              className="absolute right-3 top-1/2 z-20 -translate-y-1/2 rounded-full border border-border bg-card/90 p-2 text-foreground transition-colors hover:bg-card disabled:cursor-not-allowed disabled:opacity-30 md:right-6"
          >
            <svg
              className="h-6 w-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
