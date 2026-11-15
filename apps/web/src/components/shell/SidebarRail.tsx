/**
 * SidebarRail — la barre latérale de l'application de bureau repliée en icônes (56 px).
 *
 * Mêmes sections et mêmes onglets que `SidebarNav` déplié, une icône par entrée : un lien avec
 * infobulle (au survol et au focus) pour une section sans onglets, un menu volant (`RailMenu`) listant
 * les onglets pour les autres. La bascule de jeu et le joueur se réduisent au sigle et à l'initiale (leur
 * menu volant porte les noms), la cloche garde sa place. Le pied de barre porte les référentiels, un menu
 * « Plus » (Aide, signalement, Paramètres, thème, « À propos ») et le bouton qui la déplie. L'état (replié ou non) est décidé par
 * `useSidebarCollapse`, le contenu des sections et du joueur est calculé par `SidebarNav`.
 */
import { Link } from '@tanstack/react-router'
import type { ButtonHTMLAttributes, ReactNode } from 'react'
import {
  BookOpenIcon,
  BookOpenTextIcon,
  BugIcon,
  EllipsisIcon,
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
import { HomeLink } from './HomeLink'
import type { L1Section } from './navL1Sections'
import { RailMenu, RailTip } from './RailPopover'
import { playerInitial, titleInitials } from './railLabels'
import { SIDEBAR_TOGGLE_ATTR } from './sidebarCollapse'
import { SIDEBAR_SHORTCUT_LABEL, useSidebarFooterActions } from './sidebarFooterActions'
import { SectionIcon } from './sidebarIcons'
import { SIDEBAR_SCROLL_CLASS, SIDEBAR_SCROLL_CONTENT_CLASS } from './sidebarScroll'
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

const MORE_ITEM_CLASS =
  'flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm whitespace-nowrap text-popover-foreground hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:outline-none'

const MORE_ICON_CLASS = 'h-4 w-4 shrink-0'

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
  /** Ouvre la fenêtre « À propos » (tenue par `SidebarNav`). */
  onOpenAbout: () => void
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

interface RailButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  /** Nom accessible du bouton, et texte de l'infobulle sauf `tip`. */
  label: string
  /** Texte de l'infobulle quand il diffère du nom accessible (raccourci clavier annoncé). */
  tip?: string
  active?: boolean
  children: ReactNode
}

/** Un bouton d'icône du pied de barre repliée, avec son infobulle. */
function RailButton({ label, tip, active = false, children, ...rest }: RailButtonProps) {
  return (
    <RailTip label={tip ?? label}>
      <button type="button" aria-label={label} className={railItemClass(active)} {...rest}>
        {children}
      </button>
    </RailTip>
  )
}

/**
 * Les actions secondaires du pied de barre repliée (Aide, signalement, Paramètres, thème, « À propos »)
 * regroupées dans un menu volant « Plus » : cinq icônes de plus ne tiendraient pas dans une fenêtre basse.
 * Le menu monte depuis le bas de son icône.
 */
function RailMoreMenu({
  pathname,
  locale,
  onOpenAbout,
}: Pick<SidebarRailProps, 'pathname' | 'locale' | 'onOpenAbout'>) {
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const { openFeedback, isDark, toggleTheme } = useSidebarFooterActions()
  const helpActive = pathname.startsWith('/help')
  const settingsActive = pathname.startsWith('/settings')

  return (
    <RailMenu
      label={t('common.shell.nav_more')}
      trigger={<EllipsisIcon className={ICON_CLASS} />}
      triggerClassName={railItemClass(helpActive || settingsActive)}
      current={helpActive || settingsActive}
      placement="bottom"
    >
      {(close) => (
        <>
          <Link
            to="/help"
            search={{ tab: 'glossary' }}
            role="menuitem"
            onClick={close}
            aria-current={helpActive ? 'page' : undefined}
            className={`${MORE_ITEM_CLASS} ${helpActive ? 'bg-accent text-accent-foreground' : ''}`}
          >
            <BookOpenTextIcon className={MORE_ICON_CLASS} />
            {t('common.shell.nav_help')}
          </Link>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              openFeedback()
              close()
            }}
            className={MORE_ITEM_CLASS}
          >
            <BugIcon className={MORE_ICON_CLASS} />
            {t('common.shell.nav_feedback')}
          </button>
          <Link
            to="/settings"
            search={{ tab: 'appearance' }}
            role="menuitem"
            onClick={close}
            aria-current={settingsActive ? 'page' : undefined}
            className={`${MORE_ITEM_CLASS} ${settingsActive ? 'bg-accent text-accent-foreground' : ''}`}
          >
            <SettingsIcon className={MORE_ICON_CLASS} />
            {t('common.shell.nav_settings')}
          </Link>
          <button
            type="button"
            role="menuitemcheckbox"
            aria-checked={isDark}
            onClick={() => {
              toggleTheme()
              close()
            }}
            className={MORE_ITEM_CLASS}
          >
            <SunMoonIcon className={MORE_ICON_CLASS} />
            {t('common.shell.nav_theme')}
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onOpenAbout()
              close()
            }}
            className={MORE_ITEM_CLASS}
          >
            <InfoIcon className={MORE_ICON_CLASS} />
            {t('common.shell.nav_about')}
          </button>
        </>
      )}
    </RailMenu>
  )
}

function RailFooter({ pathname, locale, onExpand, onOpenAbout }: Pick<SidebarRailProps, 'pathname' | 'locale' | 'onExpand' | 'onOpenAbout'>) {
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const { assetsOpen, toggleAssets } = useSidebarFooterActions()
  const expandLabel = t('common.shell.sidebar_expand')

  return (
    <div
      className="flex shrink-0 flex-col items-center gap-1 border-t border-border py-2"
      role="group"
      aria-label={t('common.shell.sidebar_footer_aria')}
    >
      <RailButton
        label={formatMessage(assetDrawerManifest, 'asset_drawer.mini_tab', locale)}
        active={assetsOpen}
        id={ASSET_PANEL_TRIGGER_ID}
        onClick={toggleAssets}
        aria-expanded={assetsOpen}
        aria-controls={assetsOpen ? ASSET_PANEL_ID : undefined}
      >
        <BookOpenIcon className={ICON_CLASS} />
      </RailButton>
      <RailMoreMenu pathname={pathname} locale={locale} onOpenAbout={onOpenAbout} />
      <RailButton
        label={expandLabel}
        tip={`${expandLabel} (${SIDEBAR_SHORTCUT_LABEL})`}
        onClick={onExpand}
        {...{ [SIDEBAR_TOGGLE_ATTR]: '' }}
        aria-keyshortcuts="Control+B"
      >
        <PanelLeftOpenIcon className={ICON_CLASS} />
      </RailButton>
    </div>
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
          <HomeLink locale={locale} titleSlug={titleSlug} playerSlug={playerSlug} className={logoClass}>
            {logo}
          </HomeLink>
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

      <nav className={`w-full flex-1 overflow-y-auto ${SIDEBAR_SCROLL_CLASS}`} aria-label={t('common.nav.sections_aria')}>
        <div className={`flex flex-col items-center gap-1 py-2 ${SIDEBAR_SCROLL_CONTENT_CLASS}`}>
          {playerSlug && (
            <RailSections
              sections={sections}
              pathname={pathname}
              locale={locale}
              titleSlug={titleSlug}
              playerSlug={playerSlug}
            />
          )}
        </div>
      </nav>

      <RailFooter pathname={pathname} locale={locale} onExpand={props.onExpand} onOpenAbout={props.onOpenAbout} />
    </aside>
  )
}
