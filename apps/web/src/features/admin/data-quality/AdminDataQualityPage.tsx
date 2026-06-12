/**
 * AdminDataQualityPage — onglet Qualité données : compteurs d'inconnus (avec
 * delta vs visite précédente), action globale « Résoudre les noms registry »
 * (+ scan à blanc), et les quatre listes actionnables (modes, assets UUID,
 * playlists hors catalogue, xuids orphelins).
 */
import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import { KpiCard } from '@/components/cards/KpiCard'
import { Button } from '@/components/ui/button'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { apiErrorMessage } from '@/lib/api/client'
import type {
  AdminDataQualityCounts,
  LyingBitsResetResult,
  RegistryNamesBackfillResult,
} from '@/lib/api/types'
import { AdminActionButton } from '../components/AdminActionButton'
import { useDataQualityCounts } from './queries'
import {
  invalidateDataQuality,
  useRunCatalogRefresh,
  useRunCatalogUGCDrain,
  useRunLyingBitsReset,
  useRunRegistryNamesBackfill,
} from './mutations'
import {
  OrphanPlaylistsSection,
  OrphanXuidsSection,
  RawAssetsSection,
  UntranslatedModesSection,
} from './IssueSections'
import {
  counterDelta,
  readCountersSnapshot,
  writeCountersSnapshot,
  type CountersSnapshot,
} from '../countersTrend'
import { useAdminT, type TAdmin } from '../useAdminText'

const DQ_SNAPSHOT_KEY = 'admin-dq-snapshot'

export function AdminDataQualityPage() {
  const { data, isLoading, isError } = useDataQualityCounts()
  const tA = useAdminT()

  // Baseline roulante (pattern invariantsTrend) : delta vs run précédent.
  const [previous, setPrevious] = useState<CountersSnapshot>(() => readCountersSnapshot(DQ_SNAPSHOT_KEY))
  const lastRunRef = useRef<{ generatedAt: string; snapshot: CountersSnapshot } | null>(null)
  useEffect(() => {
    if (!data) return
    const snap = buildDQSnapshot(data)
    const last = lastRunRef.current
    if (last && last.generatedAt !== data.generated_at) {
      setPrevious(last.snapshot)
    }
    lastRunRef.current = { generatedAt: data.generated_at, snapshot: snap }
    writeCountersSnapshot(DQ_SNAPSHOT_KEY, snap)
  }, [data])

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError || !data) {
    return <p className="text-sm text-destructive">{tA('admin.dq.unavailable')}</p>
  }

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.dq.counts_section')}
        </h3>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
          <DQKpi label={tA('admin.dq.kpi_raw_uuids')} value={data.raw_uuid_total} delta={counterDelta(previous, 'raw_uuids', data.raw_uuid_total)} />
          <DQKpi label={tA('admin.dq.kpi_untranslated')} value={data.untranslated_modes} delta={counterDelta(previous, 'untranslated', data.untranslated_modes)} />
          <DQKpi label={tA('admin.dq.kpi_orphan_playlists')} value={data.orphan_playlists} delta={counterDelta(previous, 'orphan_playlists', data.orphan_playlists)} />
          <DQKpi label={tA('admin.dq.kpi_orphan_xuids')} value={data.orphan_xuids} delta={counterDelta(previous, 'orphan_xuids', data.orphan_xuids)} neutral />
          <DQKpi label={tA('admin.dq.kpi_lying_bits')} value={data.lying_bits_events + data.lying_bits_weapons} delta={counterDelta(previous, 'lying_bits', data.lying_bits_events + data.lying_bits_weapons)} />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.dq.actions_section')}
        </h3>
        <RegistryNamesAction tA={tA} />
        <LyingBitsAction tA={tA} />
        <CatalogDrainAction tA={tA} />
      </section>

      <UntranslatedModesSection />
      <RawAssetsSection />
      <OrphanPlaylistsSection />
      <OrphanXuidsSection />
    </div>
  )
}

function buildDQSnapshot(data: AdminDataQualityCounts): CountersSnapshot {
  return {
    raw_uuids: data.raw_uuid_total,
    untranslated: data.untranslated_modes,
    orphan_playlists: data.orphan_playlists,
    orphan_xuids: data.orphan_xuids,
    lying_bits: data.lying_bits_events + data.lying_bits_weapons,
  }
}

/** KPI count : 0 = vert, > 0 = warning (neutral pour les compteurs informatifs). */
function DQKpi({
  label,
  value,
  delta,
  neutral,
}: {
  label: string
  value: number
  delta?: number
  neutral?: boolean
}) {
  const accent: SemanticToken | undefined = neutral
    ? value > 0
      ? 'info'
      : 'success'
    : value > 0
      ? 'warning'
      : 'success'
  return (
    <KpiCard accent={accent} className="h-full">
      <div className="p-4">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="mt-1 flex items-baseline gap-2">
          <span className="text-2xl font-semibold tabular-nums text-foreground">{value}</span>
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
      </div>
    </KpiCard>
  )
}

