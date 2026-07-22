/**
 * Route /players/$playerSlug/ — layout joueur partagé.
 *
 * Rend un <Outlet /> pour les sous-routes.
 * Vérifie que le playerSlug existe dans les joueurs disponibles.
 */
import { createFileRoute, Navigate, Outlet, redirect } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useEffect, useRef } from 'react'
import { NavL2 } from '@/components/shell/NavL2'
import { resolvePlayerFallback } from '@/components/shell/shellNavigation'
import { useFiltersResolve, useFollowLatestSession } from '@/features/filters/queries'
import { queryKeys } from '@/lib/query/keys'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug')({
  // Guard synchrone avant rendu — bloque les accès directs par URL.
  //
  // Redirige automatiquement vers le joueur courant si l'URL pointe vers un
  // slug inexistant (ex: URL héritée d'un mode démo, ou tampon navigateur après
  // suppression d'un profil joueur). Sans ce guard, le frontend boucle sur
  // l'erreur API "joueur introuvable" sans recovery automatique.
  beforeLoad: ({ params }) => {
    const {
      isBootstrapped,
      setupRequired,
      authMode,
      currentUsername,
      availablePlayers,
      currentPlayer,
    } = useAppShellStore.getState()
    if (!isBootstrapped) return // Bootstrap pas encore terminé — __root gère l'écran de chargement
    if (setupRequired) throw redirect({ to: '/setup' })
    // Rediriger vers login uniquement en mode password sans utilisateur connecté
    if (authMode === 'password' && !currentUsername) throw redirect({ to: '/login' })

    // Slug inexistant → rediriger vers le joueur courant ou le premier disponible.
    if (availablePlayers.length > 0) {
      const slugExists = availablePlayers.some((p) => p.player_slug === params.playerSlug)
      if (!slugExists) {
        const fallbackSlug = currentPlayer?.player_slug ?? availablePlayers[0].player_slug
        throw redirect({
          to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
          params: { ...params, playerSlug: fallbackSlug },
        })
      }
    }
  },
  component: PlayerLayout,
})

function PlayerLayout() {
  const params = Route.useParams()
  const { playerSlug } = params
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)

  // Synchroniser le joueur actif avec le slug de la route
  useEffect(() => {
    if (currentPlayer?.player_slug === playerSlug) return
    const player = availablePlayers.find((p) => p.player_slug === playerSlug)
    if (player) setCurrentPlayer(player)
  }, [playerSlug, availablePlayers, currentPlayer, setCurrentPlayer])

  // Résolution du filterContext SOLO côté backend → alimente le store solo
  // (consommé par NavL2/FilterOmnibar/PeriodSessionRail). Le store squad est
  // résolu séparément depuis SquadLayout (qui appelle aussi useFiltersResolve).
  useFiltersResolve(playerSlug, useSoloFilterStore)

  // Atterrissage sur la dernière session : piloté par l'état (resolvedContext),
  // tant que rien n'est épinglé manuellement. Le snap squad est monté côté
  // SquadLayout (où le store squad est résolu). Cf. useFollowLatestSession.
  useFollowLatestSession(playerSlug, useSoloFilterStore, 'solo')

  // À la fin d'un sync (transition activeSyncJobId string → null), invalider la
  // résolution de filtres pour rafraîchir `resolvedContext` (rien d'autre ne
  // l'invalide — useJobToasts n'émet que des toasts). Le snap proprement dit est
  // ensuite géré par useFollowLatestSession quand le nouveau resolvedContext arrive.
  const queryClient = useQueryClient()
  const prevSyncJobId = useRef<string | null>(activeSyncJobId)
  useEffect(() => {
    const prev = prevSyncJobId.current
    prevSyncJobId.current = activeSyncJobId
    if (prev === null || activeSyncJobId !== null) return
    // Préfixe ['filters-resolve', playerSlug] : couvre les stores solo ET squad
    // (cf. queryKeys.filtersResolve, lib/query/keys.ts).
    queryClient.invalidateQueries({ queryKey: queryKeys.filtersResolveAll(playerSlug) })
  }, [activeSyncJobId, playerSlug, queryClient])

  // Filet fresh-load DÉCLARATIF (D-8, trou n°1 de la revue v2). Le beforeLoad ci-dessus
  // couvre les navigations SPA (store hydraté, exécuté au matching) mais NE re-tourne
  // PAS sur un simple re-render : au fresh-load, le store s'hydrate APRÈS le matching,
  // donc le beforeLoad a early-return sur `!isBootstrapped`. Une fois hydraté, si le
  // playerSlug d'URL est absent des joueurs disponibles, on redirige ici (pattern
  // index.tsx). Division : beforeLoad = SPA (synchrone) ; composant = fresh-load
  // (déclaratif). titleSlug + langue hérités via `...params`.
  const fallback = isBootstrapped
    ? resolvePlayerFallback(playerSlug, availablePlayers)
    : ({ kind: 'ok' } as const)
  if (fallback.kind === 'index') return <Navigate to="/" replace />
  if (fallback.kind === 'redirect') {
    return (
      <Navigate
        to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
        params={{ ...params, playerSlug: fallback.slug }}
        replace
      />
    )
  }

  return (
    <div className="flex flex-col">
      {/* NavL2 = onglets Stats + FilterOmnibar (Stats/Squad uniquement). Le rail
          de navigation période/session est rendu DANS NavL2 (Stats) et DANS
          SquadLayout (Squad), juste après leurs filtres respectifs, pour
          apparaître toujours en dessous des filtres. */}
      <NavL2 />
      <Outlet />
    </div>
  )
}
