/**
 * i18n.ts — traductions FR + EN du composant SessionBriefing.
 *
 * Les libellés outcomes (Victoire / Défaite / Égalité / Abandon) ne sont PAS
 * dans ce fichier — ils proviennent de outcomes.toml via useFieldMappings()
 * pour rester title-aware (multi-titres).
 */

export type BriefingLocale = 'fr' | 'en'

export interface BriefingTexts {
  /** Libellé "Résultats" + helper pluralisation outcomes — utilisés dans la
   *  Results bar de SquadVerdict (squad mode uniquement). */
  rail: {
    resultsLabel: string
  }
  verdict: {
    teamScore: string
    deltaBonusPositive: (delta: number) => string
    deltaBonusNegative: (delta: number) => string
  }
  drill: {
    activeView: (gamertag: string) => string
    resetButton: string
  }
  grid: {
    titleSelf: string
    trendHint: string
    matchesPlayed: string
    totalDuration: string
    avgMatchDuration: string
    fragsPerMatch: string
    deathsPerMatch: string
    assistsPerMatch: string
    accuracy: string
    lifespan: string
    perMin: string
    avgPerformance: string
    rankDeltaCSR: string
    rankDeltaLUSR: string
  }
  /** Format pluriel pour les libellés outcomes — utilisé dans la Results bar */
  pluralize: (count: number, singular: string) => string
}

const FR: BriefingTexts = {
  rail: {
    resultsLabel: 'Résultats',
  },
  verdict: {
    teamScore: "Score d'équipe",
    deltaBonusPositive: (d) => `Bonus équipe +${d}`,
    deltaBonusNegative: (d) => `Malus équipe ${d}`,
  },
  drill: {
    activeView: (gt) => `Vue active : ${gt}`,
    resetButton: '✕ revenir à mes stats',
  },
  grid: {
    titleSelf: 'Mes stats sur cette session',
    trendHint: "▲/▼ vs moyenne d'équipe sur la session",
    matchesPlayed: 'Matchs joués',
    totalDuration: 'Durée totale',
    avgMatchDuration: 'Durée moyenne par match',
    fragsPerMatch: 'Frags par partie',
    deathsPerMatch: 'Morts par partie',
    assistsPerMatch: 'Assistances par partie',
    accuracy: 'Précision moyenne',
    lifespan: 'Durée de vie moyenne',
    perMin: '/min',
    avgPerformance: 'Performance moyenne',
    rankDeltaCSR: 'Delta CSR',
    rankDeltaLUSR: 'Delta LUSR',
  },
  // FR : ajout "s" si count > 1, sauf "Abandon" qui prend aussi "s".
  // Toutes nos labels (Victoire/Défaite/Égalité/Abandon) suivent la même règle.
  pluralize: (count, singular) => (count > 1 ? `${singular}s` : singular),
}

const EN: BriefingTexts = {
  rail: {
    resultsLabel: 'Results',
  },
  verdict: {
    teamScore: 'Team score',
    deltaBonusPositive: (d) => `Team bonus +${d}`,
    deltaBonusNegative: (d) => `Team penalty ${d}`,
  },
  drill: {
    activeView: (gt) => `Viewing: ${gt}`,
    resetButton: '✕ back to my stats',
  },
  grid: {
    titleSelf: 'My stats this session',
    trendHint: '▲/▼ vs team average on this session',
    matchesPlayed: 'Matches played',
    totalDuration: 'Total duration',
    avgMatchDuration: 'Avg duration per match',
    fragsPerMatch: 'Frags per match',
    deathsPerMatch: 'Deaths per match',
    assistsPerMatch: 'Assists per match',
    accuracy: 'Avg accuracy',
    lifespan: 'Avg lifespan',
    perMin: '/min',
    avgPerformance: 'Avg performance',
    rankDeltaCSR: 'CSR change',
    rankDeltaLUSR: 'LUSR change',
  },
  // EN : pluralisation par "s" couvre nos labels outcomes (Win → Wins, Loss → Losses
  // est géré séparément si needed — ici on assume singulier sans -s).
  pluralize: (count, singular) => {
    if (count <= 1) return singular
    if (singular.endsWith('s') || singular.endsWith('x')) return `${singular}es`
    return `${singular}s`
  },
}

export function getBriefingTexts(locale: BriefingLocale): BriefingTexts {
  return locale === 'en' ? EN : FR
}
