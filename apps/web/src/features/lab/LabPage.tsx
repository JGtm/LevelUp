import { startTransition, useDeferredValue, useEffect, useMemo, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PageLoader } from '@/components/ui/spinner'
import type {
  LabAssetSummary,
  LabContractsResponse,
  LabDiagnosticsResponse,
  LabFileStatus,
  LabGuardResult,
  LabMedalSummary,
  LabResourcesResponse,
  LabRouteMethods,
} from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  LabIntroNotice,
  LabSelectedToolNotice,
} from './LabHelp'
import { getLabText, normalizeLabLocale, type LabLocale, type LabTab, type LabText } from './i18n'
import { useLabContracts, useLabDiagnostics, useLabResources } from './queries'

const TAB_VALUES: LabTab[] = ['resources', 'contracts', 'diagnostics']

const RESOURCE_LIMIT = 12

function getStatusVariant(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === 'ok' || normalized === 'passed' || normalized === 'present') {
    return 'success' as const
  }
  if (
    normalized === 'failed'
    || normalized === 'ko'
    || normalized === 'divergence'
    || normalized === 'divergences'
    || normalized === 'absent'
  ) {
    return 'destructive' as const
  }
  return 'secondary' as const
}

function translateStatus(status: string, text: LabText) {
  switch (status.toLowerCase()) {
    case 'ok':
      return text.common.statuses.ok
    case 'ko':
      return text.common.statuses.ko
    case 'passed':
      return text.common.statuses.passed
    case 'failed':
      return text.common.statuses.failed
    case 'skipped':
      return text.common.statuses.skipped
    case 'divergence':
    case 'divergences':
      return text.common.statuses.divergence
    default:
      return status
  }
}

function formatDate(value: string | null | undefined, locale: LabLocale, text: LabText) {
  if (!value) {
    return text.common.notAvailable
  }
  return new Date(value).toLocaleString(getLabText(locale).intlLocale)
}

function formatNumber(value: number | null | undefined, locale: LabLocale, text: LabText) {
  if (value == null) {
    return text.common.notAvailable
  }
  return value.toLocaleString(getLabText(locale).intlLocale)
}

function formatDecimal(value: number, locale: LabLocale) {
  return value.toLocaleString(getLabText(locale).intlLocale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })
}

function formatBytes(value: number | null | undefined, locale: LabLocale, text: LabText) {
  if (value == null) {
    return text.common.notAvailable
  }
  if (value < 1024) {
    return `${value} B`
  }
  if (value < 1024 * 1024) {
    return `${formatDecimal(value / 1024, locale)} KB`
  }
  return `${formatDecimal(value / (1024 * 1024), locale)} MB`
}

function StatusBadge({ status, text }: { status: string; text: LabText }) {
  return <Badge variant={getStatusVariant(status)}>{translateStatus(status, text)}</Badge>
}

