/**
 * replayLoadState — ce que la page de rejeu affiche selon l'état de chargement de l'artefact et
 * de la vue du match.
 *
 * L'ABSENCE ET LA PANNE NE SONT PAS LA MÊME CHOSE. Seul un 404 dit que le match n'a pas de
 * rejeu ; toute autre erreur (serveur occupé, réseau) laisse le rejeu exister — la page le dit
 * et propose de réessayer, au lieu d'annoncer « aucun rejeu » sur un match qui en a un et dont
 * la tuile affichait l'icône.
 *
 * LE REJEU NE S'AFFICHE QU'AVEC SES IDENTITÉS. L'artefact dit ce qui s'est passé, la vue du match
 * qui sont les joueurs (camps, noms, tableau des scores) : montré avant elle, le rejeu n'a ni
 * pions, ni fiches, ni frise — un fond de carte seul. La page attend donc que la vue du match soit
 * TRANCHÉE (servie ou en échec) ; en échec, le rejeu se montre avec ce que l'artefact porte seul.
 */
export type ReplayLoadState = 'loading' | 'ready' | 'empty' | 'error'

interface ReplayQueryView {
  isLoading: boolean
  /** L'artefact porte au moins une piste. */
  hasTracks: boolean
  error: unknown
  /** La vue du match n'est ni servie ni en échec. */
  identityPending: boolean
}

const STATUS_NOT_FOUND = 404

export function replayLoadState({ isLoading, hasTracks, error, identityPending }: ReplayQueryView): ReplayLoadState {
  if (isLoading) return 'loading'
  if (hasTracks) return identityPending ? 'loading' : 'ready'
  if (error == null) return 'empty'
  const status = (error as { status?: unknown }).status
  return status === STATUS_NOT_FOUND ? 'empty' : 'error'
}
