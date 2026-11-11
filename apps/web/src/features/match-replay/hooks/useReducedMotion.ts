/**
 * useReducedMotion — LA PRÉFÉRENCE DE MOUVEMENT RÉDUIT, lue une fois au montage.
 *
 * La feuille de style la respecte pour le DOM ; un canvas dessiné en JS n'est atteint par aucune
 * règle CSS, la préférence se lit donc ici. Extrait de `ReplayCanvas.tsx` sans changement de
 * comportement (le fichier est au plafond de taille : le panneau de joueur y ajoute son câblage).
 */
import { useMemo } from 'react'

export function useReducedMotion(): boolean {
  return useMemo(
    () => typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    [],
  )
}
