/**
 * SynthesisKillTypesDonut — adaptateur fin : « Répartition des frags » sur Synthesis.
 *
 * Délègue au composant partagé KillTypesDonutCard (components/charts) en
 * injectant les libellés depuis le manifest synthesis. La logique de partition
 * (mêlée / arme lourde / grenade / autres) et le rendu vivent dans le composant
 * partagé, réutilisé aussi sur Timeseries.
 */
import { KillTypesDonutCard } from '@/components/charts/KillTypesDonutCard'
import type { SynthesisDetailedStats } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  stats: SynthesisDetailedStats
  /** Total des frags (overview.total_kills) — sert à dériver la part « Autres ». */
  totalKills: number
}

export function SynthesisKillTypesDonut({ stats, totalKills }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  return (
    <KillTypesDonutCard
      title={formatMessage(synthesisManifest, 'synthesis.charts.kill_types_title', appLocale)}
      otherLabel={formatMessage(synthesisManifest, 'synthesis.charts.kill_type_other', appLocale)}
      melee={stats.total_melee_kills}
      powerWeapon={stats.total_power_weapon_kills}
      grenade={stats.total_grenade_kills}
      totalKills={totalKills}
    />
  )
}
