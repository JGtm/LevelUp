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
      // Tout ce qui dépend de la liste : dropdown Escouade, coloration vue match,
      // barres Prestige d'escouade.
      queryClient.invalidateQueries({ queryKey: queryKeys.playerFriends(slug) })
      queryClient.invalidateQueries({ queryKey: queryKeys.teammatesAll })
      queryClient.invalidateQueries({ queryKey: ['prestige'] })
      queryClient.invalidateQueries({ queryKey: ['match-view'] })
    },
  })
}

/**
 * Raccourci des consommateurs qui n'ont besoin que des gamertags (coloration de
 * la vue match, présélection Escouade, escouade Prestige) : liste vide tant que
 * la requête n'a pas abouti.
 */
export function useFriendGamertags(slug: string | undefined): string[] {
  const { data } = usePlayerFriends(slug)
  return data?.gamertags ?? []
}
