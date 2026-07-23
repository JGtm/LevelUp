/**
 * StatCard — carte sémantique label + valeur (+ hint optionnel).
 *
 * P8.13 (revue 2026-04-29 gap #13) : consolidation des 3 implémentations
 * dispersées (l'ancienne MetricCard de features/lab/ a été retirée avec le
 * panneau Diagnostics d'instance, Lot D 2026-07-23) :
 *   - features/synthesis/SynthesisPage.tsx::StatCell (border simple + xl value)
 *   - features/home/HomeKPICard (retiré 2026-06-06 — home migré sur components/cards/KpiCard)
 *
 * 3 variants couvrent ces cas tout en partageant l'API. Le composant vit
 * dans `components/cards/` (pas dans `features/*`) pour respecter la
 * frontière (P8.5).
 *
 * Usage :
 *   <StatCard label="Win Rate" value="62%" />                     // default
 *   <StatCard label="K/D" value="1.42" variant="kpi" />           // home
 *   <StatCard label="Latence p95" value="142ms" hint="API health" variant="metric" />
 */
import { Card, CardContent } from '@/components/ui/card'

export type StatCardVariant = 'default' | 'kpi' | 'metric'

interface StatCardProps {
  label: string
  value: string | number
  hint?: string
  /**
   * Style visuel :
   * - 'default' : div simple bordée (synthesis StatCell historique)
   * - 'kpi'     : centré sur fond muted, valeur en text-primary (home KPI)
   * - 'metric'  : shadcn Card avec uppercase-tracking heading (lab MetricCard)
   */
  variant?: StatCardVariant
  /**
   * Variante compact pour les grilles denses (kpi uniquement). Réduit
   * le padding horizontal.
   */
  compact?: boolean
}

export function StatCard({
  label,
  value,
  hint,
  variant = 'default',
  compact = false,
}: StatCardProps) {
  if (variant === 'metric') {
    return (
      <Card>
        <CardContent className="p-4">
          <p className="text-xs font-semibold uppercase tracking-label text-muted-foreground">
            {label}
          </p>
          <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
          {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
        </CardContent>
      </Card>
    )
  }

  if (variant === 'kpi') {
    return (
      <div
        className={`flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted py-3 text-center ${
          compact ? 'px-2' : 'px-4'
        }`}
      >
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-xl font-bold text-primary">{value}</p>
        {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
      </div>
    )
  }

  // default : div bordé simple
  return (
    <div className="rounded-lg border p-3">
      <span className="text-xs text-muted-foreground block">{label}</span>
      <span className="text-xl font-bold">{value}</span>
      {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  )
}
