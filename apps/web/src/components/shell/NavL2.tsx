/**
 * NavL2 — bandeau de contexte analytique (niveau 2).
 *
 * Visible dans les sections Stats, Escouade et Carrière.
 *
 * Stats legacy (timeseries, history) : FilterOmnibar + PeriodSessionRail.
 * Stats perso (_personal.*) : NavL2 absent — PersonalStatsLayout gère sa propre barre.
 * Escouade : FilterOmnibar uniquement (SquadLayout gère sa propre barre).
 * Carrière : sous-onglets (Progression · Citations · Médailles · Pass saisonnier) uniquement.
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
import { isCommunityPath, type RouteTo } from './shellNavigation'
import { playerRelativePath, routeTemplateSuffix, useTitleSlug } from '@/lib/title-routing'

// Onglet « Classements » de la section Communauté (gaté sur world.leaderboard).
const COMMUNITY_LEADERBOARD_PATH: RouteTo = '/{-$lang}/t/$titleSlug/players/$playerSlug/community'

// Onglet NavL2 : libellé référencé par clé i18n (résolu au rendu via `t()`).
type NavTab = { readonly labelKey: CommonManifestKey; readonly path: RouteTo }

// ─── Sous-onglets de la section Carrière ──────────────────────────────────────

const CAREER_TABS = [
  { labelKey: 'common.nav.tab_progression', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career' },
  { labelKey: 'common.nav.tab_citations', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations' },
  { labelKey: 'common.nav.tab_medals', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/medals' },
  { labelKey: 'common.nav.tab_season_pass', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/season-pass' },
] as const satisfies readonly NavTab[]

// Halo 5 : les commendations sont NATIVES (carnage) — l'onglet « Citations » (moteur
// dérivé d'Infinite, capability `citations.engine` not_exposed pour h5) est remplacé
// par « Commendations » (totaux à vie natifs). Le slug courant est le SEUL signal
// front pour distinguer h5 : aucune capability COARSE ne le fait (Infinite les
// déclare toutes) et `commendations.native` est une capability FINE non exposée au nav.
// Halo 5 n'a PAS de pass saisonnier / Battlepass (capability `season_pass` absente).
// L'inventaire REQ personnel n'étant pas servi (sonde 404), aucune surface de
// remplacement n'est câblée → pas d'onglet « Pass saisonnier » pour h5.
const CAREER_TABS_H5 = [
  { labelKey: 'common.nav.tab_progression', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career' },
  // Halo 5 : commendations natives. Clé `tab_citations` = FR « Citations » (terme
  // officiel Halo FR, cohérent Infinite et l'onglet L1) / EN « Commendations ».
  { labelKey: 'common.nav.tab_citations', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/commendations' },
  // Halo 5 a des médailles natives (ingest medals.go) — page title-agnostic partagée.
  { labelKey: 'common.nav.tab_medals', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/medals' },
] as const satisfies readonly NavTab[]

// Communauté : aligné sur le dropdown L1 (NavL1 section 'community'). Face-à-face
// pointe vers /compare (hors /palmares), d'où des chemins absolus par onglet.
const COMMUNITY_TABS = [
  { labelKey: 'common.nav.tab_leaderboard', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/community' },
  { labelKey: 'common.nav.tab_relations', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/relations' },
  { labelKey: 'common.nav.tab_compare', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/compare' },
] as const satisfies readonly NavTab[]

// ─── Helpers ──────────────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | 'career' | 'community' | null

// Routes _personal : PersonalStatsLayout gère sa propre barre de filtres. Matchers
// sur le SUFFIXE relatif au joueur (playerRelativePath) — aucun littéral `/players/`.
const PERSONAL_STATS_RE = /^\/stats\/(summary|maps-modes|distributions|progression|advanced)/

function detectSection(pathname: string): ActiveSection {
  const suffix = playerRelativePath(pathname)
  if (suffix === null) return null
  if (PERSONAL_STATS_RE.test(suffix)) return null
  // Synthèse gère sa propre barre de filtres (PeriodePill/SaisonPill) → pas de NavL2.
  if (/^\/stats\/synthesis/.test(suffix)) return null
  if (/^\/stats\//.test(suffix)) return 'stats'
  if (/^\/squad/.test(suffix)) return 'squad'
  if (/^\/(career|citations|commendations)/.test(suffix)) return 'career'
  if (isCommunityPath(pathname)) return 'community'
  return null
}

// ─── Barre d'onglets réutilisable (sticky, soulignée) ─────────────────────────

function NavTabBar({
  tabs,
  pathname,
  titleSlug,
  playerSlug,
  ariaLabel,
  t,
}: {
  tabs: readonly NavTab[]
  pathname: string
  titleSlug: string
  playerSlug: string
  ariaLabel: string
  t: (key: CommonManifestKey) => string
}) {
  const suffix = playerRelativePath(pathname)
  return (
    <div
      className="sticky top-0 z-30 shrink-0 bg-background px-6"
      role="navigation"
      aria-label={ariaLabel}
    >
      <div className="flex items-center gap-0 border-b border-border">
        {tabs.map((tab) => {
          const isActive = suffix !== null && suffix === routeTemplateSuffix(tab.path)
          return (
            <Link
              key={tab.path}
              to={tab.path}
              params={{ titleSlug, playerSlug }}
              className={[
                'border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground',
              ].join(' ')}
              aria-current={isActive ? 'page' : undefined}
            >
              {t(tab.labelKey)}
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
  const titleSlug = useTitleSlug()
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
        titleSlug={titleSlug}
        playerSlug={playerSlug}
        ariaLabel={t('common.shell.nav_career_aria')}
        t={t}
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
        titleSlug={titleSlug}
        playerSlug={playerSlug}
        ariaLabel={t('common.shell.nav_community_aria')}
        t={t}
      />
    )
  }

  // Stats : FilterOmnibar + PeriodSessionRail, sans onglets (virés).
  // match_context='solo' : Timeseries/History ne concernent que les matchs solo.
  // Page Sessions : on active le sélecteur de contexte (Solo/Escouade/Mixte) —
  // les autres pages stats restent figées sur 'solo' (aucune régression).
  const isSessionsPage = /^\/stats\/sessions/.test(playerRelativePath(pathname) ?? '')
  return (
    <div
      className="sticky top-0 z-30 shrink-0 bg-background px-6"
      role="navigation"
      aria-label={t('common.shell.nav_analytics_aria')}
    >
      <FilterOmnibar matchContext="solo" contextSelectable={isSessionsPage} />
      <PeriodSessionRail />
    </div>
  )
}
