/**
 * SidebarFooter — pied de la barre latérale de bureau (`SidebarNav`).
 *
 * Aide, référentiels (volet ancré, raccourci Ctrl+Maj+L), signalement d'un problème, Paramètres,
 * bascule de thème, « À propos » (qui porte tout ce que le pied de page du site portait), version
 * de l'application et bouton de repliement de la barre (Ctrl+B). Le chemin courant, la langue et la
 * version viennent du parent ; l'état des deux panneaux (référentiels, signalement) est lu dans leurs
 * stores de feature, l'ouverture de « À propos » reste locale (`useSidebarFooterActions`, partagé avec la
 * barre repliée).
 */
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { ASSET_PANEL_ID, ASSET_PANEL_TRIGGER_ID } from '@/features/asset-drawer'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'
import {
  BookOpenIcon,
  BookOpenTextIcon,
  BugIcon,
  InfoIcon,
  PanelLeftCloseIcon,
  SettingsIcon,
  SunMoonIcon,
} from '@/components/icons/lucide'
import { AboutDialog } from './AboutDialog'
import { SIDEBAR_SHORTCUT_LABEL, useSidebarFooterActions } from './sidebarFooterActions'
import { ThemeToggle } from './ThemeToggle'

const FOOT_ITEM_CLASS =
  'flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-left text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'

const ICON_CLASS = 'h-4 w-4 shrink-0'

function footItemClass(active: boolean): string {
  return `${FOOT_ITEM_CLASS} ${active ? 'bg-sidebar-accent text-sidebar-accent-foreground' : 'text-sidebar-foreground/70'}`
}

function FootRow({ children }: { children: ReactNode }) {
  return <div className="flex items-center justify-between gap-2 px-3 py-1.5">{children}</div>
}

export interface SidebarFooterProps {
  pathname: string
  locale: Locale
  /** Version de l'application ; vide : la ligne de version n'est pas rendue. */
  appVersion: string
  /** Replie la barre en icônes. */
  onCollapse: () => void
}

export function SidebarFooter({ pathname, locale, appVersion, onCollapse }: SidebarFooterProps) {
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  const { aboutOpen, setAboutOpen, assetsOpen, toggleAssets, openFeedback } = useSidebarFooterActions()

  return (
    <>
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
          <BookOpenTextIcon className={ICON_CLASS} />
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
          <BookOpenIcon className={ICON_CLASS} />
          {formatMessage(assetDrawerManifest, 'asset_drawer.mini_tab', locale)}
        </button>
        <button type="button" onClick={openFeedback} className={footItemClass(false)}>
          <BugIcon className={ICON_CLASS} />
          {t('common.shell.nav_feedback')}
        </button>
        <Link
          to="/settings"
          search={{ tab: 'appearance' }}
          className={footItemClass(pathname.startsWith('/settings'))}
          aria-current={pathname.startsWith('/settings') ? 'page' : undefined}
        >
          <SettingsIcon className={ICON_CLASS} />
          {t('common.shell.nav_settings')}
        </Link>
        <FootRow>
          <span className="flex items-center gap-2 text-sm text-sidebar-foreground/70">
            <SunMoonIcon className={ICON_CLASS} />
            {t('common.shell.nav_theme')}
          </span>
          <ThemeToggle />
        </FootRow>
        <button type="button" onClick={() => setAboutOpen(true)} className={footItemClass(false)}>
          <InfoIcon className={ICON_CLASS} />
          {t('common.shell.nav_about')}
        </button>
        <button
          type="button"
          onClick={onCollapse}
          title={`${t('common.shell.sidebar_collapse')} (${SIDEBAR_SHORTCUT_LABEL})`}
          aria-keyshortcuts="Control+B"
          className={footItemClass(false)}
        >
          <PanelLeftCloseIcon className={ICON_CLASS} />
          {t('common.shell.sidebar_collapse')}
        </button>
        {appVersion && (
          <p className="px-3 pt-1 font-mono text-xs text-sidebar-foreground/60" data-testid="sidebar-version">
            {t('common.shell.sidebar_version', { version: appVersion })}
          </p>
        )}
      </div>

      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </>
  )
}
