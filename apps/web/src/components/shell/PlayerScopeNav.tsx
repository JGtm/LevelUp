import { Link } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  PLAYER_PRIMARY_NAV_ITEMS,
  PLAYER_SECONDARY_NAV_ITEMS,
  type ShellNavItem,
} from './shellNavigation'

interface PlayerNavLinkProps {
  item: ShellNavItem
  playerSlug: string
  tone: 'primary' | 'secondary'
}

function PlayerNavLink({ item, playerSlug, tone }: PlayerNavLinkProps) {
  const base =
    tone === 'primary'
      ? 'group flex min-w-[15rem] shrink-0 flex-col rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-left transition hover:border-violet-200 hover:bg-violet-50/70 [&.active]:border-slate-950 [&.active]:bg-slate-950 [&.active]:text-white'
      : 'group inline-flex items-start gap-3 rounded-full border border-slate-200 bg-white px-4 py-2 text-left text-sm font-medium text-slate-700 transition hover:border-violet-200 hover:bg-violet-50/80 hover:text-violet-700 [&.active]:border-slate-950 [&.active]:bg-slate-950 [&.active]:text-white'

  return (
    <Link to={item.to} params={{ playerSlug }} className={base}>
      {tone === 'primary' ? (
        <>
          <span className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-500 transition group-[&.active]:text-white/60">
            {item.eyebrow}
          </span>
          <span className="mt-1 text-base font-semibold tracking-tight">{item.label}</span>
          <span className="mt-1 text-sm leading-5 text-slate-600 transition group-[&.active]:text-white/72">
            {item.description}
          </span>
        </>
      ) : (
        <>
          <span className="text-[10px] font-semibold uppercase tracking-[0.22em] text-slate-500 transition group-[&.active]:text-white/60">
            {item.eyebrow}
          </span>
          <span>{item.label}</span>
        </>
      )}
    </Link>
  )
}

export function PlayerScopeNav() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  if (!currentPlayer) return null

  return (
    <nav aria-label="Navigation joueur" className="space-y-3">
      <div className="rounded-[28px] border border-slate-200/80 bg-white/90 p-3 shadow-sm backdrop-blur">
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-2 px-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-500">
                Navigation joueur
              </p>
              <h2 className="mt-1 text-base font-semibold tracking-tight text-slate-950">
                Parcours principal
              </h2>
            </div>
            <Badge variant="outline" className="w-fit border-slate-200 bg-slate-50 text-slate-600">
              {currentPlayer.gamertag}
            </Badge>
          </div>

          <div className="flex gap-3 overflow-x-auto px-1 pb-1">
            {PLAYER_PRIMARY_NAV_ITEMS.map((item) => (
              <PlayerNavLink
                key={item.to}
                item={item}
                playerSlug={currentPlayer.player_slug}
                tone="primary"
              />
            ))}
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200/80 bg-white/80 p-3 shadow-sm backdrop-blur">
        <div className="mb-2 flex items-center justify-between px-2">
          <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-500">
            Vues secondaires
          </p>
          <span className="text-xs text-slate-400">Accès plus ciblés</span>
        </div>

        <div className="flex flex-wrap gap-2">
          {PLAYER_SECONDARY_NAV_ITEMS.map((item) => (
            <PlayerNavLink
              key={item.to}
              item={item}
              playerSlug={currentPlayer.player_slug}
              tone="secondary"
            />
          ))}
        </div>
      </div>
    </nav>
  )
}