function MetricCard({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
        {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
      </CardContent>
    </Card>
  )
}

function JsonViewer({
  title,
  content,
  text,
}: {
  title: string
  content?: string | null
  text: LabText
}) {
  if (!content) {
    return (
      <EmptyStateNotice
        title={text.common.payloadUnavailableTitle(title)}
        description={text.common.payloadUnavailableDescription}
      />
    )
  }

  return (
    <div className="space-y-2">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{title}</p>
      <pre className="max-h-[420px] overflow-auto rounded-xl bg-card p-4 text-xs leading-6 text-muted-foreground">
        {content}
      </pre>
    </div>
  )
}

function FileStatusRow({
  label,
  file,
  locale,
  text,
}: {
  label: string
  file: LabFileStatus
  locale: LabLocale
  text: LabText
}) {
  return (
    <div className="rounded-xl border border-border bg-muted p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">{label}</p>
          <p className="mt-1 break-all text-xs text-muted-foreground">{file.path}</p>
        </div>
        <Badge variant={file.exists ? 'success' : 'destructive'}>
          {file.exists ? text.common.present : text.common.absent}
        </Badge>
      </div>
      <div className="mt-3 flex flex-wrap gap-4 text-xs text-muted-foreground">
        <span>{text.common.size}: {formatBytes(file.size_bytes, locale, text)}</span>
        <span>{text.common.modified}: {formatDate(file.modified_at ?? null, locale, text)}</span>
      </div>
    </div>
  )
}

function RouteList({
  title,
  items,
  emptyTitle,
  emptyDescription,
}: {
  title: string
  items: LabRouteMethods[]
  emptyTitle: string
  emptyDescription: string
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyStateNotice title={emptyTitle} description={emptyDescription} />
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <div key={item.path} className="rounded-xl border border-border p-3">
                <div className="flex flex-wrap items-center gap-2">
                  {item.methods.map((method) => (
                    <Badge key={`${item.path}-${method}`} variant="outline">
                      {method.toUpperCase()}
                    </Badge>
                  ))}
                </div>
                <p className="mt-2 break-all text-sm text-foreground">{item.path}</p>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function GuardRow({
  label,
  result,
  text,
}: {
  label: string
  result: LabGuardResult
  text: LabText
}) {
  return (
    <div className="rounded-xl border border-border p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-foreground">{label}</p>
        <Badge variant={result.passed ? 'success' : 'destructive'}>
          {result.passed ? text.common.statuses.ok : text.common.statuses.ko}
        </Badge>
      </div>
      <p className="mt-2 text-sm text-muted-foreground">{result.reason}</p>
      {result.details.length > 0 ? (
        <div className="mt-2 space-y-1 text-xs text-muted-foreground">
          {result.details.slice(0, 5).map((detail) => (
            <p key={detail}>{detail}</p>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function TabButton({
  active,
  label,
  onClick,
}: {
  active: boolean
  label: string
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={[
        'rounded-full px-4 py-2 text-sm font-medium transition-colors',
        active
          ? 'bg-primary text-primary-foreground'
          : 'bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
      ].join(' ')}
    >
      {label}
    </button>
  )
}

function SelectableAssetList({
  items,
  selectedID,
  onSelect,
  text,
}: {
  items: LabAssetSummary[]
  selectedID: string
  onSelect: (assetID: string) => void
  text: LabText
}) {
  if (items.length === 0) {
    return (
      <EmptyStateNotice
        title={text.resources.noAssetsTitle}
        description={text.resources.noAssetsDescription}
      />
    )
  }

  return (
    <div className="space-y-2">
      {items.map((item) => {
        const active = item.asset_id === selectedID
        return (
          <button
            key={item.asset_id}
            onClick={() => onSelect(item.asset_id)}
            className={[
              'w-full rounded-xl border p-3 text-left transition-colors',
              active
                ? 'border-primary/30 bg-primary/10'
                : 'border-border hover:border-border hover:bg-muted',
            ].join(' ')}
          >
            <div className="flex items-center justify-between gap-3">
              <p className="text-sm font-semibold text-foreground">{item.name || item.asset_id}</p>
              <Badge variant="outline">{item.asset_type}</Badge>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">{item.asset_id}</p>
            <p className="mt-1 text-xs text-muted-foreground">{text.common.version}: {item.version_id}</p>
          </button>
        )
      })}
    </div>
  )
}

function SelectableMedalList({
  items,
  selectedID,
  onSelect,
  text,
}: {
  items: LabMedalSummary[]
  selectedID: number | null
  onSelect: (medalID: number) => void
  text: LabText
}) {
  if (items.length === 0) {
    return (
      <EmptyStateNotice
        title={text.resources.noMedalsTitle}
        description={text.resources.noMedalsDescription}
      />
    )
  }

  return (
    <div className="space-y-2">
      {items.map((item) => {
        const active = item.medal_id === selectedID
        return (
          <button
            key={item.medal_id}
            onClick={() => onSelect(item.medal_id)}
            className={[
              'w-full rounded-xl border p-3 text-left transition-colors',
              active
                ? 'border-primary/30 bg-primary/10'
                : 'border-border hover:border-border hover:bg-muted',
            ].join(' ')}
          >
            <div className="flex items-center justify-between gap-3">
              <p className="text-sm font-semibold text-foreground">{item.name_id || `${text.common.id} ${item.medal_id}`}</p>
              <Badge variant="outline">{item.medal_type || text.common.rawValue}</Badge>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">{text.common.id}: {item.medal_id}</p>
            <p className="mt-1 text-xs text-muted-foreground">{text.common.sprite}: {item.sprite_index}</p>
          </button>
        )
      })}
    </div>
  )
}

function ResourcesPanel({
  data,
  isLoading,
  isError,
  onRetry,
  locale,
  text,
  assetSearch,
  setAssetSearch,
  medalSearch,
  setMedalSearch,
  selectedSnapshotKey,
  setSelectedSnapshotKey,
  selectedAssetID,
  setSelectedAssetID,
  selectedMedalID,
  setSelectedMedalID,
}: {
  data: LabResourcesResponse | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  locale: LabLocale
  text: LabText
  assetSearch: string
  setAssetSearch: (value: string) => void
  medalSearch: string
  setMedalSearch: (value: string) => void
  selectedSnapshotKey: string
  setSelectedSnapshotKey: (value: string) => void
  selectedAssetID: string
  setSelectedAssetID: (value: string) => void
  selectedMedalID: number | null
  setSelectedMedalID: (value: number) => void
}) {
  if (isLoading) {
    return (
      <PageLoader label={text.resources.loading} />
    )
  }

  if (isError || !data) {
    return (
      <EmptyStateCard
        title={text.resources.unavailableTitle}
        description={text.resources.unavailableDescription}
        actionLabel={text.common.retry}
        onAction={onRetry}
      />
    )
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label={text.resources.currentTitle} value={data.title_slug} hint={text.resources.currentTitleHint} />
        <MetricCard
          label={text.resources.activeSeason}
          value={data.current_season?.name ?? text.resources.activeSeasonFallback}
          hint={text.resources.snapshotsHint(data.snapshots.length)}
        />
        <MetricCard
          label={text.resources.localAssets}
          value={formatNumber(data.assets.total, locale, text)}
          hint={data.assets.search ? `${text.common.filterPrefix}: ${data.assets.search}` : 'waypoint_assets_raw'}
        />
        <MetricCard
          label={text.resources.localMedals}
          value={formatNumber(data.medals.total, locale, text)}
          hint={data.medals.search ? `${text.common.filterPrefix}: ${data.medals.search}` : 'waypoint_medals_raw'}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{text.resources.metadataBaseTitle}</CardTitle>
          <CardDescription>{data.metadata_db_path}</CardDescription>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{text.resources.snapshotsTitle}</CardTitle>
          <CardDescription>{text.resources.snapshotsDescription}</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-[minmax(0,320px)_minmax(0,1fr)]">
          <div className="space-y-2">
            {data.snapshots.length === 0 ? (
              <EmptyStateNotice
                title={text.resources.noSnapshotsTitle}
                description={text.resources.noSnapshotsDescription}
              />
            ) : (
              data.snapshots.map((snapshot) => (
                <button
                  key={`${snapshot.resource_key}-${snapshot.version}`}
                  onClick={() => setSelectedSnapshotKey(snapshot.resource_key)}
                  className={[
                    'w-full rounded-xl border p-3 text-left transition-colors',
                    snapshot.resource_key === selectedSnapshotKey
                      ? 'border-primary/30 bg-primary/10'
                      : 'border-border hover:border-border hover:bg-muted',
                  ].join(' ')}
                >
                  <p className="text-sm font-semibold text-foreground">{snapshot.resource_key}</p>
                  <p className="mt-1 text-xs text-muted-foreground">{text.common.version}: {snapshot.version}</p>
                  <p className="mt-1 text-xs text-muted-foreground">{text.resources.snapshotPayloadTitle}: {formatBytes(snapshot.payload_size, locale, text)}</p>
                </button>
              ))
            )}
          </div>

          <div className="space-y-4">
            {data.selected_snapshot ? (
              <Card className="border-border">
                <CardContent className="space-y-4 p-4">
                  <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                    <p>{text.common.version}: <span className="font-medium text-foreground">{data.selected_snapshot.version}</span></p>
                    <p>{text.common.fetchedAt}: <span className="font-medium text-foreground">{formatDate(data.selected_snapshot.fetched_at, locale, text)}</span></p>
                    <p className="break-all">{text.common.hash}: <span className="font-medium text-foreground">{data.selected_snapshot.content_hash}</span></p>
                    <p className="break-all">{text.common.source}: <span className="font-medium text-foreground">{data.selected_snapshot.source_url || text.common.notAvailable}</span></p>
                  </div>
                  <JsonViewer title={text.resources.snapshotPayloadTitle} content={data.selected_snapshot.payload} text={text} />
                </CardContent>
              </Card>
            ) : (
              <EmptyStateNotice
                title={text.resources.selectSnapshotTitle}
                description={text.resources.selectSnapshotDescription}
              />
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle className="text-base">{text.resources.assetsTitle}</CardTitle>
              <CardDescription>{text.resources.assetsDescription}</CardDescription>
            </div>
            <input
              value={assetSearch}
              onChange={(event) => setAssetSearch(event.target.value)}
              placeholder={text.resources.assetsPlaceholder}
              className="w-full rounded-xl border border-input px-3 py-2 text-sm md:max-w-sm"
            />
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-[minmax(0,320px)_minmax(0,1fr)]">
          <SelectableAssetList items={data.assets.items} selectedID={selectedAssetID} onSelect={setSelectedAssetID} text={text} />
          <div className="space-y-4">
            {data.assets.selected ? (
              <Card className="border-border">
                <CardContent className="space-y-4 p-4">
                  <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                    <p>{text.common.asset}: <span className="font-medium text-foreground">{data.assets.selected.asset_id}</span></p>
                    <p>{text.common.type}: <span className="font-medium text-foreground">{data.assets.selected.asset_type}</span></p>
                    <p>{text.common.version}: <span className="font-medium text-foreground">{data.assets.selected.version_id}</span></p>
                    <p>{text.common.fetchedAt}: <span className="font-medium text-foreground">{formatDate(data.assets.selected.fetched_at, locale, text)}</span></p>
                  </div>
                  <JsonViewer title={text.resources.rawAssetTitle} content={data.assets.selected.raw_json} text={text} />
                </CardContent>
              </Card>
            ) : (
              <EmptyStateNotice
                title={text.resources.selectAssetTitle}
                description={text.resources.selectAssetDescription}
              />
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle className="text-base">{text.resources.medalsTitle}</CardTitle>
              <CardDescription>{text.resources.medalsDescription}</CardDescription>
            </div>
            <input
              value={medalSearch}
              onChange={(event) => setMedalSearch(event.target.value)}
              placeholder={text.resources.medalsPlaceholder}
              className="w-full rounded-xl border border-input px-3 py-2 text-sm md:max-w-sm"
            />
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-[minmax(0,320px)_minmax(0,1fr)]">
          <SelectableMedalList items={data.medals.items} selectedID={selectedMedalID} onSelect={setSelectedMedalID} text={text} />
          <div className="space-y-4">
            {data.medals.selected ? (
              <Card className="border-border">
                <CardContent className="space-y-4 p-4">
                  <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                    <p>{text.common.id}: <span className="font-medium text-foreground">{data.medals.selected.medal_id}</span></p>
                    <p>{text.common.type}: <span className="font-medium text-foreground">{data.medals.selected.medal_type || text.common.notAvailable}</span></p>
                    <p>{text.common.score}: <span className="font-medium text-foreground">{formatNumber(data.medals.selected.personal_score, locale, text)}</span></p>
                    <p>{text.common.fetchedAt}: <span className="font-medium text-foreground">{formatDate(data.medals.selected.fetched_at, locale, text)}</span></p>
                  </div>
                  <JsonViewer title={text.resources.rawMedalTitle} content={data.medals.selected.raw_json} text={text} />
                </CardContent>
              </Card>
            ) : (
              <EmptyStateNotice
                title={text.resources.selectMedalTitle}
                description={text.resources.selectMedalDescription}
              />
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function ContractsPanel({
  data,
  isLoading,
  isError,
  onRetry,
  locale,
  text,
}: {
  data: LabContractsResponse | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  locale: LabLocale
  text: LabText
}) {
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
          items={data.missing_in_go}
          emptyTitle={text.contracts.missingRoutesEmptyTitle}
          emptyDescription={text.contracts.missingRoutesEmptyDescription}
        />
        <RouteList
          title={text.contracts.extraRoutesTitle}
          items={data.extra_in_go}
          emptyTitle={text.contracts.extraRoutesEmptyTitle}
          emptyDescription={text.contracts.extraRoutesEmptyDescription}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{text.contracts.mismatchTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          {data.method_mismatches.length === 0 ? (
            <EmptyStateNotice
              title={text.contracts.mismatchEmptyTitle}
              description={text.contracts.mismatchEmptyDescription}
            />
          ) : (
            <div className="space-y-3">
              {data.method_mismatches.map((item) => (
                <div key={item.fastapi_path} className="rounded-xl border border-border p-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <StatusBadge status="DIVERGENCES" text={text} />
                    <p className="text-sm font-semibold text-foreground">{item.fastapi_path}</p>
                  </div>
                  <div className="mt-3 grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
                    <div>
                      <p className="font-medium text-foreground">{text.contracts.fastapiLabel}</p>
                      <p>{item.fastapi_methods.join(', ')}</p>
                    </div>
                    <div>
                      <p className="font-medium text-foreground">{text.contracts.goLabel}</p>
                      <p>{item.go_methods.join(', ')}</p>
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

function DiagnosticsPanel({
  data,
  isLoading,
  isError,
  onRetry,
  locale,
  text,
}: {
  data: LabDiagnosticsResponse | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  locale: LabLocale
  text: LabText
}) {
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
                {data.parity_report.results.map((item) => (
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

export function LabPage() {
  const capabilities = useAppShellStore((state) => state.capabilities)
  const currentTitleSlug = useAppShellStore((state) => state.currentTitleSlug)
  const locale = normalizeLabLocale(useAppShellStore((state) => state.locale))
  const canManageInstance = capabilities?.can_manage_instance ?? false
  const text = useMemo(() => getLabText(locale), [locale])

  const [activeTab, setActiveTab] = useState<LabTab>('resources')
  const [assetSearch, setAssetSearch] = useState('')
  const [medalSearch, setMedalSearch] = useState('')
  const [selectedSnapshotKey, setSelectedSnapshotKey] = useState('')
  const [selectedAssetID, setSelectedAssetID] = useState('')
  const [selectedMedalID, setSelectedMedalID] = useState<number | null>(null)

  const deferredAssetSearch = useDeferredValue(assetSearch)
  const deferredMedalSearch = useDeferredValue(medalSearch)

  const resourceParams = useMemo(
    () => ({
      snapshotKey: selectedSnapshotKey || undefined,
      assetID: selectedAssetID || undefined,
      assetSearch: deferredAssetSearch || undefined,
      medalID: selectedMedalID,
      medalSearch: deferredMedalSearch || undefined,
      limit: RESOURCE_LIMIT,
    }),
    [deferredAssetSearch, deferredMedalSearch, selectedAssetID, selectedMedalID, selectedSnapshotKey],
  )

  const resourcesQuery = useLabResources(resourceParams, canManageInstance)
  const contractsQuery = useLabContracts(canManageInstance && activeTab === 'contracts')
  const diagnosticsQuery = useLabDiagnostics(canManageInstance && activeTab === 'diagnostics')

  useEffect(() => {
    const first = resourcesQuery.data?.snapshots[0]
    if (first && !selectedSnapshotKey) {
      setSelectedSnapshotKey(first.resource_key)
    }
  }, [resourcesQuery.data?.snapshots, selectedSnapshotKey])

  useEffect(() => {
    const first = resourcesQuery.data?.assets.items[0]
    if (first && !selectedAssetID) {
      setSelectedAssetID(first.asset_id)
    }
  }, [resourcesQuery.data?.assets.items, selectedAssetID])

  useEffect(() => {
    const first = resourcesQuery.data?.medals.items[0]
    if (first && selectedMedalID == null) {
      setSelectedMedalID(first.medal_id)
    }
  }, [resourcesQuery.data?.medals.items, selectedMedalID])

  if (!canManageInstance) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={text.page.accessDeniedTitle}
          description={text.page.accessDeniedDescription}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-end">
        <Badge variant="outline">{text.page.currentTitleBadge}: {currentTitleSlug}</Badge>
      </div>

      <div className="flex flex-wrap gap-2 rounded-full bg-muted p-1">
        {TAB_VALUES.map((tab) => (
          <TabButton
            key={tab}
            active={activeTab === tab}
            label={text.tabs[tab]}
            onClick={() => startTransition(() => setActiveTab(tab))}
          />
        ))}
      </div>

      <LabIntroNotice locale={locale} />
      <LabSelectedToolNotice tab={activeTab} locale={locale} />

      {activeTab === 'resources' ? (
        <ResourcesPanel
          data={resourcesQuery.data}
          isLoading={resourcesQuery.isLoading}
          isError={resourcesQuery.isError}
          onRetry={() => void resourcesQuery.refetch()}
          locale={locale}
          text={text}
          assetSearch={assetSearch}
          setAssetSearch={setAssetSearch}
          medalSearch={medalSearch}
          setMedalSearch={setMedalSearch}
          selectedSnapshotKey={selectedSnapshotKey}
          setSelectedSnapshotKey={setSelectedSnapshotKey}
          selectedAssetID={selectedAssetID}
          setSelectedAssetID={setSelectedAssetID}
          selectedMedalID={selectedMedalID}
          setSelectedMedalID={setSelectedMedalID}
        />
      ) : null}

      {activeTab === 'contracts' ? (
        <ContractsPanel
          data={contractsQuery.data}
          isLoading={contractsQuery.isLoading}
          isError={contractsQuery.isError}
          onRetry={() => void contractsQuery.refetch()}
          locale={locale}
          text={text}
        />
      ) : null}

      {activeTab === 'diagnostics' ? (
        <DiagnosticsPanel
          data={diagnosticsQuery.data}
          isLoading={diagnosticsQuery.isLoading}
          isError={diagnosticsQuery.isError}
          onRetry={() => void diagnosticsQuery.refetch()}
          locale={locale}
          text={text}
        />
      ) : null}
    </div>
  )
}
