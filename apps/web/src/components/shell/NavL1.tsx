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
import { useRef, useEffect, useState, type ReactNode } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { ThemeToggle } from './ThemeToggle'
import { buildPlayerDestination, isCommunityPath } from './shellNavigation'
import { HelpSplitButton } from './HelpSplitButton'
import { LogoutButton } from './LogoutButton'
import { useSettings } from '@/features/settings/queries'
import { NotificationsBell } from '@/features/notifications/NotificationsBell'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
// ─── Icône flamme (label Ascension) ──────────────────────────────────────────

function NavFlameIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="13"
      height="13"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z" />
    </svg>
  )
}

// ─── Définition des sections L1 ───────────────────────────────────────────────

interface L1Tab {
  key: string
  label: string
  /** Chemin avec $playerSlug en placeholder. */
  path: string
}

interface L1Section {
  key: string
  label: string
  /** Icône optionnelle affichée avant le label (ex: flamme pour Ascension). */
  icon?: ReactNode
  /** Route par défaut lors du clic sur le label (avec $playerSlug en placeholder). */
  defaultPath: string
  /** Retourne true si le pathname courant appartient à cette section. */
  matchPathname: (pathname: string) => boolean
  /** Onglets du dropdown (optionnel — si absent, bouton simple). */
  tabs?: L1Tab[]
}

// Refonte nav L1 (Phase 4 Prestige) :
// - Synthèse devient un onglet de Stats (transverse : sa famille naturelle)
// - Pass saisonnier devient un onglet de Carrière (progression temporelle)
// - Palmarès renommé en "Communauté" + ajout onglet Leaderboard PP
// - Nouvelle entrée L1 "Ascension" (page Prestige : Objectifs + Parcours)
//   Route path /objectifs conservé. Label rétabli "Objectifs" (≠ "Défis" in-game Halo).
const L1_SECTIONS: L1Section[] = [
  {
    key: 'home',
    label: 'Accueil',
    defaultPath: '/players/$playerSlug/home',
    matchPathname: (p) => /\/players\/[^/]+\/home/.test(p),
  },
  {
    key: 'stats',
    label: 'Solo',
    defaultPath: '/players/$playerSlug/stats/timeseries',
    matchPathname: (p) => /\/players\/[^/]+\/(stats\/|synthesis)/.test(p),
    tabs: [
      { key: 'synthesis', label: 'Synthèse', path: '/players/$playerSlug/synthesis' },
      { key: 'timeseries', label: 'Séries temporelles', path: '/players/$playerSlug/stats/timeseries' },
      { key: 'sessions', label: 'Sessions', path: '/players/$playerSlug/stats/sessions' },
    ],
  },
  {
    key: 'squad',
    label: 'Escouade',
    defaultPath: '/players/$playerSlug/squad/synergies',
    matchPathname: (p) => /\/players\/[^/]+\/squad/.test(p),
    tabs: [
      { key: 'synergies', label: 'Synergies', path: '/players/$playerSlug/squad/synergies' },
      { key: 'contributions', label: 'Contributions', path: '/players/$playerSlug/squad/contributions' },
    ],
  },
  {
    key: 'career',
    label: 'Carrière',
    defaultPath: '/players/$playerSlug/career',
    matchPathname: (p) => /\/players\/[^/]+\/(career|citations|profile)/.test(p),
    tabs: [
      { key: 'progression', label: 'Progression', path: '/players/$playerSlug/career' },
      { key: 'citations', label: 'Citations', path: '/players/$playerSlug/citations' },
      { key: 'season-pass', label: 'Pass saisonnier', path: '/players/$playerSlug/career/season-pass' },
    ],
  },
  {
    key: 'ascension',
    label: 'Ascension',
    icon: <NavFlameIcon />,
    defaultPath: '/players/$playerSlug/ascension',
    matchPathname: (p) => /\/players\/[^/]+\/(objectifs|ascension)/.test(p),
    tabs: [
      { key: 'profile', label: 'Profil & objectifs', path: '/players/$playerSlug/ascension' },
      { key: 'realisations', label: 'Réalisations', path: '/players/$playerSlug/ascension/realisations' },
    ],
  },
  {
    key: 'community',
    label: 'Communauté',
    defaultPath: '/players/$playerSlug/palmares',
    matchPathname: isCommunityPath,
    tabs: [
      { key: 'leaderboard', label: 'Classements', path: '/players/$playerSlug/palmares' },
      { key: 'relations', label: 'Relations', path: '/players/$playerSlug/palmares/relations' },
      { key: 'compare', label: 'Face-à-face', path: '/players/$playerSlug/compare' },
      { key: 'prestige-leaderboard', label: 'Leaderboard PP', path: '/players/$playerSlug/palmares/prestige' },
    ],
  },
  {
    key: 'media',
    label: 'Médias',
    defaultPath: '/players/$playerSlug/media',
    matchPathname: (p) => /\/players\/[^/]+\/media/.test(p),
  },
  {
    key: 'explorer',
    label: 'Explorer',
    defaultPath: '/players/$playerSlug/explorer',
    matchPathname: (p) => /\/players\/[^/]+\/explorer/.test(p),
  },
]

