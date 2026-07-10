/**
 * CronsPanel — statut unifié des crons + feature liveness (A6.3, onglet État).
 * Chaque ligne cron = dernier succès, échecs consécutifs, durée ; accent
 * destructive si échecs consécutifs >= seuil ou heartbeat jamais vu.
 * Couleurs exclusivement via tokens sémantiques.
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { CronStatusEntry, FeatureHeartbeat } from '@/lib/api/types'
import { useMonitoringCrons } from '../monitoring/queries'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'

/** Token d'accent d'un statut cron/feature (never = destructive, A6.3). */
function cronToken(status: string): SemanticToken | undefined {
  switch (status) {
    case 'ok':
      return 'success'
    case 'warn':
      return 'warning'
    case 'critical':
    case 'never':
      return 'destructive'
    default:
      return undefined
  }
}

export function CronsPanel() {
  const { data, isError } = useMonitoringCrons()
  const tA = useAdminT()

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        {tA('admin.crons.section')}
      </h3>
      {isError ? (
        <p className="text-sm text-destructive">{tA('admin.crons.unavailable')}</p>
      ) : !data ? (
        <p className="text-sm text-muted-foreground">…</p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <CronsTable crons={data.crons ?? []} />
          <FeaturesTable features={data.features ?? []} />
        </div>
      )}
    </section>
  )
}

function CronsTable({ crons }: { crons: CronStatusEntry[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_cron')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_last_success')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_failures')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_status')}</th>
          </tr>
        </thead>
        <tbody>
          {crons.length === 0 ? (
            <tr>
              <td className="px-3 py-2 text-xs text-muted-foreground" colSpan={4}>
                {tA('admin.crons.empty')}
              </td>
            </tr>
          ) : (
            crons.map((c) => (
              <tr key={c.name} className="border-b last:border-b-0 hover:bg-muted/30">
                <td className="px-3 py-2 font-mono text-xs text-foreground" title={c.last_error || undefined}>
                  {c.name}
                  {!c.since_boot && (
                    <span className="ml-1 text-muted-foreground" title={tA('admin.crons.since_restart')}>
                      *
                    </span>
                  )}
                </td>
                <td
                  className="px-3 py-2 text-xs text-muted-foreground"
                  title={adminAbsoluteTime(c.last_success_at, locale)}
                >
                  {c.last_success_at ? adminRelativeTime(c.last_success_at, locale) : tA('admin.crons.never')}
                </td>
                <td className="px-3 py-2 tabular-nums text-muted-foreground">{c.consecutive_failures}</td>
                <td className="px-3 py-2">
                  <StatusPill status={c.status} label={c.status} title={c.last_error || undefined} />
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

function FeaturesTable({ features }: { features: FeatureHeartbeat[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_feature')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_heartbeat')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.crons.col_status')}</th>
          </tr>
        </thead>
        <tbody>
          {features.map((f) => (
            <tr key={f.feature} className="border-b last:border-b-0 hover:bg-muted/30">
              <td className="px-3 py-2 font-mono text-xs text-foreground">{f.feature}</td>
              <td className="px-3 py-2 text-xs text-muted-foreground" title={adminAbsoluteTime(f.last_seen_at, locale)}>
                {f.last_seen_at ? adminRelativeTime(f.last_seen_at, locale) : tA('admin.crons.never')}
              </td>
              <td className="px-3 py-2">
                <StatusPill
                  status={f.status}
                  label={f.status === 'never' ? tA('admin.crons.never_seen') : f.status}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function StatusPill({ status, label, title }: { status: string; label: string; title?: string }) {
  const token = cronToken(status)
  const color = token ? tokenCssVar(token) : undefined
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-sm bg-muted px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground"
      style={color ? { color } : undefined}
      title={title}
    >
      {color && <span aria-hidden className="inline-block h-2 w-2 flex-none" style={{ backgroundColor: color }} />}
      {label}
    </span>
  )
}
