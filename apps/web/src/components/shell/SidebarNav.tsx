/**
 * SidebarNav — barre latérale de l'application de bureau (capacité `can_use_compact_chrome`).
 *
 * Remplace la barre haute du site (NavL1) : mêmes sections et mêmes onglets que le tiroir de la vue
 * mobile (`NavSectionList`, `useVisibleNavSections`), toujours dépliée. De haut en bas : marque,
 * sélecteur de jeu, joueur actif et cloche, sections, puis le pied de barre — Aide, Paramètres,
 * bascule de thème, « À propos » (qui porte tout ce que le pied de page du site portait) et la
 * version de l'application. Le pied porte aussi les deux panneaux qui remplacent les mini-onglets
 * du site : les référentiels (volet ancré, raccourci Ctrl+Maj+L) et le signalement d'un problème.
 */
import { Link, useRouterState } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ASSET_PANEL_ID, ASSET_PANEL_TRIGGER_ID, useAssetDrawerStore } from '@/features/asset-drawer'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer'
import { NotificationsBell } from '@/features/notifications/NotificationsBell'
import { APP_NAME } from '@/lib/appBrand'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { useAppShellStore } from '@/stores/appShellStore'
import { AboutDialog } from './AboutDialog'
import { NavSectionList } from './NavSectionList'
import { PlayerSwitcher } from './PlayerSwitcher'
import { ThemeToggle } from './ThemeToggle'
import { TitleSwitcher } from './TitleSwitcher'
import { usePlayerSwitch } from './usePlayerSwitch'
import { useVisibleNavSections } from './useVisibleNavSections'

const FOOT_ITEM_CLASS =
  'flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-left text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'

function footItemClass(active: boolean): string {
  return `${FOOT_ITEM_CLASS} ${active ? 'bg-sidebar-accent text-sidebar-accent-foreground' : 'text-sidebar-foreground/70'}`
}

function FootRow({ children }: { children: ReactNode }) {
  return <div className="flex items-center justify-between gap-2 px-3 py-1.5">{children}</div>
}

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
  const [aboutOpen, setAboutOpen] = useState(false)
  const assetsOpen = useAssetDrawerStore((s) => s.isOpen)
  const toggleAssets = useAssetDrawerStore((s) => s.toggle)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)
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

      <div
        className="flex shrink-0 flex-col gap-0.5 border-t border-border p-2"
        role="group"
        aria-label={t('common.shell.sidebar_footer_aria')}
      >
        <Link
          to="/help"
          search={{ tab: 'glossary' }}
          className={footItemClass(pathname.startsWith('/help'))}
          aria-current={pathname.startsWith('/help') ? 'page' : undefined}
        >
          {t('common.shell.nav_help')}
        </Link>
        <button
          type="button"
          id={ASSET_PANEL_TRIGGER_ID}
          onClick={toggleAssets}
          aria-expanded={assetsOpen}
          aria-controls={assetsOpen ? ASSET_PANEL_ID : undefined}
          title={formatMessage(assetDrawerManifest, 'asset_drawer.trigger.title', locale)}
          className={footItemClass(assetsOpen)}
        >
          {formatMessage(assetDrawerManifest, 'asset_drawer.mini_tab', locale)}
        </button>
        <button type="button" onClick={openFeedback} className={footItemClass(false)}>
          {t('common.shell.nav_feedback')}
        </button>
        <Link
          to="/settings"
          search={{ tab: 'appearance' }}
          className={footItemClass(pathname.startsWith('/settings'))}
          aria-current={pathname.startsWith('/settings') ? 'page' : undefined}
        >
          {t('common.shell.nav_settings')}
        </Link>
        <FootRow>
          <span className="text-sm text-sidebar-foreground/70">{t('common.shell.nav_theme')}</span>
          <ThemeToggle />
        </FootRow>
        <button type="button" onClick={() => setAboutOpen(true)} className={footItemClass(false)}>
          {t('common.shell.nav_about')}
        </button>
        {appVersion && (
          <p className="px-3 pt-1 font-mono text-xs text-sidebar-foreground/60" data-testid="sidebar-version">
            {t('common.shell.sidebar_version', { version: appVersion })}
          </p>
        )}
      </div>

      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </aside>
  )
}
