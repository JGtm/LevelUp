/**
 * ContractsPanel — onglet "Contracts" de LabPage : OpenAPI vs FastAPI parity.
 *
 * P8.4 (revue 2026-04-29) : extrait de LabPage.tsx (~100L).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { LabContractsResponse } from '@/lib/api/types'
import type { LabLocale, LabText } from './i18n'
import {
  FileStatusRow,
  formatNumber,
  MetricCard,
  RouteList,
  StatusBadge,
  translateStatus,
} from './_labShared'

interface ContractsPanelProps {
  data: LabContractsResponse | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  locale: LabLocale
  text: LabText
}

export function ContractsPanel({ data, isLoading, isError, onRetry, locale, text }: ContractsPanelProps) {
  if (isLoading) {
    return (
      <div className="flex min-h-[320px] items-center justify-center">
        <Spinner size="lg" label={text.contracts.loading} />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <EmptyStateCard
        title={text.contracts.unavailableTitle}
        description={text.contracts.unavailableDescription}
        actionLabel={text.common.retry}
        onAction={onRetry}
      />
    )
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        <MetricCard label={text.contracts.summaryStatus} value={translateStatus(data.summary.status, text)} hint={text.contracts.summaryHint} />
        <MetricCard label={text.contracts.fastapiRoutes} value={formatNumber(data.summary.fastapi_route_count, locale, text)} />
        <MetricCard label={text.contracts.goRoutes} value={formatNumber(data.summary.go_route_count, locale, text)} />
        <MetricCard label={text.contracts.missingInGo} value={formatNumber(data.summary.missing_in_go, locale, text)} />
        <MetricCard label={text.contracts.methodMismatches} value={formatNumber(data.summary.method_mismatches, locale, text)} />
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <FileStatusRow label={text.contracts.goSpec} file={data.go_openapi} locale={locale} text={text} />
        <FileStatusRow label={text.contracts.fastapiReference} file={data.fastapi_reference} locale={locale} text={text} />
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <RouteList
          title={text.contracts.missingRoutesTitle}
          items={data.missing_in_go ?? []}
          emptyTitle={text.contracts.missingRoutesEmptyTitle}
          emptyDescription={text.contracts.missingRoutesEmptyDescription}
        />
        <RouteList
          title={text.contracts.extraRoutesTitle}
          items={data.extra_in_go ?? []}
          emptyTitle={text.contracts.extraRoutesEmptyTitle}
          emptyDescription={text.contracts.extraRoutesEmptyDescription}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{text.contracts.mismatchTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          {(data.method_mismatches ?? []).length === 0 ? (
            <EmptyStateNotice
              title={text.contracts.mismatchEmptyTitle}
              description={text.contracts.mismatchEmptyDescription}
            />
          ) : (
            <div className="space-y-3">
              {(data.method_mismatches ?? []).map((item) => (
                <div key={item.fastapi_path} className="rounded-xl border border-border p-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <StatusBadge status="DIVERGENCES" text={text} />
                    <p className="text-sm font-semibold text-foreground">{item.fastapi_path}</p>
                  </div>
                  <div className="mt-3 grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
                    <div>
                      <p className="font-medium text-foreground">{text.contracts.fastapiLabel}</p>
                      <p>{(item.fastapi_methods ?? []).join(', ')}</p>
                    </div>
                    <div>
                      <p className="font-medium text-foreground">{text.contracts.goLabel}</p>
                      <p>{(item.go_methods ?? []).join(', ')}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
