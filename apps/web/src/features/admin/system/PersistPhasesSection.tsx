/**
 * PersistPhasesSection — temps d'écriture DB du worker persist par phase
 * (attente writer shared, écriture shared+pve/meta, lease player, écriture
 * player). Complète la carte Contention : décompose le coût réel des writes.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminPerfStats, PerfCallStats } from '@/lib/api/types'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import { usePerfStats } from '../monitoring/queries'
import { formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'
import type { Locale } from '@/lib/i18n/locale'

const PHASE_LABELS: Record<string, AdminManifestKey> = {
  shared_acquire: 'admin.persist.phase_shared_acquire',
  shared_write: 'admin.persist.phase_shared_write',
  player_lease: 'admin.persist.phase_player_lease',
  player_write: 'admin.persist.phase_player_write',
}

export function PersistPhasesSection() {
  const { data } = usePerfStats()
  const tA = useAdminT()
  const locale = useAdminLocale()

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.persist.section')} />
      {!data || (data.persist_phases?.length ?? 0) === 0 ? (
        <EmptyStateNotice title={tA('admin.persist.empty')} description="" />
      ) : (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          {(data.persist_phases ?? []).map((p) => (
            <PhaseCell key={p.name} phase={p} label={labelFor(p, tA)} locale={locale} />
          ))}
        </div>
      )}
    </section>
  )
}

function labelFor(p: PerfCallStats, tA: (key: AdminManifestKey) => string): string {
  const key = PHASE_LABELS[p.name]
  return key ? tA(key) : p.name
}

function PhaseCell({
  phase,
  label,
  locale,
}: {
  phase: NonNullable<AdminPerfStats['persist_phases']>[number]
  label: string
  locale: Locale
}) {
  const hasErrors = (phase.errors ?? 0) > 0
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="text-lg font-semibold tabular-nums text-foreground">
        {formatDurationMs(phase.avg_ms, locale)}
      </div>
      <div className="font-mono text-[11px] text-muted-foreground">
        max {formatDurationMs(phase.max_ms, locale)} · ×{phase.count}
        {hasErrors && (
          <span className="ml-1 font-semibold" style={{ color: tokenCssVar('destructive') }}>
            · {phase.errors} err
          </span>
        )}
      </div>
    </div>
  )
}
