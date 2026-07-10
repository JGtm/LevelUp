/**
 * KillTypesDonutCard — carte « Répartition des frags » par TYPE D'ARME.
 *
 * Wrapper de carte partagé autour du donut SVG (KillTypesDonut). Partition
 * mutuellement exclusive : mêlée / arme lourde / grenade / autres (arme normale)
 * = total − (mêlée + lourde + grenade). Les headshots sont ORTHOGONAUX au type
 * d'arme → hors donut (restent un KPI à part).
 *
 * Mécaniques de kill NATIVES (assassinat / frappe au sol / charge d'épaule) :
 * props OPTIONNELLES, injectées seulement par les titres qui exposent la
 * capability `native_kill_mechanics` (ex. Halo 5). Quand elles sont absentes
 * (cas Infinite), le composant est strictement identique à sa forme à 4 parts.
 *
 * i18n injecté par l'appelant (title + otherLabel) → réutilisable hors Synthesis
 * (Timeseries, Explorer…). Les libellés des 3 types viennent des field mappings
 * globaux. Color-blind friendly (tokens chart-series 1/6/7/8 pour les armes,
 * 2/3/4 pour les mécaniques — surtout PAS un dégradé séquentiel illisible en
 * catégoriel). Renvoie null si aucune part > 0.
 */
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'

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
  /**
   * Mécaniques de kill natives (capability-gated). Si omis (Infinite), aucune
   * part mécanique n'est émise et « Autres » n'est pas amputé. Chaque valeur
   * doit être fournie avec son libellé pour apparaître.
   */
  assassinations?: number
  assassinationLabel?: string
  groundPound?: number
  groundPoundLabel?: string
  shoulderBash?: number
  shoulderBashLabel?: string
}

export function KillTypesDonutCard({
  title,
  otherLabel,
  melee,
  powerWeapon,
  grenade,
  totalKills,
  assassinations,
  assassinationLabel,
  groundPound,
  groundPoundLabel,
  shoulderBash,
  shoulderBashLabel,
}: KillTypesDonutCardProps) {
  const { data: fieldMappings } = useFieldMappings()
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const labelOf = (key: string, fallback: string) => fieldMappings?.fields[key]?.label ?? fallback

  // Mécaniques de kill (tokens 2/3/4) — DISJOINTES des catégories d'arme ; émises
  // uniquement si l'appelant fournit la valeur ET son libellé. Sans elles → 0,
  // donc « Autres » et la partition restent identiques à la forme à 4 parts.
  const mechanicSlices: DonutSlice[] = [
    { label: assassinationLabel, count: assassinations, token: 'chart-series-2' as SemanticToken },
    { label: groundPoundLabel, count: groundPound, token: 'chart-series-3' as SemanticToken },
    { label: shoulderBashLabel, count: shoulderBash, token: 'chart-series-4' as SemanticToken },
  ]
    .filter((s): s is DonutSlice => typeof s.label === 'string' && (s.count ?? 0) > 0)

  const mechanicKills = mechanicSlices.reduce((acc, s) => acc + s.count, 0)
  const other = Math.max(0, totalKills - (melee + powerWeapon + grenade + mechanicKills))

  // Tokens DISTINCTS (1/6/7/8 pour les armes), color-blind friendly dans toutes
  // les palettes. Les mécaniques s'intercalent avant « Autres » (2/3/4).
  const slices: DonutSlice[] = [
    { label: labelOf('melee_kills', 'Mêlée'), count: melee, token: 'chart-series-1' as SemanticToken },
    { label: labelOf('power_weapon_kills', 'Arme lourde'), count: powerWeapon, token: 'chart-series-6' as SemanticToken },
    { label: labelOf('grenade_kills', 'Grenade'), count: grenade, token: 'chart-series-7' as SemanticToken },
    ...mechanicSlices,
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
