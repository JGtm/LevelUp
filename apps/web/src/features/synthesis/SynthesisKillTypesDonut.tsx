/**
 * SynthesisKillTypesDonut — adaptateur fin : « Répartition des frags » sur Synthesis.
 *
 * Délègue au composant partagé KillTypesDonutCard (components/charts) en
 * injectant les libellés depuis le manifest synthesis. La logique de partition
 * (mêlée / arme lourde / grenade / autres) et le rendu vivent dans le composant
 * partagé, réutilisé aussi sur Timeseries.
 *
 * Mécaniques de kill NATIVES (assassinat / frappe au sol / charge d'épaule) :
 * passées au composant partagé UNIQUEMENT si le titre courant expose la
 * capability `native_kill_mechanics` (ex. Halo 5). Sans la cap (Infinite), la
 * carte rendue est strictement identique à sa forme à 4 parts.
 */
import { KillTypesDonutCard } from '@/components/charts/KillTypesDonutCard'
import type { SynthesisDetailedStats } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCapability } from '@/lib/capabilities/capabilities'

interface Props {
  stats: SynthesisDetailedStats
  /** Total des frags (overview.total_kills) — sert à dériver la part « Autres ». */
  totalKills: number
}

export function SynthesisKillTypesDonut({ stats, totalKills }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const hasKillMechanics = useCapability('native_kill_mechanics')

  // Mécaniques natives Halo 5, capability-gated : on ne passe les props (et leurs
  // libellés) que si la cap est présente. Sinon `undefined` → byte-équivalent main.
  const mechanics = hasKillMechanics
    ? {
        assassinations: stats.total_assassinations,
        assassinationLabel: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_assassination', appLocale),
        groundPound: stats.total_ground_pound_kills,
        groundPoundLabel: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_ground_pound', appLocale),
        shoulderBash: stats.total_shoulder_bash_kills,
        shoulderBashLabel: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_shoulder_bash', appLocale),
      }
    : {}

  return (
    <KillTypesDonutCard
      title={formatMessage(synthesisManifest, 'synthesis.charts.kill_types_title', appLocale)}
      otherLabel={formatMessage(synthesisManifest, 'synthesis.charts.kill_type_other', appLocale)}
      melee={stats.total_melee_kills}
      powerWeapon={stats.total_power_weapon_kills}
      grenade={stats.total_grenade_kills}
      totalKills={totalKills}
      {...mechanics}
    />
  )
}
