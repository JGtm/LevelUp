import type { Locale } from '@/lib/i18n/locale'
import { playerRelativePath } from '@/lib/title-routing'

/** Titre localisé — parité FR/EN obligatoire par typage (CLAUDE.md règle 1). */
interface LocalizedTitle {
  fr: string
  en: string
}

interface RouteTitleRule {
  pattern: string
  title: LocalizedTitle
}

// Overrides de titre par SUFFIXE relatif au joueur : le pathname title-scoped a la
// forme `/{-lang}/t/{slug}/players/{playerSlug}{suffix}` et seul le suffixe identifie
// la page (cf. playerRelativePath). Patterns ancrés (^…$) → l'ordre n'est indicatif
// que de l'intention (le plus spécifique déclaré avant le plus générique). Aucun
// littéral `/players/` (garde-rail D-10).
//
// Table EXHAUSTIVE (I18, 2026-07-24) : couvre CHAQUE route réelle sous le scope joueur
// (garde-rail `pageTitle.test.ts` — un nouveau fichier de route sans entrée ici fait
// échouer le test). Remplace l'ancienne dérivation depuis
// `shellNavigation.{PLAYER_PRIMARY,PLAYER_SECONDARY}_NAV_ITEMS` : ces deux exports
// n'avaient plus AUCUN autre consommateur (la nav réelle vit dans `navL1Sections.tsx` /
// `NavL2.tsx`, locale-aware via `commonManifest`), portaient des libellés FR figés sans
// variante EN, et sont supprimés du même coup avec `shellNavigation.ts` (règle CLAUDE.md
// n°7, 0 code mort). Les libellés EN ci-dessous reprennent VERBATIM les traductions
// canoniques déjà établies ailleurs (`lib/i18n/generated/common.ts` `common.nav.*`,
// `features/citations/i18n` via `citationsManifest`, `features/compare/i18n.ts`,
// `features/squad/i18n.ts`) pour rester cohérentes avec la barre d'onglets réellement
// affichée.
const PLAYER_SUFFIX_OVERRIDES: RouteTitleRule[] = [
  // Accueil
  { pattern: '', title: { fr: 'Accueil', en: 'Home' } }, // racine joueur nue
  { pattern: '/home', title: { fr: 'Accueil', en: 'Home' } },
  // Solo
  { pattern: '/stats/timeseries', title: { fr: 'Séries temporelles', en: 'Time series' } },
  { pattern: '/stats/sessions', title: { fr: 'Sessions', en: 'Sessions' } },
  { pattern: '/stats/synthesis', title: { fr: 'Synthèse', en: 'Summary' } },
  { pattern: '/stats', title: { fr: 'Solo', en: 'Solo' } },
  // Escouade
  { pattern: '/squad/synergies', title: { fr: 'Synergies', en: 'Synergies' } },
  { pattern: '/squad/contributions', title: { fr: 'Contributions', en: 'Contributions' } },
  { pattern: '/squad/dynamique', title: { fr: 'Dynamique', en: 'Dynamics' } },
  { pattern: '/squad', title: { fr: 'Escouade', en: 'Squad' } },
  // Carrière — nuance Citations/Commendations (I18) : la source est fixée par la ROUTE
  // (/career/citations = moteur dérivé Infinite, /career/commendations = totaux natifs
  // H5), jamais par une donnée runtime — même distinction que l'ex-effet local de
  // `UnifiedCitationsPage` (titleKey), désormais supprimé au profit de cette table
  // (source unique). FR identique dans les deux cas (« Citations » est le terme
  // officiel Halo FR pour les deux titres, cf. `common.nav.tab_citations`).
  { pattern: '/career/citations', title: { fr: 'Citations', en: 'Citations' } },
  { pattern: '/career/commendations', title: { fr: 'Citations', en: 'Commendations' } },
  { pattern: '/career/medals', title: { fr: 'Médailles', en: 'Medals' } },
  { pattern: '/career/season-pass', title: { fr: 'Pass saisonnier', en: 'Season pass' } },
  { pattern: '/career', title: { fr: 'Carrière', en: 'Career' } },
  // Ascension (refonte 4 onglets 2026-07 : Profil + Objectifs + Entraînement + Réalisations)
  {
    pattern: '/ascension/objectifs',
    title: { fr: 'Ascension — Objectifs', en: 'Ascension — Objectives' },
  },
  {
    pattern: '/ascension/coaching',
    title: { fr: 'Ascension — Entraînement', en: 'Ascension — Coaching' },
  },
  {
    pattern: '/ascension/realisations',
    title: { fr: 'Ascension — Réalisations', en: 'Ascension — Achievements' },
  },
  { pattern: '/ascension', title: { fr: 'Ascension', en: 'Ascension' } },
  // Route historique /objectifs redirect → /ascension/objectifs (préservée pour bookmarks).
  { pattern: '/objectifs', title: { fr: 'Ascension', en: 'Ascension' } },
  // Communauté / Palmarès
  { pattern: '/community/compare', title: { fr: 'Face-à-face', en: 'Head-to-head' } },
  { pattern: '/community/relations', title: { fr: 'Relations', en: 'Relations' } },
  { pattern: '/community/prestige', title: { fr: 'Leaderboard PP', en: 'Leaderboard PP' } },
  { pattern: '/community', title: { fr: 'Communauté', en: 'Community' } },
  // Médias / Explorer
  { pattern: '/media', title: { fr: 'Médias', en: 'Media' } },
  { pattern: '/explorer', title: { fr: 'Explorer', en: 'Explorer' } },
  // Divers
  { pattern: '/matches/$matchId/replay', title: { fr: 'Replay', en: 'Replay' } },
  { pattern: '/matches/$matchId', title: { fr: 'Match', en: 'Match' } },
  { pattern: '/notifications', title: { fr: 'Notifications', en: 'Notifications' } },
]

