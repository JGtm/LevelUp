/**
 * BuildQueueSection — la file durable de construction des rejeux ET l'état des
 * ouvriers, dans le dashboard admin (piste F §4bis).
 *
 * POURQUOI CE PANNEAU EXISTE À CÔTÉ DES JOBS ASYNCHRONES. Les jobs asynchrones
 * sont le travail que CE serveur fait lui-même (ring mémoire, perdu au restart).
 * La file de construction est le travail qu'il DÉLÈGUE, éventuellement à une
 * autre machine — persisté, repris après un redémarrage, et pris par un ouvrier
 * qui n'est peut-être même pas allumé. Deux natures, deux tableaux.
 *
 * L'état vit côté serveur : ce panneau voit le travail distant sans jamais
 * interroger l'ouvrier. Un ouvrier disparu ne fait donc perdre aucune ligne — son
 * job repasse simplement en attente.
 *
 * Couleurs exclusivement via tokens sémantiques (StatusBadge).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { BuildQueueJob, BuildQueueWorker } from '@/lib/api/types'
import { useBuildQueue } from '../monitoring/queries'
import { AdminTable, AdminTd, AdminTh, AdminTr } from '../components/AdminTable'
import { SectionHeader } from '../components/SectionHeader'
import { StatusBadge } from '../components/StatusBadge'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { jobStatusToAdminStatus } from '../statusDisplay'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'
import type { AdminLocale } from '../format'

export function BuildQueueSection() {
  const { data, isError } = useBuildQueue()
  const tA = useAdminT()
  const locale = useAdminLocale()

  if (isError) {
    return (
      <section className="space-y-3">
        <SectionHeader title={tA('admin.buildqueue.section')} />
        <p className="text-sm text-destructive">{tA('admin.buildqueue.unavailable')}</p>
      </section>
    )
  }
  if (!data) {
    return (
      <section className="space-y-3">
        <SectionHeader title={tA('admin.buildqueue.section')} />
        <p className="text-sm text-muted-foreground">…</p>
      </section>
    )
  }

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.buildqueue.section')} />
      {!data.enabled && (
        <p className="text-sm text-muted-foreground">{tA('admin.buildqueue.disabled_notice')}</p>
      )}
      <CountsRow data={data.counts} tA={tA} />
      {/* Le serveur sert toujours des tableaux, jamais null — mais le schéma
          OpenAPI autorise null, et un `!` mentirait au premier changement. */}
      <JobsTable jobs={data.jobs ?? []} tA={tA} locale={locale} />
      <SectionHeader title={tA('admin.buildqueue.workers_section')} />
      <WorkersTable workers={data.workers ?? []} tA={tA} locale={locale} />
    </section>
  )
}

/** Compteurs de file : l'état d'ensemble sans lire le tableau ligne à ligne. */
function CountsRow({
  data,
  tA,
}: {
  data: { queued: number; running: number; succeeded: number; failed: number }
  tA: TAdmin
}) {
  const cells: Array<[string, number]> = [
    [tA('admin.buildqueue.counts_queued'), data.queued],
    [tA('admin.buildqueue.counts_running'), data.running],
    [tA('admin.buildqueue.counts_succeeded'), data.succeeded],
    [tA('admin.buildqueue.counts_failed'), data.failed],
  ]
  return (
    <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted-foreground">
      {cells.map(([label, value]) => (
        <span key={label}>
          {label}{' '}
          <span className="font-semibold tabular-nums text-foreground">{value}</span>
        </span>
      ))}
    </div>
  )
}

function JobsTable({ jobs, tA, locale }: { jobs: BuildQueueJob[]; tA: TAdmin; locale: AdminLocale }) {
  if (!jobs.length) {
    return (
      <EmptyStateNotice
        title={tA('admin.buildqueue.empty_title')}
        description={tA('admin.buildqueue.empty_desc')}
      />
    )
  }
  return (
    <AdminTable
      head={
        <>
          <AdminTh>{tA('admin.buildqueue.col_match')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_status')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_worker')}</AdminTh>
          <AdminTh className="text-right">{tA('admin.buildqueue.col_attempt')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_updated')}</AdminTh>
        </>
      }
    >
      {jobs.map((job) => (
        <AdminTr key={job.job_id}>
          <AdminTd>
            <span className="font-mono text-xs text-foreground">{job.match_id ?? '—'}</span>
            {job.error_message && (
              <div className="max-w-[24rem] truncate text-xs text-muted-foreground" title={job.error_message}>
                {job.error_message}
              </div>
            )}
          </AdminTd>
          <AdminTd>
            <StatusBadge status={jobStatusToAdminStatus(job.status)} />
          </AdminTd>
          <AdminTd className="font-mono text-xs text-muted-foreground">{job.worker_id || '—'}</AdminTd>
          <AdminTd className="text-right font-mono text-xs tabular-nums text-muted-foreground">
            {job.attempt > 0 ? job.attempt + 1 : 1}
          </AdminTd>
          <AdminTd
            className="text-xs text-muted-foreground"
            title={adminAbsoluteTime(job.updated_at ?? undefined, locale)}
          >
            {adminRelativeTime(job.updated_at ?? undefined, locale)}
          </AdminTd>
        </AdminTr>
      ))}
    </AdminTable>
  )
}

function WorkersTable({
  workers,
  tA,
  locale,
}: {
  workers: BuildQueueWorker[]
  tA: TAdmin
  locale: AdminLocale
}) {
  if (!workers.length) {
    return (
      <EmptyStateNotice
        title={tA('admin.buildqueue.workers_empty_title')}
        description={tA('admin.buildqueue.workers_empty_desc')}
      />
    )
  }
  return (
    <AdminTable
      head={
        <>
          <AdminTh>{tA('admin.buildqueue.col_worker')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_status')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_hostname')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_current_job')}</AdminTh>
          <AdminTh className="text-right">{tA('admin.buildqueue.col_done')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.col_last_beat')}</AdminTh>
        </>
      }
    >
      {workers.map((w) => (
        <AdminTr key={w.worker_id}>
          <AdminTd className="font-mono text-xs text-foreground">{w.worker_id}</AdminTd>
          <AdminTd>
            <StatusBadge
              status={w.online ? 'running' : 'idle'}
              label={w.online ? tA('admin.buildqueue.worker_online') : tA('admin.buildqueue.worker_offline')}
            />
          </AdminTd>
          <AdminTd className="text-xs text-muted-foreground">{w.hostname || '—'}</AdminTd>
          <AdminTd className="font-mono text-xs text-muted-foreground">
            {w.current_job_id || tA('admin.buildqueue.idle')}
          </AdminTd>
          <AdminTd className="text-right font-mono text-xs tabular-nums text-muted-foreground">
            {w.jobs_done}
            {w.jobs_failed > 0 && <span className="text-destructive"> / {w.jobs_failed}</span>}
          </AdminTd>
          <AdminTd
            className="text-xs text-muted-foreground"
            title={adminAbsoluteTime(w.last_beat_at ?? undefined, locale)}
          >
            {adminRelativeTime(w.last_beat_at ?? undefined, locale)}
          </AdminTd>
        </AdminTr>
      ))}
    </AdminTable>
  )
}
