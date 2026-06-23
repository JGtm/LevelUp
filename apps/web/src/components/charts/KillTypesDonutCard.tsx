/**
 * KillTypesDonutCard — carte « Répartition des frags » par TYPE D'ARME.
 *
 * Wrapper de carte partagé autour du donut SVG (KillTypesDonut). Partition
 * mutuellement exclusive : mêlée / arme lourde / grenade / autres (arme normale)
 * = total − (mêlée + lourde + grenade). Les headshots sont ORTHOGONAUX au type
 * d'arme → hors donut (restent un KPI à part).
 *
 * i18n injecté par l'appelant (title + otherLabel) → réutilisable hors Synthesis
 * (Timeseries, Explorer…). Les libellés des 3 types viennent des field mappings
 * globaux. Color-blind friendly (tokens chart-series 1/6/7/8 — surtout PAS 2-5,
 * un dégradé séquentiel illisible en catégoriel). Renvoie null si aucune part > 0.
 */
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'

interface KillTypesDonutCardProps {
  /** Titre de la carte. */
  title: string
  /** Libellé de la part « Autres » (arme normale). */
  otherLabel: string
  melee: number
  powerWeapon: number
  grenade: number
  /** Total des frags — sert à dériver la part « Autres ». */
  totalKills: number
}

export function KillTypesDonutCard({
  title,
  otherLabel,
  melee,
  powerWeapon,
  grenade,
  totalKills,
}: KillTypesDonutCardProps) {
  const { data: fieldMappings } = useFieldMappings()
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const labelOf = (key: string, fallback: string) => fieldMappings?.fields[key]?.label ?? fallback

  const other = Math.max(0, totalKills - (melee + powerWeapon + grenade))

  // Tokens DISTINCTS (1/6/7/8), color-blind friendly dans toutes les palettes.
  const slices: DonutSlice[] = [
    { label: labelOf('melee_kills', 'Mêlée'), count: melee, token: 'chart-series-1' as SemanticToken },
    { label: labelOf('power_weapon_kills', 'Arme lourde'), count: powerWeapon, token: 'chart-series-6' as SemanticToken },
    { label: labelOf('grenade_kills', 'Grenade'), count: grenade, token: 'chart-series-7' as SemanticToken },
    { label: otherLabel, count: other, token: 'chart-series-8' as SemanticToken },
  ].filter((s) => s.count > 0)

  if (slices.length === 0) return null

  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{title}</div>
      <div className="flex w-full flex-1 flex-col items-center justify-center gap-2 p-3">
        <KillTypesDonut slices={slices} locale={locale} />
      </div>
    </div>
  )
}
