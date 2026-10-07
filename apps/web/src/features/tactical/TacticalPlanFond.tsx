/**
 * TacticalPlanFond — le CADRE du plan et son FOND de carte (`<img>`), rien d'autre.
 *
 * IL NE REÇOIT QUE LA CARTE, LE CALAGE (`aspect`), LA HAUTEUR DISPONIBLE ET LE CADRAGE DU ZOOM,
 * JAMAIS UNE PROP DE LA LECTURE : le fond ne dépend que de la carte, et le rendre sous la condition
 * de la lecture le faisait démonter puis remonter à chaque changement de lecture ou de filtre
 * (constat utilisateur « le fond clignote »). Le calque de chaleur, les états et les indicateurs
 * sont des ENFANTS posés par-dessus : ils changent, le `<img>` reste le même nœud du premier
 * chargement au départ de la carte — le zoom ne fait que le déplacer (`cadrageDuFond`).
 *
 * LE FOND ET LE CALQUE PARTAGENT LE MÊME CADRE MONDE (celui du calage) : la boîte prend son
 * rapport (`boiteDuPlan`), donc l'image n'y est ni rognée ni déformée. Sans carte (`mapId` vide),
 * la boîte garde le rapport par défaut, sans image.
 */
import type { ReactNode } from 'react'

import { boiteDuPlan, type CadrageDuFond } from './plan.logic'
import { useTacticalMapBackgroundUrl } from './queries'

export interface TacticalPlanFondProps {
  playerSlug: string
  mapId: string
  /** Rapport largeur/hauteur du cadre — celui du calage du fond (`aspectDuPlan`). */
  aspect: number
  /** La hauteur que la fenêtre laisse au cadre, en px (`useHauteurDuPlan`). */
  hauteurMax: number
  /** La place de l'image dans le cadre au zoom courant (`cadrageDuFond`). */
  cadrage: CadrageDuFond
  children?: ReactNode
}

export function TacticalPlanFond({ playerSlug, mapId, aspect, hauteurMax, cadrage, children }: TacticalPlanFondProps) {
  const fond = useTacticalMapBackgroundUrl(playerSlug, mapId)
  return (
    <div
      className="relative flex-none overflow-hidden rounded-md bg-muted"
      style={boiteDuPlan(aspect, hauteurMax)}
      data-testid="tactical-plan-frame"
    >
      {fond && <img src={fond} alt="" aria-hidden className="absolute max-w-none" style={cadrage} />}
      {children}
    </div>
  )
}
