/**
 * SessionFragCard — carte « Répartition des frags » v2 d'UNE session : compose le
 * sunburst hiérarchique classe→rôle (FragSunburst) et, en dessous, le breakdown par
 * arme recoloré par classe (FragWeaponBreakdown). Calque SynthesisFragCard/MatchFragCard
 * — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P5.2.
 *
 * Alimentée par l'agrégat de session (entry.frag_distribution + entry.top_weapon_kills,
 * nouveau chemin de données P5). Non gatée : Infinite = classes sans Spartan ; Halo 5 =
 * avec (la capability native_kill_mechanics décide côté backend). Rend null si aucune
 * donnée (sunburst null ET aucune arme) — pas de carte vide dans la pile.
 */
import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import type { SessionCompareEntry } from '@/lib/api/types'

interface Props {
  entry: SessionCompareEntry | null
}

export function SessionFragCard({ entry }: Props) {
  const distribution = entry?.frag_distribution ?? null
  const weapons = entry?.top_weapon_kills ?? []

  const hasSunburst = (distribution?.total_kills ?? 0) > 0
  if (!hasSunburst && weapons.length === 0) return null

  return (
    <div className="flex flex-col gap-3">
      <FragSunburst distribution={distribution} />
      <FragWeaponBreakdown weapons={weapons} />
    </div>
  )
}
