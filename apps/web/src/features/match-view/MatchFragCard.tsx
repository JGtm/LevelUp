/**
 * MatchFragCard — carte « Répartition des frags » v2 du viewer sur la Match view :
 * compose le sunburst hiérarchique classe→rôle (FragSunburst) et, en dessous, le
 * breakdown par arme recoloré par classe (FragWeaponBreakdown). Remplace les deux
 * anciens graphes de frags du viewer (MatchWeaponPieChart « Frags par arme » +
 * MatchKillTypesDonut « Frags par technique ») — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P3.3.
 *
 * Non gaté : Infinite = classes sans Spartan ; Halo 5 = avec (la capability décide
 * côté backend via native_kill_mechanics). Rend null si aucune donnée (sunburst null
 * ET aucune arme) — le viewer sans frags n'affiche pas de carte vide.
 */
import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import type { FragDistribution, MatchWeaponKill, SynthesisWeaponKillEntry } from '@/lib/api/types'

interface Props {
  distribution?: FragDistribution | null
  weapons?: MatchWeaponKill[]
}

export function MatchFragCard({ distribution, weapons }: Props) {
  // Mappe les kills par arme du viewer (MatchWeaponKill) vers la forme attendue par
  // FragWeaponBreakdown (SynthesisWeaponKillEntry) : la classe recolore chaque barre.
  const breakdown: SynthesisWeaponKillEntry[] = (weapons ?? []).map((w) => ({
    label: w.weapon_label,
    kills: w.kill_count,
    class: w.class,
  }))

  const hasSunburst = (distribution?.total_kills ?? 0) > 0
  if (!hasSunburst && breakdown.length === 0) return null

  return (
    <div className="flex flex-col gap-3">
      <FragSunburst distribution={distribution} />
      <FragWeaponBreakdown weapons={breakdown} />
    </div>
  )
}
