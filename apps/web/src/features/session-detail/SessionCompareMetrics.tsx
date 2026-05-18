/**
 * SessionCompareMetrics — vue tabulaire des métriques A vs B.
 *
 * Header = 4 DeltaCards (kd_ratio, win_rate, kills_per_match, score) + table complète
 * de toutes les métriques retournées par le backend (`compare_metrics`).
 */
import { DeltaCard } from '@/components/ui/delta-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SessionCompareMetricRow } from '@/lib/api/types'

import { parseDelta, useSessionT } from './_shared'

interface Props {
  metrics: SessionCompareMetricRow[]
}

export function SessionCompareMetrics({ metrics }: Props) {
  const t = useSessionT()
  const summaryKeys = ['kd_ratio', 'win_rate', 'kills_per_match', 'score']
  const summaryRows = summaryKeys
    .map((key) => metrics.find((row) => row.key === key))
    .filter((row): row is SessionCompareMetricRow => Boolean(row))

  return (
    <div className="space-y-4">
      {summaryRows.length > 0 ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {summaryRows.map((row) => (
            <DeltaCard
              key={row.key}
              label={row.label}
              value={row.value_a}
              delta={parseDelta(row.delta)}
              lowerIsBetter={false}
            />
          ))}
        </div>
      ) : null}

      {metrics.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[680px] text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs uppercase tracking-[0.16em] text-muted-foreground">
                <th className="px-3 py-3 font-medium">{t('session.detail.compare_col_metric')}</th>
                <th className="px-3 py-3 text-right font-medium">{t('session.detail.compare_col_session_active')}</th>
                <th className="px-3 py-3 text-right font-medium">{t('session.detail.compare_col_session_compared')}</th>
                <th className="px-3 py-3 text-right font-medium">{t('session.detail.compare_col_delta')}</th>
              </tr>
            </thead>
            <tbody>
              {metrics.map((row) => (
                <tr key={row.key} className="border-b border-border/60 last:border-0">
                  <td className="px-3 py-3 text-muted-foreground">{row.label}</td>
                  <td className="px-3 py-3 text-right font-medium text-foreground">{row.value_a}</td>
                  <td className="px-3 py-3 text-right font-medium text-compare-b">{row.value_b}</td>
                  <td className="px-3 py-3 text-right text-xs text-muted-foreground">{row.delta ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyStateNotice
          title={t('session.detail.compare_metrics_empty_title')}
          description={t('session.detail.compare_metrics_empty_description')}
        />
      )}
    </div>
  )
}
