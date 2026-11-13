/**
 * DBContentionSection — Contention DB (B-swap shared, diagnostic du stall
 * pendant le sync). Extraction 1:1 depuis l'ancienne AdminPage.
 */
import { Button } from '@/components/ui/button'
import { SortableTh } from '@/components/ui/sortable-th'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColumnSort, type SortColumn } from '@/lib/table/useColumnSort'
import type { DBContentionResponse } from '@/lib/api/types'
import { useAdminDBContention } from '../queries'
import { useT } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

// DBContentionHolder n'est pas ré-exporté séparément par lib/api/types.ts
// (seul DBContentionResponse l'est) — dérivé par indexation du tableau.
type DBContentionHolder = NonNullable<DBContentionResponse['holders']>[number]

type HolderSortKey = 'label' | 'count' | 'total_ms' | 'avg_ms' | 'max_ms' | 'watchdog_fired'

const SORT_COLUMNS: Record<HolderSortKey, SortColumn<DBContentionHolder>> = {
  label: { value: (h) => h.label, firstDir: 'asc' },
  count: { value: (h) => h.count },
  total_ms: { value: (h) => h.total_ms },
  avg_ms: { value: (h) => h.avg_ms },
  max_ms: { value: (h) => h.max_ms },
  watchdog_fired: { value: (h) => h.watchdog_fired },
}

export function DBContentionSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminDBContention()
  const t = useT()
  // I16 : tri CLIENT — table entièrement chargée. Aucun tri actif par défaut :
  // l'ordre serveur (total_ms DESC) reste affiché tant qu'aucun en-tête n'a été
  // cliqué (cf. commentaire ci-dessous « trié total desc côté API »).
  const sort = useColumnSort(data?.holders ?? [], SORT_COLUMNS)
  const sortedHolders = sort.rows

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
          {(data.holders?.length ?? 0) > 0 && (
            <div>
              <h3 className="mb-2 text-sm font-medium text-foreground">
                {t('common.admin.contention_holders_title')}
              </h3>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-xs text-muted-foreground">
                    <SortableTh label={t('common.admin.contention_holder_label')} {...sort.header('label')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_count')} {...sort.header('count')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_total')} {...sort.header('total_ms')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_avg')} {...sort.header('avg_ms')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_holder_max')} {...sort.header('max_ms')} className="py-1 pr-2 font-medium" />
                    <SortableTh label={t('common.admin.contention_watchdog')} {...sort.header('watchdog_fired')} className="py-1 font-medium" />
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
