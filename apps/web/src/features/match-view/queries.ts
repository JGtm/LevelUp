/**
 * Queries TanStack Query — Match View (Slice 4B).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchObjectiveEvent, MatchPlayerPosition, MatchViewResponse } from '@/lib/api/types'

export function useMatchView(playerSlug: string, matchId: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchView(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<MatchViewResponse>(`/players/${playerSlug}/matches/${matchId}`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
  })
}

/**
 * useMatchObjectiveEvents — événements d'objectif filmés (CTF captures, etc.).
 *
 * Endpoint : GET /players/{slug}/matches/{matchId}/objective-events.
 *
 * TROIS CONDITIONS, et la première est une porte de TITRE. La timeline d'objectif est une
 * PROJECTION DE L'ARTEFACT DE REJEU : un titre qui ne déclare pas `film.replay_artifact`
 * n'en produira jamais, et son endpoint rend un 503 `capability_not_supported` (depuis
 * v2(C.2)). Partir quand même, c'est demander à coup sûr une donnée qui n'existe pas —
 * deux requêtes vouées au 503 à chaque ouverture de l'onglet Chronologie sur halo_5 (revue
 * C-R1, condition n°2 : « sur halo_5, aucune requête de film n'est émise »). La porte est
 * ici, dans `enabled`, et non chez les consommateurs : c'est le seul endroit qui empêche la
 * requête de partir.
 *
 * Le 503 reste géré (`retry: false`, consommateurs en `data ?? []`) : la capability peut
 * être connue trop tard, ou la table manquer pour un titre qui a pourtant la clé.
 *
 * `enabled` (paramètre) : la donnée n'alimente que les charts de l'onglet Chronologie — la
 * page ne la tire donc que quand cet onglet est actif (défaut `true` pour les appelants qui
 * l'affichent inconditionnellement).
 */
export function useMatchObjectiveEvents(playerSlug: string, matchId: string, enabled = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const titreProduitDesArtefacts = useDataCapability('film.replay_artifact')
  return useQuery({
    queryKey: queryKeys.matchObjectiveEvents(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<MatchObjectiveEvent[]>(
        `/players/${playerSlug}/matches/${matchId}/objective-events`,
      ),
    enabled: enabled && titreProduitDesArtefacts && !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
    retry: false,
  })
}

/**
 * useMatchPositions — positions joueurs keyframe v3 décodées du film (match-level).
 *
 * Endpoint : GET /players/{slug}/matches/{matchId}/positions.
 *
 * MÊME PORTE DE TITRE que sa jumelle ci-dessus, et la MÊME clé : les positions aux
 * images-clés sont l'autre projection de l'artefact de rejeu (`film.replay_artifact`), pas
 * les positions PAR KILL (`film.kill_positions`, une famille distincte qui alimente la
 * distance par arme). Sans la clé, l'endpoint rend un 503 : la requête ne part pas.
 *
 * Le 503 reste géré (`retry: false`, consommateurs en `data ?? []`) : un match non
 * backfillé d'un titre qui A la clé rend une réponse vide, pas une erreur — et la carte de
 * chaleur rend `null` sur une liste vide.
 *
 * `enabled` (paramètre) : la carte de chaleur qui consomme ces positions vit dans l'onglet
 * Chronologie — la page ne tire la donnée que quand cet onglet est actif.
 */
export function useMatchPositions(playerSlug: string, matchId: string, enabled = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const titreProduitDesArtefacts = useDataCapability('film.replay_artifact')
  return useQuery({
    queryKey: queryKeys.matchPositions(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<MatchPlayerPosition[]>(
        `/players/${playerSlug}/matches/${matchId}/positions`,
      ),
    enabled: enabled && titreProduitDesArtefacts && !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
    retry: false,
  })
}

interface MatchFavoriteResponse {
  player_slug: string
  match_id: string
  favorited: boolean
}

/**
 * useToggleMatchFavorite — bascule l'état favori d'un match.
 *
 * Endpoint : PATCH /players/{slug}/matches/{matchId}/favorite, body { favorited }.
 * Invalide la query matchView pour refléter le nouvel état dans le header.
 */
export function useToggleMatchFavorite(playerSlug: string, matchId: string) {
  const queryClient = useQueryClient()
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useMutation({
    mutationFn: (favorited: boolean) =>
      api.patch<MatchFavoriteResponse>(
        `/players/${playerSlug}/matches/${matchId}/favorite`,
        { favorited },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.matchView(playerSlug, titleSlug, matchId),
      })
    },
  })
}
