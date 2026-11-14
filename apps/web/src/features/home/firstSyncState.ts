/**
 * État réel de la synchronisation d'un joueur dont l'accueil n'a aucun match.
 *
 * Avant : « Synchronisation en cours » s'affichait dès que le total de matchs était nul, qu'une
 * synchronisation tourne ou non (un joueur dont rien ne partait la lisait pendant une heure).
 * Les sources sont celles que l'API sert déjà : connexion du compte (bootstrap), titre en pause
 * (profil), job de synchronisation connu de la session (`GET /jobs/{id}`).
 */
export type FirstSyncKind =
  /** Le compte Xbox n'est pas (ou plus) connecté : aucune synchronisation ne peut partir. */
  | 'waiting_connection'
  /** La synchronisation de ce titre est en pause pour ce joueur. */
  | 'paused'
  /** Un job de synchronisation est en cours. */
  | 'running'
  /** Le dernier job connu a échoué, a été interrompu ou annulé. */
  | 'failed'
  /** Le dernier job connu est terminé et il n'y a toujours aucun match. */
  | 'empty'
  /** Aucun job connu : rien n'a été lancé depuis cette session. */
  | 'idle'

export interface FirstSyncInput {
  reauthRequired: boolean
  authState: string
  /** `sync_enabled` du profil courant ; `undefined` = profil inconnu (pas de pause déclarée). */
  syncEnabled: boolean | undefined
  /** Identifiant du job de synchronisation connu de la session ; `null` = aucun. */
  activeJobId: string | null
  /** Statut du job lu par `GET /jobs/{id}` ; `undefined` tant qu'il n'est pas arrivé. */
  jobStatus: string | undefined
}

export interface FirstSyncState {
  kind: FirstSyncKind
}

const FAILED_STATUSES: ReadonlySet<string> = new Set(['failed', 'cancelled', 'interrupted'])

/**
 * Ordre de priorité : une connexion manquante et une pause expliquent à elles seules l'absence de
 * synchronisation, elles priment sur un job ancien ; viennent ensuite le job de la session, puis
 * « rien lancé ».
 */
export function resolveFirstSyncState(input: FirstSyncInput): FirstSyncState {
  if (input.reauthRequired || input.authState !== 'ready') return { kind: 'waiting_connection' }
  if (input.syncEnabled === false) return { kind: 'paused' }
  if (!input.activeJobId) return { kind: 'idle' }
  if (input.jobStatus === undefined) return { kind: 'running' }
  if (input.jobStatus === 'succeeded') return { kind: 'empty' }
  if (FAILED_STATUSES.has(input.jobStatus)) return { kind: 'failed' }
  return { kind: 'running' }
}
