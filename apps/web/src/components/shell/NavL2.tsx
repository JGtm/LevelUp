/**
 * NavL2 — bandeau de contexte analytique (niveau 2).
 *
 * Visible dans les sections Stats, Escouade et Carrière.
 *
 * Stats legacy (timeseries, history) : FilterOmnibar + PeriodSessionRail.
 * Stats perso (_personal.*) : NavL2 absent — PersonalStatsLayout gère sa propre barre.
 * Escouade : FilterOmnibar uniquement (SquadLayout gère sa propre barre).
 * Carrière : sous-onglets (Progression · Citations · Pass saisonnier) uniquement.
 *
 * Sticky en dessous de NavL1 (top-0 dans le conteneur scrollable). Les filtres
 * (Session, Période, cascade) sont gérés par FilterOmnibar.tsx via des pills.
 */
import { Link, useRouterState, useParams } from '@tanstack/react-router'
import { FilterOmnibar } from './FilterOmnibar'
import { PeriodSessionRail } from './PeriodSessionRail'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { isCommunityPath } from './shellNavigation'

// ─── Sous-onglets de la section Carrière ──────────────────────────────────────

const CAREER_TABS = [
  { label: 'Progression', path: '/players/$playerSlug/career' },
  { label: 'Citations', path: '/players/$playerSlug/citations' },
  { label: 'Pass saisonnier', path: '/players/$playerSlug/career/season-pass' },
] as const

// Communauté : aligné sur le dropdown L1 (NavL1 section 'community'). Face-à-face
// pointe vers /compare (hors /palmares), d'où des chemins absolus par onglet.
const COMMUNITY_TABS = [
  { label: 'Classements', path: '/players/$playerSlug/palmares' },
  { label: 'Relations', path: '/players/$playerSlug/palmares/relations' },
  { label: 'Face-à-face', path: '/players/$playerSlug/compare' },
] as const

// ─── Helpers ──────────────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | 'career' | 'community' | null

// Routes _personal : PersonalStatsLayout gère sa propre barre de filtres.
const PERSONAL_STATS_RE = /\/players\/[^/]+\/stats\/(summary|maps-modes|distributions|progression|advanced)/

function detectSection(pathname: string): ActiveSection {
  if (PERSONAL_STATS_RE.test(pathname)) return null
  if (/\/players\/[^/]+\/stats\//.test(pathname)) return 'stats'
  if (/\/players\/[^/]+\/squad/.test(pathname)) return 'squad'
  if (/\/players\/[^/]+\/(career|citations)/.test(pathname)) return 'career'
  if (isCommunityPath(pathname)) return 'community'
  return null
}

// ─── Barre d'onglets réutilisable (sticky, soulignée) ─────────────────────────

function NavTabBar({
  tabs,
  pathname,
  resolvePath,
  ariaLabel,
}: {
  tabs: readonly { readonly label: string; readonly path: string }[]
  pathname: string
  resolvePath: (tpl: string) => string
  ariaLabel: string
}) {
  return (
    <div
      className="sticky top-0 z-30 shrink-0 border-b border-border bg-background"
      role="navigation"
      aria-label={ariaLabel}
    >
      <div className="flex items-center gap-0 px-4">
        {tabs.map((tab) => {
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

// ─── Composant principal ──────────────────────────────────────────────────────

export function NavL2() {
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = params.playerSlug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const section = detectSection(pathname)
  if (!section) return null
  // La section squad gère sa propre barre de filtres complète dans SquadLayout.
  if (section === 'squad') return null

  function resolvePath(tpl: string): string {
    return tpl.replace('$playerSlug', playerSlug)
  }

  // Carrière & Communauté : barre d'onglets uniquement, pas de filtres analytiques.
  if (section === 'career') {
    return (
      <NavTabBar
        tabs={CAREER_TABS}
        pathname={pathname}
        resolvePath={resolvePath}
        ariaLabel={t('common.shell.nav_career_aria')}
      />
    )
  }

  if (section === 'community') {
    return (
      <NavTabBar
        tabs={COMMUNITY_TABS}
        pathname={pathname}
        resolvePath={resolvePath}
        ariaLabel={t('common.shell.nav_community_aria')}
      />
    )
  }

  // Stats : FilterOmnibar + PeriodSessionRail, sans onglets (virés).
  // match_context='solo' : Timeseries/History ne concernent que les matchs solo.
  // Page Sessions : on active le sélecteur de contexte (Solo/Escouade/Mixte) —
  // les autres pages stats restent figées sur 'solo' (aucune régression).
  const isSessionsPage = /\/players\/[^/]+\/stats\/sessions/.test(pathname)
  return (
    <div
      className="sticky top-0 z-30 shrink-0 border-b border-border bg-background"
      role="navigation"
      aria-label={t('common.shell.nav_analytics_aria')}
    >
      <FilterOmnibar matchContext="solo" contextSelectable={isSessionsPage} />
      <PeriodSessionRail />
    </div>
  )
}
