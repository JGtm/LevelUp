/**
 * MatchFragCard — « Répartition des frags » v2 du viewer sur la Match view : rend
 * DEUX cartes CÔTE À CÔTE dans sa propre grille — le sunburst
 * hiérarchique classe→rôle (FragSunburst) et le « Détails des frags » recoloré par
 * classe (FragWeaponBreakdown) — armes du registre + détail mêlée/grenade/capacités
 * issu de la distribution. Remplace les deux anciens graphes de frags du viewer
 * (MatchWeaponPieChart « Frags par arme » + MatchKillTypesDonut « Frags par
 * technique ») — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P3.3.
 *
 * Chaque enfant est une ChartCard autonome (bordure/titre). La grille (2/3 sunburst,
 * 1/3 breakdown) est portée ICI : quand le sunburst n'a pas de données (distribution
 * absente/vide), le breakdown occupe seul la pleine largeur — jamais de cellule
 * orpheline à 1/3, ni de wrapper grille vide côté parent.
 *
 * Survol LIÉ : un état `hoveredClass` PARTAGÉ est remonté ici. Survoler une classe/rôle
 * du sunburst estompe les armes des autres classes dans le breakdown, et réciproquement
 * survoler une barre estompe les autres classes du sunburst. Les deux composants restent
 * autonomes (le lien passe par les callbacks optionnels — cf. Synthesis/Sessions non liés).
 *
 * Non gaté : Infinite = classes sans Spartan ; Halo 5 = avec (la capability décide
 * côté backend via native_kill_mechanics). Rend null si aucune donnée (sunburst null
 * ET aucune arme) — le viewer sans frags n'affiche aucune carte.
 */
import { useState } from 'react'

import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import { buildFragDetailBreakdown } from '@/components/charts/fragDetailBreakdown'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { FragDistribution, MatchWeaponKill, SynthesisWeaponKillEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  distribution?: FragDistribution | null
  weapons?: MatchWeaponKill[]
}

export function MatchFragCard({ distribution, weapons }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  // Survol partagé entre les deux cartes (sunburst ↔ breakdown).
  const [hoveredClass, setHoveredClass] = useState<string | null>(null)
  const classLabel = (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale)
  const roleLabel = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale)
  // Titre du bloc, SCOPÉ Match view (les autres surfaces gardent « Frags par arme »).
  const detailTitle = formatMessage(fragsManifest, 'frags.charts.detail_title', appLocale)

  // « Détails des frags » = armes (per-arme du viewer) + détail mêlée/grenade/capacités depuis
  // la distribution (source unique buildFragDetailBreakdown). On normalise d'abord la liste
  // per-arme du viewer (MatchWeaponKill) vers la forme {label, kills, class}.
  const weaponsNorm: SynthesisWeaponKillEntry[] = (weapons ?? []).map((w) => ({
    label: w.weapon_label,
    kills: w.kill_count,
    class: w.class,
  }))
  const breakdown = buildFragDetailBreakdown(distribution, weaponsNorm, { roleLabel, classLabel, locale: appLocale })

  // Miroir EXACT du prédicat de rendu de FragSunburst (total > 0 ET classes non
  // vides) : si le sunburst rendrait null, on ne réserve pas sa colonne.
  const hasSunburst =
    (distribution?.total_kills ?? 0) > 0 && (distribution?.classes?.length ?? 0) > 0
  if (!hasSunburst && breakdown.length === 0) return null

  return (
    <div className={hasSunburst ? 'grid grid-cols-1 gap-4 lg:grid-cols-3' : ''}>
      {hasSunburst && (
        <FragSunburst
          distribution={distribution}
          externalHoveredClass={hoveredClass}
          onClassHover={setHoveredClass}
          className="lg:col-span-2"
          hideCenterLabel
          maxWidthPx={480}
          legendSide="left"
        />
      )}
      <FragWeaponBreakdown
        weapons={breakdown}
        title={detailTitle}
        hoveredClass={hoveredClass}
        onClassHover={setHoveredClass}
        className={hasSunburst ? 'lg:col-span-1' : ''}
        heightScale={1.1}
      />
    </div>
  )
}
