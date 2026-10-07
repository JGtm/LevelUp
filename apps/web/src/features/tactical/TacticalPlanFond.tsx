/**
 * TacticalPlanFond — le CADRE du plan et son FOND de carte (`<img>`), rien d'autre.
 *
 * IL NE REÇOIT QUE LA CARTE ET LE CALAGE (`aspect`), JAMAIS UNE PROP DE LA LECTURE : le fond ne
 * dépend que de la carte, et le rendre sous la condition de la lecture le faisait démonter puis
 * remonter à chaque changement de lecture ou de filtre (constat utilisateur « le fond clignote »).
 * Le calque de chaleur, les états et les indicateurs sont des ENFANTS posés par-dessus : ils
 * changent, le `<img>` reste le même nœud du premier chargement au départ de la carte.
 *
 * LE FOND ET LE CALQUE PARTAGENT LE MÊME CADRE MONDE (celui du calage) : la boîte prend son
 * rapport (`boiteDuPlan`), donc `object-cover` n'y rogne rien. Sans carte (`mapId` vide), la boîte
 * garde le rapport par défaut, sans image.
 */
import type { ReactNode } from 'react'

import { boiteDuPlan } from './plan.logic'
import { useTacticalMapBackgroundUrl } from './queries'

export interface TacticalPlanFondProps {
  playerSlug: string
  mapId: string
  /** Rapport largeur/hauteur du cadre — celui du calage du fond (`aspectDuPlan`). */
  aspect: number
  children?: ReactNode
}

export function TacticalPlanFond({ playerSlug, mapId, aspect, children }: TacticalPlanFondProps) {
  const fond = useTacticalMapBackgroundUrl(playerSlug, mapId)
  return (
    <div
      className="relative flex-none overflow-hidden rounded-md bg-muted"
      style={boiteDuPlan(aspect)}
      data-testid="tactical-plan-frame"
    >
      {fond && <img src={fond} alt="" aria-hidden className="h-full w-full object-cover" />}
      {children}
    </div>
  )
}
