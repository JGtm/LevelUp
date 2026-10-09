/**
 * Queries TanStack Query — amis PAR JOUEUR.
 *
 * La liste d'amis appartient au profil joueur (clé xuid côté serveur), plus aux
 * réglages d'instance : elle se lit sur `/players/{slug}/friends`, accessible à
 * un utilisateur standard, là où `GET /settings` est réservé aux admins.
 *
 * `can_edit` vient de la réponse : le front n'interprète JAMAIS un 403 pour
 * décider d'afficher la liste en lecture seule.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

export interface PlayerFriends {
  xuid: string
  gamertags: string[]
  updated_at?: string
  can_edit: boolean
}

/**
 * Liste d'amis du joueur `slug`. Slug vide → requête désactivée.
 *
 * `enabled: !!slug` : l'endpoint `/players/{slug}/friends` est servi par
 * l'API depuis l'étape 3 du chantier « amis par joueur » (2026-09-15).
 */
export function usePlayerFriends(slug: string | undefined) {
  return useQuery({
    queryKey: queryKeys.playerFriends(slug ?? ''),
    queryFn: () => api.get<PlayerFriends>(`/players/${encodeURIComponent(slug ?? '')}/friends`),
    enabled: !!slug,
    staleTime: 30_000,
  })
}

/**
 * Remplacement COMPLET de la liste d'amis du joueur `slug` (PUT). Le serveur
 * normalise (trim, dédoublonnage, exclusion de soi) et relance le recompute
 * `is_with_friends` du joueur.
 */
export function useUpdatePlayerFriends(slug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (gamertags: string[]) =>
      api.put<PlayerFriends>(`/players/${encodeURIComponent(slug)}/friends`, { gamertags }),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.playerFriends(slug), data)
      // La coloration de la vue match, les barres Prestige d escouade et le rejeu
      // DERIVENT de cette liste cote client (usePlayerFriends). Ce que le SERVEUR
      // filtre sur les amis doit etre recharge : l Escouade, les rencontres de
      // Carriere (« hors amis »), les tuiles escouade de l Accueil (revue 2026-09-16 :
      // un ami tout juste ajoute restait « adversaire croise » 30 min en Carriere).
      queryClient.invalidateQueries({ queryKey: queryKeys.playerFriends(slug) })
      queryClient.invalidateQueries({ queryKey: queryKeys.teammatesAll })
      queryClient.invalidateQueries({ queryKey: queryKeys.careerAll(slug) })
      queryClient.invalidateQueries({ queryKey: queryKeys.homeAll(slug) })
    },
  })
}

// Reference STABLE : un `?? []` neuf a chaque rendu casserait les useMemo aval
// (le rejeu 2D se re-rend toutes les 150 ms en lecture).
const NO_FRIENDS: readonly string[] = []

/** Gamertags d'amis + l'issue de leur lecture (cf. `useFriendGamertagsState`). */
export interface FriendGamertagsState {
  gamertags: readonly string[]
  /** La liste est arrivée — éventuellement VIDE, ce qui est une réponse. */
  isSuccess: boolean
  /** La lecture a échoué : `gamertags` vaut alors la liste vide. */
  isError: boolean
}

/**
 * Gamertags d'amis AVEC l'issue de la requête. Sert au consommateur qui doit
 * distinguer « liste vide » de « pas encore là » : l'Escouade n'envoie sa requête
 * lourde qu'une fois la composition initiale connue (lot perf L4a, D4.2,
 * 2026-09-23) — une liste d'amis vide RÉSOLUE est une composition (exploration
 * sans coéquipier), une liste pas encore arrivée n'en est pas une.
 */
export function useFriendGamertagsState(slug: string | undefined): FriendGamertagsState {
  const { data, isSuccess, isError } = usePlayerFriends(slug)
  return { gamertags: data?.gamertags ?? NO_FRIENDS, isSuccess, isError }
}

/**
 * Raccourci des consommateurs qui n'ont besoin que des gamertags (coloration de
 * la vue match, escouade Prestige, rejeu) : liste vide tant que la requête n'a
 * pas abouti.
 */
export function useFriendGamertags(slug: string | undefined): readonly string[] {
  return useFriendGamertagsState(slug).gamertags
}
