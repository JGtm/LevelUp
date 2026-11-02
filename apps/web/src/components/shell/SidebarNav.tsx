/**
 * SidebarNav — barre latérale de l'application de bureau (capacité `can_use_compact_chrome`).
 *
 * Remplace la barre haute du site (NavL1) : mêmes sections et mêmes onglets que le tiroir de la vue
 * mobile (`NavSectionList`, `useVisibleNavSections`), toujours dépliée. De haut en bas : marque,
 * sélecteur de jeu, joueur actif et cloche, sections, puis le pied de barre (`SidebarFooter` : Aide,
 * Paramètres, bascule de thème, « À propos », version, et les deux panneaux qui remplacent les
 * mini-onglets du site : référentiels et signalement d'un problème).
 */
import { Link, useRouterState } from '@tanstack/react-router'
import { NotificationsBell } from '@/features/notifications/NotificationsBell'
import { APP_NAME } from '@/lib/appBrand'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { useAppShellStore } from '@/stores/appShellStore'
import { NavSectionList } from './NavSectionList'
import { PlayerSwitcher } from './PlayerSwitcher'
import { SidebarFooter } from './SidebarFooter'
import { TitleSwitcher } from './TitleSwitcher'
import { usePlayerSwitch } from './usePlayerSwitch'
import { useVisibleNavSections } from './useVisibleNavSections'

export function SidebarNav() {
  const locale = useAppShellStore((s) => s.locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const appVersion = useAppShellStore((s) => s.appInfo?.version ?? '')
  const pathname = useRouterState().location.pathname
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  const sections = useVisibleNavSections()
  const handlePlayerChange = usePlayerSwitch()
  const playerSlug = currentPlayer?.player_slug ?? ''
  // Joueur courant = profil d'un ami : ses notifications ne sont pas celles de la session, pas de cloche.
  const canWriteCurrent = useCanWritePlayer(playerSlug)

  const logo = (
    <>
      <img src="/logo.png" alt={APP_NAME} className="h-7 w-7 shrink-0 object-contain" />
      <span className="text-sm font-bold text-sidebar-foreground">{APP_NAME}</span>
    </>
  )
  const logoClass = 'flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-sidebar-accent'

  return (
    <aside
      className="flex h-full w-60 shrink-0 flex-col border-r border-border bg-sidebar text-sidebar-foreground"
      aria-label={t('common.shell.nav_main_aria')}
    >
      <div className="shrink-0 border-b border-border px-2 py-2">
        {playerSlug ? (
          <Link
            to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
            params={{ titleSlug: currentTitleSlug, playerSlug }}
            className={logoClass}
            aria-label={t('common.shell.logo_aria')}
          >
            {logo}
          </Link>
        ) : (
          <Link to="/{-$lang}" className={logoClass} aria-label={t('common.shell.logo_aria')}>
            {logo}
          </Link>
        )}
        {/* Sélecteur de jeu : s'auto-masque en mono-titre. */}
        <TitleSwitcher />
      </div>

      {currentPlayer && (
        <div className="flex shrink-0 items-center gap-1 border-b border-border px-3 py-2.5">
          <div className="min-w-0 flex-1">
            <PlayerSwitcher
              players={availablePlayers}
              currentPlayer={currentPlayer}
              locale={locale}
              onPlayerChange={handlePlayerChange}
              menuAlign="start"
            />
          </div>
          {canWriteCurrent && <NotificationsBell playerSlug={currentPlayer.player_slug} panelAlign="start" />}
        </div>
      )}

      <nav className="flex-1 overflow-y-auto py-2" aria-label={t('common.nav.sections_aria')}>
        {playerSlug && (
          <NavSectionList
            variant="sidebar"
            sections={sections}
            pathname={pathname}
            titleSlug={currentTitleSlug}
            playerSlug={playerSlug}
          />
        )}
      </nav>

      <SidebarFooter pathname={pathname} locale={locale} appVersion={appVersion} />
    </aside>
  )
}
