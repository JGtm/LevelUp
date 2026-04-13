/**
 * NavBar — barre de navigation latérale principale de LevelUp.
 */
import { Link } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'

const PLAYER_NAV_ITEMS = [
  { to: '/players/$playerSlug/home', label: 'Accueil', icon: '🏠' },
  { to: '/players/$playerSlug/last-match', label: 'Dernier Match', icon: '⚡' },
  { to: '/players/$playerSlug/career', label: 'Carrière', icon: '🎖️' },
  { to: '/players/$playerSlug/profile/citations', label: 'Citations', icon: '🏅' },
  { to: '/players/$playerSlug/stats/history', label: 'Historique', icon: '📊' },
  { to: '/players/$playerSlug/stats/timeseries', label: 'Séries', icon: '📈' },
  { to: '/players/$playerSlug/stats/sessions', label: 'Sessions', icon: '🔄' },
  { to: '/players/$playerSlug/explorer', label: 'Explorer', icon: '🔍' },
  { to: '/players/$playerSlug/squad', label: 'Escouade', icon: '👥' },
  { to: '/players/$playerSlug/synthesis', label: 'Synthèse', icon: '🗂️' },
  { to: '/players/$playerSlug/media', label: 'Médias', icon: '🎬' },
] as const

interface NavLinkProps {
  to: string
  label: string
  icon: string
  params: Record<string, string>
}

function NavLink({ to, label, icon, params }: NavLinkProps) {
  return (
    <Link
      to={to}
      params={params}
      className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-purple-50 hover:text-purple-700 [&.active]:bg-purple-100 [&.active]:text-purple-800"
    >
      <span className="text-base">{icon}</span>
      <span>{label}</span>
    </Link>
  )
}

export function NavBar() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)

  return (
    <nav className="flex h-full w-56 flex-col gap-1 border-r border-gray-200 bg-white p-3">
      {/* Logo + titre */}
      <div className="mb-4 flex items-center gap-2 px-3 py-2">
        <span className="text-2xl">⚔️</span>
        <span className="text-base font-bold text-gray-900">LevelUp</span>
      </div>

      {/* Sélecteur joueur */}
      {availablePlayers.length > 0 && (
        <div className="mb-2 px-1">
          <label htmlFor="player-select" className="mb-1 block text-xs font-medium text-gray-500">
            Joueur actif
          </label>
          <select
            id="player-select"
            value={currentPlayer?.player_slug ?? ''}
            onChange={(e) => {
              const p = availablePlayers.find((pl) => pl.player_slug === e.target.value)
              if (p) setCurrentPlayer(p)
            }}
            className="w-full rounded-md border border-gray-300 bg-white px-2 py-1.5 text-sm focus:ring-2 focus:ring-purple-500 outline-none"
          >
            {availablePlayers.map((p) => (
              <option key={p.player_slug} value={p.player_slug}>
                {p.gamertag}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Liens navigation joueur */}
      {currentPlayer &&
        PLAYER_NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            label={item.label}
            icon={item.icon}
            params={{ playerSlug: currentPlayer.player_slug }}
          />
        ))}

      {/* Spacer */}
      <div className="flex-1" />

      {/* Liens globaux */}
      <div className="border-t border-gray-100 pt-2">
        <Link
          to="/settings"
          className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 [&.active]:bg-gray-100"
        >
          <span className="text-base">⚙️</span>
          <span>Paramètres</span>
        </Link>
      </div>
    </nav>
  )
}
