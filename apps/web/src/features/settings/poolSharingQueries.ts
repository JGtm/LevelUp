/**
 * Mutation TanStack Query — décision de mise en commun du jeton (PUT /me/pool-sharing).
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { components } from '@/lib/api/generated'

type SetPoolSharingOutput = components['schemas']['SetPoolSharingOutputBody']

/** Enregistre l'accord (true) ou le retrait (false) et reflète le résultat dans le shell. */
export function useSetPoolSharing() {
  const queryClient = useQueryClient()
  const setPoolSharing = useAppShellStore((s) => s.setPoolSharing)
  return useMutation({
    mutationFn: (granted: boolean) =>
      api.put<SetPoolSharingOutput>('/me/pool-sharing', { granted }),
    onSuccess: (out) => {
      setPoolSharing(out.pool_sharing)
      return queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    },
  })
}
