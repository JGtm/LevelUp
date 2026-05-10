/**
 * NavL2 — bandeau de contexte analytique (niveau 2).
 *
 * Visible dans les sections Stats, Escouade et Carrière.
 *
 * Stats    : sous-onglets (Séries · Sessions) + FilterOmnibar + PeriodSessionRail.
 * Escouade : FilterOmnibar uniquement (SquadLayout gère sa propre barre).
 * Carrière : sous-onglets (Progression · Citations · Pass saisonnier) uniquement.
 *
 * Sticky en dessous de NavL1 (top-0 dans le conteneur scrollable). Les filtres
 * (Session, Période, cascade) sont gérés par FilterOmnibar.tsx via des pills.
 */
import { Link, useRouterState, useParams } from '@tanstack/react-router'
import { FilterOmnibar } from './FilterOmnibar'
import { PeriodSessionRail } from './PeriodSessionRail'

// ─── Sous-onglets de la section Stats ─────────────────────────────────────────

const STATS_TABS = [
  { label: 'Séries', path: '/players/$playerSlug/stats/timeseries' },
  { label: 'Sessions', path: '/players/$playerSlug/stats/sessions' },
] as const

// ─── Sous-onglets de la section Carrière ──────────────────────────────────────

const CAREER_TABS = [
  { label: 'Progression', path: '/players/$playerSlug/career' },
  { label: 'Citations', path: '/players/$playerSlug/citations' },
  { label: 'Pass saisonnier', path: '/players/$playerSlug/career/season-pass' },
] as const

// ─── Helpers ──────────────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | 'career' | null

function detectSection(pathname: string): ActiveSection {
  if (/\/players\/[^/]+\/stats\//.test(pathname)) return 'stats'
  if (/\/players\/[^/]+\/squad/.test(pathname)) return 'squad'
  if (/\/players\/[^/]+\/(career|citations)/.test(pathname)) return 'career'
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

  // Carrière : barre d'onglets uniquement, pas de filtres analytiques.
  if (section === 'career') {
    return (
      <div
        className="sticky top-0 z-30 shrink-0 border-b border-border bg-background"
        role="navigation"
        aria-label="Navigation carrière"
      >
        <div className="flex items-center gap-0 px-4">
          {CAREER_TABS.map((tab) => {
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
      </div>
    )
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
