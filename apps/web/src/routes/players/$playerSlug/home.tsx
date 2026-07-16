/**
 * Route /players/$playerSlug/home — Accueil Mission Control.
 *
 * P8.6 (revue 2026-04-29) : loader prefetch la home page au navigation.
 * Élimine le flicker `useEffect` initial — le payload est déjà chaud quand
 * `<HomePage>` monte. La query React reste source de vérité (loader =
 * warmup, pas de retour de données dans le composant).
 */
import { createFileRoute } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomePageResponse } from '@/lib/api/types'
import { HomePage } from '@/features/home/HomePage'

export const Route = createFileRoute('/players/$playerSlug/home')({
  loader: ({ params, context }) => {
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.home(params.playerSlug, useAppShellStore.getState().currentTitleSlug),
      queryFn: () =>
        api.get<HomePageResponse>(`/players/${params.playerSlug}/pages/home`),
    })
  },
  component: HomePage,
})
