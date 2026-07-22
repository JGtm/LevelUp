/**
 * DiagnosticsPanel — onglet "Diagnostics" de LabPage : parity report + medal guards.
 *
 * P8.4 (revue 2026-04-29) : extrait de LabPage.tsx (~115L).
 */
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { LabDiagnosticsResponse } from '@/lib/api/types'
import type { LabLocale, LabText } from './i18n'
import {
  FileStatusRow,
  GuardRow,
  MetricCard,
  StatusBadge,
} from './_labShared'
import { formatNumber } from './_labFormatters'

interface DiagnosticsPanelProps {
  data: LabDiagnosticsResponse | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  locale: LabLocale
  text: LabText
}

export function DiagnosticsPanel({
  data,
  isLoading,
  isError,
  onRetry,
  locale,
  text,
}: DiagnosticsPanelProps) {
  if (isLoading) {
    return (
      <div className="flex min-h-[320px] items-center justify-center">
        <Spinner size="lg" label={text.diagnostics.loading} />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <EmptyStateCard
        title={text.diagnostics.unavailableTitle}
        description={text.diagnostics.unavailableDescription}
        actionLabel={text.common.retry}
        onAction={onRetry}
      />
    )
  }

  const paritySummary = data.parity_report?.summary

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label={text.diagnostics.titleMetric} value={data.title_slug} hint={text.diagnostics.titleMetricHint} />
        <MetricCard label={text.diagnostics.endpointsVerified} value={formatNumber(paritySummary?.total, locale, text)} />
        <MetricCard label={text.diagnostics.passedCount} value={formatNumber(paritySummary?.passed, locale, text)} />
        <MetricCard
          label={text.diagnostics.failedCount}
          value={formatNumber(paritySummary?.failed, locale, text)}
          hint={text.diagnostics.skippedHint(formatNumber(paritySummary?.skipped, locale, text))}
        />
      </div>

      <FileStatusRow label={text.diagnostics.parityReportFile} file={data.parity_report_file} locale={locale} text={text} />

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{text.diagnostics.parityReportTitle}</CardTitle>
            <CardDescription>
              {data.parity_report
                ? `${data.parity_report.player} · ${data.parity_report.go_url}`
                : text.diagnostics.parityReportMissingDescription}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {data.parity_report ? (
              <div className="space-y-3">
                {(data.parity_report.results ?? []).map((item) => (
                  <div key={item.name} className="rounded-xl border border-border p-3">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-semibold text-foreground">{item.name}</p>
                      <StatusBadge status={item.status} text={text} />
                    </div>
                    <div className="mt-2 flex flex-wrap gap-4 text-xs text-muted-foreground">
                      <span>{text.common.http}: {item.http_status ?? text.common.notAvailable}</span>
                      <span>{text.common.mode}: {item.mode || text.common.defaultMode}</span>
                    </div>
                    {item.error ? <p className="mt-2 text-xs text-destructive">{item.error}</p> : null}
                  </div>
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title={text.diagnostics.reportAbsentTitle}
                description={text.diagnostics.reportAbsentDescription}
              />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">{text.diagnostics.medalGuardsTitle}</CardTitle>
            <CardDescription>{text.diagnostics.medalGuardsDescription}</CardDescription>
          </CardHeader>
          <CardContent>
            {data.medal_guards ? (
              <div className="space-y-3">
                <MetricCard label={text.diagnostics.entriesAnalyzed} value={formatNumber(data.medal_guards.entry_count, locale, text)} />
                <GuardRow label={text.diagnostics.cardinality} result={data.medal_guards.cardinality} text={text} />
                <GuardRow label={text.diagnostics.requiredFields} result={data.medal_guards.required_fields} text={text} />
                <GuardRow label={text.diagnostics.images} result={data.medal_guards.images} text={text} />
                <GuardRow label={text.diagnostics.overallVerdict} result={data.medal_guards.overall} text={text} />
              </div>
            ) : (
              <EmptyStateNotice
                title={text.diagnostics.noGuardsTitle}
                description={text.diagnostics.noGuardsDescription}
              />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
