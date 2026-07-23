/**
 * NavL1 — barre de navigation principale (niveau 1).
 *
 * Barre horizontale fixée en haut de l'application, visible sur toutes les pages.
 * Contient : logo · sections joueur · sélecteur de joueur actif · switch thème · lien paramètres.
 *
 * Les sections avec plusieurs onglets (Palmarès, Stats, Escouade, Carrière) utilisent un
 * split button : clic sur le label → landing, clic sur ▾ → dropdown des onglets.
 */
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { useRef, useEffect, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { ThemeToggle } from './ThemeToggle'
import { TitleSwitcher } from './TitleSwitcher'
import { resolvePlayerSwitch } from './shellNavigation'
import { HelpSplitButton } from './HelpSplitButton'
import { LogoutButton } from './LogoutButton'
import { NavL1MobileMenu } from './NavL1MobileMenu'
import { NavL1MobileActions, type SettingsTabItem } from './NavL1MobileActions'
import { L1_SECTIONS, type L1Section, type L1Tab } from './navL1Sections'
import { useTitleCapabilities, hasCapabilityIn } from '@/lib/capabilities/capabilities'
import { useSettings } from '@/features/settings/queries'
import { getSettingsText } from '@/features/settings/i18n'
import { NotificationsBell } from '@/features/notifications/NotificationsBell'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

// ─── SplitButton ──────────────────────────────────────────────────────────────

interface SplitButtonProps {
  section: L1Section & { tabs: L1Tab[] }
  isActive: boolean
  titleSlug: string
  playerSlug: string
}

function SplitButton({ section, isActive, titleSlug, playerSlug }: SplitButtonProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const sectionLabel = t(section.labelKey)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const wrapperClass = [
    'flex items-stretch rounded-md overflow-hidden text-sm font-medium transition-colors',
    isActive
      ? 'bg-sidebar-primary text-sidebar-primary-foreground'
      : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
  ].join(' ')

  return (
    <div ref={ref} className="relative">
      <div className={wrapperClass}>
        <Link
          to={section.defaultPath}
          params={{ titleSlug, playerSlug }}
          className="flex items-center gap-1.5 px-3 py-1.5 whitespace-nowrap"
          aria-current={isActive ? 'page' : undefined}
        >
          {section.icon}
          {sectionLabel}
        </Link>

        <span className="mx-0.5 h-4 w-px self-center rounded-full bg-current opacity-20" aria-hidden="true" />

        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="px-1.5 py-1.5 cursor-pointer"
          aria-label={formatMessage(commonManifest, 'common.nav.section_tabs_aria', locale, { section: sectionLabel })}
          aria-expanded={open}
          aria-haspopup="menu"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className={`h-3 w-3 transition-transform ${open ? 'rotate-180' : ''}`}
            viewBox="0 0 12 12"
            fill="currentColor"
            aria-hidden="true"
          >
            <path d="M6 8L1 3h10z" />
          </svg>
        </button>
      </div>

      {open && (
        <div
          role="menu"
          className="absolute left-0 top-full mt-1 z-50 min-w-[10rem] rounded-md border border-border bg-popover py-1 shadow-lg"
        >
          {section.tabs.map((tab) => (
            <Link
              key={tab.key}
              to={tab.path}
              params={{ titleSlug, playerSlug }}
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground whitespace-nowrap"
            >
              {t(tab.labelKey)}
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── SettingsSplitButton ─────────────────────────────────────────────────────

interface SettingsSplitButtonProps {
  tabs: SettingsTabItem[]
  isActive: boolean
  isAdmin: boolean
}

function SettingsSplitButton({ tabs, isActive, isAdmin }: SettingsSplitButtonProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const wrapperClass = [
    'flex items-stretch rounded-md overflow-hidden text-sm font-medium transition-colors',
    isActive
      ? 'bg-sidebar-primary text-sidebar-primary-foreground'
      : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
  ].join(' ')

  return (
    <div ref={ref} className="relative ml-1">
      <div className={wrapperClass}>
        <Link
          to="/settings"
          search={{ tab: 'appearance' }}
          className="px-2 py-1.5"
          aria-label={t('common.shell.nav_settings')}
          title={t('common.shell.nav_settings')}
          aria-current={isActive ? 'page' : undefined}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="h-4 w-4"
            viewBox="0 0 20 20"
            fill="currentColor"
            aria-hidden="true"
          >
            <path
              fillRule="evenodd"
              d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z"
              clipRule="evenodd"
            />
          </svg>
        </Link>

        <span className="mx-0.5 h-4 w-px self-center rounded-full bg-current opacity-20" aria-hidden="true" />

        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="px-1.5 py-1.5 cursor-pointer"
          aria-label={t('common.shell.settings_tabs_aria')}
          aria-expanded={open}
          aria-haspopup="menu"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className={`h-3 w-3 transition-transform ${open ? 'rotate-180' : ''}`}
            viewBox="0 0 12 12"
            fill="currentColor"
            aria-hidden="true"
          >
            <path d="M6 8L1 3h10z" />
          </svg>
        </button>
      </div>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full mt-1 z-50 min-w-[12rem] rounded-md border border-border bg-popover py-1 shadow-lg"
        >
          {/* Sélecteur de jeu (titre) — NO-OP mono-titre, apparaît dès qu'un 2e
              titre est configuré (PMT-8/MT-22). Placé avant le toggle de thème. */}
          <TitleSwitcher onSwitched={() => setOpen(false)} />
          <div className="flex items-center justify-between gap-4 px-3 py-1.5">
            <span className="text-sm text-popover-foreground">{t('common.shell.nav_theme')}</span>
            <ThemeToggle variant="menu" />
          </div>
          <div role="separator" className="my-1 h-px bg-foreground/20" />
          {tabs.map((item) => (
            <Link
              key={item.key}
              to="/settings"
              search={{ tab: item.tab }}
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground whitespace-nowrap"
            >
              {item.label}
            </Link>
          ))}
          {isAdmin && (
            <>
              <div role="separator" className="my-1 h-px bg-foreground/20" />
              <Link
                to="/admin"
                role="menuitem"
                onClick={() => setOpen(false)}
                className="block px-3 py-1.5 text-sm font-medium text-popover-foreground hover:bg-accent hover:text-accent-foreground whitespace-nowrap"
              >
                {t('common.admin.page_title')}
              </Link>
            </>
          )}
          <div role="separator" className="my-1 h-px bg-foreground/20" />
          <LogoutButton variant="menu" />
        </div>
      )}
    </div>
  )
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function NavL1() {
  const navigate = useNavigate()
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const { data: settings } = useSettings()
  const showProgression = settings?.show_progression ?? true
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const playerSlug = currentPlayer?.player_slug ?? ''

  // Gating multi-titre (Phase 5) : on masque les sections / onglets dont la
  // capability n'est pas déclarée par le titre courant. Fail-open (titleCaps null
  // pendant le bootstrap → tout visible) ⇒ NO-OP pour halo_infinite. Le filtre
  // alimente à la fois le rendu desktop inline ET le drawer mobile (même tableau).
  const titleCaps = useTitleCapabilities()
  const sectionVisible = (s: L1Section) => s.capability == null || hasCapabilityIn(titleCaps, s.capability)
  const tabVisible = (tab: L1Tab) => tab.capability == null || hasCapabilityIn(titleCaps, tab.capability)
  const visibleSections = (showProgression
    ? L1_SECTIONS
    : L1_SECTIONS.filter((s) => s.key !== 'ascension'))
    .filter(sectionVisible)
    .map((s) => {
      if (!s.tabs) return s
      // h5 : « Citations » (moteur dérivé d'Infinite) → « Commendations » natif —
      // cohérent avec NavL2. Le slug est le seul signal front (aucune capability
      // coarse ne distingue h5, cf. CAREER_TABS_H5).
      const tabs = s.tabs.filter(tabVisible).map((tab): L1Tab =>
        currentTitleSlug === 'halo_5' && tab.key === 'citations'
          ? {
              // Halo 5 : commendations natives. On conserve la clé i18n d'origine
              // (`tab_citations` = FR « Citations » / EN « Commendations », terme
              // officiel Halo) et on ne change QUE la clé de tab + le chemin.
              ...tab,
              key: 'commendations',
              path: '/{-$lang}/t/$titleSlug/players/$playerSlug/commendations',
            }
          : tab,
      )
      // Si l'onglet servant de landing par défaut est masqué (ex: « Classements »
      // gaté sur world.leaderboard), replier le défaut sur le 1er onglet visible.
      const defaultPath = tabs.some((tab) => tab.path === s.defaultPath)
        ? s.defaultPath
        : (tabs[0]?.path ?? s.defaultPath)
      return { ...s, tabs: tabs.length > 0 ? tabs : undefined, defaultPath }
    })

  // Onglets Paramètres — source unique partagée entre le split button desktop
  // et le menu compte/outils mobile.
  // « Synchronisation » et « Utilisateurs » ont migré vers la page Admin
  // (lien « Administration » ci-dessous, gardé pour les admins).
  // Libellés localisés (FR/EN) via le bundle Settings partagé — alimentent à la
  // fois le dropdown desktop et le drawer mobile.
  const st = getSettingsText(locale)
  const settingsTabs: SettingsTabItem[] = [
    { key: 'appearance', label: st.tabAppearance, tab: 'appearance' },
    { key: 'analyse', label: st.tabAnalyse, tab: 'analyse' },
    { key: 'notifications', label: st.tabNotifications, tab: 'notifications' },
    { key: 'data', label: st.tabData, tab: 'data' },
    { key: 'account', label: st.tabAccount, tab: 'account' },
  ]

  function handlePlayerChange(slug: string) {
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

  const logoInner = (
    <>
      <img src="/logo.png" alt="LevelUp" className="h-7 w-7 shrink-0 object-contain" />
      <span className="hidden text-sm font-bold text-sidebar-foreground sm:block">LevelUp</span>
    </>
  )
  const logoClass =
    'mr-3 flex shrink-0 items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-sidebar-accent'

  return (
    <nav
      className="flex h-12 shrink-0 items-center gap-0.5 border-b border-border bg-sidebar px-3"
      role="navigation"
      aria-label={t('common.shell.nav_main_aria')}
    >
      {/* ── Logo ────────────────────────────────────────────────────────── */}
      {playerSlug ? (
        <Link
          to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
          params={{ titleSlug: currentTitleSlug, playerSlug }}
          className={logoClass}
          aria-label={t('common.shell.logo_aria')}
        >
          {logoInner}
        </Link>
      ) : (
        <Link to="/" className={logoClass} aria-label={t('common.shell.logo_aria')}>
          {logoInner}
        </Link>
      )}

      {/* ── Hamburger mobile (< md) — ouvre le drawer des sections ────────── */}
      {playerSlug && (
        <NavL1MobileMenu
          sections={visibleSections}
          pathname={pathname}
          titleSlug={currentTitleSlug}
          playerSlug={playerSlug}
        />
      )}

      {/* ── Sections inline desktop (≥ md, seulement si un joueur est actif) ─ */}
      {playerSlug && (
        <div className="hidden items-center gap-0.5 md:flex">
          {visibleSections.map((section) => {
            const isActive = section.matchPathname(pathname)

            if (section.tabs) {
              return (
                <SplitButton
                  key={section.key}
                  section={section as L1Section & { tabs: L1Tab[] }}
                  isActive={isActive}
                  titleSlug={currentTitleSlug}
                  playerSlug={playerSlug}
                />
              )
            }

            return (
              <Link
                key={section.key}
                to={section.defaultPath}
                params={{ titleSlug: currentTitleSlug, playerSlug }}
                className={[
                  'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors whitespace-nowrap',
                  isActive
                    ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                    : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                ].join(' ')}
                aria-current={isActive ? 'page' : undefined}
              >
                {section.icon}
                {t(section.labelKey)}
              </Link>
            )
          })}
        </div>
      )}

      {/* ── Spacer ──────────────────────────────────────────────────────── */}
      <div className="flex-1" />

      {/* ── Gamertag / sélecteur joueur (visible mobile, tronqué si large) ── */}
      {availablePlayers.length > 1 ? (
        <div className="flex min-w-0 items-center gap-1.5">
          <select
            value={playerSlug}
            onChange={(e) => handlePlayerChange(e.target.value)}
            className="max-w-[32vw] truncate rounded-md border border-sidebar-border bg-sidebar-accent px-2 py-1 text-sm text-sidebar-foreground transition-colors focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/30 sm:max-w-none"
            aria-label={t('common.shell.player_select')}
          >
            {availablePlayers.map((p) => (
              <option key={p.player_slug} value={p.player_slug}>
                {p.gamertag}
              </option>
            ))}
          </select>
        </div>
      ) : (
        currentPlayer && (
          <span className="max-w-[32vw] truncate text-sm font-medium text-sidebar-foreground/70 sm:max-w-none">
            {currentPlayer.gamertag}
          </span>
        )
      )}

      {/* ── Cloche notifications (per-player) ────────────────────────────── */}
      {currentPlayer && <NotificationsBell playerSlug={currentPlayer.player_slug} />}

      {/* ── Cluster droit desktop (≥ md) : aide · paramètres (déconnexion dans le menu) ── */}
      <div className="hidden items-center gap-0.5 md:flex">
        <div className="ml-1">
          <HelpSplitButton isActive={pathname.startsWith('/help')} />
        </div>
        <SettingsSplitButton isActive={pathname.startsWith('/settings')} tabs={settingsTabs} isAdmin={isAdmin} />
      </div>

      {/* ── Menu compte & outils mobile (< md) : regroupe le cluster droit ── */}
      <NavL1MobileActions settingsTabs={settingsTabs} pathname={pathname} isAdmin={isAdmin} />
    </nav>
  )
}
