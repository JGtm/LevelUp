/**
 * TimeseriesKillTypesDonut — « répartition des frags » par TYPE sur la période
 * filtrée (1er onglet), pendant du « Top armes ». Réutilise le donut SVG partagé
 * (components/charts/KillTypesDonut).
 *
 * Capability-gated `native_kill_mechanics` : ne s'affiche QUE pour les titres qui
 * fournissent les mécaniques natives (Halo 5). Partition : mêlée / arme lourde /
 * grenade / assassinat / frappe au sol / charge d'épaule / autres
 * (= total − somme des catégories connues).
 */
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { TimeseriesKillTypes } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'

interface Props {
  killTypes: TimeseriesKillTypes | null | undefined
  t: (key: TimeseriesManifestKey) => string
}

export function TimeseriesKillTypesDonut({ killTypes, t }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const hasKillMechanics = useCapability('native_kill_mechanics')

  // Surface H5-only (mécaniques natives) : null pour les titres sans la capability
  // (le « Top armes » reste affiché à côté).
  if (!hasKillMechanics || !killTypes) return null

  const total = killTypes.total_kills
  const known =
    killTypes.melee_kills +
    killTypes.power_weapon_kills +
    killTypes.grenade_kills +
    killTypes.assassinations +
    killTypes.ground_pound_kills +
    killTypes.shoulder_bash_kills
  const other = Math.max(0, total - known)

  const slices: DonutSlice[] = [
    { label: t('timeseries.summary.kill_melee'), count: killTypes.melee_kills, token: 'chart-series-1' as SemanticToken },
    { label: t('timeseries.summary.kill_power'), count: killTypes.power_weapon_kills, token: 'chart-series-6' as SemanticToken },
    { label: t('timeseries.summary.kill_grenade'), count: killTypes.grenade_kills, token: 'chart-series-7' as SemanticToken },
    { label: t('timeseries.summary.kill_assassination'), count: killTypes.assassinations, token: 'chart-series-2' as SemanticToken },
    { label: t('timeseries.summary.kill_ground_pound'), count: killTypes.ground_pound_kills, token: 'chart-series-3' as SemanticToken },
    { label: t('timeseries.summary.kill_shoulder_bash'), count: killTypes.shoulder_bash_kills, token: 'chart-series-4' as SemanticToken },
    { label: t('timeseries.summary.kill_other'), count: other, token: 'chart-series-8' as SemanticToken },
  ].filter((s) => s.count > 0)

  if (slices.length === 0) return null

  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {t('timeseries.summary.kill_types_title')}
      </div>
      <div className="flex w-full flex-col items-center gap-2 p-3">
        <KillTypesDonut slices={slices} locale={locale} />
      </div>
    </div>
  )
}
