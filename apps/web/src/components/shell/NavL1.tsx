/**
 * NavL1 — barre de navigation principale (niveau 1).
 *
 * Barre horizontale fixée en haut de l'application, visible sur toutes les pages.
 * Contient : logo · 6 sections joueur · sélecteur de joueur actif · lien paramètres.
 *
 * Les 6 sections correspondent au plan V7 : Accueil · Stats · Escouade ·
 * Explorer · Médias · Profil. La détection de la section active se fait sur
 * le pathname courant (pas sur les classes CSS du Link, pour gérer les
 * groupes de sous-routes, ex. /stats/history = section Stats).
 */
import { Link, useRouterState } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'

// ─── Définition des sections L1 ───────────────────────────────────────────────

interface L1Section {
  key: string
  label: string
  /** Route par défaut lors du clic (avec $playerSlug en placeholder). */
  defaultPath: string
  /** Retourne true si le pathname courant appartient à cette section. */
  matchPathname: (pathname: string) => boolean
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
  },
  {
    key: 'squad',
    label: 'Escouade',
    defaultPath: '/players/$playerSlug/squad',
    matchPathname: (p) => /\/players\/[^/]+\/squad/.test(p),
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
    key: 'career',
    label: 'Carrière',
    defaultPath: '/players/$playerSlug/career',
    matchPathname: (p) => /\/players\/[^/]+\/(career|profile)/.test(p),
  },
]

// ─── Composant ────────────────────────────────────────────────────────────────

export function NavL1() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const playerSlug = currentPlayer?.player_slug ?? ''

  function resolvedPath(templatePath: string): string {
    return templatePath.replace('$playerSlug', playerSlug)
  }

  function handlePlayerChange(slug: string) {
    const player = availablePlayers.find((p) => p.player_slug === slug)
    if (player) setCurrentPlayer(player)
    // La navigation vers la route équivalente pour le nouveau joueur
    // est gérée par buildPlayerDestination si besoin (on laisse le
    // comportement natif du sélecteur ici — l'URL sera mise à jour
    // par le useEffect dans $playerSlug.tsx).
  }

  return (
    <nav
      className="flex h-12 shrink-0 items-center gap-0.5 border-b border-gray-200 bg-white px-3"
      role="navigation"
      aria-label="Navigation principale"
    >
      {/* ── Logo ────────────────────────────────────────────────────────── */}
      <Link
        to={playerSlug ? resolvedPath('/players/$playerSlug/home') : '/'}
        className="mr-3 flex shrink-0 items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-gray-50"
        aria-label="LevelUp — retour à l'accueil"
      >
        <span className="text-lg leading-none">⚔️</span>
        <span className="hidden text-sm font-bold text-gray-900 sm:block">LevelUp</span>
      </Link>

      {/* ── Sections (seulement si un joueur est actif) ──────────────────── */}
      {playerSlug &&
        L1_SECTIONS.map((section) => {
          const isActive = section.matchPathname(pathname)
          return (
            <Link
              key={section.key}
              to={resolvedPath(section.defaultPath)}
              className={[
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors whitespace-nowrap',
                isActive
                  ? 'bg-purple-100 text-purple-800'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900',
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
          className="rounded-md border border-gray-200 bg-white px-2 py-1 text-sm text-gray-700 transition-colors focus:border-purple-400 focus:outline-none focus:ring-2 focus:ring-purple-200"
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
          <span className="shrink-0 text-sm font-medium text-gray-600">
            {currentPlayer.gamertag}
          </span>
        )
      )}

      {/* ── Lien Paramètres ─────────────────────────────────────────────── */}
      <Link
        to="/settings"
        className="ml-1 shrink-0 rounded-md p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 [&.active]:text-gray-700"
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
