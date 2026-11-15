/**
 * SidebarFooter — pied de la barre latérale de bureau (`SidebarNav`).
 *
 * Une seule rangée de sept boutons d'icône de 32 px (la barre fait 240 px, marge comprise) : Aide,
 * référentiels (volet ancré, raccourci Ctrl+Maj+L), signalement d'un problème, Paramètres, bascule de
 * thème, « À propos » (qui porte tout ce que le pied de page du site portait) et bouton de repliement
 * de la barre (Ctrl+B), puis la version de l'application. Chaque bouton porte son nom en `aria-label` et en
 * `title` : la rangée remplace sept lignes de texte, dont la hauteur faisait déborder les fenêtres basses.
 * Le chemin courant, la langue et la version viennent du parent ; l'état des deux panneaux (référentiels,
 * signalement) et du thème est lu par `useSidebarFooterActions`, partagé avec la barre repliée ; « À propos » est
 * ouvert par `onOpenAbout`, son état et sa fenêtre étant tenus par `SidebarNav`.
 */
import { Link } from '@tanstack/react-router'
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
import { SIDEBAR_TOGGLE_ATTR } from './sidebarCollapse'
import { SIDEBAR_SHORTCUT_LABEL, useSidebarFooterActions } from './sidebarFooterActions'

const FOOT_BUTTON_BASE =
  'flex h-8 w-8 items-center justify-center rounded-md transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'

const ICON_CLASS = 'h-4 w-4 shrink-0'

function footButtonClass(active: boolean): string {
  return `${FOOT_BUTTON_BASE} ${active ? 'bg-sidebar-accent text-sidebar-accent-foreground' : 'text-sidebar-foreground/70'}`
}

export interface SidebarFooterProps {
  pathname: string
  locale: Locale
  /** Version de l'application ; vide : la ligne de version n'est pas rendue. */
  appVersion: string
  /** Replie la barre en icônes. */
  onCollapse: () => void
  /** Ouvre la fenêtre « À propos ». */
  onOpenAbout: () => void
}

export function SidebarFooter({ pathname, locale, appVersion, onCollapse, onOpenAbout }: SidebarFooterProps) {
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  const { assetsOpen, toggleAssets, openFeedback, isDark, toggleTheme } = useSidebarFooterActions()
  const helpLabel = t('common.shell.nav_help')
  const settingsLabel = t('common.shell.nav_settings')
  const helpActive = pathname.startsWith('/help')
  const settingsActive = pathname.startsWith('/settings')
  const assetsLabel = formatMessage(assetDrawerManifest, 'asset_drawer.mini_tab', locale)
  const feedbackLabel = t('common.shell.nav_feedback')
  const themeLabel = t('common.shell.nav_theme')
  const aboutLabel = t('common.shell.nav_about')
  const collapseLabel = t('common.shell.sidebar_collapse')

  return (
    <div className="shrink-0 border-t border-border p-2" role="group" aria-label={t('common.shell.sidebar_footer_aria')}>
      <div className="grid grid-cols-7 justify-items-center">
        <Link
          to="/help"
          search={{ tab: 'glossary' }}
          aria-label={helpLabel}
          title={helpLabel}
          className={footButtonClass(helpActive)}
          aria-current={helpActive ? 'page' : undefined}
        >
          <BookOpenTextIcon className={ICON_CLASS} />
        </Link>
        <button
          type="button"
          id={ASSET_PANEL_TRIGGER_ID}
          onClick={toggleAssets}
          aria-expanded={assetsOpen}
          aria-controls={assetsOpen ? ASSET_PANEL_ID : undefined}
          aria-label={assetsLabel}
          title={formatMessage(assetDrawerManifest, 'asset_drawer.trigger.title', locale)}
          className={footButtonClass(assetsOpen)}
        >
          <BookOpenIcon className={ICON_CLASS} />
        </button>
        <button
          type="button"
          onClick={openFeedback}
          aria-label={feedbackLabel}
          title={feedbackLabel}
          className={footButtonClass(false)}
        >
          <BugIcon className={ICON_CLASS} />
        </button>
        <Link
          to="/settings"
          search={{ tab: 'appearance' }}
          aria-label={settingsLabel}
          title={settingsLabel}
          className={footButtonClass(settingsActive)}
          aria-current={settingsActive ? 'page' : undefined}
        >
          <SettingsIcon className={ICON_CLASS} />
        </Link>
        <button
          type="button"
          role="switch"
          aria-checked={isDark}
          onClick={toggleTheme}
          aria-label={themeLabel}
          title={themeLabel}
          className={footButtonClass(false)}
        >
          <SunMoonIcon className={ICON_CLASS} />
        </button>
        <button type="button" onClick={onOpenAbout} aria-label={aboutLabel} title={aboutLabel} className={footButtonClass(false)}>
          <InfoIcon className={ICON_CLASS} />
        </button>
        <button
          type="button"
          onClick={onCollapse}
          {...{ [SIDEBAR_TOGGLE_ATTR]: '' }}
          aria-label={collapseLabel}
          title={`${collapseLabel} (${SIDEBAR_SHORTCUT_LABEL})`}
          aria-keyshortcuts="Control+B"
          className={footButtonClass(false)}
        >
          <PanelLeftCloseIcon className={ICON_CLASS} />
        </button>
      </div>
      {appVersion && (
        <p className="px-1 pt-1.5 text-center font-mono text-xs text-sidebar-foreground/60" data-testid="sidebar-version">
          {t('common.shell.sidebar_version', { version: appVersion })}
        </p>
      )}
    </div>
  )
}
