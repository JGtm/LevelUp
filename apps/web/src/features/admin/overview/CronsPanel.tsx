/**
 * CronsPanel — statut unifié des crons + feature liveness (A6.3, onglet État).
 * Chaque ligne cron = dernier succès, échecs consécutifs, durée ; accent
 * destructive si échecs consécutifs >= seuil ou heartbeat jamais vu.
 * Couleurs exclusivement via tokens sémantiques.
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { CronStatusEntry, FeatureHeartbeat } from '@/lib/api/types'
import { useMonitoringCrons } from '../monitoring/queries'
import { AdminTable, AdminTd, AdminTh, AdminTr } from '../components/AdminTable'
import { SectionHeader } from '../components/SectionHeader'
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
      <SectionHeader title={tA('admin.crons.section')} />
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
    <AdminTable
      head={
        <>
          <AdminTh>{tA('admin.crons.col_cron')}</AdminTh>
          <AdminTh>{tA('admin.crons.col_last_success')}</AdminTh>
          <AdminTh>{tA('admin.crons.col_failures')}</AdminTh>
          <AdminTh>{tA('admin.crons.col_status')}</AdminTh>
        </>
      }
    >
      {crons.length === 0 ? (
        <tr>
          <AdminTd className="text-xs text-muted-foreground" colSpan={4}>
            {tA('admin.crons.empty')}
          </AdminTd>
        </tr>
      ) : (
        crons.map((c) => (
          <AdminTr key={c.name}>
            <AdminTd className="font-mono text-xs text-foreground" title={c.last_error || undefined}>
              {c.name}
              {!c.since_boot && (
                <span className="ml-1 text-muted-foreground" title={tA('admin.crons.since_restart')}>
                  *
                </span>
              )}
            </AdminTd>
            <AdminTd className="text-xs text-muted-foreground" title={adminAbsoluteTime(c.last_success_at, locale)}>
              {c.last_success_at ? adminRelativeTime(c.last_success_at, locale) : tA('admin.crons.never')}
            </AdminTd>
            <AdminTd className="tabular-nums text-muted-foreground">{c.consecutive_failures}</AdminTd>
            <AdminTd>
              <StatusPill status={c.status} label={c.status} title={c.last_error || undefined} />
            </AdminTd>
          </AdminTr>
        ))
      )}
    </AdminTable>
  )
}

function FeaturesTable({ features }: { features: FeatureHeartbeat[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  return (
    <AdminTable
      head={
        <>
          <AdminTh>{tA('admin.crons.col_feature')}</AdminTh>
          <AdminTh>{tA('admin.crons.col_heartbeat')}</AdminTh>
          <AdminTh>{tA('admin.crons.col_status')}</AdminTh>
        </>
      }
    >
      {features.map((f) => (
        <AdminTr key={f.feature}>
          <AdminTd className="font-mono text-xs text-foreground">{f.feature}</AdminTd>
          <AdminTd className="text-xs text-muted-foreground" title={adminAbsoluteTime(f.last_seen_at, locale)}>
            {f.last_seen_at ? adminRelativeTime(f.last_seen_at, locale) : tA('admin.crons.never')}
          </AdminTd>
          <AdminTd>
            <StatusPill
              status={f.status}
              label={f.status === 'never' ? tA('admin.crons.never_seen') : f.status}
            />
          </AdminTd>
        </AdminTr>
      ))}
    </AdminTable>
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
