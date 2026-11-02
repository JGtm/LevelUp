/**
 * SidebarFooter — pied de la barre latérale de bureau (`SidebarNav`).
 *
 * Aide, référentiels (volet ancré, raccourci Ctrl+Maj+L), signalement d'un problème, Paramètres,
 * bascule de thème, « À propos » (qui porte tout ce que le pied de page du site portait) et version
 * de l'application. Le chemin courant, la langue et la version viennent du parent ; l'état des deux
 * panneaux (référentiels, signalement) est lu dans leurs stores de feature, l'ouverture de « À propos »
 * reste locale.
 */
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ASSET_PANEL_ID, ASSET_PANEL_TRIGGER_ID, useAssetDrawerStore } from '@/features/asset-drawer'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'
import { AboutDialog } from './AboutDialog'
import { ThemeToggle } from './ThemeToggle'

const FOOT_ITEM_CLASS =
  'flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-left text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'

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
}

export function SidebarFooter({ pathname, locale, appVersion }: SidebarFooterProps) {
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  const [aboutOpen, setAboutOpen] = useState(false)
  const assetsOpen = useAssetDrawerStore((s) => s.isOpen)
  const toggleAssets = useAssetDrawerStore((s) => s.toggle)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)

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
    </>
  )
}
