/**
 * DBContentionSection — Contention DB (B-swap shared, diagnostic du stall
 * pendant le sync). Extraction 1:1 depuis l'ancienne AdminPage.
 */
import { useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { SortableTh } from '@/components/ui/sortable-th'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { DBContentionResponse } from '@/lib/api/types'
import { useAdminDBContention } from '../queries'
import { useT } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

// DBContentionHolder n'est pas ré-exporté séparément par lib/api/types.ts
// (seul DBContentionResponse l'est) — dérivé par indexation du tableau.
type DBContentionHolder = DBContentionResponse['holders'][number]

type HolderSortKey = 'label' | 'count' | 'total_ms' | 'avg_ms' | 'max_ms' | 'watchdog_fired'

function holderRawValue(h: DBContentionHolder, key: HolderSortKey): string | number {
  return h[key]
}

function compareHolders(a: DBContentionHolder, b: DBContentionHolder, key: HolderSortKey, dir: 'asc' | 'desc'): number {
  const va = holderRawValue(a, key)
  const vb = holderRawValue(b, key)
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

export function DBContentionSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminDBContention()
  const t = useT()
  // I16 : tri CLIENT — table entièrement chargée. Aucun tri actif par défaut :
  // l'ordre serveur (total_ms DESC) reste affiché tant qu'aucun en-tête n'a été
  // cliqué (cf. commentaire ci-dessous « trié total desc côté API »).
  const [sortKey, setSortKey] = useState<HolderSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: HolderSortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'label' ? 'asc' : 'desc')
    }
  }
  const sortedHolders = useMemo(() => {
    const holders = data?.holders ?? []
    if (!sortKey) return holders
    return [...holders].sort((a, b) => compareHolders(a, b, sortKey, sortDir))
  }, [data?.holders, sortKey, sortDir])

  return (
    <section className="space-y-3">
      <SectionHeader
        title={t('common.admin.contention_section')}
        description={t('common.admin.contention_desc')}
        actions={
          <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? t('common.admin.loading') : t('common.admin.refresh')}
          </Button>
        }
      />

      {isLoading ? (
        <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
      ) : isError || !data ? (
        <p className="text-sm text-destructive">{t('common.admin.contention_unavailable')}</p>
      ) : (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Metric label={t('common.admin.contention_swaps')} value={String(data.swaps)} />
            <Metric label={t('common.admin.contention_acquire')} value={`${data.avg_acquire_ms} ms`} />
            <Metric label={t('common.admin.contention_release')} value={`${data.avg_release_ms} ms`} />
            <Metric label={t('common.admin.contention_drain')} value={`${data.drain_ms_total} ms`} />
            <Metric label={t('common.admin.contention_blocked_avg')} value={`${data.avg_blocked_ms} ms`} />
            <Metric
              label={t('common.admin.contention_blocked_max')}
              value={`${data.max_blocked_ms} ms`}
              alert={data.max_blocked_ms >= 1000}
            />
            <Metric
              label={t('common.admin.contention_503')}
              value={String(data.reads_rejected)}
              alert={data.reads_rejected > 0}
            />
            <Metric
              label={t('common.admin.contention_failures')}
              value={String(data.swap_failures)}
              alert={data.swap_failures > 0}
            />
            <Metric label={t('common.admin.contention_readers')} value={String(data.readers_in_use)} />
            <Metric label={t('common.admin.contention_state')} value={data.state} />
            <Metric label={t('common.admin.contention_rw_avg')} value={`${data.avg_rw_window_ms} ms`} />
            <Metric
              label={t('common.admin.contention_rw_max')}
              value={`${data.max_rw_window_ms} ms`}
              alert={data.max_rw_window_ms >= 2000}
            />
            <Metric
              label={t('common.admin.contention_watchdog')}
              value={String(data.watchdog_fired)}
              alert={data.watchdog_fired > 0}
            />
          </div>

          {/* Étape 0 attribution : ventilation de la détention writer par label
              (trié total desc côté API) — désigne les cibles du refactor
              « writer non tenu pendant I/O ». */}
          {data.holders.length > 0 && (
            <div>
              <h3 className="mb-2 text-sm font-medium text-foreground">
                {t('common.admin.contention_holders_title')}
              </h3>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-xs text-muted-foreground">
                    <SortableTh label={t('common.admin.contention_holder_label')} active={sortKey === 'label'} dir={sortDir} onClick={() => toggleSort('label')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_count')} active={sortKey === 'count'} dir={sortDir} onClick={() => toggleSort('count')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_total')} active={sortKey === 'total_ms'} dir={sortDir} onClick={() => toggleSort('total_ms')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_avg')} active={sortKey === 'avg_ms'} dir={sortDir} onClick={() => toggleSort('avg_ms')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_max')} active={sortKey === 'max_ms'} dir={sortDir} onClick={() => toggleSort('max_ms')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_watchdog')} active={sortKey === 'watchdog_fired'} dir={sortDir} onClick={() => toggleSort('watchdog_fired')} className="py-1 font-medium" />
                  </tr>
                </thead>
                <tbody>
                  {sortedHolders.map((h) => (
                    <tr key={h.label} className="border-b border-border/50">
                      <td className="py-1 pr-2 font-mono text-xs">{h.label}</td>
                      <td className="py-1 pr-2">{h.count}</td>
                      <td className="py-1 pr-2">{h.total_ms} ms</td>
                      <td className="py-1 pr-2">{h.avg_ms} ms</td>
                      <td
                        className="py-1 pr-2"
                        style={h.max_ms >= 2000 ? { color: tokenCssVar('destructive') } : undefined}
                      >
                        {h.max_ms} ms
                      </td>
                      <td
                        className="py-1"
                        style={h.watchdog_fired > 0 ? { color: tokenCssVar('destructive') } : undefined}
                      >
                        {h.watchdog_fired}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </section>
  )
}

function Metric({ label, value, alert }: { label: string; value: string; alert?: boolean }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className="text-lg font-semibold text-foreground"
        style={alert ? { color: tokenCssVar('destructive') } : undefined}
      >
        {value}
      </div>
    </div>
  )
}
