/**
 * Route /players/$playerSlug/ — layout joueur partagé.
 *
 * Rend un <Outlet /> pour les sous-routes.
 * Vérifie que le playerSlug existe dans les joueurs disponibles.
 */
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useEffect, useRef } from 'react'
import { NavL2 } from '@/components/shell/NavL2'
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

  // Résolution du filterContext SOLO côté backend → alimente le store solo
  // (consommé par NavL2/FilterOmnibar/PeriodSessionRail). Le store squad est
  // résolu séparément depuis SquadLayout (qui appelle aussi useFiltersResolve).
  const filtersResolve = useFiltersResolve(playerSlug, useSoloFilterStore)

  // Auto-snap-to-latest contextuel : sur la transition activeSyncJobId string
  // → null (sync vient de terminer), on scanne `all_sessions` et on snap chaque
  // store sur la dernière session de son `is_squad`. Permet à la page Stats
  // Solo d'auto-snap aux nouvelles sessions solo et à la page Escouade aux
  // nouvelles sessions squad — sans pollution croisée.
  const prevSyncJobId = useRef<string | null>(activeSyncJobId)
  useEffect(() => {
    const prev = prevSyncJobId.current
    prevSyncJobId.current = activeSyncJobId
    if (prev === null || activeSyncJobId !== null) return
    filtersLog.debug(`auto_snap:sync_complete prev_job=${prev}`)
    filtersResolve.refetch().then((r) => {
      const sessions = r.data?.session_options?.all_sessions ?? []
      const latestSolo = sessions.find((s) => !s.is_squad)
      const latestSquad = sessions.find((s) => s.is_squad)

      if (latestSolo) {
        const soloStore = useSoloFilterStore.getState()
        if (latestSolo.session_id !== soloStore.lastKnownLatestSessionId) {
          soloStore.autoSnapToLatestSession(latestSolo.session_id, true)
        } else {
          filtersLog.debug(`auto_snap:skipped scope=solo reason=unchanged session=${latestSolo.session_id}`)
        }
      } else {
        filtersLog.debug('auto_snap:skipped scope=solo reason=no_session')
      }

      if (latestSquad) {
        const squadStore = useSquadFilterStore.getState()
        if (latestSquad.session_id !== squadStore.lastKnownLatestSessionId) {
          squadStore.autoSnapToLatestSession(latestSquad.session_id, true)
        } else {
          filtersLog.debug(`auto_snap:skipped scope=squad reason=unchanged session=${latestSquad.session_id}`)
        }
      } else {
        filtersLog.debug('auto_snap:skipped scope=squad reason=no_session')
      }
    })
  }, [activeSyncJobId, filtersResolve])

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
