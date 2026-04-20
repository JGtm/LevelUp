export type PalmaresLocale = 'fr' | 'en'
export type PalmaresTab = 'leaderboard' | 'relations' | 'season-pass' | 'compare'

export interface PalmaresText {
  intlLocale: string
  page: {
    title: string
    subtitle: string
  }
  tabs: Record<PalmaresTab, string>
  relations: {
    retry: string
    unavailableTitle: string
    unavailableDescription: string
    overview: {
      distinctPlayers: string
      frequentAllies: string
      repeatRivals: string
      closedCircle: string
    }
    sections: {
      frequentAlliesTitle: string
      frequentAlliesDescription: string
      bestSynergiesTitle: string
      bestSynergiesDescription: string
      nemesesTitle: string
      nemesesDescription: string
      victimsTitle: string
      victimsDescription: string
      closedCircleTitle: string
      closedCircleDescription: string
    }
    labels: {
      with: string
      against: string
      winRate: string
      avgKDA: string
      lastSeen: string
    }
    emptyTitle: string
    emptyDescription: string
  }
  seasonPass: {
    retry: string
    unavailableTitle: string
    unavailableDescription: string
    activeCard: string
    completedCard: string
    inProgressCard: string
    challengesCard: string
    challengesTitle: string
    challengesUnavailable: string
    completed: string
    total: string
    xpAvailable: string
    nextExpiry: string
    noExpiry: string
    premium: string
    active: string
    cardRank: string
    cardProgress: string
    activePassTitle: string
    activeTierTitle: string
    activeTierProgress: string
    activeTierFallback: string
    obtained: string
    upcoming: string
    otherPassesTitle: string
    noDescription: string
    noPassesTitle: string
    noPassesDescription: string
    status: Record<string, string>
  }
}

const FR_TEXT: PalmaresText = {
  intlLocale: 'fr-FR',
  page: {
    title: 'Palmarès',
    subtitle: 'Classements, relations de jeu, passes saisonniers et comparaison joueur à joueur regroupés dans un même hub.',
  },
  tabs: {
    leaderboard: 'Classements',
    relations: 'Relations',
    'season-pass': 'Pass saisonnier',
    compare: 'Face-à-face',
  },
  relations: {
    retry: 'Réessayer',
    unavailableTitle: 'Relations indisponibles',
    unavailableDescription: 'Les agrégats relationnels n\'ont pas pu être chargés pour ce joueur.',
    overview: {
      distinctPlayers: 'Joueurs recroisés',
      frequentAllies: 'Alliés fréquents',
      repeatRivals: 'Rivaux suivis',
      closedCircle: 'Cercle fermé',
    },
    sections: {
      frequentAlliesTitle: 'Alliés fréquents',
      frequentAlliesDescription: 'Les joueurs qui reviennent le plus souvent à tes côtés, quel que soit le résultat.',
      bestSynergiesTitle: 'Meilleures synergies',
      bestSynergiesDescription: 'Les duos qui convertissent le mieux les matchs en victoires avec un KDA solide.',
      nemesesTitle: 'Némésis',
      nemesesDescription: 'Les adversaires qui reviennent souvent et contre qui tu lâches le plus de terrain.',
      victimsTitle: 'Victimes favorites',
      victimsDescription: 'Les visages récurrents que tu domines le plus souvent quand ils sont en face.',
      closedCircleTitle: 'Cercle fermé',
      closedCircleDescription: 'Le noyau dur de joueurs que tu retrouves parfois avec toi, parfois contre toi.',
    },
    labels: {
      with: 'Avec',
      against: 'Contre',
      winRate: 'Win rate',
      avgKDA: 'KDA moyen',
      lastSeen: 'Dernière rencontre',
    },
    emptyTitle: 'Aucune relation exploitable',
    emptyDescription: 'Il faut au moins quelques matchs répétés contre ou avec les mêmes joueurs pour faire émerger les liens.',
  },
  seasonPass: {
    retry: 'Réessayer',
    unavailableTitle: 'Pass saisonnier indisponible',
    unavailableDescription: 'Les données live du battle pass n\'ont pas pu être chargées pour cette session.',
    activeCard: 'Pass actif',
    completedCard: 'Terminés',
    inProgressCard: 'En cours',
    challengesCard: 'Défis actifs',
    challengesTitle: 'Défis live',
    challengesUnavailable: 'Défis indisponibles',
    completed: 'Complétés',
    total: 'Total',
    xpAvailable: 'XP disponible',
    nextExpiry: 'Prochaine expiration',
    noExpiry: 'Aucune échéance connue',
    premium: 'Premium',
    active: 'Actif',
    cardRank: 'Rang',
    cardProgress: 'Progression',
    activePassTitle: 'Pass actif',
    activeTierTitle: 'Palier actif',
    activeTierProgress: 'Progression du palier',
    activeTierFallback: 'Palier à venir',
    obtained: 'Obtenu',
    upcoming: 'À venir',
    otherPassesTitle: 'Autres passes',
    noDescription: 'Aucune description disponible pour ce pass.',
    noPassesTitle: 'Aucun pass détecté',
    noPassesDescription: 'Aucune progression de pass saisonnier n\'a été renvoyée pour ce joueur.',
    status: {
      active: 'Actif',
      in_progress: 'En cours',
      completed: 'Terminé',
      not_started: 'Non commencé',
    },
  },
}

