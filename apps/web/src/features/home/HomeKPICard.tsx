/**
 * HomeKPICard — wrapper léger autour de StatCard variant="kpi".
 *
 * P8.13 (revue 2026-04-29) : la logique a été consolidée dans
 * `components/cards/StatCard.tsx` ; ce wrapper conserve le nom historique
 * pour ne pas casser les imports existants côté HomePage.
 */
import { StatCard } from '@/components/cards/StatCard'

interface HomeKPICardProps {
  label: string
  value: string | number
  compact?: boolean
}

export function HomeKPICard({ label, value, compact = false }: HomeKPICardProps) {
  return <StatCard label={label} value={value} variant="kpi" compact={compact} />
}
