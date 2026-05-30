/**
 * TipsTicker — affiche un tip à la fois avec un fondu enchaîné.
 *
 * Cycle : fade-in → lecture → fade-out → tip suivant.
 * Respecte `prefers-reduced-motion` (liste statique scrollable).
 *
 * Usage typique : indices visuels sur une page complexe, pointant vers le
 * glossaire (`/help?tab=glossary#glossary-entry-<slug>`).
 */
import { useEffect, useState, type ReactNode } from 'react'

export interface Tip {
  id: string
  term: string
  shortDef: string
  href?: string
}

interface TipsTickerProps {
  tips: Tip[]
  /** Durée d'affichage de chaque tip en secondes. Défaut : 6s. */
  displaySeconds?: number
  /** Durée du fondu enchaîné en secondes. Défaut : 0.5s. */
  transitionSeconds?: number
  /** Label aria pour la région. */
  ariaLabel?: string
  /** Pictogramme optionnel affiché en tête de la pill. */
  leadingIcon?: ReactNode
}

function useReducedMotion(): boolean {
  const query = '(prefers-reduced-motion: reduce)'
  const getMatch = () =>
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia(query).matches
  const [reduced, setReduced] = useState(getMatch)
  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return
    const mq = window.matchMedia(query)
    const handler = (e: MediaQueryListEvent) => setReduced(e.matches)
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])
  return reduced
}

export function TipsTicker({
  tips,
  displaySeconds = 6,
  transitionSeconds = 0.5,
  ariaLabel,
  leadingIcon,
}: TipsTickerProps) {
  const [currentIndex, setCurrentIndex] = useState(0)
  const [visible, setVisible] = useState(true)
  const reducedMotion = useReducedMotion()

  useEffect(() => {
    if (reducedMotion || tips.length <= 1) return

    const cycleMs = (displaySeconds + transitionSeconds * 2) * 1000
    const timer = setInterval(() => {
      setVisible(false)
      setTimeout(() => {
        setCurrentIndex((i) => (i + 1) % tips.length)
        setVisible(true)
      }, transitionSeconds * 1000)
    }, cycleMs)

    return () => clearInterval(timer)
  }, [tips.length, displaySeconds, transitionSeconds, reducedMotion])

  if (tips.length === 0) return null

  if (reducedMotion) {
    return (
      <section
        role="region"
        aria-label={ariaLabel}
        className="flex w-full flex-col gap-0.5 py-1"
      >
        {tips.map((tip) => (
          <TipPill key={tip.id} tip={tip} leadingIcon={leadingIcon} />
        ))}
      </section>
    )
  }

  return (
    <section
      role="region"
      aria-label={ariaLabel}
      className="relative w-full min-h-[3.25rem]"
    >
      <div
        style={{
          opacity: visible ? 1 : 0,
          transition: `opacity ${transitionSeconds}s ease`,
        }}
      >
        <TipPill tip={tips[currentIndex % tips.length]} leadingIcon={leadingIcon} />
      </div>
    </section>
  )
}

interface TipPillProps {
  tip: Tip
  leadingIcon?: ReactNode
}

function TipPill({ tip, leadingIcon }: TipPillProps) {
  const inner = (
    <span className="flex flex-col gap-0.5 text-xs">
      <span className="flex items-center gap-1.5 font-semibold">
        {leadingIcon && (
          <span className="shrink-0 text-muted-foreground" aria-hidden="true">
            {leadingIcon}
          </span>
        )}
        {tip.term}
      </span>
      <span className="text-muted-foreground">{tip.shortDef}</span>
    </span>
  )

  if (tip.href) {
    return (
      <a
        href={tip.href}
        className="block hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 rounded-sm"
      >
        {inner}
      </a>
    )
  }
  return <>{inner}</>
}
