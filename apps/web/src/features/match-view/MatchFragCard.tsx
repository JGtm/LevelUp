/**
 * MatchFragCard — la rangée « Répartition des frags » | « Outils de destruction » du joueur de la page
 * sur la Vue match (plan PLAN_MATCHVIEW_EMPRISE_2026-10-06, cartes A et B, D10).
 *
 * A : l'anneau hiérarchique classe → rôle (`FragSunburst`), inchangé (décision utilisateur du
 * 2026-10-06). B : les outils de destruction (`MatchToolsCard`, le graphe de l'Escouade et de
 * Sessions). Deux colonnes égales ; une carte seule prend toute la largeur. Rend null sans aucune
 * des deux — le MÊME prédicat (`hasMatchFragData`) que celui lu par l'onglet pour poser son titre.
 *
 * Non gaté : Infinite = classes sans Spartan ; Halo 5 = avec (la capability décide côté backend via
 * native_kill_mechanics).
 */
import { FragSunburst } from '@/components/charts/FragSunburst'
import type { FragDistribution, SquadWeaponTools } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { hasFragSunburst, hasMatchFragData, hasWeaponTools } from './blockPredicates'
import { MatchToolsCard } from './MatchToolsCard'

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
      {hasFragSunburst(distribution) && <FragSunburst distribution={distribution} hideCenterLabel maxWidthPx={480} legendSide="left" />}
      {hasWeaponTools(tools) && <MatchToolsCard tools={tools} locale={locale} />}
    </div>
  )
}
