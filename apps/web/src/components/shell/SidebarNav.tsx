/**
 * SidebarNav — barre latérale de l'application de bureau (capacité `can_use_compact_chrome`).
 *
 * Remplace la barre haute du site (NavL1) : mêmes sections et mêmes onglets que le tiroir de la vue
 * mobile (`NavSectionList`, `useVisibleNavSections`). Dépliée (240 px) ou repliée en icônes
 * (`SidebarRail`, 56 px) selon `useSidebarCollapse` : bouton au pied de la barre, `Ctrl+B`, repli
 * automatique d'une fenêtre étroite. Dépliée, de haut en bas : marque,
 * sélecteur de jeu, joueur actif et cloche, sections, puis le pied de barre (`SidebarFooter` : Aide,
 * Paramètres, bascule de thème, « À propos », version, et les deux panneaux qui remplacent les
 * mini-onglets du site : référentiels et signalement d'un problème).
 */
import { useRouterState } from '@tanstack/react-router'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { useAppShellStore } from '@/stores/appShellStore'
import { SidebarPanel } from './SidebarPanel'
import { SidebarRail } from './SidebarRail'
import { useSidebarCollapse } from './sidebarCollapse'
import { usePlayerSwitch } from './usePlayerSwitch'
import { useVisibleNavSections } from './useVisibleNavSections'

export function SidebarNav() {
  const locale = useAppShellStore((s) => s.locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const appVersion = useAppShellStore((s) => s.appInfo?.version ?? '')
  const pathname = useRouterState().location.pathname
  const sections = useVisibleNavSections()
  const handlePlayerChange = usePlayerSwitch()
  const playerSlug = currentPlayer?.player_slug ?? ''
  // Joueur courant = profil d'un ami : ses notifications ne sont pas celles de la session, pas de cloche.
  const canWriteCurrent = useCanWritePlayer(playerSlug)
  const { collapsed, toggle } = useSidebarCollapse()

  const shared = {
    sections,
    pathname,
    locale,
    titleSlug: currentTitleSlug,
    playerSlug,
    currentPlayer,
    availablePlayers,
    canWriteCurrent,
    onPlayerChange: handlePlayerChange,
  }
  return collapsed ? (
    <SidebarRail {...shared} onExpand={toggle} />
  ) : (
    <SidebarPanel {...shared} appVersion={appVersion} onCollapse={toggle} />
  )
}
