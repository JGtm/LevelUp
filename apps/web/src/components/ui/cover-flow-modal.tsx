'use client'

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { MediaItemRow } from '@/lib/api/types'
import { MediaLikeButton } from '@/features/media/MediaViewer'
import { getMediaModalsText } from '@/features/media/i18n-modals'
import { useAppShellStore } from '@/stores/appShellStore'

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
  const dateStr = item.match_start_time || item.capture_end_utc
    ? new Date((item.match_start_time ?? item.capture_end_utc)!).toLocaleDateString('fr-FR', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : null
  return [
    `${index + 1}/${total}`,
    item.map_name,
    dateStr,
  ].filter(Boolean).join(' · ')
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
  const autoChainTimerRef = useRef<NodeJS.Timeout | null>(null)

  // L'index courant est calculé dynamiquement depuis le file_path
  const committedIdx = useMemo(() => {
    if (!currentFilePath) return startIndex
    const idx = items.findIndex((item) => item.file_path === currentFilePath)
    return idx >= 0 ? idx : startIndex
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
          void vid.play().catch(() => undefined)
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
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90"
      onClick={onClose}
    >
      <div
        className="relative mx-4 flex max-h-[90vh] w-full max-w-5xl flex-col"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between bg-black/60 px-4 py-2 text-white">
          <div className="flex min-w-0 items-center gap-3">
            <span className="truncate text-sm opacity-80">{heading}</span>
            {onReassociate && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onReassociate(currentItem)
                }}
                className="shrink-0 rounded border border-white/20 px-2 py-0.5 text-xs text-white/80 hover:border-white/50 hover:text-white whitespace-nowrap"
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
                    ? 'bg-green-500/20 border-green-500/50 text-green-400'
                    : 'bg-red-500/20 border-red-500/50 text-red-400'
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
              className="text-xl leading-none text-white/70 hover:text-white"
              aria-label={text.coverFlow.closeAriaLabel}
            >
              ✕
            </button>
          </div>
        </div>

        <div className="relative w-full flex-1 overflow-visible bg-black">
          <div className="relative aspect-video w-full overflow-visible mx-auto">
            <button
              onClick={(e) => {
                e.stopPropagation()
                navigate('prev')
              }}
              disabled={!canPrev}
              className="absolute left-3 top-1/2 z-20 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white transition-colors hover:bg-black/80 disabled:cursor-not-allowed disabled:opacity-30 md:left-6"
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
                    <video
                      ref={setVideoRef(item.file_path)}
                      src={item.file_path}
                      controls={isCenter}
                      muted={!isCenter}
                      preload={Math.abs(relPos) <= 1 ? 'auto' : 'metadata'}
                      onEnded={isCenter && autoChain && canAdvanceFurther ? handleVideoEnded : undefined}
                      className="h-full w-full bg-black"
                    />
                  ) : (
                    <img
                      src={item.file_path}
                      alt={item.basename}
                      className="h-full w-full object-contain bg-black"
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
              className="absolute right-3 top-1/2 z-20 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white transition-colors hover:bg-black/80 disabled:cursor-not-allowed disabled:opacity-30 md:right-6"
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
