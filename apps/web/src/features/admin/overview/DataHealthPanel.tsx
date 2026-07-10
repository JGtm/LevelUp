/**
 * DataHealthPanel — compteurs du dernier audit data health (UUIDs bruts,
 * lying bits, orphelins, garbage URLs). État vide explicite si aucun audit
 * complet depuis le boot (l'action « Lancer un audit data » est juste au-dessus).
 * KPI = composant canonique AdminKpi (A8.1) : 0 = neutre, > 0 = warning
 * ('info' pour les compteurs informatifs comme orphan_xuids).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { MonitoringDataHealth } from '@/lib/api/types'
import { AdminKpi } from '../components/AdminKpi'
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
  const warnAccent = (v: number) => (v > 0 ? ('warning' as const) : undefined)
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <AdminKpi size="sm" label={tA('admin.dh.uuids_raw')} value={dataHealth.uuids_raw_count} accent={warnAccent(dataHealth.uuids_raw_count)} />
        <AdminKpi size="sm" label={tA('admin.dh.lying_events')} value={dataHealth.lying_bits_events} accent={warnAccent(dataHealth.lying_bits_events)} />
        <AdminKpi size="sm" label={tA('admin.dh.lying_weapons')} value={dataHealth.lying_bits_weapon_kills} accent={warnAccent(dataHealth.lying_bits_weapon_kills)} />
        <AdminKpi size="sm" label={tA('admin.dh.orphan_xuids')} value={dataHealth.orphan_xuids} accent={dataHealth.orphan_xuids > 0 ? 'info' : undefined} />
        <AdminKpi size="sm" label={tA('admin.dh.garbage_urls')} value={dataHealth.garbage_banner_urls} accent={warnAccent(dataHealth.garbage_banner_urls)} />
        <AdminKpi size="sm" label={tA('admin.dh.warnings_total')} value={dataHealth.warnings_total} accent={warnAccent(dataHealth.warnings_total)} />
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