// Titres des pages agnostiques (hors scope joueur). Table EXHAUSTIVE (I18) — couvre
// chaque route STATIQUE réelle, dont les 6 sous-onglets Administration (DC-8) qui
// n'avaient PAS de titre propre : le pattern `/admin` est ANCRÉ (`^\/admin\/?$`) et ne
// matchait jamais `/admin/xxx`, donc `/admin/management`, `/admin/data`,
// `/admin/detections`, `/admin/sync` et `/admin/system` retombaient tous sur le
// fallback 'LevelUp'. Remplace aussi la dérivation depuis
// `shellNavigation.GLOBAL_SHELL_LINKS` (même sort que PLAYER_*_NAV_ITEMS ci-dessus —
// supprimé, 0 autre consommateur).
const STATIC_ROUTE_TITLES: RouteTitleRule[] = [
  { pattern: '/', title: { fr: 'Accueil', en: 'Home' } },
  {
    pattern: '/admin/detections',
    title: { fr: 'Administration — Détections', en: 'Administration — Detections' },
  },
  {
    pattern: '/admin/data',
    title: { fr: 'Administration — Données', en: 'Administration — Data' },
  },
  {
    pattern: '/admin/sync',
    title: { fr: 'Administration — Sync', en: 'Administration — Sync' },
  },
  {
    pattern: '/admin/system',
    title: { fr: 'Administration — Système', en: 'Administration — System' },
  },
  {
    pattern: '/admin/management',
    title: { fr: 'Administration — Gestion', en: 'Administration — Management' },
  },
  { pattern: '/admin', title: { fr: 'Administration', en: 'Administration' } },
  { pattern: '/changelog', title: { fr: 'Changelog', en: 'Changelog' } },
  { pattern: '/groups', title: { fr: 'Mes groupes', en: 'My groups' } },
  { pattern: '/help', title: { fr: 'Aide', en: 'Help' } },
  { pattern: '/join', title: { fr: 'Rejoindre un groupe', en: 'Join a group' } },
  // Sandbox dev interne, jamais lié depuis la nav prod (cf. ChartsShowcasePage) —
  // conservé pour que l'onglet ne reste pas nu si on y accède en direct.
  { pattern: '/lab/charts', title: { fr: 'Aperçu graphiques', en: 'Charts gallery' } },
  { pattern: '/login', title: { fr: 'Connexion', en: 'Sign in' } },
  { pattern: '/onboarding/openspartan', title: { fr: 'Bienvenue', en: 'Welcome' } },
  { pattern: '/privacy', title: { fr: 'Confidentialité', en: 'Privacy' } },
  { pattern: '/register', title: { fr: 'Inscription', en: 'Register' } },
  { pattern: '/settings', title: { fr: 'Paramètres', en: 'Settings' } },
  { pattern: '/setup', title: { fr: 'Configuration', en: 'Setup' } },
]

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

const PLAYER_RULES = PLAYER_SUFFIX_OVERRIDES.map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

const STATIC_RULES = STATIC_ROUTE_TITLES.map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

/**
 * Titre d'onglet navigateur pour un pathname donné, dans la locale active.
 *
 * MÉCANISME UNIQUE (I18, 2026-07-24) : consommé exclusivement par l'effet
 * `[pathname, locale]` de `__root.tsx`. Les anciens effets locaux dupliqués
 * (`MedalsPage`, `ComparePage`, `UnifiedCitationsPage`) sont supprimés — ils étaient de
 * toute façon systématiquement écrasés par CE résolveur, rejoué à chaque navigation
 * après le montage des effets enfants (cause du symptôme « onglet figé/nu »).
 */
export function resolvePageTitle(pathname: string, locale: Locale): string {
  // Sous un scope joueur : on matche le SUFFIXE ; sinon (page agnostique) le pathname.
  const suffix = playerRelativePath(pathname)
  const rules = suffix !== null ? PLAYER_RULES : STATIC_RULES
  const target = suffix !== null ? suffix : pathname
  const matchingRule = rules.find((rule) => rule.regex.test(target))
  return matchingRule ? `LevelUp - ${matchingRule.title[locale]}` : 'LevelUp'
}
