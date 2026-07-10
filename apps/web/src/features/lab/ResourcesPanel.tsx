/**
 * ResourcesPanel — onglet "Resources" de LabPage : snapshots, assets, médailles.
 *
 * P8.4 (revue 2026-04-29) : extrait de LabPage.tsx (~215L).
 */
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import type { LabResourcesResponse } from '@/lib/api/types'
import type { LabLocale, LabText } from './i18n'
import {
  formatBytes,
  formatLabDateTime,
  formatNumber,
  JsonViewer,
  MetricCard,
  SelectableAssetList,
  SelectableMedalList,
} from './_labShared'

interface ResourcesPanelProps {
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
}

export function ResourcesPanel({
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
}: ResourcesPanelProps) {
  if (isLoading) return null

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

  // Défense : un slice nil côté Go sérialise en JSON `null` → `.length` / `.map`
  // crashent (« data.snapshots is null »). On normalise les listes avant rendu.
  const snapshots = data.snapshots ?? []
  const assetItems = data.assets?.items ?? []
  const medalItems = data.medals?.items ?? []

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label={text.resources.currentTitle} value={data.title_slug} hint={text.resources.currentTitleHint} />
        <MetricCard
          label={text.resources.activeSeason}
          value={data.current_season?.name ?? text.resources.activeSeasonFallback}
          hint={text.resources.snapshotsHint(snapshots.length)}
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
            {snapshots.length === 0 ? (
              <EmptyStateNotice
                title={text.resources.noSnapshotsTitle}
                description={text.resources.noSnapshotsDescription}
              />
            ) : (
              snapshots.map((snapshot) => (
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
                    <p>{text.common.fetchedAt}: <span className="font-medium text-foreground">{formatLabDateTime(data.selected_snapshot.fetched_at, locale, text)}</span></p>
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
          <SelectableAssetList items={assetItems} selectedID={selectedAssetID} onSelect={setSelectedAssetID} text={text} />
          <div className="space-y-4">
            {data.assets.selected ? (
              <Card className="border-border">
                <CardContent className="space-y-4 p-4">
                  <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                    <p>{text.common.asset}: <span className="font-medium text-foreground">{data.assets.selected.asset_id}</span></p>
                    <p>{text.common.type}: <span className="font-medium text-foreground">{data.assets.selected.asset_type}</span></p>
                    <p>{text.common.version}: <span className="font-medium text-foreground">{data.assets.selected.version_id}</span></p>
                    <p>{text.common.fetchedAt}: <span className="font-medium text-foreground">{formatLabDateTime(data.assets.selected.fetched_at, locale, text)}</span></p>
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
          <SelectableMedalList items={medalItems} selectedID={selectedMedalID} onSelect={setSelectedMedalID} text={text} />
          <div className="space-y-4">
            {data.medals.selected ? (
              <Card className="border-border">
                <CardContent className="space-y-4 p-4">
                  <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                    <p>{text.common.id}: <span className="font-medium text-foreground">{data.medals.selected.medal_id}</span></p>
                    <p>{text.common.type}: <span className="font-medium text-foreground">{data.medals.selected.medal_type || text.common.notAvailable}</span></p>
                    <p>{text.common.score}: <span className="font-medium text-foreground">{formatNumber(data.medals.selected.personal_score, locale, text)}</span></p>
                    <p>{text.common.fetchedAt}: <span className="font-medium text-foreground">{formatLabDateTime(data.medals.selected.fetched_at, locale, text)}</span></p>
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
