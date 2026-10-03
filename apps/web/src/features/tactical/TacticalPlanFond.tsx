/**
 * TacticalPlanFond — le CADRE du plan et son FOND de carte (`<img>`), rien d'autre
 * (retours rejeu L2, 2026-09-23).
 *
 * IL NE REÇOIT QUE LA CARTE ET LE CALAGE (`aspect`), JAMAIS UNE PROP DE LA LECTURE : le fond
 * ne dépend que de la carte, et le rendre sous la condition de la lecture le faisait
 * démonter puis remonter à chaque changement de question ou de filtre (le DOM reconstruit,
 * l'image redécodée — constat utilisateur « le fond clignote »). Le calque de chaleur,
 * l'état vide et les indicateurs sont des ENFANTS posés par-dessus : ils changent, le
 * `<img>` reste le même nœud du premier chargement au départ de la carte.
 *
 * LE FOND ET LE CALQUE PARTAGENT LE MÊME CADRE MONDE (celui du calage, cf.
 * `TacticalPlanCard`) : le conteneur prend son rapport, donc `object-cover` n'y rogne rien.
 */
import type { ReactNode } from 'react'

import { useTacticalMapBackgroundUrl } from './queries'
import { PLAN_HAUTEUR_MAX_PX } from './tacticalView.logic'

export interface TacticalPlanFondProps {
  playerSlug: string
  mapId: string
  /** Rapport largeur/hauteur du cadre — celui du calage du fond (`aspectDuPlan`). */
  aspect: number
  children?: ReactNode
}

export function TacticalPlanFond({ playerSlug, mapId, aspect, children }: TacticalPlanFondProps) {
  const fond = useTacticalMapBackgroundUrl(playerSlug, mapId)
  // Hauteur BORNÉE par une largeur maximale (`rapport x plafond`) : le rapport n'est jamais
  // déformé (cf. `PLAN_HAUTEUR_MAX_PX`).
  const cadre = { aspectRatio: aspect, maxWidth: `${aspect * PLAN_HAUTEUR_MAX_PX}px` }
  return (
    <div
      className="relative w-full overflow-hidden rounded-md bg-muted"
      style={cadre}
      data-testid="tactical-plan-frame"
    >
      {fond && <img src={fond} alt="" aria-hidden className="h-full w-full object-cover" />}
      {children}
    </div>
  )
}
