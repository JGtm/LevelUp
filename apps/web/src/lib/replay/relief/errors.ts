/**
 * relief/errors — L'ERREUR TYPEE DU RELIEF : tout ce qui rend un relief illisible.
 *
 * Une erreur du relief n'est jamais fatale pour la vue : l'appelant journalise et RETOMBE SUR LE PLAN
 * (conception §6.1). Le `code` dit quelle garde a refusé, pour le journal et pour les tests ; le texte
 * n'est pas affiché à l'utilisateur.
 */

/** Les gardes qui refusent un relief. */
export type ReliefErrorCode =
  /** L'image n'a pas pu être décodée ou relue par le navigateur. */
  | 'image-decode'
  /** Les dimensions de l'image ne sont pas celles du calage. */
  | 'image-size'
  /** Une alpha intermédiaire : le navigateur a prémultiplié ou converti. */
  | 'image-alpha'
  /** Un bit réservé de B est posé, ou une hauteur inconnue porte du contenu. */
  | 'image-flags'
  /** Le JSON n'est pas un relief lisible. */
  | 'data-invalid'
  /** Le JSON porte une version de schéma que cette version du web ne sait pas lire. */
  | 'data-version'
  /** La couleur de base de la teinte n'est pas lisible. */
  | 'tint-invalid'
  /** Aucune carte n'est chargée dans le moteur. */
  | 'not-loaded'
  /** Le worker n'a pas pu servir une demande. */
  | 'worker-failed'

/** L'erreur levée par les décodeurs et le calculateur du relief. */
export class ReliefError extends Error {
  readonly code: ReliefErrorCode

  constructor(code: ReliefErrorCode, message: string) {
    super(message)
    this.name = 'ReliefError'
    this.code = code
  }
}

/** isReliefError reconnaît une erreur du relief (le client du worker en reconstruit une à la réception). */
export function isReliefError(err: unknown): err is ReliefError {
  return err instanceof ReliefError
}
