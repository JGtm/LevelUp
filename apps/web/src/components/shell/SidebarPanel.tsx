/**
 * SidebarPanel — la barre latérale de l'application de bureau dépliée (240 px) : marque, sélecteur de jeu,
 * joueur actif et cloche, sections, pied de barre (`SidebarFooter`). Les sections, le joueur et les
 * actions viennent de `SidebarNav`, qui choisit entre cette barre et `SidebarRail`.
 */
import { Link } from '@tanstack/react-router'
import { NotificationsBell } from '@/features/notifications/NotificationsBell'
import { APP_NAME } from '@/lib/appBrand'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { NavSectionList } from './NavSectionList'
import { PlayerSwitcher } from './PlayerSwitcher'
import { SidebarFooter } from './SidebarFooter'
import type { SidebarRailProps } from './SidebarRail'
import { TitleSwitcher } from './TitleSwitcher'

export type SidebarPanelProps = Omit<SidebarRailProps, 'onExpand'> & {
  /** Version de l'application ; vide : la ligne de version n'est pas rendue. */
  appVersion: string
  /** Replie la barre en icônes. */
  onCollapse: () => void
}

export function SidebarPanel({
  sections,
  pathname,
  locale,
  titleSlug,
  playerSlug,
  currentPlayer,
  availablePlayers,
  canWriteCurrent,
  onPlayerChange,
  appVersion,
  onCollapse,
}: SidebarPanelProps) {
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  const logo = (
    <>
      <img src="/logo.png" alt={APP_NAME} className="h-7 w-7 shrink-0 object-contain" />
      <span className="text-sm font-bold text-sidebar-foreground">{APP_NAME}</span>
    </>
  )
  const logoClass = 'flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-sidebar-accent'

  return (
    <aside
      data-sidebar="expanded"
      className="flex h-full w-60 shrink-0 flex-col border-r border-border bg-sidebar text-sidebar-foreground"
      aria-label={t('common.shell.nav_main_aria')}
    >
      <div className="shrink-0 border-b border-border px-2 py-2">
        {playerSlug ? (
          <Link
            to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
            params={{ titleSlug, playerSlug }}
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
              onPlayerChange={onPlayerChange}
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
            titleSlug={titleSlug}
            playerSlug={playerSlug}
          />
        )}
      </nav>

      <SidebarFooter pathname={pathname} locale={locale} appVersion={appVersion} onCollapse={onCollapse} />
    </aside>
  )
}