const EN_TEXT: PalmaresText = {
  intlLocale: 'en-GB',
  page: {
    title: 'Palmares',
    subtitle: 'Ranked standings, play relationships, seasonal passes and player-vs-player comparison grouped into one hub.',
  },
  tabs: {
    leaderboard: 'Leaderboard',
    relations: 'Relations',
    'season-pass': 'Season Pass',
    compare: 'Head-to-head',
  },
  relations: {
    retry: 'Retry',
    unavailableTitle: 'Relations unavailable',
    unavailableDescription: 'Relationship aggregates could not be loaded for this player.',
    overview: {
      distinctPlayers: 'Players re-encountered',
      frequentAllies: 'Frequent allies',
      repeatRivals: 'Tracked rivals',
      closedCircle: 'Closed circle',
    },
    sections: {
      frequentAlliesTitle: 'Frequent allies',
      frequentAlliesDescription: 'The players you most often queue into on your side, regardless of result.',
      bestSynergiesTitle: 'Best synergies',
      bestSynergiesDescription: 'The pairings that convert matches into wins most efficiently.',
      nemesesTitle: 'Nemeses',
      nemesesDescription: 'Recurring opponents that tend to take the edge when they are on the other team.',
      victimsTitle: 'Favorite victims',
      victimsDescription: 'Recurring opponents you consistently beat when they are on the other side.',
      closedCircleTitle: 'Closed circle',
      closedCircleDescription: 'The small set of players you keep finding with you and against you.',
    },
    labels: {
      with: 'With',
      against: 'Against',
      winRate: 'Win rate',
      avgKDA: 'Avg KDA',
      lastSeen: 'Last seen',
    },
    emptyTitle: 'No meaningful relationships yet',
    emptyDescription: 'You need a few repeated encounters with the same players before relationship patterns become useful.',
  },
  seasonPass: {
    retry: 'Retry',
    unavailableTitle: 'Season pass unavailable',
    unavailableDescription: 'Live battle pass data could not be loaded for this session.',
    activeCard: 'Active pass',
    completedCard: 'Completed',
    inProgressCard: 'In progress',
    challengesCard: 'Active challenges',
    challengesTitle: 'Live challenges',
    challengesUnavailable: 'Challenges unavailable',
    completed: 'Completed',
    total: 'Total',
    xpAvailable: 'XP available',
    nextExpiry: 'Next expiry',
    noExpiry: 'No known deadline',
    premium: 'Premium',
    active: 'Active',
    cardRank: 'Rank',
    cardProgress: 'Progress',
    activePassTitle: 'Active pass',
    activeTierTitle: 'Active tier',
    activeTierProgress: 'Tier progress',
    activeTierFallback: 'Upcoming tier',
    obtained: 'Unlocked',
    upcoming: 'Upcoming',
    otherPassesTitle: 'Other passes',
    noDescription: 'No description is available for this pass.',
    noPassesTitle: 'No pass detected',
    noPassesDescription: 'No seasonal pass progression was returned for this player.',
    status: {
      active: 'Active',
      in_progress: 'In progress',
      completed: 'Completed',
      not_started: 'Not started',
    },
  },
}

const TEXT: Record<PalmaresLocale, PalmaresText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizePalmaresLocale(locale?: string | null): PalmaresLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getPalmaresText(locale?: string | null): PalmaresText {
  return TEXT[normalizePalmaresLocale(locale)]
}
