/**
 * Route /players/$playerSlug/ — layout joueur partagé.
 *
 * Rend un <Outlet /> pour les sous-routes.
 * Vérifie que le playerSlug existe dans les joueurs disponibles.
 */
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useEffect } from 'react'
import { NavL2 } from '@/components/shell/NavL2'
import { useFiltersResolve } from '@/features/filters/queries'

export const Route = createFileRoute('/players/$playerSlug')({
  // Guard synchrone avant rendu — bloque les accès directs par URL
  beforeLoad: () => {
    const { isBootstrapped, setupRequired, authMode, currentUsername } = useAppShellStore.getState()
    if (!isBootstrapped) return // Bootstrap pas encore terminé — __root gère l'écran de chargement
    if (setupRequired) throw redirect({ to: '/setup' })
    // Rediriger vers login uniquement en mode password sans utilisateur connecté
    if (authMode === 'password' && !currentUsername) throw redirect({ to: '/login' })
  },
  component: PlayerLayout,
})

function PlayerLayout() {
  const { playerSlug } = Route.useParams()
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)

  // Synchroniser le joueur actif avec le slug de la route
  useEffect(() => {
    if (currentPlayer?.player_slug === playerSlug) return
    const player = availablePlayers.find((p) => p.player_slug === playerSlug)
    if (player) setCurrentPlayer(player)
  }, [playerSlug, availablePlayers, currentPlayer, setCurrentPlayer])

  // Résolution du filterContext côté backend → alimente
  // resolvedContext.session_options et resolvedContext.available_options
  // dans le globalFilterStore, consommés par NavL2/FilterPanel/SquadLayout.
  useFiltersResolve(playerSlug)

  return (
    <div className="flex flex-col">
      {/* Bandeau contextuel L2 : sous-onglets Stats + filtres (Stats/Escouade uniquement) */}
      <NavL2 />
      <Outlet />
    </div>
  )
}
