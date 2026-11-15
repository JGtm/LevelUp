/**
 * État réel de la synchronisation d'un joueur dont l'accueil n'a aucun match.
 *
 * Avant : « Synchronisation en cours » s'affichait dès que le total de matchs était nul, qu'une
 * synchronisation tourne ou non. Les sources sont celles du serveur : la connexion du compte
 * (`reauth_required` du bootstrap, seul signal d'un jeton mort), le titre en pause (profil) et
 * `GET /players/{slug}/sync/status` (en vol toutes sources, job du JOUEUR, délai du sync manuel).
 */
export type FirstSyncKind =
  /** L'état du serveur n'est pas encore arrivé : rien à dire. */
  | 'loading'
  /** L'état du serveur n'a pas pu être lu (panne, réseau, réponse d'un autre joueur) : le dire et proposer de relire. */
  | 'status_error'
  /** Le jeton du compte est mort : aucune synchronisation ne peut partir (compte du visiteur, qui peut reconnecter). */
  | 'waiting_connection'
  /** La synchronisation de ce titre est en pause pour ce joueur. */
  | 'paused'
  /** Une synchronisation de ce joueur tourne (job de l'API, synchronisation automatique ou suivi en direct). */
  | 'running'
  /** Le dernier job connu a échoué, a été interrompu ou annulé. */
  | 'failed'
  /** Le dernier job connu est terminé et il n'y a toujours aucun match. */
  | 'empty'
  /** Rien ne tourne et aucun job récent n'est connu. */
  | 'idle'

export interface FirstSyncInput {
  /** `reauth_required` du bootstrap : le jeton du compte connecté est mort. */
  reauthRequired: boolean
  /**
   * Le joueur affiché est celui du compte connecté (`own_halo_xuid` du bootstrap) : `reauth_required` ne concerne que
   * ce compte. La synchronisation d'un AUTRE joueur passe par le pool de jetons, pas par la session, même pour un
   * administrateur ou quand la propriété des profils n'est pas appliquée (`canWrite` y est vrai).
   */
  isOwnAccount: boolean
  /** La session peut écrire sur le joueur affiché (son profil) : seul cas où reconnecter a un sens. */
  canWrite: boolean
  /** `sync_enabled` du profil affiché ; `undefined` = profil inconnu (pas de pause déclarée). */
  syncEnabled: boolean | undefined
  /** État servi pour le JOUEUR affiché ; `null` tant qu'il n'est pas arrivé. */
  server: { syncing: boolean; jobStatus: string | undefined } | null
  /** La lecture de l'état a échoué, ou rend l'état d'un autre joueur. */
  statusError: boolean
}

const FAILED_STATUSES: ReadonlySet<string> = new Set(['failed', 'cancelled', 'interrupted'])

/**
 * Ordre de priorité : un jeton mort et une pause expliquent à eux seuls l'absence de synchronisation,
 * ils priment sur un job ancien ; viennent ensuite ce qui tourne, puis le dernier job, puis « rien ».
 * « En attente de connexion » ne s'affiche que sur le profil du compte connecté, et pour qui peut y écrire : le
 * jeton mort est celui de ce compte, pas celui du joueur qu'on regarde.
 */
export function resolveFirstSyncState(input: FirstSyncInput): { kind: FirstSyncKind } {
  if (input.reauthRequired && input.canWrite && input.isOwnAccount) return { kind: 'waiting_connection' }
  if (input.syncEnabled === false) return { kind: 'paused' }
  const server = input.server
  if (server === null) return { kind: input.statusError ? 'status_error' : 'loading' }
  const status = server.jobStatus
  const jobActive = status !== undefined && status !== 'succeeded' && !FAILED_STATUSES.has(status)
  if (server.syncing || jobActive) return { kind: 'running' }
  if (status === 'succeeded') return { kind: 'empty' }
  if (status !== undefined) return { kind: 'failed' }
  return { kind: 'idle' }
}
