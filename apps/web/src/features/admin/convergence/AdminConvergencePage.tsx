/**
 * AdminConvergencePage — onglet Convergence : backlog d'enrichissement par
 * joueur (deltas vs visite précédente, action « Converger » par joueur) +
 * compteurs post-sync du dernier cycle. Pas de polling continu (le backlog
 * résout les DBs de tous les joueurs) : staleTime 60 s + refetch au focus.
 */
import { useEffect, useRef, useState } from 'react'

import { KpiCard } from '@/components/cards/KpiCard'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminConvergenceReport } from '@/lib/api/types'
import { useMonitoringConvergence, useMonitoringScheduler } from '../monitoring/queries'
import {
  counterDelta,
  readCountersSnapshot,
  writeCountersSnapshot,
  type CountersSnapshot,
} from '../countersTrend'
import { useAdminT, type TAdmin } from '../useAdminText'
import { ConvergencePlayersTable } from './ConvergencePlayersTable'
import { PostSyncMatrix } from './PostSyncMatrix'
import { PostSyncTimeline } from './PostSyncTimeline'

const CONVERGENCE_SNAPSHOT_KEY = 'admin-convergence-snapshot'

export function AdminConvergencePage() {
  const { data, isLoading, isError } = useMonitoringConvergence()
  const scheduler = useMonitoringScheduler()
  const tA = useAdminT()

  // Baseline roulante (pattern invariantsTrend) : delta vs run précédent.
  const [previous, setPrevious] = useState<CountersSnapshot>(() =>
    readCountersSnapshot(CONVERGENCE_SNAPSHOT_KEY),
  )
  const lastRunRef = useRef<{ generatedAt: string; snapshot: CountersSnapshot } | null>(null)
  useEffect(() => {
    if (!data) return
    const snap = buildConvergenceSnapshot(data)
    const last = lastRunRef.current
    if (last && last.generatedAt !== data.generated_at) {
      setPrevious(last.snapshot)
    }
    lastRunRef.current = { generatedAt: data.generated_at, snapshot: snap }
    writeCountersSnapshot(CONVERGENCE_SNAPSHOT_KEY, snap)
  }, [data])

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError || !data) {
    return <p className="text-sm text-destructive">{tA('admin.convergence.unavailable')}</p>
  }

  const totals = sumBacklog(data)
  const allGreen = totals.enrichment + totals.psa + totals.events + totals.weapons === 0

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <div>
          <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
            {tA('admin.convergence.backlog_section')}
          </h3>
          <p className="mt-0.5 max-w-2xl text-xs text-muted-foreground">
            {tA('admin.convergence.backlog_desc')} (horizon : {data.horizon})
          </p>
        </div>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <BacklogKpi label={tA('admin.convergence.kpi_enrichment')} value={totals.enrichment} capped={false} delta={counterDelta(previous, 'enrichment', totals.enrichment)} />
          <BacklogKpi label={tA('admin.convergence.kpi_psa')} value={totals.psa} capped={totals.psaCapped} delta={counterDelta(previous, 'psa', totals.psa)} />
          <BacklogKpi label={tA('admin.convergence.kpi_events')} value={totals.events} capped={totals.eventsCapped} delta={counterDelta(previous, 'events', totals.events)} />
          <BacklogKpi label={tA('admin.convergence.kpi_weapons')} value={totals.weapons} capped={totals.weaponsCapped} delta={counterDelta(previous, 'weapons', totals.weapons)} />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.convergence.players_section')}
        </h3>
        {allGreen && data.players.every((p) => !p.check_error) ? (
          <EmptyStateNotice
            title={tA('admin.convergence.all_green_title')}
            description={tA('admin.convergence.all_green_desc')}
          />
        ) : (
          <ConvergencePlayersTable report={data} />
        )}
      </section>

      <CaughtUpPanel data={data} players={scheduler.data?.snapshot?.players ?? []} tA={tA} />

      <section className="space-y-3">
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.convergence.timeline_section')}
        </h3>
        <PostSyncTimeline players={scheduler.data?.snapshot?.players ?? []} />
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.convergence.postsync_section')}
        </h3>
        <PostSyncMatrix players={scheduler.data?.snapshot?.players ?? []} />
      </section>
    </div>
  )
}

/**
 * CaughtUpPanel — « rattrapé par la convergence » : dernier passage (somme
 * des compteurs post-sync par joueur) + cumuls depuis le boot (expvar).
 */
function CaughtUpPanel({
  data,
  players,
  tA,
}: {
  data: AdminConvergenceReport
  players: import('@/lib/api/types').SchedulerPlayerOutcome[]
  tA: TAdmin
}) {
  const lastEvents = players.reduce((acc, p) => acc + (p.post_sync?.converged_events ?? 0), 0)
  const lastPSA = players.reduce((acc, p) => acc + (p.post_sync?.converged_psa ?? 0), 0)
  const lastWeapons = players.reduce((acc, p) => acc + (p.post_sync?.weapon_kills_processed ?? 0), 0)
  const boot = data.totals_since_boot

  const rows: Array<{ label: string; last: number; sinceBoot: number }> = [
    { label: tA('admin.convergence.caught_events'), last: lastEvents, sinceBoot: boot.events_processed },
    { label: tA('admin.convergence.caught_weapons'), last: lastWeapons, sinceBoot: boot.weapons_processed },
    { label: tA('admin.convergence.caught_psa'), last: lastPSA, sinceBoot: boot.psa_processed },
    { label: tA('admin.convergence.caught_aliases'), last: -1, sinceBoot: boot.aliases_upserted },
  ]

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        {tA('admin.convergence.caught_up_section')}
      </h3>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {rows.map((r) => (
          <div key={r.label} className="rounded-md border px-3 py-2">
            <div className="text-xs text-muted-foreground">{r.label}</div>
            <div className="text-lg font-semibold tabular-nums text-foreground">
              {r.sinceBoot}
              <span className="ml-1 text-xs font-normal text-muted-foreground">
                {tA('admin.convergence.caught_since_boot')}
              </span>
            </div>
            {r.last >= 0 && (
              <div className="font-mono text-[11px] text-muted-foreground">
                +{r.last} {tA('admin.convergence.caught_last_cycle')}
              </div>
            )}
          </div>
        ))}
      </div>
    </section>
  )
}

function sumBacklog(data: AdminConvergenceReport) {
  const totals = {
    enrichment: 0,
    psa: 0,
    events: 0,
    weapons: 0,
    psaCapped: false,
    eventsCapped: false,
    weaponsCapped: false,
  }
  for (const p of data.players) {
    totals.enrichment += p.missing_enrichment
    totals.psa += p.missing_psa
    totals.events += p.missing_events
    totals.weapons += p.missing_weapons
    if (p.missing_psa >= data.horizon) totals.psaCapped = true
    if (p.missing_events >= data.horizon) totals.eventsCapped = true
    if (p.missing_weapons >= data.horizon) totals.weaponsCapped = true
  }
  return totals
}

function buildConvergenceSnapshot(data: AdminConvergenceReport): CountersSnapshot {
  const t = sumBacklog(data)
  return { enrichment: t.enrichment, psa: t.psa, events: t.events, weapons: t.weapons }
}

function BacklogKpi({
  label,
  value,
  capped,
  delta,
}: {
  label: string
  value: number
  capped: boolean
  delta?: number
}) {
  return (
    <KpiCard accent={value > 0 ? 'warning' : 'success'} className="h-full">
      <div className="p-4">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="mt-1 flex items-baseline gap-2">
          <span className="text-2xl font-semibold tabular-nums text-foreground">
            {value}
            {capped ? '+' : ''}
          </span>
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
