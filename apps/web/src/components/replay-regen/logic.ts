/**
 * Logique pure de la régénération des rejeux : la phase d'affichage tirée de l'état du serveur, la visibilité du
 * bandeau et le pourcentage de la barre de progression. Aucun accès réseau ni React.
 */
import type { ReplayRegenStatus } from './queries'

export type RegenPhase =
  | 'checking' // le serveur compte encore
  | 'upToDate' // rien en retard
  | 'offer' // des rejeux en retard, la file n'est pas lancée
  | 'preparing' // lancée, la liste se prépare
  | 'running'
  | 'compressing' // fin de passe : compression des rejeux bruts que la file n'a pas pu refaire
  | 'paused'
  | 'waitingGame'
  | 'compressionPaused' // la pause demandée retient la compression, pas une refonte
  | 'compressionWaitingGame' // la partie en cours retient la compression, pas une refonte

export function regenPhase(status: ReplayRegenStatus | undefined): RegenPhase {
  if (!status) return 'checking'
  if (status.running) {
    if (status.total === 0 && !status.compressing) return 'preparing'
    // La compression passe par la même porte que la refonte (pause, partie en cours) : quand elle tourne, c'est
    // elle qui attend, même si total vaut 0.
    if (status.paused) return status.compressing ? 'compressionPaused' : 'paused'
    if (status.waiting_for_game) return status.compressing ? 'compressionWaitingGame' : 'waitingGame'
    return status.compressing ? 'compressing' : 'running'
  }
  if (!status.checked) return 'checking'
  return status.stale > 0 || status.raw > 0 ? 'offer' : 'upToDate'
}

/** La file est lancée (préparation, refonte, compression, pause ou attente comprises). */
export function regenPhaseActive(phase: RegenPhase): boolean {
  return (
    phase === 'preparing' ||
    phase === 'running' ||
    phase === 'compressing' ||
    phase === 'paused' ||
    phase === 'waitingGame' ||
    phase === 'compressionPaused' ||
    phase === 'compressionWaitingGame'
  )
}

/**
 * Les phases où le bandeau de la page de rejeu parle. « Plus tard » ne masque que l'offre : une file lancée par
 * l'utilisateur reste visible tant qu'elle tourne.
 */
export function bannerPhaseVisible(phase: RegenPhase, dismissed: boolean): boolean {
  if (phase === 'offer') return !dismissed
  return regenPhaseActive(phase)
}

/** Part faite de la file, de 0 à 100 ; 0 tant que la liste se prépare. */
export function regenPercent(status: Pick<ReplayRegenStatus, 'done' | 'total'>): number {
  if (status.total <= 0) return 0
  return Math.min(100, Math.round((status.done / status.total) * 100))
}
