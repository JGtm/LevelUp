/**
 * Route /players/$playerSlug/ — layout joueur partagé.
 *
 * Rend un <Outlet /> pour les sous-routes.
 * Vérifie que le playerSlug existe dans les joueurs disponibles.
 */
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useEffect, useRef } from 'react'
import { NavL2 } from '@/components/shell/NavL2'
import { SessionNavBar } from '@/components/shell/SessionNavBar'
import { useFiltersResolve } from '@/features/filters/queries'
import { log as filtersLog } from '@/features/filters/_logger'

export const Route = createFileRoute('/players/$playerSlug')({
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
          to: '/players/$playerSlug/home',
          params: { playerSlug: fallbackSlug },
        })
      }
    }
  },
  component: PlayerLayout,
})

function PlayerLayout() {
  const { playerSlug } = Route.useParams()
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

  // Résolution du filterContext côté backend → alimente
  // resolvedContext.session_options et resolvedContext.available_options
  // dans le globalFilterStore, consommés par SessionNavBar/FilterOmnibar/SquadLayout.
  const filtersResolve = useFiltersResolve(playerSlug)

  // Auto-snap-to-latest : sur la transition activeSyncJobId string → null
  // (sync vient de terminer), si une nouvelle session a été ingérée, on
  // bascule automatiquement le filtre sur la session la plus récente.
  // Le user reste libre de changer ensuite (manuel reset isAutoSnappingToLatest).
  const prevSyncJobId = useRef<string | null>(activeSyncJobId)
  useEffect(() => {
    const prev = prevSyncJobId.current
    prevSyncJobId.current = activeSyncJobId
    // Transition "était actif" → "idle"
    if (prev === null || activeSyncJobId !== null) return
    filtersLog.debug(`auto_snap:sync_complete prev_job=${prev}`)
    // Force un re-fetch immédiat pour récupérer les nouvelles sessions
    filtersResolve.refetch().then((r) => {
      const latestId = r.data?.session_options?.all_sessions?.[0]?.session_id
      if (!latestId) {
        filtersLog.debug('auto_snap:skipped reason=no_session')
        return
      }
      const store = useGlobalFilterStore.getState()
      if (latestId === store.lastKnownLatestSessionId) {
        filtersLog.debug(`auto_snap:skipped reason=unchanged session=${latestId}`)
        return
      }
      store.autoSnapToLatestSession(latestId, true)
    })
  }, [activeSyncJobId, filtersResolve])

  return (
    <div className="flex flex-col">
      {/* Barre de navigation de session (sticky h-12) — Stats/Escouade uniquement */}
      <SessionNavBar />
      {/* Bandeau contextuel L2 : sous-onglets Stats + filtres (Stats/Escouade uniquement) */}
      <NavL2 />
      <Outlet />
    </div>
  )
}
