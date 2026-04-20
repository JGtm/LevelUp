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
    if (!el) return
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
      {/* Flèche gauche + fade */}
      <div
        className={`pointer-events-none absolute inset-y-0 left-0 z-10 flex items-center transition-opacity duration-200 ${
          canLeft ? 'opacity-100' : 'opacity-0'
        }`}
      >
        <div className="h-full w-16 bg-gradient-to-r from-card via-card/60 to-transparent" />
      </div>
      <button
        type="button"
        onClick={() => scroll('left')}
        aria-label="Précédent"
        className={`absolute left-1 top-1/2 z-20 -translate-y-1/2 rounded-full border border-border bg-background/90 p-1.5 shadow-md backdrop-blur-sm transition-all duration-200 hover:bg-muted hover:scale-110 active:scale-95 ${
          canLeft ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
      >
        <ChevronLeftIcon />
      </button>

      {/* Track scrollable */}
      <div
        ref={trackRef}
        onScroll={updateScrollState}
        className="flex gap-3 overflow-x-auto scroll-smooth snap-x snap-mandatory pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {children}
      </div>

      {/* Flèche droite + fade */}
      <div
        className={`pointer-events-none absolute inset-y-0 right-0 z-10 flex items-center transition-opacity duration-200 ${
          canRight ? 'opacity-100' : 'opacity-0'
        }`}
      >
        <div className="h-full w-16 bg-gradient-to-l from-card via-card/60 to-transparent" />
      </div>
      <button
        type="button"
        onClick={() => scroll('right')}
        aria-label="Suivant"
        className={`absolute right-1 top-1/2 z-20 -translate-y-1/2 rounded-full border border-border bg-background/90 p-1.5 shadow-md backdrop-blur-sm transition-all duration-200 hover:bg-muted hover:scale-110 active:scale-95 ${
          canRight ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
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
