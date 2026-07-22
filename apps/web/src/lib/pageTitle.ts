import {
  GLOBAL_SHELL_LINKS,
  PLAYER_PRIMARY_NAV_ITEMS,
  PLAYER_SECONDARY_NAV_ITEMS,
} from '@/components/shell/shellNavigation'
import { playerRelativePath, routeTemplateSuffix } from '@/lib/title-routing'

interface RouteTitleRule {
  pattern: string
  title: string
}

// Overrides de titre par SUFFIXE relatif au joueur : le pathname title-scoped a la
// forme `/{-lang}/t/{slug}/players/{playerSlug}{suffix}` et seul le suffixe identifie
// la page (cf. playerRelativePath). Patterns ancrés (^…$) → l'ordre n'est indicatif
// que de l'intention. Aucun littéral `/players/` (garde-rail D-10).
const PLAYER_SUFFIX_OVERRIDES: RouteTitleRule[] = [
  // Solo
  { pattern: '/stats/timeseries', title: 'Séries temporelles' },
  { pattern: '/stats/sessions', title: 'Sessions' },
  { pattern: '/stats/synthesis', title: 'Synthèse' },
  { pattern: '/stats', title: 'Solo' },
  // Carrière
  { pattern: '/career/season-pass', title: 'Pass saisonnier' },
  { pattern: '/career/citations', title: 'Citations' },
  { pattern: '/career/commendations', title: 'Citations' },
  { pattern: '/career', title: 'Carrière' },
  // Communauté / Palmarès
  { pattern: '/community/compare', title: 'Face-à-face' },
  { pattern: '/community/relations', title: 'Relations' },
  { pattern: '/community/prestige', title: 'Leaderboard PP' },
  // Ascension (refonte 4 onglets 2026-07 : Profil + Objectifs + Entraînement + Réalisations)
  { pattern: '/ascension/objectifs', title: 'Ascension — Objectifs' },
  { pattern: '/ascension/coaching', title: 'Ascension — Entraînement' },
  { pattern: '/ascension/realisations', title: 'Ascension — Réalisations' },
  { pattern: '/ascension', title: 'Ascension' },
  // Route historique /objectifs redirect → /ascension/objectifs (préservée pour bookmarks).
  { pattern: '/objectifs', title: 'Ascension' },
  // Escouade
  { pattern: '/squad/contributions', title: 'Contributions' },
  { pattern: '/squad/synergies', title: 'Synergies' },
  // Divers
  { pattern: '/matches/$matchId/replay', title: 'Replay' },
  { pattern: '/matches/$matchId', title: 'Match' },
  { pattern: '/notifications', title: 'Notifications' },
  // Racine joueur nue → Accueil.
  { pattern: '', title: 'Accueil' },
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
  ...GLOBAL_SHELL_LINKS.map((item) => ({ pattern: item.to, title: item.label })),
]

// Titres dérivés des items de nav (labels des sections) — patterns = SUFFIXE de la
// cible de route typée (routeTemplateSuffix). Complètent les overrides ci-dessus.
const PLAYER_NAV_TITLES: RouteTitleRule[] = [
  ...PLAYER_PRIMARY_NAV_ITEMS,
  ...PLAYER_SECONDARY_NAV_ITEMS,
].map((item) => ({ pattern: routeTemplateSuffix(item.to), title: item.label }))

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function compileRoutePattern(pattern: string): RegExp {
  if (pattern === '/') return /^\/$/
  if (pattern === '') return /^$/

  const escaped = escapeRegex(pattern)
  const withParams = escaped.replace(/\\\$[A-Za-z][A-Za-z0-9]*/g, '[^/]+')
  return new RegExp(`^${withParams}/?$`)
}

const PLAYER_RULES = [...PLAYER_SUFFIX_OVERRIDES, ...PLAYER_NAV_TITLES].map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

const STATIC_RULES = STATIC_ROUTE_TITLES.map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

export function resolvePageTitle(pathname: string): string {
  // Sous un scope joueur : on matche le SUFFIXE ; sinon (page agnostique) le pathname.
  const suffix = playerRelativePath(pathname)
  const rules = suffix !== null ? PLAYER_RULES : STATIC_RULES
  const target = suffix !== null ? suffix : pathname
  const matchingRule = rules.find((rule) => rule.regex.test(target))
  return matchingRule ? `LevelUp - ${matchingRule.title}` : 'LevelUp'
}
