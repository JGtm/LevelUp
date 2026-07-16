/**
 * HomeHighlightTile — tuile "Faits marquants" avec auto-rotation des slides.
 *
 * P8.4 (revue 2026-04-29) : extrait de HomePage.tsx (SerieTile + HighlightTile
 * + helpers de couleur). Réduit la god page de ~95L.
 */
import { useEffect, useState, type CSSProperties } from 'react'
import type { HighlightItem, HighlightSlide, HighlightValueColor } from '@/lib/api/types'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { KpiCard } from '@/components/cards/KpiCard'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { resolveTitle, resolveLabel, resolveDetail, resolveColSpan, resolveUnit } from './highlights.i18n'

// Rangée flex sur lg+ : chaque tuile grandit proportionnellement à son poids
// (flex-grow) sur une base NULLE (flex-basis: 0). Largeur finale ∝ poids →
// largeurs relatives conservées ET la ligne remplit 100 % quel que soit le
// sous-ensemble de tuiles (ex. H5 sans la case MMR/skill), sans jamais wrapper.
// Régression corrigée : une flex-basis en % (poids×5 %) sommait ~100 % (8 tuiles,
// poids total 20) puis débordait de la somme des gaps (gap-2) — la dernière tuile
// « Séries » passait alors seule sur une 2e ligne. flex-basis: 0 laisse flex-grow
// absorber les gaps, donc une seule ligne.
// Classes LITTÉRALES (pas de template dynamique) pour être détectées par le JIT.
const HIGHLIGHT_GROW_CLASS: Record<number, string> = {
  1: 'lg:grow-[1]',
  2: 'lg:grow-[2]',
  3: 'lg:grow-[3]',
  4: 'lg:grow-[4]',
  5: 'lg:grow-[5]',
}

function highlightFlexClass(titleKey: string | undefined): string {
  const weight = resolveColSpan(titleKey)
  const grow = HIGHLIGHT_GROW_CLASS[weight] ?? 'lg:grow'
  return `${grow} lg:basis-0 lg:min-w-0`
}

// Mapping unique sentiment → token sémantique. Sert à la FOIS la couleur de la
// valeur (inline style) ET l'accent dynamique de la KpiCard (barre 3px en haut).
const HIGHLIGHT_TOKEN_MAP: Record<string, SemanticToken> = {
  positive: 'divergent-pos',
  warning: 'perf-tier-2',
  negative: 'divergent-neg',
  neutral: 'perf-tier-3',
  'perf-excellent': 'perf-tier-1',
  'perf-good': 'perf-tier-2',
  'perf-ok': 'perf-tier-3',
  'perf-low': 'perf-tier-4',
  'perf-bad': 'perf-tier-5',
}

function highlightToken(color?: HighlightValueColor): SemanticToken | undefined {
  return color ? HIGHLIGHT_TOKEN_MAP[color] : undefined
}

function highlightColorStyle(color?: HighlightValueColor): CSSProperties | undefined {
  const token = highlightToken(color)
  return token ? { color: tokenCssVar(token) } : undefined
}

// Accent dynamique de la tuile (type 4 du catalogue) : suit le sentiment de la
// valeur ; neutre (perf-tier-3) à défaut.
function highlightAccent(color?: HighlightValueColor): SemanticToken {
  return highlightToken(color) ?? 'perf-tier-3'
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
    <KpiCard accent={highlightAccent(s.value_color)} className={className}>
      <div className="p-3">
      <p className="text-xs font-medium text-muted-foreground leading-tight">{title}</p>
      <div
        className={`transition-opacity duration-200 ${fading ? 'opacity-0' : 'opacity-100'}`}
        aria-live="polite"
      >
        <p className="truncate text-base font-bold" title={s.value} style={highlightColorStyle(s.value_color)}>
          {s.value}
        </p>
        <p className="line-clamp-1 text-3xs leading-tight text-muted-foreground/80" title={slideLabel}>{slideLabel}</p>
        <p className="line-clamp-1 h-[1rem] text-xs text-muted-foreground" title={slideDetail || undefined}>{slideDetail}</p>
      </div>
      {slides.length > 1 ? (
        <div className="mt-1 flex gap-1" aria-hidden="true">
          {slides.map((_, i) => (
            <span key={i} className={`h-1 w-4 rounded-full ${i === idx ? 'bg-primary' : 'bg-border'}`} />
          ))}
        </div>
      ) : null}
      </div>
    </KpiCard>
  )
}

interface HomeHighlightTileProps {
  h: HighlightItem
  locale: string | null | undefined
}

export function HomeHighlightTile({ h, locale }: HomeHighlightTileProps) {
  const title = h.title_key ? resolveTitle(locale, h.title_key) : (h.title ?? '')
  const spanClass = highlightFlexClass(h.title_key)
  if (h.slides && h.slides.length > 0) {
    return <SerieTile title={title} slides={h.slides} locale={locale} className={spanClass} />
  }
  const detail = h.detail_key
    ? resolveDetail(locale, h.detail_key, h.detail_params)
    : (h.detail ?? '')
  const unit = resolveUnit(locale, h.title_key)
  return (
    <KpiCard accent={highlightAccent(h.value_color)} className={spanClass}>
      <div className="p-3">
        <p className="text-xs font-medium text-muted-foreground">{title}</p>
        <p className="text-base font-bold" style={highlightColorStyle(h.value_color)}>
          {h.value}
          {unit ? <span className="ml-1 text-xs font-medium text-muted-foreground">{unit}</span> : null}
        </p>
        {detail ? <p className="text-xs text-muted-foreground">{detail}</p> : null}
      </div>
    </KpiCard>
  )
}
