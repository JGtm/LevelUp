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
import { buildPlayerDestination } from './shellNavigation'

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
  /** Route par défaut lors du clic sur le label (avec $playerSlug en placeholder). */
  defaultPath: string
  /** Retourne true si le pathname courant appartient à cette section. */
  matchPathname: (pathname: string) => boolean
  /** Onglets du dropdown (optionnel — si absent, bouton simple). */
  tabs?: L1Tab[]
}

const L1_SECTIONS: L1Section[] = [
  {
    key: 'home',
    label: 'Accueil',
    defaultPath: '/players/$playerSlug/home',
    matchPathname: (p) => /\/players\/[^/]+\/home/.test(p),
  },
  {
    key: 'stats',
    label: 'Stats',
    defaultPath: '/players/$playerSlug/stats/history',
    matchPathname: (p) => /\/players\/[^/]+\/stats\//.test(p),
    tabs: [
      { key: 'history', label: 'Historique', path: '/players/$playerSlug/stats/history' },
      { key: 'timeseries', label: 'Séries', path: '/players/$playerSlug/stats/timeseries' },
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
    key: 'explorer',
    label: 'Explorer',
    defaultPath: '/players/$playerSlug/explorer',
    matchPathname: (p) => /\/players\/[^/]+\/explorer/.test(p),
  },
  {
    key: 'media',
    label: 'Médias',
    defaultPath: '/players/$playerSlug/media',
    matchPathname: (p) => /\/players\/[^/]+\/media/.test(p),
  },
  {
    key: 'palmares',
    label: 'Palmarès',
    defaultPath: '/players/$playerSlug/palmares',
    matchPathname: (p) => /\/players\/[^/]+\/palmares(?:\/|$)/.test(p),
    tabs: [
      { key: 'leaderboard', label: 'Classements', path: '/players/$playerSlug/palmares' },
      { key: 'relations', label: 'Relations', path: '/players/$playerSlug/palmares/relations' },
      { key: 'compare', label: 'Face-à-face', path: '/players/$playerSlug/palmares/compare' },
      { key: 'season-pass', label: 'Pass saisonnier', path: '/players/$playerSlug/palmares/season-pass' },
    ],
  },
  {
    key: 'career',
    label: 'Carrière',
    defaultPath: '/players/$playerSlug/career',
    matchPathname: (p) => /\/players\/[^/]+\/(career|profile)/.test(p),
    tabs: [
      { key: 'progression', label: 'Progression', path: '/players/$playerSlug/career' },
      { key: 'citations', label: 'Citations', path: '/players/$playerSlug/profile/citations' },
    ],
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
          className="px-3 py-1.5 whitespace-nowrap"
          aria-current={isActive ? 'page' : undefined}
        >
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

// ─── Composant principal ──────────────────────────────────────────────────────

export function NavL1() {
  const navigate = useNavigate()
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const authMode = useAppShellStore((s) => s.authMode)
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const playerSlug = currentPlayer?.player_slug ?? ''

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
      aria-label="Navigation principale"
    >
      {/* ── Logo ────────────────────────────────────────────────────────── */}
      <Link
        to={playerSlug ? resolvePath('/players/$playerSlug/home') as never : '/'}
        className="mr-3 flex shrink-0 items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-sidebar-accent"
        aria-label="LevelUp — retour à l'accueil"
      >
        <img src="/logo.png" alt="LevelUp" className="h-7 w-7 shrink-0 object-contain" />
        <span className="hidden text-sm font-bold text-sidebar-foreground sm:block">LevelUp</span>
      </Link>

      {/* ── Sections (seulement si un joueur est actif) ──────────────────── */}
      {playerSlug &&
        L1_SECTIONS.map((section) => {
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
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors whitespace-nowrap',
                isActive
                  ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              ].join(' ')}
              aria-current={isActive ? 'page' : undefined}
            >
              {section.label}
            </Link>
          )
        })}

      {/* ── Spacer ──────────────────────────────────────────────────────── */}
      <div className="flex-1" />

      {/* ── Gamertag / sélecteur joueur ──────────────────────────────────── */}
      {availablePlayers.length > 1 ? (
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
      ) : (
        currentPlayer && (
          <span className="shrink-0 text-sm font-medium text-sidebar-foreground/70">
            {currentPlayer.gamertag}
          </span>
        )
      )}

      <ThemeToggle className="ml-2" />

      {/* ── Lien Admin (mode password, rôle admin) ─────────────────────── */}
      {authMode === 'password' && isAdmin && (
        <Link
          to="/admin"
          className="ml-1 shrink-0 rounded-md px-2 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground [&.active]:text-sidebar-foreground"
          title="Administration"
          aria-label="Administration"
        >
          Admin
        </Link>
      )}

      {/* ── Lien Paramètres ─────────────────────────────────────────────── */}
      <Link
        to="/settings"
        className="ml-1 shrink-0 rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground [&.active]:text-sidebar-foreground"
        title="Paramètres"
        aria-label="Paramètres"
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
    </nav>
  )
}