// ─── SplitButton ──────────────────────────────────────────────────────────────

interface SplitButtonProps {
  section: L1Section & { tabs: L1Tab[] }
  isActive: boolean
  resolvedDefaultPath: string
  resolvePath: (tpl: string) => string
}

function SplitButton({ section, isActive, resolvedDefaultPath, resolvePath }: SplitButtonProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

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
          to={resolvedDefaultPath as never}
          className="flex items-center gap-1.5 px-3 py-1.5 whitespace-nowrap"
          aria-current={isActive ? 'page' : undefined}
        >
          {section.icon}
          {section.label}
        </Link>

        <span className="mx-0.5 h-4 w-px self-center rounded-full bg-current opacity-20" aria-hidden="true" />

        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="px-1.5 py-1.5 cursor-pointer"
          aria-label={`Onglets ${section.label}`}
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
              to={resolvePath(tab.path) as never}
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground whitespace-nowrap"
            >
              {tab.label}
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── SettingsSplitButton ─────────────────────────────────────────────────────

interface SettingsTabItem {
  key: string
  label: string
  tab: string
}

interface SettingsSplitButtonProps {
  tabs: SettingsTabItem[]
  isActive: boolean
}

function SettingsSplitButton({ tabs, isActive }: SettingsSplitButtonProps) {
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
          search={{ tab: 'general' }}
          className="px-2 py-1.5"
          aria-label="Paramètres"
          title="Paramètres"
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
          <div className="flex items-center justify-between gap-4 px-3 py-1.5">
            <span className="text-sm text-popover-foreground">Thème</span>
            <ThemeToggle variant="menu" />
          </div>
          <div role="separator" className="my-1 h-px bg-border" />
          {tabs.map((item) => (
            <Link
              key={item.key}
              to="/settings"
              search={{ tab: item.tab as 'general' | 'sync' | 'analyse' | 'accessibility' | 'notifications' | 'lab' | 'users' }}
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground whitespace-nowrap"
            >
              {item.label}
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function NavL1() {
  const navigate = useNavigate()
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const canManageInstance = useAppShellStore((s) => s.capabilities?.can_manage_instance ?? false)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const { data: settings } = useSettings()
  const showProgression = settings?.show_progression ?? true
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const playerSlug = currentPlayer?.player_slug ?? ''
  const visibleSections = showProgression
    ? L1_SECTIONS
    : L1_SECTIONS.filter((s) => s.key !== 'ascension')

  function resolvePath(templatePath: string): string {
    return templatePath.replace('$playerSlug', playerSlug)
  }

  function handlePlayerChange(slug: string) {
    const player = availablePlayers.find((p) => p.player_slug === slug)
    if (!player) return
    setCurrentPlayer(player)
    const nextPath = buildPlayerDestination(pathname, playerSlug, player.player_slug)
    navigate({ to: nextPath as never })
  }

  return (
    <nav
      className="flex h-12 shrink-0 items-center gap-0.5 border-b border-border bg-sidebar px-3"
      role="navigation"
      aria-label={t('common.shell.nav_main_aria')}
    >
      {/* ── Logo ────────────────────────────────────────────────────────── */}
      <Link
        to={playerSlug ? resolvePath('/players/$playerSlug/home') as never : '/'}
        className="mr-3 flex shrink-0 items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-sidebar-accent"
        aria-label={t('common.shell.logo_aria')}
      >
        <img src="/logo.png" alt="LevelUp" className="h-7 w-7 shrink-0 object-contain" />
        <span className="hidden text-sm font-bold text-sidebar-foreground sm:block">LevelUp</span>
      </Link>

      {/* ── Sections (seulement si un joueur est actif) ──────────────────── */}
      {playerSlug &&
        visibleSections.map((section) => {
          const isActive = section.matchPathname(pathname)
          const resolvedDefaultPath = resolvePath(section.defaultPath)

          if (section.tabs) {
            return (
              <SplitButton
                key={section.key}
                section={section as L1Section & { tabs: L1Tab[] }}
                isActive={isActive}
                resolvedDefaultPath={resolvedDefaultPath}
                resolvePath={resolvePath}
              />
            )
          }

          return (
            <Link
              key={section.key}
              to={resolvedDefaultPath as never}
              className={[
                'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors whitespace-nowrap',
                isActive
                  ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              ].join(' ')}
              aria-current={isActive ? 'page' : undefined}
            >
              {section.icon}
              {section.label}
            </Link>
          )
        })}

      {/* ── Spacer ──────────────────────────────────────────────────────── */}
      <div className="flex-1" />

      {/* ── Gamertag / sélecteur joueur ──────────────────────────────────── */}
      {availablePlayers.length > 1 ? (
        <div className="flex items-center gap-1.5">
          <select
            value={playerSlug}
            onChange={(e) => handlePlayerChange(e.target.value)}
            className="rounded-md border border-sidebar-border bg-sidebar-accent px-2 py-1 text-sm text-sidebar-foreground transition-colors focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/30"
            aria-label="Joueur actif"
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
          <span className="text-sm font-medium text-sidebar-foreground/70">
            {currentPlayer.gamertag}
          </span>
        )
      )}

      {/* ── Cloche notifications (per-player) ────────────────────────────── */}
      {currentPlayer && <NotificationsBell playerSlug={currentPlayer.player_slug} />}

      {/* ── Aide ────────────────────────────────────────────────────────── */}
      <div className="ml-1">
        <HelpSplitButton isActive={pathname.startsWith('/help')} />
      </div>

      {/* ── Split button Paramètres ──────────────────────────────────────── */}
      <SettingsSplitButton
        isActive={pathname.startsWith('/settings')}
        tabs={[
          { key: 'general', label: 'Général', tab: 'general' },
          { key: 'sync', label: 'Synchronisation', tab: 'sync' },
          { key: 'analyse', label: 'Analyse', tab: 'analyse' },
          { key: 'accessibility', label: 'Accessibilité', tab: 'accessibility' },
          { key: 'notifications', label: 'Notifications', tab: 'notifications' },
          ...(canManageInstance ? [{ key: 'lab', label: 'Lab', tab: 'lab' }] : []),
          ...(isAdmin ? [{ key: 'users', label: 'Utilisateurs', tab: 'users' }] : []),
        ]}
      />

      {/* ── Déconnexion (visible si session ouverte) ─────────────────────── */}
      <LogoutButton />
    </nav>
  )
}
