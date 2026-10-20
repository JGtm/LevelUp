/**
 * replayLoadState — ce que la page de rejeu affiche selon l'état de chargement de l'artefact.
 *
 * L'ABSENCE ET LA PANNE NE SONT PAS LA MÊME CHOSE. Seul un 404 dit que le match n'a pas de
 * rejeu ; toute autre erreur (serveur occupé, réseau) laisse le rejeu exister — la page le dit
 * et propose de réessayer, au lieu d'annoncer « aucun rejeu » sur un match qui en a un et dont
 * la tuile affichait l'icône.
 */
export type ReplayLoadState = 'loading' | 'ready' | 'empty' | 'error'

interface ReplayQueryView {
  isLoading: boolean
  /** L'artefact porte au moins une piste. */
  hasTracks: boolean
  error: unknown
}

const STATUS_NOT_FOUND = 404

export function replayLoadState({ isLoading, hasTracks, error }: ReplayQueryView): ReplayLoadState {
  if (isLoading) return 'loading'
  if (hasTracks) return 'ready'
  if (error == null) return 'empty'
  const status = (error as { status?: unknown }).status
  return status === STATUS_NOT_FOUND ? 'empty' : 'error'
}
