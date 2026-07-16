export interface ShellNavItem {
  to: string
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
    to: '/players/$playerSlug/home',
    label: 'Accueil',
    eyebrow: 'Mission',
    description: 'Briefing, signaux chauds et accès prioritaires.',
  },
  {
    to: '/players/$playerSlug/community',
    label: 'Communauté',
    eyebrow: 'Prestige',
    description: 'Classements, relations et face-à-face joueur à joueur.',
  },
  {
    to: '/players/$playerSlug/career',
    label: 'Carrière',
    eyebrow: 'Progression',
    description: 'Rang, stabilité et lecture globale du niveau.',
  },
  {
    to: '/players/$playerSlug/squad',
    label: 'Escouade',
    eyebrow: 'Relations',
    description: 'Cohortes, synergies et contexte d’équipe.',
  },
  {
    to: '/players/$playerSlug/media',
    label: 'Médias',
    eyebrow: 'Captures',
    description: 'Moments visuels, extraits et preuve terrain.',
  },
]

export const PLAYER_SECONDARY_NAV_ITEMS: ShellNavItem[] = [
  {
    to: '/players/$playerSlug/profile/citations',
    label: 'Citations',
    eyebrow: 'Référentiel',
    description: 'Signatures de jeu, médailles et profils.',
  },
  {
    to: '/players/$playerSlug/explorer',
    label: 'Explorer',
    eyebrow: 'Drilldown',
    description: 'Approfondir, filtrer et descendre dans le détail.',
  },
  {
    to: '/players/$playerSlug/stats/synthesis',
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
 * Section Communauté : pages /palmares (sauf l'ancien /palmares/season-pass, passé
 * sous Carrière) + la route /compare (Face-à-face), qui n'est pas sous /palmares.
 *
 * Source unique partagée par NavL1 (surlignage du bouton) et NavL2 (affichage des
 * sous-onglets) pour garder les deux navs synchronisées.
 */
export function isCommunityPath(pathname: string): boolean {
  const community = /\/players\/[^/]+\/community(?:\/|$)/.test(pathname)
  // Legacy : ancien /palmares (hors season-pass passé sous Carrière) + ancien /compare,
  // encore surlignés Communauté le temps que les redirections vers /community s'appliquent.
  const legacy =
    (/\/players\/[^/]+\/palmares(?:\/|$)/.test(pathname) &&
      !/\/palmares\/season-pass/.test(pathname)) ||
    /\/players\/[^/]+\/compare(?:\/|$)/.test(pathname)
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

export function buildPlayerDestination(
  pathname: string,
  currentPlayerSlug: string | null | undefined,
  nextPlayerSlug: string,
): string {
  if (!currentPlayerSlug) {
    return `/players/${nextPlayerSlug}/home`
  }

  const currentPrefix = `/players/${currentPlayerSlug}`
  if (!pathname.startsWith(currentPrefix)) {
    return `/players/${nextPlayerSlug}/home`
  }

  const suffix = pathname.slice(currentPrefix.length)
  if (!suffix) {
    return `/players/${nextPlayerSlug}/home`
  }

  return `/players/${nextPlayerSlug}${suffix}`
}
