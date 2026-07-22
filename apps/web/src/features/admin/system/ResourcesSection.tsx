/**
 * ResourcesSection — panneau détaillé des ressources machine & process (A5.2,
 * onglet Système) : runtime Go, restarts, disque, tailles des bases DuckDB +
 * WAL, budgets/pool DuckDB (expvar enfin surfacés). Le verdict compact vit sur
 * l'onglet État. Couleurs exclusivement via tokens sémantiques.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminResourcesResponse } from '@/lib/api/types'
import { useMonitoringResources } from '../monitoring/queries'
import { AdminTable, AdminTd, AdminTh, AdminTr } from '../components/AdminTable'
import { SectionHeader } from '../components/SectionHeader'
import { formatBytes, formatDurationMs } from '../format'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'
import type { AdminLocale } from '../format'
import { diskToken } from './diskStatusToken'

export function ResourcesSection() {
  const { data, isError } = useMonitoringResources()
  const tA = useAdminT()
  const locale = useAdminLocale()

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.resources.section')} />
      {isError ? (
        <p className="text-sm text-destructive">{tA('admin.resources.unavailable')}</p>
      ) : !data ? (
        <p className="text-sm text-muted-foreground">…</p>
      ) : (
        <div className="space-y-4">
          <RuntimeSummary data={data} tA={tA} locale={locale} />
          <DatabasesTable data={data} tA={tA} locale={locale} />
          <BudgetsSummary data={data} tA={tA} />
        </div>
      )}
    </section>
  )
}

function RuntimeSummary({ data, tA, locale }: { data: AdminResourcesResponse; tA: TAdmin; locale: AdminLocale }) {
  const disk = data.disk
  const color = diskToken(disk.status)
  return (
    <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted-foreground">
      <span>
        {tA('admin.resources.disk_free')}{' '}
        <span
          className="font-semibold tabular-nums"
          style={color ? { color: tokenCssVar(color) } : undefined}
          title={disk.error || disk.path}
        >
          {disk.status === 'unknown' ? '—' : formatBytes(disk.free_bytes, locale)}
        </span>
        {disk.total_bytes > 0 && <span> / {formatBytes(disk.total_bytes, locale)}</span>}
      </span>
      <span>
        {tA('admin.resources.heap')}{' '}
        <span className="font-semibold tabular-nums text-foreground">{formatBytes(data.runtime.heap_alloc_bytes, locale)}</span>
        {' '}({tA('admin.resources.sys')} {formatBytes(data.runtime.sys_bytes, locale)})
      </span>
      <span>
        {tA('admin.resources.goroutines')}{' '}
        <span className="font-semibold tabular-nums text-foreground">{data.runtime.goroutines}</span>
      </span>
      <span>
        {tA('admin.resources.uptime')}{' '}
        <span className="font-semibold tabular-nums text-foreground">{formatDurationMs(data.uptime_s * 1000, locale)}</span>
      </span>
      <span>
        {tA('admin.resources.restarts')}{' '}
        <span className="font-semibold tabular-nums text-foreground">{data.restarts}</span>
      </span>
    </div>
  )
}

function DatabasesTable({ data, tA, locale }: { data: AdminResourcesResponse; tA: TAdmin; locale: AdminLocale }) {
  const dbs = data.databases ?? []
  return (
    <AdminTable
      head={
        <>
          <AdminTh>{tA('admin.resources.col_db')}</AdminTh>
          <AdminTh>{tA('admin.resources.col_size')}</AdminTh>
          <AdminTh>{tA('admin.resources.col_wal')}</AdminTh>
        </>
      }
    >
      {dbs.map((db) => (
        <AdminTr key={db.name}>
          <AdminTd className="font-mono text-xs text-foreground" title={db.path}>
            {db.name}
          </AdminTd>
          <AdminTd className="tabular-nums text-muted-foreground">{formatBytes(db.size_bytes, locale)}</AdminTd>
          <AdminTd className="tabular-nums text-muted-foreground">
            {db.wal_bytes ? formatBytes(db.wal_bytes, locale) : '—'}
          </AdminTd>
        </AdminTr>
      ))}
      <tr className="bg-muted/40">
        <AdminTd className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {tA('admin.resources.total')}
        </AdminTd>
        <AdminTd className="font-semibold tabular-nums text-foreground" colSpan={2}>
          {formatBytes(data.db_total_bytes, locale)}
        </AdminTd>
      </tr>
    </AdminTable>
  )
}

function BudgetsSummary({ data, tA }: { data: AdminResourcesResponse; tA: TAdmin }) {
  const budgets = data.budgets ?? {}
  const pool = data.pool_stats ?? {}
  return (
    <details className="text-xs text-muted-foreground">
      <summary className="cursor-pointer select-none font-medium">
        {tA('admin.resources.budgets')} ({Object.keys(budgets).length}) · {tA('admin.resources.pool')} ({Object.keys(pool).length})
      </summary>
      <pre className="mt-2 overflow-x-auto rounded-md border bg-muted/40 p-3 font-mono text-[11px] leading-relaxed">
        {JSON.stringify({ budgets, pool_stats: pool }, null, 2)}
      </pre>
    </details>
  )
}
