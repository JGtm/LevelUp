/**
 * MatchKillTypesDonut — « Répartition des frags » par TYPE pour le match courant
 * (viewer), pendant du donut « par arme ». Réutilise le donut SVG partagé
 * (components/charts/KillTypesDonut), color-blind friendly.
 *
 * Capability-gated `native_kill_mechanics` : ne s'affiche QUE pour les titres qui
 * fournissent les mécaniques natives (Halo 5). Partition du viewer : mêlée / arme
 * lourde / grenade / assassinat / frappe au sol / charge d'épaule / autres
 * (= kills − somme des catégories connues).
 */
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { MatchScoreboardRow } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

interface Props {
  /** Ligne de scoreboard du viewer (is_me) — porte les compteurs de kills par type. */
  me: MatchScoreboardRow | null | undefined
  t: MatchViewText
}

export function MatchKillTypesDonut({ me, t }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const hasKillMechanics = useCapability('native_kill_mechanics')

  // Surface H5-only (mécaniques natives) : pour les titres sans la capability,
  // pas de donut kill-types ici (le donut « par arme » reste affiché à côté).
  if (!hasKillMechanics || !me) return null

  const total = me.kills ?? 0
  const melee = me.melee_kills ?? 0
  const power = me.power_weapon_kills ?? 0
  const grenade = me.grenade_kills ?? 0
  const assassination = me.assassination_kills ?? 0
  const groundPound = me.ground_pound_kills ?? 0
  const shoulderBash = me.shoulder_bash_kills ?? 0
  const other = Math.max(0, total - (melee + power + grenade + assassination + groundPound + shoulderBash))

  const slices: DonutSlice[] = [
    { label: t.labelMelee, count: melee, token: 'chart-series-1' as SemanticToken },
    { label: t.labelPowerWeapon, count: power, token: 'chart-series-6' as SemanticToken },
    { label: t.labelGrenade, count: grenade, token: 'chart-series-7' as SemanticToken },
    { label: t.labelAssassination, count: assassination, token: 'chart-series-2' as SemanticToken },
    { label: t.labelGroundPound, count: groundPound, token: 'chart-series-3' as SemanticToken },
    { label: t.labelShoulderBash, count: shoulderBash, token: 'chart-series-4' as SemanticToken },
    { label: t.labelOtherKills, count: other, token: 'chart-series-8' as SemanticToken },
  ].filter((s) => s.count > 0)

  if (slices.length === 0) return null

  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{t.chartKillTypesTitle}</div>
      <div className="flex w-full flex-col items-center gap-2 p-3">
        <KillTypesDonut slices={slices} locale={locale} />
      </div>
    </div>
  )
}
