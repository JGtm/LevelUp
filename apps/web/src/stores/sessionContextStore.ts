/**
 * useSessionContextStore — contexte de match (Solo / Escouade / Mixte) propre à
 * la page Sessions.
 *
 * Volontairement SÉPARÉ du `soloFilterStore` (filterContext partagé par
 * Timeseries / History / Explorer) : le choix de contexte ne doit affecter que
 * la page Sessions, jamais les autres pages qui consomment le même FilterOmnibar.
 *
 * Défaut 'solo' = comportement historique (le sélecteur de sessions ne montrait
 * que les sessions solo). En mémoire uniquement (réinitialisé à 'solo' au
 * rechargement) pour rester sur un défaut sûr.
 */
import { create } from 'zustand'

export type SessionMatchContext = 'solo' | 'squad' | 'all'

interface SessionContextState {
  matchContext: SessionMatchContext
  setMatchContext: (c: SessionMatchContext) => void
}

export const useSessionContextStore = create<SessionContextState>((set) => ({
  matchContext: 'solo',
  setMatchContext: (matchContext) => set({ matchContext }),
}))
