import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Gamertag de référence d'une page qui montre des médias : celui du joueur courant, sinon
 * celui du profil `playerSlug` dans la liste des profils. Les vues médias le comparent au
 * propriétaire d'un média (`owner_gamertag`) pour n'offrir réassociation et suppression que
 * sur les siens, comme le serveur (propriétaire ou admin, `domain.CanModifyMedia`).
 *
 * Source unique du calcul : garde-rail `usePlayerGamertag.guard.test.ts`.
 */
export function usePlayerGamertag(playerSlug: string): string | undefined {
  return useAppShellStore(
    (s) => s.currentPlayer?.gamertag ?? s.availablePlayers.find((p) => p.player_slug === playerSlug)?.gamertag,
  )
}
