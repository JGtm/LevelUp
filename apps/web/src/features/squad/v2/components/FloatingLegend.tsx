/**
 * FloatingLegend — légende sticky des joueurs du squad (chunk S12).
 *
 * Affichée en haut de la page Squad V2 et reste sticky pendant le scroll
 * via `position: sticky` + `top-0`. Optionnellement, peut être contrôlée
 * via `IntersectionObserver` pour disparaître quand l'utilisateur quitte la
 * zone des charts (cf. audit § 22 : sentinelles `#llp-squad-start`).
 *
 * Conventions :
 *   - 1 pastille couleur par joueur, ordre stable (main puis coéquipiers)
 *   - Couleurs résolues via `seriesColor(idx)` (chart-series-1..8 cyclées)
 *   - Aucune string hardcodée : labels passés en prop ou résolus par caller
 */
import { useEffect, useRef, useState } from 'react'

import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility'

const SERIES_TOKENS: SemanticToken[] = [
  'chart-series-1',
  'chart-series-2',
  'chart-series-3',
  'chart-series-4',
  'chart-series-5',
  'chart-series-6',
  'chart-series-7',
  'chart-series-8',
]

export interface FloatingLegendProps {
  /** Ordre stable des gamertags (main puis coequipiers). */
  squadOrder: string[]
  /**
   * ID DOM de la sentinelle de fin de zone surveillee. Si fourni, la legende
   * disparait quand cette sentinelle entre dans le viewport (l'utilisateur a
   * scrolle au-dela des sections concernees). Default : non observe.
   */
  endSentinelId?: string
}

export function FloatingLegend({ squadOrder, endSentinelId }: FloatingLegendProps) {
  const [visible, setVisible] = useState(true)
  const observerRef = useRef<IntersectionObserver | null>(null)

  useEffect(() => {
    if (!endSentinelId) return
    if (typeof IntersectionObserver === 'undefined') return // jsdom safe
    const el = document.getElementById(endSentinelId)
    if (!el) return

    observerRef.current = new IntersectionObserver(
      (entries) => {
        // La sentinelle entre dans le viewport -> on cache la legende.
        // Elle ressort -> on la remontre.
        for (const entry of entries) {
          setVisible(!entry.isIntersecting)
        }
      },
      { rootMargin: '0px' },
    )
    observerRef.current.observe(el)
    return () => {
      observerRef.current?.disconnect()
    }
  }, [endSentinelId])

  if (squadOrder.length === 0) {
    return null
  }
  if (!visible) {
    return null
  }
  return (
    <div
      className="sticky top-0 z-10 flex flex-wrap items-center gap-3 rounded-md border border-border bg-card/90 px-3 py-2 backdrop-blur"
      data-testid="floating-legend"
    >
      {squadOrder.map((gt, idx) => (
        <div key={gt} className="flex items-center gap-2 text-sm">
          <span
            className="inline-block h-2.5 w-2.5 rounded-full"
            style={{
              backgroundColor: tokenCssVar(SERIES_TOKENS[idx % SERIES_TOKENS.length]),
            }}
            data-testid={`floating-legend-dot-${idx}`}
          />
          <span className="font-medium">{gt}</span>
        </div>
      ))}
    </div>
  )
}
