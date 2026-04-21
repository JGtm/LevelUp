/**
 * Carousel — défilement horizontal scroll-snap avec flèches de navigation.
 *
 * Pas de dépendance externe : scroll-snap CSS natif + ResizeObserver.
 * Compatible Tailwind v4 / shadcn design system.
 */
import { useRef, useState, useEffect, useCallback, type ReactNode } from 'react'

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
}

export function Carousel({ children, className = '' }: CarouselProps) {
  const trackRef = useRef<HTMLDivElement>(null)
  const [canLeft, setCanLeft] = useState(false)
  const [canRight, setCanRight] = useState(false)

  const updateScrollState = useCallback(() => {
    const el = trackRef.current
    if (!el) return
    setCanLeft(el.scrollLeft > 4)
    setCanRight(el.scrollLeft + el.offsetWidth < el.scrollWidth - 4)
  }, [])

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
    <div className={`relative ${className}`}>
      {/* Bouton gauche */}
      <button
        type="button"
        onClick={() => scroll('left')}
        disabled={!canLeft}
        aria-label="Précédent"
        className="absolute left-0 inset-y-0 z-20 flex items-center justify-center w-9 rounded-l-lg border-y border-l border-border bg-background/95 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-30 disabled:hover:bg-background/95"
      >
        <ChevronLeftIcon />
      </button>

      {/* Track scrollable — padding horizontal pour laisser place aux boutons */}
      <div
        ref={trackRef}
        onScroll={updateScrollState}
        className="flex gap-3 overflow-x-auto scroll-smooth snap-x snap-mandatory pb-1 px-10 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {children}
      </div>

      {/* Bouton droit */}
      <button
        type="button"
        onClick={() => scroll('right')}
        disabled={!canRight}
        aria-label="Suivant"
        className="absolute right-0 inset-y-0 z-20 flex items-center justify-center w-9 rounded-r-lg border-y border-r border-border bg-background/95 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-30 disabled:hover:bg-background/95"
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
    <div className={`shrink-0 snap-start ${className}`}>
      {children}
    </div>
  )
}
