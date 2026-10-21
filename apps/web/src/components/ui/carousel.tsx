/**
 * Carousel — défilement horizontal scroll-snap avec flèches de navigation.
 *
 * Pas de dépendance externe : scroll-snap CSS natif + ResizeObserver.
 * Compatible Tailwind v4 / shadcn design system.
 */
import { useRef, useState, useEffect, useCallback, type ReactNode } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { CAROUSEL_NEAR_END_TILES, isNearEnd, itemSpanOf } from './carousel.logic'

function ChevronLeftIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="m15 18-6-6 6-6" />
    </svg>
  )
}

function ChevronRightIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

interface CarouselProps {
  children: ReactNode
  className?: string
  /** Appelé (sans arguments) quand le défilement arrive à `nearEndTiles` tuiles de la fin et que
   *  `loadingMore` est faux : le parent charge la suite. Idempotent côté parent (appelé à chaque
   *  rendu tant que la fin est proche). Le carrousel ne monte que les enfants reçus. */
  onNearEnd?: () => void
  /** Nombre de tuiles restantes à partir duquel `onNearEnd` est appelé. */
  nearEndTiles?: number
  /** La suite est en cours de chargement : un indicateur ferme la piste. */
  loadingMore?: boolean
}

export function Carousel({
  children,
  className = '',
  onNearEnd,
  nearEndTiles = CAROUSEL_NEAR_END_TILES,
  loadingMore = false,
}: CarouselProps) {
  const locale = useAppShellStore((s) => s.locale)
  const prevLabel = formatMessage(commonManifest, 'common.carousel.prev', locale)
  const nextLabel = formatMessage(commonManifest, 'common.carousel.next', locale)
  const loadingLabel = formatMessage(commonManifest, 'common.carousel.loading_more', locale)
  const trackRef = useRef<HTMLDivElement>(null)
  const onNearEndRef = useRef(onNearEnd)
  useEffect(() => {
    onNearEndRef.current = onNearEnd
  })
  const [canLeft, setCanLeft] = useState(false)
  const [canRight, setCanRight] = useState(false)

  const updateScrollState = useCallback(() => {
    const el = trackRef.current
    if (!el) return
    setCanLeft(el.scrollLeft > 4)
    setCanRight(el.scrollLeft + el.offsetWidth < el.scrollWidth - 4)
    if (
      onNearEndRef.current &&
      !loadingMore &&
      isNearEnd(
        { scrollLeft: el.scrollLeft, clientWidth: el.clientWidth, scrollWidth: el.scrollWidth, itemSpan: itemSpanOf(el) },
        nearEndTiles,
      )
    ) {
      onNearEndRef.current()
    }
  }, [loadingMore, nearEndTiles])

  useEffect(() => {
    updateScrollState()
    const el = trackRef.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(updateScrollState)
    ro.observe(el)
    return () => ro.disconnect()
  }, [updateScrollState])

  // Re-check quand les enfants changent (données chargées après mount)
  useEffect(() => { updateScrollState() })

  function scroll(direction: 'left' | 'right') {
    const el = trackRef.current
    if (!el) return
    const step = el.offsetWidth * 0.75
    el.scrollBy({ left: direction === 'right' ? step : -step, behavior: 'smooth' })
  }

  return (
    <div className={`flex items-stretch gap-0 ${className}`}>
      {/* Bouton gauche — dans le flux, ne chevauche pas les tuiles */}
      <button
        type="button"
        onClick={() => scroll('left')}
        disabled={!canLeft}
        aria-label={prevLabel}
        className="shrink-0 flex items-center justify-center w-9 rounded-l-lg border-y border-l border-border bg-background/95 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-30 disabled:hover:bg-background/95"
      >
        <ChevronLeftIcon />
      </button>

      {/* Track scrollable */}
      <div
        ref={trackRef}
        onScroll={updateScrollState}
        className="flex-1 flex gap-3 overflow-x-auto scroll-smooth snap-x snap-mandatory pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {children}
        {loadingMore && (
          <div
            role="status"
            className="flex w-24 shrink-0 items-center justify-center text-xs text-muted-foreground"
          >
            {loadingLabel}
          </div>
        )}
      </div>

      {/* Bouton droit — dans le flux, ne chevauche pas les tuiles */}
      <button
        type="button"
        onClick={() => scroll('right')}
        disabled={!canRight}
        aria-label={nextLabel}
        className="shrink-0 flex items-center justify-center w-9 rounded-r-lg border-y border-r border-border bg-background/95 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-30 disabled:hover:bg-background/95"
      >
        <ChevronRightIcon />
      </button>
    </div>
  )
}

interface CarouselItemProps {
  children: ReactNode
  className?: string
}

export function CarouselItem({ children, className = '' }: CarouselItemProps) {
  return (
    <div className={`shrink-0 snap-start flex flex-col h-full ${className}`}>
      {children}
    </div>
  )
}
