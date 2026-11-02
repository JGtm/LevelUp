/**
 * SidebarRail — la barre latérale de l'application de bureau repliée en icônes (56 px).
 *
 * Mêmes sections et mêmes onglets que `SidebarNav` déplié, une icône par entrée : un lien avec
 * infobulle (au survol et au focus) pour une section sans onglets, un menu volant (`RailMenu`) listant
 * les onglets pour les autres. La bascule de jeu et le joueur se réduisent au sigle et à l'initiale (leur
 * menu volant porte les noms), la cloche garde sa place. Le pied de barre garde les mêmes actions que la
 * barre dépliée, en icônes, et le bouton qui la déplie. L'état (replié ou non) est décidé par
 * `useSidebarCollapse`, le contenu des sections et du joueur est calculé par `SidebarNav`.
 */
import { Link } from '@tanstack/react-router'
import {
  BookOpenIcon,
  BookOpenTextIcon,
  BugIcon,
  InfoIcon,
  PanelLeftOpenIcon,
  SettingsIcon,
  SunMoonIcon,
} from '@/components/icons/lucide'
import { ASSET_PANEL_ID, ASSET_PANEL_TRIGGER_ID } from '@/features/asset-drawer'
import { NotificationsBell } from '@/features/notifications/NotificationsBell'
import type { PlayerSummary } from '@/lib/api/types'
import { APP_NAME } from '@/lib/appBrand'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'
import { routeTemplateSuffix, playerRelativePath } from '@/lib/title-routing'
import { buildTitleSwitcherEntries, useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { AboutDialog } from './AboutDialog'
import type { L1Section } from './navL1Sections'
import { RailMenu, RailTip } from './RailPopover'
import { playerInitial, titleInitials } from './railLabels'
import { SIDEBAR_SHORTCUT_LABEL, useSidebarFooterActions } from './sidebarFooterActions'
import { SectionIcon } from './sidebarIcons'
import { useSelectTitle } from './useSelectTitle'

const RAIL_ITEM_BASE =
  'flex h-10 w-10 items-center justify-center rounded-md text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'

function railItemClass(active: boolean): string {
  return `${RAIL_ITEM_BASE} ${
    active
      ? 'bg-sidebar-primary text-sidebar-primary-foreground'
      : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
  }`
}

const MENU_ITEM_CLASS =
  'flex w-full items-center justify-between gap-3 px-3 py-1.5 text-left text-sm whitespace-nowrap text-popover-foreground hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-60'

const ICON_CLASS = 'h-5 w-5'

export interface SidebarRailProps {
  sections: L1Section[]
  pathname: string
  locale: Locale
  titleSlug: string
  playerSlug: string
  currentPlayer: PlayerSummary | null
  availablePlayers: PlayerSummary[]
  /** Le joueur courant est le sien : sa cloche de notifications est offerte. */
  canWriteCurrent: boolean
  onPlayerChange: (playerSlug: string) => void
  /** Déplie la barre. */
  onExpand: () => void
}

function RailTitleMenu({ locale }: { locale: Locale }) {
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const isTitleSwitching = useAppShellStore((s) => s.isTitleSwitching)
  const onSelect = useSelectTitle()
  const entries = buildTitleSwitcherEntries(availableTitles, currentTitleSlug)
  if (entries.length <= 1) return null
  const label = formatMessage(commonManifest, 'common.shell.nav_game', locale)
  const current = entries.find((e) => e.isCurrent)
  return (
    <RailMenu label={label} trigger={titleInitials(current?.name ?? '')} triggerClassName={railItemClass(false)}>
      {(close) =>
        entries.map((e) => (
          <button
            key={e.slug}
            type="button"
            role="menuitemradio"
            aria-checked={e.isCurrent}
            disabled={e.disabled || isTitleSwitching}
            onClick={() => {
              onSelect(e.slug, e.disabled, e.isCurrent)
              close()
            }}
            className={`${MENU_ITEM_CLASS} ${e.isCurrent ? 'font-semibold text-primary' : ''}`}
          >
            {e.name}
          </button>
        ))
      }
    </RailMenu>
  )
}

function RailPlayer({
  currentPlayer,
  availablePlayers,
  locale,
  onPlayerChange,
}: Pick<SidebarRailProps, 'currentPlayer' | 'availablePlayers' | 'locale' | 'onPlayerChange'>) {
  if (!currentPlayer) return null
  const initial = playerInitial(currentPlayer.gamertag)
  if (availablePlayers.length <= 1) {
    return (
      <RailTip label={currentPlayer.gamertag}>
        <span role="img" aria-label={currentPlayer.gamertag} tabIndex={0} className={railItemClass(false)}>
          {initial}
        </span>
      </RailTip>
    )
  }
  return (
    <RailMenu
      label={formatMessage(commonManifest, 'common.shell.player_select', locale)}
      trigger={initial}
      triggerClassName={railItemClass(false)}
    >
      {(close) =>
        availablePlayers.map((p) => (
          <button
            key={p.player_slug}
            type="button"
            role="menuitem"
            aria-current={p.player_slug === currentPlayer.player_slug ? 'true' : undefined}
            onClick={() => {
              onPlayerChange(p.player_slug)
              close()
            }}
            className={MENU_ITEM_CLASS}
          >
            {p.gamertag}
          </button>
        ))
      }
    </RailMenu>
  )
}

function RailSections({
  sections,
  pathname,
  locale,
  titleSlug,
  playerSlug,
}: Pick<SidebarRailProps, 'sections' | 'pathname' | 'locale' | 'titleSlug' | 'playerSlug'>) {
  const suffix = playerRelativePath(pathname)
  return (
    <>
      {sections.map((section) => {
        const active = section.matchPathname(pathname)
        const label = formatMessage(commonManifest, section.labelKey, locale)
        const glyph = <SectionIcon sectionKey={section.key} className={ICON_CLASS} fallback={Array.from(label)[0]} />
        if (!section.tabs) {
          return (
            <RailTip key={section.key} label={label}>
              <Link
                to={section.defaultPath}
                params={{ titleSlug, playerSlug }}
                aria-label={label}
                aria-current={active ? 'page' : undefined}
                className={railItemClass(active)}
              >
                {glyph}
              </Link>
            </RailTip>
          )
        }
        return (
          <RailMenu key={section.key} label={label} trigger={glyph} triggerClassName={railItemClass(active)} current={active}>
            {(close) =>
              section.tabs?.map((tab) => {
                const tabActive = suffix !== null && suffix === routeTemplateSuffix(tab.path)
                return (
                  <Link
                    key={tab.key}
                    to={tab.path}
                    params={{ titleSlug, playerSlug }}
                    role="menuitem"
                    onClick={close}
                    aria-current={tabActive ? 'page' : undefined}
                    className={`${MENU_ITEM_CLASS} ${tabActive ? 'bg-accent text-accent-foreground' : ''}`}
                  >
                    {formatMessage(commonManifest, tab.labelKey, locale)}
                  </Link>
                )
              })
            }
          </RailMenu>
        )
      })}
    </>
  )
}

function RailFooter({ pathname, locale, onExpand }: Pick<SidebarRailProps, 'pathname' | 'locale' | 'onExpand'>) {
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const { aboutOpen, setAboutOpen, assetsOpen, toggleAssets, openFeedback } = useSidebarFooterActions()
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const toggleTheme = useSettingsDraftStore((s) => s.toggleTheme)
  const helpLabel = t('common.shell.nav_help')
  const settingsLabel = t('common.shell.nav_settings')
  const assetsLabel = formatMessage(assetDrawerManifest, 'asset_drawer.mini_tab', locale)
  const helpActive = pathname.startsWith('/help')
  const settingsActive = pathname.startsWith('/settings')
  const expandLabel = t('common.shell.sidebar_expand')

  return (
    <>
      <div
        className="flex shrink-0 flex-col items-center gap-1 border-t border-border py-2"
        role="group"
        aria-label={t('common.shell.sidebar_footer_aria')}
      >
        <RailTip label={helpLabel}>
          <Link
            to="/help"
            search={{ tab: 'glossary' }}
            aria-label={helpLabel}
            aria-current={helpActive ? 'page' : undefined}
            className={railItemClass(helpActive)}
          >
            <BookOpenTextIcon className={ICON_CLASS} />
          </Link>
        </RailTip>
        <RailTip label={assetsLabel}>
          <button
            type="button"
            id={ASSET_PANEL_TRIGGER_ID}
            onClick={toggleAssets}
            aria-label={assetsLabel}
            aria-expanded={assetsOpen}
            aria-controls={assetsOpen ? ASSET_PANEL_ID : undefined}
            className={railItemClass(assetsOpen)}
          >
            <BookOpenIcon className={ICON_CLASS} />
          </button>
        </RailTip>
        <RailTip label={t('common.shell.nav_feedback')}>
          <button type="button" onClick={openFeedback} aria-label={t('common.shell.nav_feedback')} className={railItemClass(false)}>
            <BugIcon className={ICON_CLASS} />
          </button>
        </RailTip>
        <RailTip label={settingsLabel}>
          <Link
            to="/settings"
            search={{ tab: 'appearance' }}
            aria-label={settingsLabel}
            aria-current={settingsActive ? 'page' : undefined}
            className={railItemClass(settingsActive)}
          >
            <SettingsIcon className={ICON_CLASS} />
          </Link>
        </RailTip>
        <RailTip label={t('common.shell.nav_theme')}>
          <button
            type="button"
            role="switch"
            aria-checked={theme === 'dark'}
            onClick={toggleTheme}
            aria-label={t('common.shell.nav_theme')}
            className={railItemClass(false)}
          >
            <SunMoonIcon className={ICON_CLASS} />
          </button>
        </RailTip>
        <RailTip label={t('common.shell.nav_about')}>
          <button type="button" onClick={() => setAboutOpen(true)} aria-label={t('common.shell.nav_about')} className={railItemClass(false)}>
            <InfoIcon className={ICON_CLASS} />
          </button>
        </RailTip>
        <RailTip label={`${expandLabel} (${SIDEBAR_SHORTCUT_LABEL})`}>
          <button
            type="button"
            onClick={onExpand}
            aria-label={expandLabel}
            aria-keyshortcuts="Control+B"
            className={railItemClass(false)}
          >
            <PanelLeftOpenIcon className={ICON_CLASS} />
          </button>
        </RailTip>
      </div>
      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </>
  )
}

export function SidebarRail(props: SidebarRailProps) {
  const { sections, pathname, locale, titleSlug, playerSlug, currentPlayer, availablePlayers, canWriteCurrent } = props
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const logo = <img src="/logo.png" alt={APP_NAME} className="h-7 w-7 shrink-0 object-contain" />
  const logoClass = `${RAIL_ITEM_BASE} hover:bg-sidebar-accent`

  return (
    <aside
      data-sidebar="collapsed"
      className="flex h-full w-14 shrink-0 flex-col items-center border-r border-border bg-sidebar text-sidebar-foreground"
      aria-label={t('common.shell.nav_main_aria')}
    >
      <div className="flex shrink-0 flex-col items-center gap-1 border-b border-border py-2">
        <RailTip label={APP_NAME}>
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
        </RailTip>
        <RailTitleMenu locale={locale} />
      </div>

      {currentPlayer && (
        <div className="flex shrink-0 flex-col items-center gap-1 border-b border-border py-2">
          <RailPlayer
            currentPlayer={currentPlayer}
            availablePlayers={availablePlayers}
            locale={locale}
            onPlayerChange={props.onPlayerChange}
          />
          {canWriteCurrent && <NotificationsBell playerSlug={currentPlayer.player_slug} panelAlign="start" />}
        </div>
      )}

      <nav
        className="flex w-full flex-1 flex-col items-center gap-1 overflow-y-auto py-2"
        aria-label={t('common.nav.sections_aria')}
      >
        {playerSlug && (
          <RailSections
            sections={sections}
            pathname={pathname}
            locale={locale}
            titleSlug={titleSlug}
            playerSlug={playerSlug}
          />
        )}
      </nav>

      <RailFooter pathname={pathname} locale={locale} onExpand={props.onExpand} />
    </aside>
  )
}
