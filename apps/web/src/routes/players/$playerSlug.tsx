/**
 * Route /players/$playerSlug/ — layout joueur partagé.
 *
 * Rend un <Outlet /> pour les sous-routes.
 * Vérifie que le playerSlug existe dans les joueurs disponibles.
 */
import { createFileRoute, Outlet } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useEffect } from 'react'
import { KPIBar } from '@/components/shell/KPIBar'
import { NavL2 } from '@/components/shell/NavL2'

export const Route = createFileRoute('/players/$playerSlug')({
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

  return (
    <div className="flex flex-col">
      {/* Bandeau contextuel L2 : sous-onglets Stats + filtres (Stats/Escouade uniquement) */}
      <NavL2 />
      {/* KPI bar transverse */}
      <KPIBar />
      <Outlet />
    </div>
  )
}
