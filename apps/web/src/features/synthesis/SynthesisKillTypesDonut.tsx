/**
 * SynthesisKillTypesDonut — « Répartition des frags » par TYPE D'ARME.
 *
 * Réutilise le donut SVG partagé (composants/charts/KillTypesDonut), color-blind
 * friendly (tokens chart-series 1/6/7/8) et identique à celui du profil cible
 * Explorer. Partition mutuellement exclusive : mêlée / arme lourde / grenade /
 * autres (arme normale) = kills − (mêlée + lourde + grenade). Les headshots sont
 * ORTHOGONAUX au type d'arme → hors donut (restent un KPI à part).
 */
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
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
  const { data: fieldMappings } = useFieldMappings()
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const title = formatMessage(synthesisManifest, 'synthesis.charts.kill_types_title', appLocale)
  const labelOf = (key: string, fallback: string) => fieldMappings?.fields[key]?.label ?? fallback

  const hasKillMechanics = useCapability('native_kill_mechanics')

  // Mécaniques de kill NATIVES Halo 5 (assassinats + compétences spartiate),
  // DISJOINTES des catégories d'arme. Capability-gated : un titre sans la cap
  // (ex. Infinite) ne les compte pas → mechanicKills=0, slices non émises.
  const mechanicKills = hasKillMechanics
    ? stats.total_assassinations + stats.total_ground_pound_kills + stats.total_shoulder_bash_kills
    : 0
  const weaponTyped =
    stats.total_melee_kills + stats.total_power_weapon_kills + stats.total_grenade_kills + mechanicKills
  const other = Math.max(0, totalKills - weaponTyped)

  // Slices mécaniques h5 (2/3/4) — chaque slice est étiquetée label+valeur, donc la
  // distinction se fait au texte ; les armes gardent 1/6/7 et « Autres » le 8.
  const mechanicSlices: DonutSlice[] = hasKillMechanics
    ? [
        { label: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_assassination', appLocale), count: stats.total_assassinations, token: 'chart-series-2' as SemanticToken },
        { label: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_ground_pound', appLocale), count: stats.total_ground_pound_kills, token: 'chart-series-3' as SemanticToken },
        { label: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_shoulder_bash', appLocale), count: stats.total_shoulder_bash_kills, token: 'chart-series-4' as SemanticToken },
      ]
    : []

  // Tokens DISTINCTS color-blind friendly (cf. donut Explorer).
  const slices: DonutSlice[] = [
    { label: labelOf('melee_kills', 'Mêlée'), count: stats.total_melee_kills, token: 'chart-series-1' as SemanticToken },
    { label: labelOf('power_weapon_kills', 'Arme lourde'), count: stats.total_power_weapon_kills, token: 'chart-series-6' as SemanticToken },
    { label: labelOf('grenade_kills', 'Grenade'), count: stats.total_grenade_kills, token: 'chart-series-7' as SemanticToken },
    ...mechanicSlices,
    { label: formatMessage(synthesisManifest, 'synthesis.charts.kill_type_other', appLocale), count: other, token: 'chart-series-8' as SemanticToken },
  ].filter((s) => s.count > 0)

  if (slices.length === 0) return null

  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{title}</div>
      <div className="flex w-full flex-col items-center gap-2 p-3">
        <KillTypesDonut slices={slices} locale={locale} />
      </div>
    </div>
  )
}
