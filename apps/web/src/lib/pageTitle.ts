import {
  GLOBAL_SHELL_LINKS,
  PLAYER_PRIMARY_NAV_ITEMS,
  PLAYER_SECONDARY_NAV_ITEMS,
} from '@/components/shell/shellNavigation'

interface RouteTitleRule {
  pattern: string
  title: string
}

const PLAYER_ROUTE_OVERRIDES: RouteTitleRule[] = [
  // Solo
  { pattern: '/players/$playerSlug/stats/timeseries', title: 'Séries temporelles' },
  { pattern: '/players/$playerSlug/stats/sessions', title: 'Sessions' },
  { pattern: '/players/$playerSlug/stats/synthesis', title: 'Synthèse' },
  { pattern: '/players/$playerSlug/stats', title: 'Solo' },
  // Carrière
  { pattern: '/players/$playerSlug/career/season-pass', title: 'Pass saisonnier' },
  { pattern: '/players/$playerSlug/career', title: 'Carrière' },
  { pattern: '/players/$playerSlug/career/citations', title: 'Citations' },
  { pattern: '/players/$playerSlug/career/commendations', title: 'Citations' },
  // Communauté / Palmarès
  { pattern: '/players/$playerSlug/community/compare', title: 'Face-à-face' },
  { pattern: '/players/$playerSlug/community/relations', title: 'Relations' },
  { pattern: '/players/$playerSlug/community/prestige', title: 'Leaderboard PP' },
  // Ascension (refonte 4 onglets 2026-07 : Profil + Objectifs + Entraînement + Réalisations)
  { pattern: '/players/$playerSlug/ascension/objectifs', title: 'Ascension — Objectifs' },
  { pattern: '/players/$playerSlug/ascension/coaching', title: 'Ascension — Entraînement' },
  { pattern: '/players/$playerSlug/ascension/realisations', title: 'Ascension — Réalisations' },
  { pattern: '/players/$playerSlug/ascension', title: 'Ascension' },
  // Route historique /objectifs redirect → /ascension/objectifs (préservée pour bookmarks).
  { pattern: '/players/$playerSlug/objectifs', title: 'Ascension' },
  // Escouade
  { pattern: '/players/$playerSlug/squad/contributions', title: 'Contributions' },
  { pattern: '/players/$playerSlug/squad/synergies', title: 'Synergies' },
  // Divers
  { pattern: '/players/$playerSlug/matches/$matchId/replay', title: 'Replay' },
  { pattern: '/players/$playerSlug/matches/$matchId', title: 'Match' },
  { pattern: '/players/$playerSlug/notifications', title: 'Notifications' },
  { pattern: '/players/$playerSlug', title: 'Accueil' },
]

const STATIC_ROUTE_TITLES: RouteTitleRule[] = [
  { pattern: '/', title: 'Accueil' },
  { pattern: '/admin', title: 'Administration' },
  { pattern: '/changelog', title: 'Changelog' },
  { pattern: '/lab', title: 'Lab interne' },
  { pattern: '/login', title: 'Connexion' },
  { pattern: '/register', title: 'Inscription' },
  { pattern: '/settings', title: 'Parametres' },
  { pattern: '/setup', title: 'Configuration' },
]

const ROUTE_TITLE_RULES: Array<RouteTitleRule & { regex: RegExp }> = [
  ...PLAYER_ROUTE_OVERRIDES,
  ...PLAYER_PRIMARY_NAV_ITEMS.map((item) => ({ pattern: item.to, title: item.label })),
  ...PLAYER_SECONDARY_NAV_ITEMS.map((item) => ({ pattern: item.to, title: item.label })),
  ...GLOBAL_SHELL_LINKS.map((item) => ({ pattern: item.to, title: item.label })),
  ...STATIC_ROUTE_TITLES,
].map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function compileRoutePattern(pattern: string): RegExp {
  if (pattern === '/') {
    return /^\/$/
  }

  const escaped = escapeRegex(pattern)
  const withParams = escaped.replace(/\\\$[A-Za-z][A-Za-z0-9]*/g, '[^/]+')
  return new RegExp(`^${withParams}/?$`)
}

export function resolvePageTitle(pathname: string): string {
  const matchingRule = ROUTE_TITLE_RULES.find((rule) => rule.regex.test(pathname))
  return matchingRule ? `LevelUp - ${matchingRule.title}` : 'LevelUp'
}
