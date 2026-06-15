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
  /**
   * Durée d'affichage *plancher* de chaque tip en secondes. Défaut : 6s.
   * Les tips longs restent affichés plus longtemps (cf. `readingSeconds`) pour
   * laisser le temps de lire les 3 lignes ; ce plancher borne les tips courts.
   */
  displaySeconds?: number
  /** Durée du fondu enchaîné en secondes. Défaut : 0.5s. */
  transitionSeconds?: number
  /** Label aria pour la région. */
  ariaLabel?: string
  /** Pictogramme optionnel affiché en tête de la pill. */
  leadingIcon?: ReactNode
}

/**
 * Durée de lecture estimée d'un tip, proportionnelle à sa longueur.
 * ~3 mots/s en lecture de bandeau, plancher 5s, plafond 12s — un tip de 3
 * lignes pleines ne disparaît pas avant d'être lu.
 */
function readingSeconds(text: string): number {
  const words = text.trim().split(/\s+/).length
  return Math.min(12, Math.max(5, 2 + words * 0.3))
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

    // Durée propre au tip courant : plancher `displaySeconds`, allongée pour
    // les tips longs. Chaîne de setTimeout (re-planifiée à chaque index) plutôt
    // qu'un setInterval fixe, pour que chaque tip ait sa propre durée.
    const current = tips[currentIndex % tips.length]
    const dwellMs = Math.max(displaySeconds, readingSeconds(current.shortDef)) * 1000
    const fadeMs = transitionSeconds * 1000

    let fadeTimer: ReturnType<typeof setTimeout> | undefined
    const dwellTimer = setTimeout(() => {
      setVisible(false)
      fadeTimer = setTimeout(() => {
        setCurrentIndex((i) => (i + 1) % tips.length)
        setVisible(true)
      }, fadeMs)
    }, dwellMs)

    return () => {
      clearTimeout(dwellTimer)
      if (fadeTimer) clearTimeout(fadeTimer)
    }
  }, [currentIndex, tips, displaySeconds, transitionSeconds, reducedMotion])

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
      className="relative w-full min-h-[3.5rem]"
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
  // Format « Catégorie : conseil » sur un seul flux inline (pas de saut de
  // ligne entre la catégorie et le conseil), borné à 3 lignes par line-clamp.
  const inner = (
    <span className="flex items-start gap-1.5 text-xs leading-snug">
      {leadingIcon && (
        <span className="mt-px shrink-0 text-muted-foreground" aria-hidden="true">
          {leadingIcon}
        </span>
      )}
      <span className="line-clamp-3">
        <span className="font-semibold text-foreground">{tip.term} : </span>
        <span className="text-muted-foreground">{tip.shortDef}</span>
      </span>
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
