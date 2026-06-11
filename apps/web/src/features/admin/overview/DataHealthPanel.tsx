/**
 * DataHealthPanel — compteurs du dernier audit data health (UUIDs bruts,
 * lying bits, orphelins, garbage URLs). État vide explicite si aucun audit
 * complet depuis le boot (l'action « Lancer un audit data » est juste au-dessus).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MonitoringDataHealth } from '@/lib/api/types'
import type { TAdmin } from '../useAdminText'
import { adminAbsoluteTime, adminRelativeTime, type AdminLocale } from '../format'

export function DataHealthPanel({
  dataHealth,
  tA,
  locale,
}: {
  dataHealth?: MonitoringDataHealth
  tA: TAdmin
  locale: AdminLocale
}) {
  if (!dataHealth) {
    return <EmptyStateNotice title={tA('admin.overview.never_ran')} description={tA('admin.dh.never')} />
  }
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <DataHealthMetric label={tA('admin.dh.uuids_raw')} value={dataHealth.uuids_raw_count} />
        <DataHealthMetric label={tA('admin.dh.lying_events')} value={dataHealth.lying_bits_events} />
        <DataHealthMetric label={tA('admin.dh.lying_weapons')} value={dataHealth.lying_bits_weapon_kills} />
        <DataHealthMetric label={tA('admin.dh.orphan_xuids')} value={dataHealth.orphan_xuids} info />
        <DataHealthMetric label={tA('admin.dh.garbage_urls')} value={dataHealth.garbage_banner_urls} />
        <DataHealthMetric label={tA('admin.dh.warnings_total')} value={dataHealth.warnings_total} />
      </div>
      <p className="text-xs text-muted-foreground">
        {tA('admin.dh.last_run')} :{' '}
        <span title={adminAbsoluteTime(dataHealth.ran_at, locale)}>
          {adminRelativeTime(dataHealth.ran_at, locale)}
        </span>
      </p>
    </div>
  )
}

/**
 * Compteur d'audit : 0 = neutre, > 0 = warning (info=true pour les compteurs
 * informatifs comme orphan_xuids — hors WarningsTotal par décision data_health).
 */
function DataHealthMetric({ label, value, info }: { label: string; value: number; info?: boolean }) {
  const color = value > 0 ? tokenCssVar(info ? 'info' : 'warning') : undefined
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className="text-lg font-semibold tabular-nums text-foreground"
        style={color ? { color } : undefined}
      >
        {value}
      </div>
    </div>
  )
}
