/**
 * HomeKPICard — tuile KPI compacte (label + valeur).
 *
 * P8.4 (revue 2026-04-29) : extrait de HomePage.tsx pour réduire la god page.
 */

interface HomeKPICardProps {
  label: string
  value: string | number
  compact?: boolean
}

export function HomeKPICard({ label, value, compact = false }: HomeKPICardProps) {
  return (
    <div
      className={`flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted py-3 text-center ${
        compact ? 'px-2' : 'px-4'
      }`}
    >
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-xl font-bold text-primary">{value}</p>
    </div>
  )
}
