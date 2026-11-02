/**
 * usePlayerSwitch — bascule du joueur actif depuis un sélecteur du shell.
 *
 * Partagée par la barre haute (NavL1) et la barre latérale de bureau (SidebarNav) : un seul
 * comportement de navigation pour le même geste.
 */
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { resolvePlayerSwitch } from './shellNavigation'

export function usePlayerSwitch(): (slug: string) => void {
  const navigate = useNavigate()
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const pathname = useRouterState().location.pathname

  return (slug: string) => {
    const player = availablePlayers.find((p) => p.player_slug === slug)
    if (!player) return
    setCurrentPlayer(player)
    // Changement de JOUEUR (même titre) : sur une sous-page joueur, rester sur la même
    // route en ne changeant que le playerSlug (titre + langue hérités des params
    // courants) ; sinon aller à l'accueil title-scoped du nouveau joueur.
    if (resolvePlayerSwitch(pathname).kind === 'same-route') {
      void navigate({ to: '.', params: (prev) => ({ ...prev, playerSlug: slug }) })
    } else {
      void navigate({
        to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
        params: { titleSlug: currentTitleSlug, playerSlug: slug },
      })
    }
  }
}
