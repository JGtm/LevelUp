/**
 * MatchFragCard — la rangée « Répartition des frags » | « Outils de destruction » du joueur de la page
 * sur la Vue match (plan PLAN_MATCHVIEW_EMPRISE_2026-10-06, cartes A et B, D10).
 *
 * A : l'anneau hiérarchique classe → rôle (`FragSunburst` nu), monté comme sur Sessions : titre et
 * aide, l'anneau, puis la légende des classes (`FragClassLegend`) en pied de carte, centrée, renvoyée
 * à la ligne quand la largeur manque ; survol lié entre l'anneau et la légende. B : les outils de
 * destruction (`MatchToolsCard`, le graphe de l'Escouade et de Sessions). Deux colonnes égales ; une
 * carte seule prend toute la largeur. Rend null sans aucune des deux — le MÊME prédicat
 * (`hasMatchFragData`) que celui lu par l'onglet pour poser son titre.
 *
 * Non gaté : Infinite = classes sans Spartan ; Halo 5 = avec (la capability décide côté backend via
 * native_kill_mechanics).
 */
import { useState } from 'react'

import { FragClassLegend, FragSunburst } from '@/components/charts/FragSunburst'
import { ObjectifFrame } from '@/features/squad/objectif/ObjectifFrame'
import type { FragDistribution, SquadWeaponTools } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { hasFragSunburst, hasMatchFragData, hasWeaponTools } from './blockPredicates'
import { MATCH_EMPRISE_TEXT } from './matchEmpriseText'
import { MatchToolsCard } from './MatchToolsCard'

/** Largeur maximale de l'anneau : au-delà, il grossit sans rien montrer de plus. */
const DONUT_MAX_WIDTH = 480

interface Props {
  distribution?: FragDistribution | null
  tools?: SquadWeaponTools | null
  locale: Locale
}

export function MatchFragCard({ distribution, tools, locale }: Props) {
  if (!hasMatchFragData(distribution, tools)) return null
  const both = hasFragSunburst(distribution) && hasWeaponTools(tools)
  return (
    <div className={both ? 'grid grid-cols-1 gap-4 lg:grid-cols-2' : ''}>
      {hasFragSunburst(distribution) && <MatchFragDonut distribution={distribution} locale={locale} />}
      {hasWeaponTools(tools) && <MatchToolsCard tools={tools} locale={locale} />}
    </div>
  )
}

/** Carte A : titre et aide, l'anneau nu, la légende des classes en pied, centrée. */
function MatchFragDonut({ distribution, locale }: { distribution: FragDistribution | null | undefined; locale: Locale }) {
  const [hovered, setHovered] = useState<string | null>(null)
  const t = MATCH_EMPRISE_TEXT[locale].squad.performanceCharts
  return (
    <ObjectifFrame
      title={t.fragBreakdownTitle}
      info={t.fragBreakdownInfo}
      testId="match-frag-donut"
      legend={<FragClassLegend distribution={distribution} hoveredClass={hovered} onClassHover={setHovered} />}
    >
      <FragSunburst
        distribution={distribution}
        title={t.fragBreakdownTitle}
        bare
        hideCenterLabel
        legendSide="none"
        maxWidthPx={DONUT_MAX_WIDTH}
        externalHoveredClass={hovered}
        onClassHover={setHovered}
      />
    </ObjectifFrame>
  )
}
