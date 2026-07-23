import type { FileRouteTypes } from '@/routeTree.gen'
import { playerRelativePath } from '@/lib/title-routing'

/**
 * Cible de route valide du routeur généré (union des `to`). Typer `ShellNavItem.to`
 * ainsi (et non `string`) fait ENTRER la nav L1/L2 dans le typecheck : un chemin
 * inexistant (ex. l'ancien `.../profile/citations`) devient une erreur tsc au lieu
 * d'un lien mort silencieux. Source unique réutilisée par NavL1/NavL2/navL1Sections.
 */
export type RouteTo = FileRouteTypes['to']

export interface ShellNavItem {
  to: RouteTo
  label: string
  eyebrow: string
  description: string
}

export interface ShellUtilityLink {
  to: '/settings' | '/changelog' | '/groups'
  label: string
}

export const PLAYER_PRIMARY_NAV_ITEMS: ShellNavItem[] = [
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
    label: 'Accueil',
    eyebrow: 'Mission',
    description: 'Briefing, signaux chauds et accès prioritaires.',
  },
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/community',
    label: 'Communauté',
    eyebrow: 'Prestige',
    description: 'Classements, relations et face-à-face joueur à joueur.',
  },
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/career',
    label: 'Carrière',
    eyebrow: 'Progression',
    description: 'Rang, stabilité et lecture globale du niveau.',
  },
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad',
    label: 'Escouade',
    eyebrow: 'Relations',
    description: 'Cohortes, synergies et contexte d’équipe.',
  },
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/media',
    label: 'Médias',
    eyebrow: 'Captures',
    description: 'Moments visuels, extraits et preuve terrain.',
  },
]

export const PLAYER_SECONDARY_NAV_ITEMS: ShellNavItem[] = [
  {
    // Correction mécanique (2c) : l'ancien `.../profile/citations` (route inexistante,
    // toléré des mois par `to: string`) pointe désormais vers la route réelle
    // `career/citations` — imposé par le typage `RouteTo`.
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations',
    label: 'Citations',
    eyebrow: 'Référentiel',
    description: 'Signatures de jeu, médailles et profils.',
  },
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
    label: 'Explorer',
    eyebrow: 'Drilldown',
    description: 'Approfondir, filtrer et descendre dans le détail.',
  },
  {
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/synthesis',
    label: 'Synthèse',
    eyebrow: 'Recap',
    description: 'Vue consolidée et transversale.',
  },
]

export const GLOBAL_SHELL_LINKS: ShellUtilityLink[] = [
  { to: '/groups', label: 'Groupes' },
  { to: '/settings', label: 'Paramètres' },
  { to: '/changelog', label: 'Changelog' },
]

/**
 * Section Communauté : pages /community + les legacy /palmares (hors season-pass,
 * passé sous Carrière) et /compare (Face-à-face). Raisonne sur le SUFFIXE relatif
 * au joueur (playerRelativePath) — aucun littéral `/players/` (garde-rail D-10).
 *
 * Source unique partagée par NavL1 (surlignage du bouton) et NavL2 (sous-onglets).
 */
export function isCommunityPath(pathname: string): boolean {
  const suffix = playerRelativePath(pathname)
  if (suffix === null) return false
  const community = /^\/community(?:\/|$)/.test(suffix)
  const legacy =
    (/^\/palmares(?:\/|$)/.test(suffix) && !/^\/palmares\/season-pass/.test(suffix)) ||
    /^\/compare(?:\/|$)/.test(suffix)
  return community || legacy
}

