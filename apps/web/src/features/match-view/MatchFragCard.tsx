/**
 * MatchFragCard — « Répartition des frags » v2 du viewer sur la Match view : émet
 * DEUX cartes CÔTE À CÔTE (Fragment → cellules de la grille parente) — le sunburst
 * hiérarchique classe→rôle (FragSunburst) et le breakdown par arme recoloré par
 * classe (FragWeaponBreakdown). Remplace les deux anciens graphes de frags du viewer
 * (MatchWeaponPieChart « Frags par arme » + MatchKillTypesDonut « Frags par
 * technique ») — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P3.3.
 *
 * Chaque enfant est une ChartCard autonome (bordure/titre) → placé directement comme
 * cellule de la grille du parent (pas d'empilement vertical interne).
 *
 * Non gaté : Infinite = classes sans Spartan ; Halo 5 = avec (la capability décide
 * côté backend via native_kill_mechanics). Rend null si aucune donnée (sunburst null
 * ET aucune arme) — le viewer sans frags n'affiche aucune carte.
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
    <>
      <FragSunburst distribution={distribution} />
      <FragWeaponBreakdown weapons={breakdown} />
    </>
  )
}
