/**
 * Logique pure de la carte du paquet de ressources : de l'état du serveur à ce que la carte dit et propose.
 * Aucun texte ici : les vues désignent des entrées de `resourcePackText.ts`.
 */
import type { ResourcePackStatus } from './resourcePackQueries'

/** Ce que la carte affiche. */
export type ResourcePackView =
  | 'upToDate'
  | 'mismatch'
  | 'waitingGame'
  | 'downloading'
  | 'verifying'
  | 'ready'
  | 'failed'

export function resourcePackView(status: ResourcePackStatus): ResourcePackView {
  switch (status.phase) {
    case 'waiting_game':
      return 'waitingGame'
    case 'downloading':
      return 'downloading'
    case 'verifying':
      return 'verifying'
    case 'failed':
      return 'failed'
    case 'ready':
      return status.restart_needed ? 'ready' : 'upToDate'
    case 'current':
      return 'upToDate'
    case 'idle':
      return status.active && status.expected && status.active !== status.expected ? 'mismatch' : 'upToDate'
  }
}

/** Le bouton « Réessayer » : seulement quand il n'y a rien en cours et que le paquet n'est pas le bon. */
export function canRetryResourcePack(view: ResourcePackView): boolean {
  return view === 'failed' || view === 'mismatch'
}

/** Pourcentage borné pour la barre de progression. */
export function resourcePackPercent(status: ResourcePackStatus): number {
  return Math.max(0, Math.min(100, Math.round(status.percent)))
}
