/**
 * usePresence — « qui est en jeu en ce moment ».
 *
 * Alimente le sélecteur de joueur de la NavL1 : une manette à côté des joueurs
 * en partie, et le nombre d'amis (liste des Réglages) actuellement en jeu.
 *
 * Rafraîchi toutes les 30 s, y compris onglet en arrière-plan : la valeur perd
 * tout son sens si elle date. Le serveur, lui, ne réinterroge Xbox qu'une fois
 * par TTL (45 s) — le poll ne coûte donc pas un appel Xbox par tick.
 *
 * Jamais bloquant : en cas d'échec (endpoint absent, watcher éteint), le hook
 * rend un instantané vide et l'UI n'affiche simplement aucune manette.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { PlayerPresence, PresenceSnapshot } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

/** Intervalle de rafraîchissement (ms). Cohérent avec le poll de présence du watcher. */
export const PRESENCE_REFETCH_MS = 30_000

export interface PresenceState {
  /** Joueurs suivis dont l'état est connu, indexés par player_slug. */
  byPlayerSlug: Map<string, PlayerPresence>
  /** Nombre d'amis en jeu (0 si inconnu). */
  friendsInGame: number
}

const EMPTY: PresenceState = { byPlayerSlug: new Map(), friendsInGame: 0 }

export function usePresence(): PresenceState {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)

  const { data } = useQuery({
    queryKey: queryKeys.presence(titleSlug),
    queryFn: () => api.get<PresenceSnapshot>('/presence'),
    // staleTime aligné sur l'intervalle : un remontage de la nav (navigation
    // interne) réutilise la réponse en cours de validité au lieu de re-fetcher.
    staleTime: PRESENCE_REFETCH_MS,
    refetchInterval: PRESENCE_REFETCH_MS,
    refetchIntervalInBackground: true,
    enabled: !!titleSlug,
    // Une présence indisponible n'est pas une erreur à retenter en rafale : le
    // prochain tick réessaiera de toute façon.
    retry: false,
  })

  if (!data) return EMPTY

  const byPlayerSlug = new Map<string, PlayerPresence>()
  // `players` est `PlayerPresence[] | null` au contrat (toute tranche Go l'est) :
  // comblé ici, une fois, à la frontière.
  for (const p of data.players ?? []) {
    byPlayerSlug.set(p.player_slug, p)
  }
  return { byPlayerSlug, friendsInGame: data.friends_in_game ?? 0 }
}
