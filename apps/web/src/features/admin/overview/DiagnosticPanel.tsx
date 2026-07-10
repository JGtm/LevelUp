/**
 * DiagnosticPanel — verdicts interprétés (diagnostics.ts) en tête de la Vue
 * d'ensemble : le dashboard désigne les points faibles (seuils explicites)
 * avec drill-down vers l'onglet concerné. État vert explicite quand tout va
 * bien.
 */
import { Link } from '@tanstack/react-router'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useAdminDBContention } from '../queries'
import { useMonitoringScheduler, usePerfStats } from '../monitoring/queries'
import type { AdminMonitoringOverview } from '@/lib/api/types'
import { evaluateDiagnostics, type VerdictLevel } from './diagnostics'
import { useAdminT } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

const LEVEL_TOKEN: Record<VerdictLevel, SemanticToken> = {
  crit: 'destructive',
  warn: 'warning',
  info: 'info',
}

export function DiagnosticPanel({ overview }: { overview: AdminMonitoringOverview }) {
  const tA = useAdminT()
  // Sources additionnelles légères (expvar/états mémoire côté serveur).
  const scheduler = useMonitoringScheduler()
  const contention = useAdminDBContention()
  const perf = usePerfStats()

  const verdicts = evaluateDiagnostics({
    overview,
    scheduler: scheduler.data,
    contention: contention.data,
    perf: perf.data,
  })

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.diag.section')} />
      {verdicts.length === 0 ? (
        <EmptyStateNotice title={tA('admin.diag.all_green')} description={tA('admin.diag.all_green_desc')} />
      ) : (
        <div className="rounded-md border">
          {verdicts.map((v, i) => {
            const color = tokenCssVar(LEVEL_TOKEN[v.level])
            return (
              <Link
                key={`${v.titleKey}-${i}`}
                to={v.to}
                className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5 border-b px-3 py-2 text-xs last:border-b-0 hover:bg-muted/30"
              >
                <span aria-hidden className="h-2.5 w-2.5 flex-none self-center" style={{ backgroundColor: color }} />
                <span className="font-semibold uppercase tracking-wide" style={{ color }}>
                  {v.level}
                </span>
                <span className="min-w-0 flex-1 text-foreground">{tA(v.titleKey)}</span>
                <span className="font-mono tabular-nums text-muted-foreground">{v.evidence}</span>
              </Link>
            )
          })}
        </div>
      )}
    </section>
  )
}
