import { Link, useNavigate, useRouterState } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  buildPlayerDestination,
  GLOBAL_SHELL_LINKS,
} from './shellNavigation'

function UtilityLink({ to, label }: { to: '/settings' | '/changelog'; label: string }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center justify-center rounded-full border border-slate-200 bg-white/75 px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-violet-200 hover:bg-violet-50 hover:text-violet-700 [&.active]:border-slate-950 [&.active]:bg-slate-950 [&.active]:text-white"
    >
      {label}
    </Link>
  )
}

export function AppShellHeader() {
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const setCurrentPlayer = useAppShellStore((s) => s.setCurrentPlayer)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const linkedHaloIdentity = useAppShellStore((s) => s.linkedHaloIdentity)
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)

  const currentTitle =
    availableTitles.find((title) => title.slug === currentTitleSlug)?.name ?? currentTitleSlug

  function handlePlayerChange(nextPlayerSlug: string) {
    const nextPlayer = availablePlayers.find((player) => player.player_slug === nextPlayerSlug)
    if (!nextPlayer) return

    setCurrentPlayer(nextPlayer)
    const nextPath = buildPlayerDestination(
      pathname,
      currentPlayer?.player_slug,
      nextPlayer.player_slug,
    )
    navigate({ to: nextPath as never })
  }

  return (
    <header className="sticky top-0 z-40 border-b border-slate-200/80 bg-white/85 backdrop-blur-xl">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-28 bg-[radial-gradient(circle_at_top,rgba(168,85,247,0.14),transparent_70%)]" />

      <div className="relative mx-auto flex w-full max-w-7xl flex-col gap-4 px-4 py-4 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <Link to="/" className="flex min-w-0 items-start gap-4">
            <div className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl bg-slate-950 text-sm font-semibold uppercase tracking-[0.22em] text-white shadow-[0_24px_48px_-28px_rgba(15,23,42,0.7)]">
              LU
            </div>
            <div className="min-w-0">
              <p className="text-[11px] font-semibold uppercase tracking-[0.28em] text-slate-500">
                Halo Ops Center
              </p>
              <div className="mt-1 flex flex-wrap items-center gap-2">
                <span className="text-xl font-semibold tracking-tight text-slate-950 sm:text-2xl">
                  LevelUp
                </span>
                <Badge variant="outline" className="border-slate-200 bg-white/70 text-slate-600">
                  {currentTitle}
                </Badge>
              </div>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
                Un shell plus compact, sans sidebar, pour lire vite et plonger plus loin quand
                c&apos;est utile.
              </p>
            </div>
          </Link>

          <div className="flex flex-wrap items-center gap-2 xl:justify-end">
            {linkedHaloIdentity && (
              <Badge variant="secondary" className="bg-slate-100 px-3 py-1 text-slate-700">
                Session Halo : {linkedHaloIdentity.gamertag}
              </Badge>
            )}
            {activeSyncJobId && (
              <Badge variant="default" className="bg-violet-600 px-3 py-1 text-white">
                Sync active
              </Badge>
            )}
            {GLOBAL_SHELL_LINKS.map((item) => (
              <UtilityLink key={item.to} to={item.to} label={item.label} />
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-3 rounded-[28px] border border-slate-200/80 bg-slate-950 px-4 py-4 text-white shadow-[0_30px_80px_-42px_rgba(15,23,42,0.9)] xl:flex-row xl:items-center xl:justify-between">
          <div className="min-w-0">
            <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-400">
              Scope joueur
            </p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <p className="text-xl font-semibold tracking-tight text-white">
                {currentPlayer?.gamertag ?? 'Aucun joueur sélectionné'}
              </p>
              {currentPlayer?.is_demo && (
                <Badge variant="outline" className="border-white/15 bg-white/5 text-white/80">
                  Démo
                </Badge>
              )}
            </div>
            <p className="mt-1 text-sm leading-6 text-slate-300">
              {currentPlayer
                ? `Waypoint : ${currentPlayer.waypoint_player}`
                : 'Sélectionne un joueur pour charger les analyses et conserver un contexte propre.'}
            </p>
          </div>

          {availablePlayers.length > 0 && (
            <label className="flex min-w-0 flex-col xl:min-w-[22rem]">
              <span className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-400">
                Joueur actif
              </span>
              <div className="mt-2 flex flex-col gap-2 xl:items-end">
                <select
                  value={currentPlayer?.player_slug ?? ''}
                  onChange={(event) => handlePlayerChange(event.target.value)}
                  className="w-full rounded-2xl border border-white/10 bg-white/10 px-4 py-3 text-sm text-white outline-none transition focus:border-violet-300 focus:bg-white/15"
                >
                  {!currentPlayer && <option value="">Sélectionner un joueur</option>}
                  {availablePlayers.map((player) => (
                    <option key={player.player_slug} value={player.player_slug} className="text-slate-950">
                      {player.gamertag}
                    </option>
                  ))}
                </select>
                <span className="text-xs text-slate-400 xl:text-right">
                  Le shell garde la section courante quand c&apos;est possible, sinon il revient sur
                  l&apos;accueil du joueur.
                </span>
              </div>
            </label>
          )}
        </div>
      </div>
    </header>
  )
}
