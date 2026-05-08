export interface ShellNavItem {
  to: string
  label: string
  eyebrow: string
  description: string
}

export interface ShellUtilityLink {
  to: '/settings' | '/changelog'
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
    to: '/players/$playerSlug/palmares',
    label: 'Palmarès',
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
    to: '/players/$playerSlug/stats/timeseries',
    label: 'Séries',
    eyebrow: 'Tendance',
    description: 'Lecture temporelle des performances.',
  },
  {
    to: '/players/$playerSlug/stats/sessions',
    label: 'Sessions',
    eyebrow: 'Runs',
    description: 'Séquences de jeu et variations de rythme.',
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
  { to: '/settings', label: 'Paramètres' },
  { to: '/changelog', label: 'Changelog' },
]

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
