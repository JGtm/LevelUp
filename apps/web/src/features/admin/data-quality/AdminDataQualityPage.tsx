/**
 * AdminDataQualityPage — onglet Qualité données : compteurs d'inconnus (avec
 * delta vs visite précédente), action globale « Résoudre les noms registry »
 * (+ scan à blanc), et les quatre listes actionnables (modes, assets UUID,
 * playlists hors catalogue, xuids orphelins).
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { apiErrorMessage } from '@/lib/api/client'
import type {
  AdminDataQualityCounts,
  LyingBitsResetResult,
  RegistryNamesBackfillResult,
} from '@/lib/api/types'
import { AdminActionButton } from '../components/AdminActionButton'
import { AdminKpi } from '../components/AdminKpi'
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
import { counterDelta, type CountersSnapshot } from '../countersTrend'
import { useCounterSnapshot } from '../useCounterSnapshot'
import { useAdminT, type TAdmin } from '../useAdminText'
import { useAppShellStore } from '@/stores/appShellStore'
import { DiagnosticsPanel } from '@/features/lab/DiagnosticsPanel'
import { getLabText, normalizeLabLocale } from '@/features/lab/i18n'
import { useLabDiagnostics } from '@/features/lab/queries'
import { SectionHeader } from '../components/SectionHeader'

const DQ_SNAPSHOT_KEY = 'admin-dq-snapshot'

export function AdminDataQualityPage() {
  const { data, isLoading, isError } = useDataQualityCounts()
  const tA = useAdminT()

  // Diagnostics d'instance (ex-Lab) : parité endpoints + guards médailles.
  // Réutilise le panneau Lab + son i18n local ; query gardée admin via AdminLayout.
  const labLocale = normalizeLabLocale(useAppShellStore((s) => s.locale))
  const labText = getLabText(labLocale)
  const diagnostics = useLabDiagnostics(true)

  // Baseline roulante (hook canonique A8.2) : delta vs run precedent.
  const previous = useCounterSnapshot(DQ_SNAPSHOT_KEY, data?.generated_at, () => buildDQSnapshot(data!))

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError || !data) {
    return <p className="text-sm text-destructive">{tA('admin.dq.unavailable')}</p>
  }

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <SectionHeader title={tA('admin.dq.counts_section')} />
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
          {/* Accent : 0 = vert, > 0 = warning ('info' pour les compteurs informatifs). */}
          <AdminKpi label={tA('admin.dq.kpi_raw_uuids')} value={data.raw_uuid_total} accent={data.raw_uuid_total > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'raw_uuids', data.raw_uuid_total)} />
          <AdminKpi label={tA('admin.dq.kpi_untranslated')} value={data.untranslated_modes} accent={data.untranslated_modes > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'untranslated', data.untranslated_modes)} />
          <AdminKpi label={tA('admin.dq.kpi_orphan_playlists')} value={data.orphan_playlists} accent={data.orphan_playlists > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'orphan_playlists', data.orphan_playlists)} />
          <AdminKpi label={tA('admin.dq.kpi_orphan_xuids')} value={data.orphan_xuids} accent={data.orphan_xuids > 0 ? 'info' : 'success'} delta={counterDelta(previous, 'orphan_xuids', data.orphan_xuids)} />
          <AdminKpi label={tA('admin.dq.kpi_lying_bits')} value={data.lying_bits_events + data.lying_bits_weapons} accent={data.lying_bits_events + data.lying_bits_weapons > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'lying_bits', data.lying_bits_events + data.lying_bits_weapons)} />
        </div>
      </section>

      <section className="space-y-3">
        <SectionHeader title={tA('admin.dq.actions_section')} />
        <RegistryNamesAction tA={tA} />
        <LyingBitsAction tA={tA} />
        <CatalogDrainAction tA={tA} />
      </section>

      <UntranslatedModesSection />
      <RawAssetsSection />
      <OrphanPlaylistsSection />
      <OrphanXuidsSection />

      <section className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <SectionHeader title={tA('admin.dq.diagnostics_section')} />
          <Button
            size="sm"
            variant="outline"
            onClick={() => void diagnostics.refetch()}
            disabled={diagnostics.isFetching}
          >
            {diagnostics.isFetching ? tA('admin.job.in_progress') : tA('admin.dq.diagnostics_refresh')}
          </Button>
        </div>
        <DiagnosticsPanel
          data={diagnostics.data}
          isLoading={diagnostics.isLoading}
          isError={diagnostics.isError}
          onRetry={() => void diagnostics.refetch()}
          locale={labLocale}
          text={labText}
        />
      </section>
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


/** Ligne d'explication courte sous une action globale (à quoi ça sert / ce que ça fait). */
function ActionHelp({ text }: { text: string }) {
  return <p className="max-w-3xl text-xs text-muted-foreground">{text}</p>
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
      <ActionHelp text={tA('admin.dq.run_registry_names_help')} />
      <ActionHelp text={tA('admin.dq.run_catalog_refresh_help')} />
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
      <ActionHelp text={tA('admin.dq.run_lying_bits_help')} />
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
    <div className="space-y-2">
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
      <ActionHelp text={tA('admin.dq.run_ugc_drain_help')} />
    </div>
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
