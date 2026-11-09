/**
 * « Plus tard » du bandeau de régénération des rejeux : l'offre est masquée pour la session. La session est celle
 * de la fenêtre (`sessionStorage`) : elle survit à un changement de page et à un rechargement, pas à la
 * fermeture de l'application. L'accès au stockage peut échouer (fenêtre privée, accès bloqué) : le bandeau reste
 * alors masqué jusqu'au rechargement, et réapparaît ensuite.
 */
import { create } from 'zustand'

const STORAGE_KEY = 'levelup.replayRegen.later'

function readStored(): boolean {
  try {
    return window.sessionStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

interface DismissedState {
  dismissed: boolean
  dismiss: () => void
}

export const useRegenDismissedStore = create<DismissedState>((set) => ({
  dismissed: readStored(),
  dismiss: () => {
    try {
      window.sessionStorage.setItem(STORAGE_KEY, '1')
    } catch {
      // Stockage indisponible : l'état en mémoire suffit pour la page courante.
    }
    set({ dismissed: true })
  },
}))

export function useRegenDismissed(): DismissedState {
  return useRegenDismissedStore()
}
