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
import { useCapability } from '@/lib/capabilities/capabilities'
import { isCommunityPath } from './shellNavigation'

// Onglet « Classements » de la section Communauté (gaté sur world.leaderboard).
const COMMUNITY_LEADERBOARD_PATH = '/players/$playerSlug/community'

// ─── Sous-onglets de la section Carrière ──────────────────────────────────────

const CAREER_TABS = [
  { label: 'Progression', path: '/players/$playerSlug/career' },
  { label: 'Citations', path: '/players/$playerSlug/career/citations' },
  { label: 'Pass saisonnier', path: '/players/$playerSlug/career/season-pass' },
] as const

// Halo 5 : les commendations sont NATIVES (carnage) — l'onglet « Citations » (moteur
// dérivé d'Infinite, capability `citations.engine` not_exposed pour h5) est remplacé
// par « Commendations » (totaux à vie natifs). Le slug courant est le SEUL signal
// front pour distinguer h5 : aucune capability COARSE ne le fait (Infinite les
// déclare toutes) et `commendations.native` est une capability FINE non exposée au nav.
// Halo 5 n'a PAS de pass saisonnier / Battlepass (capability `season_pass` absente).
// L'inventaire REQ personnel n'étant pas servi (sonde 404), aucune surface de
// remplacement n'est câblée → pas d'onglet « Pass saisonnier » pour h5.
const CAREER_TABS_H5 = [
  { label: 'Progression', path: '/players/$playerSlug/career' },
  // Halo 5 : commendations natives, libellé FR « Citations » (terme officiel Halo
  // FR, cohérent avec Infinite et l'onglet L1).
  { label: 'Citations', path: '/players/$playerSlug/career/commendations' },
] as const

// Communauté : aligné sur le dropdown L1 (NavL1 section 'community'). Face-à-face
// pointe vers /compare (hors /palmares), d'où des chemins absolus par onglet.
const COMMUNITY_TABS = [
  { label: 'Classements', path: '/players/$playerSlug/community' },
  { label: 'Relations', path: '/players/$playerSlug/community/relations' },
  { label: 'Face-à-face', path: '/players/$playerSlug/community/compare' },
] as const

// ─── Helpers ──────────────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | 'career' | 'community' | null

// Routes _personal : PersonalStatsLayout gère sa propre barre de filtres.
const PERSONAL_STATS_RE = /\/players\/[^/]+\/stats\/(summary|maps-modes|distributions|progression|advanced)/

function detectSection(pathname: string): ActiveSection {
  if (PERSONAL_STATS_RE.test(pathname)) return null
  // Synthèse gère sa propre barre de filtres (PeriodePill/SaisonPill) → pas de NavL2.
  if (/\/players\/[^/]+\/stats\/synthesis/.test(pathname)) return null
  if (/\/players\/[^/]+\/stats\//.test(pathname)) return 'stats'
  if (/\/players\/[^/]+\/squad/.test(pathname)) return 'squad'
  if (/\/players\/[^/]+\/(career|citations|commendations)/.test(pathname)) return 'career'
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
  // Gating multi-titre (Phase 5) — hooks appelés inconditionnellement (avant tout
  // early-return). NO-OP pour halo_infinite (déclare career + world.leaderboard).
  const hasCareer = useCapability('career')
  const hasWorldLeaderboard = useCapability('world.leaderboard')
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)

  const section = detectSection(pathname)
  if (!section) return null
  // La section squad gère sa propre barre de filtres complète dans SquadLayout.
  if (section === 'squad') return null

  function resolvePath(tpl: string): string {
    return tpl.replace('$playerSlug', playerSlug)
  }

  // Carrière & Communauté : barre d'onglets uniquement, pas de filtres analytiques.
  if (section === 'career') {
    // Titre sans capability `career` : pas de barre (la page est gatée en amont).
    if (!hasCareer) return null
    // h5 : onglets carrière avec « Commendations » natif au lieu de « Citations ».
    const careerTabs = currentTitleSlug === 'halo_5' ? CAREER_TABS_H5 : CAREER_TABS
    return (
      <NavTabBar
        tabs={careerTabs}
        pathname={pathname}
        resolvePath={resolvePath}
        ariaLabel={t('common.shell.nav_career_aria')}
      />
    )
  }

  if (section === 'community') {
    // Section transverse : on retire seulement l'onglet « Classements » si le titre
    // ne déclare pas world.leaderboard (Relations / Face-à-face restent visibles).
    const communityTabs = hasWorldLeaderboard
      ? COMMUNITY_TABS
      : COMMUNITY_TABS.filter((tab) => tab.path !== COMMUNITY_LEADERBOARD_PATH)
    return (
      <NavTabBar
        tabs={communityTabs}
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
