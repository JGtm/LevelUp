/**
 * KPIStrip — bandeau transverse de KPIs personnels.
 *
 * Conformément au PLAN_SQUAD_GO_PORTAGE § 1.1 P2 : 8 cartes (Matchs, Durée
 * totale, K/match, D/match, A/match, Précision, Vie moyenne, W/L/T/DNF) avec
 * flèches de tendance ▲▼= optionnelles vs all-time.
 *
 * i18n-naïf : tous les labels et formats sont déjà localisés en amont par le
 * consommateur (qui appelle formatMessage(commonManifest, key, locale, vars?)).
 *
 * Réutilisable Squad / MatchView / Career.
 */
import type { ReactNode } from 'react'

export type KPITrend = 'above' | 'below' | 'near' | 'none'

export interface KPICardData {
  /** Identifiant stable (utilisé comme key React + data-testid). */
  id: string
  /** Label déjà localisé. */
  label: string
  /** Valeur principale formatée (ex. "12,5", "50:30", "47 %"). */
  primary: string
  /** Sous-titre optionnel (ex. "/min", "/match"). */
  secondary?: string
  /** Tendance vs référence (par défaut "none" = pas de flèche). */
  trend?: KPITrend
  /** Slot enfant pour visualisation custom (barre W/L/T/DNF par exemple). */
  custom?: ReactNode
  /** Si true, la carte prend la largeur double (utile pour la barre W/L). */
  wide?: boolean
}

export interface KPIStripProps {
  cards: KPICardData[]
  className?: string
}

/**
 * KPIStrip rend une grille horizontale responsive de cartes KPI.
 * Largeur : 4 cartes par ligne sur desktop, 2 sur tablet, 1 sur mobile.
 */
export function KPIStrip({ cards, className = '' }: KPIStripProps) {
  return (
    <div
      className={`grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4 ${className}`}
      data-testid="kpi-strip"
    >
      {cards.map((card) => (
        <KPICard key={card.id} card={card} />
      ))}
    </div>
  )
}

function KPICard({ card }: { card: KPICardData }) {
  const wideClass = card.wide ? 'lg:col-span-2' : ''
  return (
    <div
      className={`relative flex flex-col gap-1 rounded-lg border border-border bg-card px-3 py-2 ${wideClass}`}
      data-testid="kpi-card"
      data-id={card.id}
    >
      <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {card.label}
      </div>
      <div className="flex items-baseline gap-2">
        <span className="text-xl font-semibold leading-none" data-testid="kpi-primary">
          {card.primary}
        </span>
        {card.secondary && (
          <span className="text-xs text-muted-foreground" data-testid="kpi-secondary">
            {card.secondary}
          </span>
        )}
        {card.trend && card.trend !== 'none' && (
          <KPITrendArrow trend={card.trend} />
        )}
      </div>
      {card.custom && (
        <div className="mt-1" data-testid="kpi-custom">
          {card.custom}
        </div>
      )}
    </div>
  )
}

const TREND_GLYPH: Record<Exclude<KPITrend, 'none'>, string> = {
  above: '▲',
  below: '▼',
  near: '=',
}

const TREND_VAR: Record<Exclude<KPITrend, 'none'>, string> = {
  above: '--narrative-trend-positive',
  below: '--narrative-trend-negative',
  near: '--narrative-trend-neutral',
}

function KPITrendArrow({ trend }: { trend: Exclude<KPITrend, 'none'> }) {
  return (
    <span
      className="ml-auto text-xs font-bold leading-none"
      style={{ color: `var(${TREND_VAR[trend]})` }}
      data-testid="kpi-trend"
      data-trend={trend}
      aria-hidden="true"
    >
      {TREND_GLYPH[trend]}
    </span>
  )
}
