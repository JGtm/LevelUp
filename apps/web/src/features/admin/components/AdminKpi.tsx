/**
 * AdminKpi — LE composant KPI canonique du dashboard admin (A8.1).
 * Remplace les 5 variantes locales historiques (OverviewKpi, SummaryCell,
 * DQKpi, BacklogKpi, DataHealthMetric) par un seul wrapper au-dessus de la
 * primitive foundations KpiCard. Garde-rail : admin-kpi.guard.test.ts interdit
 * la re-déclaration locale d'un composant *Kpi sous features/admin/.
 */
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'

import { KpiCard } from '@/components/cards/KpiCard'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'

export interface AdminKpiProps {
  label: string
  value: string | number
  /** Suffixe accolé à la valeur (ex. '+' pour un backlog plafonné à l'horizon). */
  valueSuffix?: string
  sub?: string
  /** title= au survol de la valeur (ex. horodatage absolu). */
  title?: string
  /** Token d'accent KpiCard — calculé par l'appelant (type 4 du catalogue). */
  accent?: SemanticToken
  /** Delta vs visite précédente (pattern countersTrend) : négatif = amélioration. */
  delta?: number
  /** Valeur compacte (états « jamais couru » : texte long en petit). */
  compactValue?: boolean
  /** Densité : 'md' (défaut, p-4 text-2xl) ou 'sm' (p-3 text-lg). */
  size?: 'md' | 'sm'
  /** Drill-down : enveloppe la carte dans un Link. */
  to?: string
}

export function AdminKpi({
  label,
  value,
  valueSuffix = '',
  sub,
  title,
  accent,
  delta,
  compactValue,
  size = 'md',
  to,
}: AdminKpiProps) {
  const valueClass = compactValue ? 'text-sm' : size === 'sm' ? 'text-lg' : 'text-2xl'
  const card = (
    <KpiCard accent={accent} className="h-full transition-colors hover:border-primary/40">
      <div className={size === 'sm' ? 'p-3' : 'p-4'}>
        <div className="text-xs text-muted-foreground">{label}</div>
        <div
          className={`mt-1 flex items-baseline gap-2 font-semibold tabular-nums text-foreground ${valueClass}`}
          title={title || undefined}
        >
          <span>
            {value}
            {valueSuffix}
          </span>
          {delta !== undefined && (
            <span
              className="text-xs font-semibold tabular-nums"
              style={{ color: tokenCssVar(delta < 0 ? 'success' : 'destructive') }}
            >
              ({delta > 0 ? '+' : ''}
              {delta})
            </span>
          )}
        </div>
        {sub && (
          <div className="mt-0.5 truncate text-xs text-muted-foreground" title={sub}>
            {sub}
          </div>
        )}
      </div>
    </KpiCard>
  )
  return wrapInLink(card, to)
}

function wrapInLink(node: ReactNode, to?: string) {
  if (!to) return <div>{node}</div>
  return (
    <Link to={to} className="block focus-visible:outline-none">
      {node}
    </Link>
  )
}
