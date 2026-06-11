/**
 * DBContentionSection — Contention DB (B-swap shared, diagnostic du stall
 * pendant le sync). Extraction 1:1 depuis l'ancienne AdminPage.
 */
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useAdminDBContention } from '../queries'
import { useT } from '../useAdminText'

export function DBContentionSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminDBContention()
  const t = useT()

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{t('common.admin.contention_section')}</h2>
            <p className="max-w-xl text-xs text-muted-foreground">{t('common.admin.contention_desc')}</p>
          </div>
          <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? t('common.admin.loading') : t('common.admin.refresh')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : isError || !data ? (
          <p className="text-sm text-destructive">{t('common.admin.contention_unavailable')}</p>
        ) : (
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
          </div>
        )}
      </CardContent>
    </Card>
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
