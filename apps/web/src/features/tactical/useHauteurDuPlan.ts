/**
 * useHauteurDuPlan — la hauteur que la fenêtre laisse au CADRE du plan : EXACTEMENT la règle de la toile
 * du rejeu 2D (`useReplayViewport`, pas une copie) — plancher 360 px, plafond 720 px, marge basse de
 * 16 px, pas de 8 px, mesure refaite à l'arrêt du redimensionnement.
 *
 * LE CHROME EST DÉDUIT, comme au rejeu : `carte` est l'élément qui porte le plan ET tout ce qui
 * l'accompagne (bandeau de titre, états, légende, zone « séquence » et sa barre de lecture), `cadre` est
 * le cadre du fond. La hauteur du cadre est ce qui reste sous le haut de `carte`, dans la zone qui défile,
 * une fois ce chrome et la marge basse retirés. Le pied du plan fait partie du chrome : la carte entière,
 * barre de lecture comprise, tient dans l'écran.
 */
import type { RefObject } from 'react'

import { useReplayViewport } from '@/features/match-replay/hooks/useReplayViewport'

export function useHauteurDuPlan(carte: RefObject<HTMLElement | null>, cadre: RefObject<HTMLElement | null>): number {
  return useReplayViewport(carte, cadre).freeHeight
}
