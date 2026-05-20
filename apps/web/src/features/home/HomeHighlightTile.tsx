/**
 * HomeHighlightTile — tuile "Faits marquants" avec auto-rotation des slides.
 *
 * P8.4 (revue 2026-04-29) : extrait de HomePage.tsx (SerieTile + HighlightTile
 * + helpers de couleur). Réduit la god page de ~95L.
 */
import { useEffect, useState, type CSSProperties } from 'react'
import type { HighlightItem, HighlightSlide, HighlightValueColor } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { resolveTitle, resolveLabel, resolveDetail, resolveColSpan, resolveUnit } from './highlights.i18n'

// Grille fine de 20 sous-unités sur lg+. Tailwind v4 arbitrary values.
const HIGHLIGHT_SPAN_CLASS: Record<number, string> = {
  1: 'lg:[grid-column:span_1/span_1]',
  2: 'lg:[grid-column:span_2/span_2]',
  3: 'lg:[grid-column:span_3/span_3]',
  4: 'lg:[grid-column:span_4/span_4]',
  5: 'lg:[grid-column:span_5/span_5]',
}

const HIGHLIGHT_COLOR_MAP: Record<string, string> = {
  positive: tokenCssVar('divergent-pos'),
  warning: tokenCssVar('perf-tier-2'),
  negative: tokenCssVar('divergent-neg'),
  neutral: tokenCssVar('perf-tier-3'),
  'perf-excellent': tokenCssVar('perf-tier-1'),
  'perf-good': tokenCssVar('perf-tier-2'),
  'perf-ok': tokenCssVar('perf-tier-3'),
  'perf-low': tokenCssVar('perf-tier-4'),
  'perf-bad': tokenCssVar('perf-tier-5'),
}

function highlightColorStyle(color?: HighlightValueColor): CSSProperties | undefined {
  if (!color) return undefined
  const cssVar = HIGHLIGHT_COLOR_MAP[color]
  return cssVar ? { color: cssVar } : undefined
}

interface SerieTileProps {
  title: string
  slides: HighlightSlide[]
  locale: string | null | undefined
  className?: string
}

function SerieTile({ title, slides, locale, className }: SerieTileProps) {
  const [idx, setIdx] = useState(0)
  const [fading, setFading] = useState(false)
  const { data: fieldMappings } = useFieldMappings()
  useEffect(() => {
    if (slides.length <= 1) return
    const iv = window.setInterval(() => {
      setFading(true)
      window.setTimeout(() => {
        setIdx((i) => (i + 1) % slides.length)
        setFading(false)
      }, 250)
    }, 4000)
    return () => window.clearInterval(iv)
  }, [slides.length])
  const s = slides[idx]
  const slideLabel = s.label_key ? resolveLabel(locale, s.label_key, fieldMappings) : (s.label ?? '')
  const slideDetail = s.detail_key
    ? resolveDetail(locale, s.detail_key, s.detail_params)
    : (s.detail ?? '')
  return (
    <div className={`rounded-md border border-border p-3 ${className ?? ''}`}>
      <p className="text-xs font-medium text-muted-foreground leading-tight">{title}</p>
      <div
        className={`transition-opacity duration-200 ${fading ? 'opacity-0' : 'opacity-100'}`}
        aria-live="polite"
      >
        <p className="text-base font-bold" style={highlightColorStyle(s.value_color)}>
          {s.value}
        </p>
        <p className="text-3xs text-muted-foreground/80 leading-tight">{slideLabel}</p>
        {slideDetail ? <p className="text-xs text-muted-foreground">{slideDetail}</p> : null}
      </div>
      {slides.length > 1 ? (
        <div className="mt-1 flex gap-1" aria-hidden="true">
          {slides.map((_, i) => (
            <span key={i} className={`h-1 w-4 rounded-full ${i === idx ? 'bg-primary' : 'bg-border'}`} />
          ))}
        </div>
      ) : null}
    </div>
  )
}

interface HomeHighlightTileProps {
  h: HighlightItem
  locale: string | null | undefined
}

export function HomeHighlightTile({ h, locale }: HomeHighlightTileProps) {
  const title = h.title_key ? resolveTitle(locale, h.title_key) : (h.title ?? '')
  const spanClass = HIGHLIGHT_SPAN_CLASS[resolveColSpan(h.title_key)] ?? ''
  if (h.slides && h.slides.length > 0) {
    return <SerieTile title={title} slides={h.slides} locale={locale} className={spanClass} />
  }
  const detail = h.detail_key
    ? resolveDetail(locale, h.detail_key, h.detail_params)
    : (h.detail ?? '')
  const unit = resolveUnit(locale, h.title_key)
  return (
    <div className={`rounded-md border border-border p-3 ${spanClass}`}>
      <p className="text-xs font-medium text-muted-foreground">{title}</p>
      <p className="text-base font-bold" style={highlightColorStyle(h.value_color)}>
        {h.value}
        {unit ? <span className="ml-1 text-xs font-medium text-muted-foreground">{unit}</span> : null}
      </p>
      {detail ? <p className="text-xs text-muted-foreground">{detail}</p> : null}
    </div>
  )
}
