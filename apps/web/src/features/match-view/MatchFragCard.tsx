/**
 * MatchFragCard — « Répartition des frags » v2 du viewer sur la Match view : émet
 * DEUX cartes CÔTE À CÔTE (Fragment → cellules de la grille parente) — le sunburst
 * hiérarchique classe→rôle (FragSunburst) et le « Détails des frags » recoloré par
 * classe (FragWeaponBreakdown) — armes du registre + détail mêlée/grenade/capacités
 * issu de la distribution. Remplace les deux anciens graphes de frags du viewer
 * (MatchWeaponPieChart « Frags par arme » + MatchKillTypesDonut « Frags par
 * technique ») — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P3.3.
 *
 * Chaque enfant est une ChartCard autonome (bordure/titre) → placé directement comme
 * cellule de la grille du parent (pas d'empilement vertical interne).
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
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { FragDistribution, MatchWeaponKill, SynthesisWeaponKillEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  distribution?: FragDistribution | null
  weapons?: MatchWeaponKill[]
}

// Classes servies par des ARMES (registre). Les autres (mêlée / grenade / capacités
// spartanes) ne sont PAS des armes : leur détail vient de `frag_distribution` (compteurs
// natifs du scoreboard), jamais de la liste per-arme → append sans double-comptage.
const GUN_CLASSES = new Set(['shoulder', 'sidearm', 'heavy'])

export function MatchFragCard({ distribution, weapons }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  // Survol partagé entre les deux cartes (sunburst ↔ breakdown).
  const [hoveredClass, setHoveredClass] = useState<string | null>(null)
  const classLabel = (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale)
  const roleLabel = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale)
  // Titre du bloc, SCOPÉ Match view (les autres surfaces gardent « Frags par arme »).
  const detailTitle = formatMessage(fragsManifest, 'frags.charts.detail_title', appLocale)

  // Armes (guns) depuis la liste per-arme du viewer. On écarte d'éventuelles lignes
  // non-arme pour ne pas doubler avec le détail issu de la distribution ci-dessous.
  const guns: SynthesisWeaponKillEntry[] = (weapons ?? [])
    .filter((w) => !w.class || GUN_CLASSES.has(w.class))
    .map((w) => ({ label: w.weapon_label, kills: w.kill_count, class: w.class }))

  // Détail des frags NON-arme depuis la distribution (autoritatif, sans recouvrement) :
  // mêlée (Assassinat / Corps-à-corps), grenade, capacités spartanes — par rôle si le
  // niveau 2 existe, sinon en feuille. On saute les classes gun (déjà dans `guns`) et le
  // résidu « Non attribué ».
  const details: SynthesisWeaponKillEntry[] = []
  for (const c of distribution?.classes ?? []) {
    if (GUN_CLASSES.has(c.class) || c.class === 'unattributed') continue
    const roles = c.roles ?? []
    if (roles.length > 0) {
      for (const r of roles) details.push({ label: roleLabel(r.role), kills: r.kills, class: c.class })
    } else {
      details.push({ label: classLabel(c.class), kills: c.kills, class: c.class })
    }
  }

  // « Détails des frags » = armes + détail mêlée/grenade/capacités, trié valeur desc.
  const breakdown: SynthesisWeaponKillEntry[] = [...guns, ...details].sort((a, b) => b.kills - a.kills)

  const hasSunburst = (distribution?.total_kills ?? 0) > 0
  if (!hasSunburst && breakdown.length === 0) return null

  return (
    <>
      <FragSunburst
        distribution={distribution}
        externalHoveredClass={hoveredClass}
        onClassHover={setHoveredClass}
        className="lg:col-span-2"
        hideCenterLabel
        maxWidthPx={480}
        legendSide="left"
      />
      <FragWeaponBreakdown
        weapons={breakdown}
        title={detailTitle}
        hoveredClass={hoveredClass}
        onClassHover={setHoveredClass}
        className="lg:col-span-1"
        heightScale={1.1}
      />
    </>
  )
}
