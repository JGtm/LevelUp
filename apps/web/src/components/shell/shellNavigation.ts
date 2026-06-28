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
    to: '/players/$playerSlug/synthesis',
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
