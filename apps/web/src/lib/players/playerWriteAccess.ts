/**
 * Droit d'écriture sur un profil joueur, vu du front (ADR 0029 : l'accès famille
 * est en LECTURE seule).
 *
 * Le serveur publie au bootstrap la liste des profils sur lesquels la session peut
 * écrire (`writable_player_slugs` : les siens, tous pour un admin, tous quand la
 * propriété n'est pas appliquée). Sur le profil d'un ami, les actions d'écriture
 * (synchronisation, exclusion, favoris, notifications, médias, Prestige, réglages du
 * titre) sont masquées : le serveur les refuserait en 403.
 *
 * Liste inconnue (`null` ou absente, bootstrap pas encore lu) : rien n'est masqué. Ce n'est
 * qu'un affichage ; l'autorisation reste décidée par le serveur.
 */
import { useAppShellStore } from '@/stores/appShellStore'

/** Règle pure : `slug` est-il inscriptible au vu de la liste publiée ? */
export function canWritePlayer(writable: readonly string[] | null | undefined, slug: string | undefined): boolean {
  if (writable == null) return true
  if (!slug) return false
  return writable.includes(slug)
}

/** Hook : la session peut-elle écrire sur le profil `slug` ? */
export function useCanWritePlayer(slug: string | undefined): boolean {
  const writable = useAppShellStore((s) => s.writablePlayerSlugs)
  return canWritePlayer(writable, slug)
}
