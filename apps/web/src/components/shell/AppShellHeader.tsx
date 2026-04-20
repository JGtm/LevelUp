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
      className="inline-flex items-center justify-center rounded-full border border-border bg-background/75 px-4 py-2 text-sm font-medium text-foreground transition hover:border-border hover:bg-primary/10 hover:text-primary [&.active]:bg-primary [&.active]:text-primary-foreground"
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
    <header className="sticky top-0 z-40 border-b border-border bg-background/85 backdrop-blur-xl">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-28 bg-[radial-gradient(circle_at_top,rgba(168,85,247,0.14),transparent_70%)]" />

      <div className="app-shell-width relative mx-auto flex w-full flex-col gap-4 px-4 py-4 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <Link to="/" className="flex min-w-0 items-start gap-4">
            <img
              src="/logo-full-inline.png"
              alt="LevelUp"
              className="h-12 shrink-0 object-contain"
            />
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="outline" className="border-border bg-background/70 text-muted-foreground">
                  {currentTitle}
                </Badge>
              </div>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
                Un shell plus compact, sans sidebar, pour lire vite et plonger plus loin quand
                c&apos;est utile.
              </p>
            </div>
          </Link>

          <div className="flex flex-wrap items-center gap-2 xl:justify-end">
            {linkedHaloIdentity && (
              <Badge variant="secondary" className="bg-muted px-3 py-1 text-foreground">
                Session Halo : {linkedHaloIdentity.gamertag}
              </Badge>
            )}
            {GLOBAL_SHELL_LINKS.map((item) => (
              <UtilityLink key={item.to} to={item.to} label={item.label} />
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-3 rounded-[28px] border border-border bg-card px-4 py-4 text-card-foreground shadow-[0_30px_80px_-42px_rgba(15,23,42,0.9)] xl:flex-row xl:items-center xl:justify-between">
          <div className="min-w-0">
            <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
              Scope joueur
            </p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <p className="text-xl font-semibold tracking-tight text-card-foreground">
                {currentPlayer?.gamertag ?? 'Aucun joueur sélectionné'}
              </p>
              {currentPlayer?.is_demo && (
                <Badge variant="outline" className="border-border/15 bg-card/5 text-card-foreground/80">
                  Démo
                </Badge>
              )}
            </div>
            <p className="mt-1 text-sm leading-6 text-muted-foreground">
              {currentPlayer
                ? `Waypoint : ${currentPlayer.waypoint_player}`
                : 'Sélectionne un joueur pour charger les analyses et conserver un contexte propre.'}
            </p>
          </div>

          {availablePlayers.length > 0 && (
            <label className="flex min-w-0 flex-col xl:min-w-[22rem]">
              <span className="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                Joueur actif
              </span>
              <div className="mt-2 flex flex-col gap-2 xl:items-end">
                <select
                  value={currentPlayer?.player_slug ?? ''}
                  onChange={(event) => handlePlayerChange(event.target.value)}
                  className="w-full rounded-2xl border border-border/10 bg-card/10 px-4 py-3 text-sm text-card-foreground outline-none transition focus:border-ring focus:bg-card/15"
                >
                  {!currentPlayer && <option value="">Sélectionner un joueur</option>}
                  {availablePlayers.map((player) => (
                    <option key={player.player_slug} value={player.player_slug} className="text-foreground">
                      {player.gamertag}
                    </option>
                  ))}
                </select>
                <span className="text-xs text-muted-foreground xl:text-right">
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
