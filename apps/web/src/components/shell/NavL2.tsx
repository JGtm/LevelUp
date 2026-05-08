/**
 * NavL2 — bandeau de contexte analytique (niveau 2).
 *
 * Visible uniquement dans les sections Stats et Escouade.
 *
 * Stats    : sous-onglets (Historique · Séries · Sessions) + FilterOmnibar.
 * Escouade : FilterOmnibar uniquement.
 *
 * Sticky en dessous de PeriodSessionRail (top-12). La nav session/période
 * vit dans PeriodSessionRail.tsx, pas ici. Les filtres (Session, Période,
 * cascade) sont gérés par FilterOmnibar.tsx via des pills cliquables.
 */
import { Link, useRouterState, useParams } from '@tanstack/react-router'
import { FilterOmnibar } from './FilterOmnibar'
import { PeriodSessionRail } from './PeriodSessionRail'

// ─── Sous-onglets de la section Stats ─────────────────────────────────────────

const STATS_TABS = [
  { label: 'Séries', path: '/players/$playerSlug/stats/timeseries' },
  { label: 'Sessions', path: '/players/$playerSlug/stats/sessions' },
] as const

// ─── Helpers ──────────────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | null

function detectSection(pathname: string): ActiveSection {
  if (/\/players\/[^/]+\/stats\//.test(pathname)) return 'stats'
  if (/\/players\/[^/]+\/squad/.test(pathname)) return 'squad'
  return null
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function NavL2() {
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = params.playerSlug ?? ''

  const section = detectSection(pathname)
  if (!section) return null
  // La section squad gère sa propre barre de filtres complète dans SquadLayout.
  if (section === 'squad') return null

  function resolvePath(tpl: string): string {
    return tpl.replace('$playerSlug', playerSlug)
  }

  return (
    <div
      className="sticky top-0 z-30 shrink-0 border-b border-border bg-background"
      role="navigation"
      aria-label="Navigation analytique"
    >
      {section === 'stats' && (
        <div className="flex items-center gap-0 border-b border-border px-4">
          {STATS_TABS.map((tab) => {
            const resolved = resolvePath(tab.path)
            const isActive = pathname === resolved
            return (
              <Link
                key={tab.label}
                to={resolved}
                className={[
                  'border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
                  isActive
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground',
                ].join(' ')}
                aria-current={isActive ? 'page' : undefined}
              >
                {tab.label}
              </Link>
            )
          })}
        </div>
      )}

      <FilterOmnibar />
      <PeriodSessionRail />
    </div>
  )
}
