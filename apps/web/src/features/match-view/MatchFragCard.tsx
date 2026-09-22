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

/**
 * Normalise la liste per-arme du viewer (MatchWeaponKill) vers la forme
 * `{label, kills, class}` attendue par le breakdown partagé.
 */
function normalizeWeapons(weapons?: MatchWeaponKill[]): SynthesisWeaponKillEntry[] {
  return (weapons ?? []).map((w) => ({
    label: w.weapon_label,
    kills: w.kill_count,
    class: w.class,
  }))
}

/**
 * hasFragSunburst — miroir EXACT du prédicat de rendu de `FragSunburst` (total > 0 ET
 * classes non vides). Exporté parce que la carte s'en sert deux fois : pour savoir si elle
 * réserve la colonne du sunburst, et comme première branche de {@link hasMatchFragData}.
 */
export function hasFragSunburst(distribution?: FragDistribution | null): boolean {
  return (distribution?.total_kills ?? 0) > 0 && (distribution?.classes?.length ?? 0) > 0
}

/**
 * Résolveurs d'identité : le COMPTE de lignes du breakdown ne dépend pas des libellés
 * (`buildFragDetailBreakdown` pousse une entrée par rôle/classe retenue, quel que soit son
 * nom). Compter sans manifeste i18n rend {@link hasMatchFragData} PUR — appelable par le
 * parent hors rendu, sans store ni locale.
 */
const COUNT_ONLY_LABELS = {
  roleLabel: (role: string) => role,
  classLabel: (className: string) => className,
  locale: 'fr' as const,
}

/**
 * hasMatchFragData — LE prédicat de rendu de la carte, en fonction pure.
 *
 * La carte s'en sert pour son `return null` et le parent (`MatchViewTabArsenal`) pour
 * décider d'afficher, ou non, le titre de section qui la coiffe : un titre ne se pose
 * jamais au-dessus de rien, et le prédicat ne s'écrit qu'ICI (règle ≤ 2 copies).
 */
export function hasMatchFragData(
  distribution?: FragDistribution | null,
  weapons?: MatchWeaponKill[],
): boolean {
  if (hasFragSunburst(distribution)) return true
  return buildFragDetailBreakdown(distribution, normalizeWeapons(weapons), COUNT_ONLY_LABELS).length > 0
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
  const breakdown = buildFragDetailBreakdown(
    distribution,
    normalizeWeapons(weapons),
    { roleLabel, classLabel, locale: appLocale },
  )

  // Le sunburst rendrait null : on ne réserve pas sa colonne.
  const hasSunburst = hasFragSunburst(distribution)
  // MÊME prédicat que celui lu par le parent pour poser (ou non) son titre de section.
  if (!hasMatchFragData(distribution, weapons)) return null

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