/**
 * Verdict de routage de la route index ('/'). Fonction pure (logique hors
 * composant, règle 7) : IndexPage se contente de projeter le résultat.
 *
 * Gardes, du plus prioritaire au moins prioritaire :
 *  1. `wait`   — bootstrap pas encore hydraté. Avant hydratation `authMode` vaut son
 *               défaut 'none', donc la garde `login` ne serait pas fiable : on attend.
 *  2. `login`  — auth requise (password|xbox) mais session anonyme. Typiquement juste
 *               après « Se déconnecter » (qui recharge sur '/'). En anonyme xbox
 *               `available_players` est vide (filtrage ownership, ADR 0029) : sans
 *               cette garde on tombait sur `setup`, rendu via le <Outlet/> nu de
 *               __root — donc SANS NavL1 — et la redirection impérative de __root
 *               (navigate dans un useEffect) pouvait se perdre dans la course avec le
 *               settle initial du routeur au rechargement plein, bloquant l'utilisateur
 *               sur une page sans barre de nav ni lien de reconnexion.
 *  3. `player` — un joueur est actif (courant, sinon 1er disponible) : sa home.
 *  4. `setup`  — aucun joueur configuré : inviter à configurer l'application.
 */
export type IndexRedirect =
  | { kind: 'wait' }
  | { kind: 'login' }
  | { kind: 'player'; slug: string }
  | { kind: 'setup' }

export interface IndexRedirectInput {
  isBootstrapped: boolean
  authMode: 'none' | 'password' | 'xbox'
  currentUsername: string | null
  currentPlayerSlug?: string | null
  firstAvailablePlayerSlug?: string | null
}

export function resolveIndexRedirect(input: IndexRedirectInput): IndexRedirect {
  if (!input.isBootstrapped) {
    return { kind: 'wait' }
  }
  const authRequired = input.authMode === 'password' || input.authMode === 'xbox'
  if (authRequired && !input.currentUsername) {
    return { kind: 'login' }
  }
  const slug = input.currentPlayerSlug ?? input.firstAvailablePlayerSlug
  if (slug) {
    return { kind: 'player', slug }
  }
  return { kind: 'setup' }
}

/**
 * Décision de navigation lors d'un changement de JOUEUR actif (même titre). Pure
 * (logique hors composant, règle 7) : NavL1 projette le résultat.
 *  - `same-route` : on est sous une sous-page joueur → rester sur la même route et ne
 *    changer que le param `playerSlug` (préserve la section, le titre ET la langue).
 *  - `home` : on n'est pas sous une sous-page joueur (page agnostique, ou racine
 *    joueur nue) → aller à l'accueil title-scoped du nouveau joueur.
 */
export type PlayerSwitchNav = { kind: 'same-route' } | { kind: 'home' }

export function resolvePlayerSwitch(pathname: string): PlayerSwitchNav {
  // suffix null (page agnostique) OU '' (racine joueur nue) → home ; sinon on préserve
  // la sous-page en ne changeant que le playerSlug.
  return playerRelativePath(pathname) ? { kind: 'same-route' } : { kind: 'home' }
}

/**
 * Filet joueur (D-8, trou n°1 de la revue v2) — décision pure projetée par le layout
 * joueur au fresh-load (une fois `isBootstrapped`). Le beforeLoad du layout couvre les
 * navigations SPA (store chaud) mais NE re-tourne PAS sur un simple re-render : ce
 * résolveur ferme le cas fresh-load où le store s'hydrate après le matching.
 *  - `index`    : aucun joueur disponible → retour à l'index (onboarding).
 *  - `ok`       : le slug d'URL existe → rendre la page.
 *  - `redirect` : slug d'URL inconnu → premier joueur disponible (même titre).
 */
export type PlayerFallback = { kind: 'ok' } | { kind: 'redirect'; slug: string } | { kind: 'index' }

export function resolvePlayerFallback(
  playerSlug: string,
  availablePlayers: readonly { player_slug: string }[],
): PlayerFallback {
  if (availablePlayers.length === 0) return { kind: 'index' }
  if (availablePlayers.some((p) => p.player_slug === playerSlug)) return { kind: 'ok' }
  return { kind: 'redirect', slug: availablePlayers[0].player_slug }
}
