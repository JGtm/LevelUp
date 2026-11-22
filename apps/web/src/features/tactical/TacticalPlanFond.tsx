/**
 * TacticalPlanFond — le CADRE du plan et son FOND de carte (`<img>`, ou la toile du relief), rien d'autre.
 *
 * IL NE REÇOIT QUE LA CARTE, LE CALAGE (`aspect`), LA HAUTEUR DISPONIBLE, LE CADRAGE DU ZOOM ET L'IMAGE DU RELIEF,
 * JAMAIS UNE PROP DE LA LECTURE : le fond ne dépend que de la carte, et le rendre sous la condition
 * de la lecture le faisait démonter puis remonter à chaque changement de lecture ou de filtre
 * (constat utilisateur « le fond clignote »). Le calque de chaleur, les états et les indicateurs
 * sont des ENFANTS posés par-dessus : ils changent, le `<img>` reste le même nœud du premier
 * chargement au départ de la carte — le zoom ne fait que le déplacer (`cadrageDuFond`).
 *
 * LE RELIEF (`surface`) : le relief teinté, ou le plan / le rendu modulé par la coupe par étage, prend la place de
 * l'`<img>` tant qu'il est prêt (l'`<img>` reste monté, invisible) ; pendant sa lecture et s'il échoue (repli sur le
 * plan, sans rien dire), l'`<img>` est le fond. La toile du relief a le même calage que l'image.
 *
 * LE FOND ET LE CALQUE PARTAGENT LE MÊME CADRE MONDE (celui du calage) : la boîte prend son
 * rapport (`boiteDuPlan`), donc l'image n'y est ni rognée ni déformée. Sans carte (`mapId` vide),
 * la boîte garde le rapport par défaut, sans image.
 */
import type { ReactNode, RefObject } from 'react'

import type { ReliefSurface } from '@/lib/replay/useReliefView'

import { boiteDuPlan, type CadrageDuFond } from './plan.logic'
import { TacticalPlanSurface } from './TacticalPlanSurface'
import { useTacticalFondVariante } from './useTacticalFondVariante'

export interface TacticalPlanFondProps {
  playerSlug: string
  mapId: string
  /** Rapport largeur/hauteur du cadre — celui du calage du fond (`aspectDuPlan`). */
  aspect: number
  /** Le cadre lui-même : sa hauteur est celle que `useHauteurDuPlan` règle (le chrome s'en déduit). */
  cadreRef?: RefObject<HTMLDivElement | null>
  /** La hauteur que la fenêtre laisse au cadre, en px (`useHauteurDuPlan`). */
  hauteurMax: number
  /** La place de l'image dans le cadre au zoom courant (`cadrageDuFond`). */
  cadrage: CadrageDuFond
  /** L'image du relief (`TacticalRelief.surface`) qui prend la place du fond de base ; absente = le fond de base. */
  surface?: ReliefSurface | null
  children?: ReactNode
}

export function TacticalPlanFond({ playerSlug, mapId, aspect, hauteurMax, cadrage, cadreRef, surface = null, children }: TacticalPlanFondProps) {
  // L'image plan se superpose exactement au rendu (même calage, mêmes dimensions) : seul l'`<img>` change.
  const { url: fond } = useTacticalFondVariante(playerSlug, mapId)
  return (
    <div
      ref={cadreRef}
      className="relative flex-none overflow-hidden rounded-md bg-muted"
      style={boiteDuPlan(aspect, hauteurMax)}
      data-testid="tactical-plan-frame"
    >
      {/* `cadrage` porte la rotation de la carte (`transform`) : l'image tourne autour de son centre. */}
      {/* Sous la toile du relief l'image reste montée (même nœud, pas de clignotement) mais invisible : le relief la REMPLACE, il ne s'y superpose pas. */}
      {fond && <img src={fond} alt="" aria-hidden className="absolute max-w-none" style={surface ? { ...cadrage, visibility: 'hidden' } : cadrage} />}
      {surface && <TacticalPlanSurface surface={surface} cadrage={cadrage} />}
      {children}
    </div>
  )
}