/** Action backfill registry names : scan à blanc + run réel + résultat compact. */
function RegistryNamesAction({ tA }: { tA: TAdmin }) {
  const run = useRunRegistryNamesBackfill()
  const catalogRefresh = useRunCatalogRefresh()
  const [lastResult, setLastResult] = useState<RegistryNamesBackfillResult | null>(null)

  function launch(dryRun: boolean) {
    if (!dryRun && !confirm(tA('admin.dq.run_registry_names_confirm'))) return
    run.mutate(dryRun, {
      onSuccess: (res) => {
        setLastResult(res)
        if (!res.dry_run) toast.success(`${tA('admin.actions.done')} — ${tA('admin.dq.registry_names_result')} : ${res.total_fixed}`)
      },
      onError: (err) => toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed')),
    })
  }

  function launchCatalogRefresh() {
    if (!confirm(tA('admin.dq.run_catalog_refresh_confirm'))) return
    catalogRefresh.mutate(undefined, {
      onSuccess: (res) => {
        toast.success(
          `${tA('admin.actions.done')} — ${tA('admin.dq.catalog_refresh_result')} : playlists ${res.playlists} · pairs ${res.pairs} · maps ${res.maps} · variants ${res.game_variants}`,
        )
      },
      onError: (err) => toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed')),
    })
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-3">
        <Button size="sm" variant="outline" onClick={() => launch(false)} disabled={run.isPending}>
          {run.isPending ? tA('admin.job.in_progress') : tA('admin.dq.run_registry_names')}
        </Button>
        <Button size="sm" variant="ghost" onClick={() => launch(true)} disabled={run.isPending}>
          {tA('admin.dq.run_registry_names_dry')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={launchCatalogRefresh}
          disabled={catalogRefresh.isPending}
        >
          {catalogRefresh.isPending ? tA('admin.job.in_progress') : tA('admin.dq.run_catalog_refresh')}
        </Button>
      </div>
      {lastResult && <RegistryNamesResult result={lastResult} tA={tA} />}
    </div>
  )
}

function RegistryNamesResult({ result, tA }: { result: RegistryNamesBackfillResult; tA: TAdmin }) {
  const parts: Array<[string, number, number]> = [
    ['playlists', result.playlists_scanned, result.playlists_fixed],
    ['maps', result.maps_scanned, result.maps_fixed],
    ['pairs', result.pairs_scanned, result.pairs_fixed],
    ['variants', result.variants_scanned, result.variants_fixed],
  ]
  return (
    <p className="font-mono text-xs text-muted-foreground">
      {result.dry_run ? `${tA('admin.dq.registry_names_scanned')} : ` : `${tA('admin.dq.registry_names_result')} : `}
      {parts
        .map(([k, scanned, fixed]) => (result.dry_run ? `${k} ${scanned}` : `${k} ${fixed}/${scanned}`))
        .join(' · ')}
    </p>
  )
}

/**
 * Action reset des bits menteurs (events/weapons/events_loaded) : scan à blanc
 * + run réel (writer shared sérialisé) + résultat compact. Débloque le heal au
 * prochain sync delta.
 */
function LyingBitsAction({ tA }: { tA: TAdmin }) {
  const run = useRunLyingBitsReset()
  const [lastResult, setLastResult] = useState<LyingBitsResetResult | null>(null)

  function launch(dryRun: boolean) {
    if (!dryRun && !confirm(tA('admin.dq.run_lying_bits_confirm'))) return
    run.mutate(dryRun, {
      onSuccess: (res) => {
        setLastResult(res)
        if (!res.dry_run) toast.success(`${tA('admin.actions.done')} — ${tA('admin.dq.lying_bits_result')} : ${res.total}`)
      },
      onError: (err) => toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed')),
    })
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-3">
        <Button size="sm" variant="outline" onClick={() => launch(false)} disabled={run.isPending}>
          {run.isPending ? tA('admin.job.in_progress') : tA('admin.dq.run_lying_bits')}
        </Button>
        <Button size="sm" variant="ghost" onClick={() => launch(true)} disabled={run.isPending}>
          {tA('admin.dq.run_lying_bits_dry')}
        </Button>
      </div>
      {lastResult && <LyingBitsResult result={lastResult} tA={tA} />}
    </div>
  )
}

/**
 * Action drain DiscoveryUGC (réseau, rate-limité) : job asynchrone suivi inline
 * via AdminActionButton. Confirm avertit des appels API réels ; au succès,
 * invalide les compteurs (assets résolus → moins d'UUID bruts).
 */
function CatalogDrainAction({ tA }: { tA: TAdmin }) {
  const drain = useRunCatalogUGCDrain()
  const queryClient = useQueryClient()
  return (
    <AdminActionButton
      label={tA('admin.dq.run_ugc_drain')}
      busyLabel={tA('admin.dq.run_ugc_drain_busy')}
      confirmMessage={tA('admin.dq.run_ugc_drain_confirm')}
      onRun={async () => {
        try {
          const job = await drain.mutateAsync()
          return job.job_id
        } catch (err) {
          toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed'))
          return null
        }
      }}
      onJobTerminal={(job) => {
        if (job.status === 'succeeded') {
          invalidateDataQuality(queryClient)
          toast.success(tA('admin.dq.ugc_drain_done'))
        }
        // Échec : JobProgressInline affiche déjà le détail, pas de toast redondant.
      }}
    />
  )
}

function LyingBitsResult({ result, tA }: { result: LyingBitsResetResult; tA: TAdmin }) {
  const label = result.dry_run ? tA('admin.dq.lying_bits_scanned') : tA('admin.dq.lying_bits_result')
  // Catégories techniques (noms de colonnes backend) assemblées en expression
  // JS — comme RegistryNamesResult — pour éviter les littéraux JSX texte.
  const parts: Array<[string, number]> = [
    ['events', result.events_bits_cleared],
    ['weapons', result.weapons_bits_cleared],
    ['events_loaded', result.events_loaded_cleared],
  ]
  return (
    <p className="font-mono text-xs text-muted-foreground">
      {`${label} : ${parts.map(([k, n]) => `${k} ${n}`).join(' · ')}`}
    </p>
  )
}
