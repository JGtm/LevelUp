export type CompareLocale = 'fr' | 'en'

export interface CompareText {
  intlLocale: string
  drawerTitle: string
  formLabel: string
  placeholder: string
  submit: string
  emptyPrompt: string
  loading: string
  metricColumn: string
  notFoundTitle: string
  notFoundDescription: string
  errorTitle: string
  errorDescription: string
  detailsTitle: string
  partialWarning: (gamertag: string) => string
  close: string
  localBadge: string
  stats: {
    matches: string
    winRate: string
    kda: string
    kdr: string
    accuracy: string
    killsPerGame: string
    currentCsr: string
  }
  metrics: Record<string, string>
}

const FR_TEXT: CompareText = {
  intlLocale: 'fr-FR',
  drawerTitle: 'Face-à-face avec un joueur',
  formLabel: 'Gamertag du joueur à affronter',
  placeholder: 'Ex : HaloPlayer123',
  submit: 'Face-à-face',
  emptyPrompt: 'Entrez un gamertag pour lancer le face-à-face.',
  loading: 'Chargement de la comparaison…',
  metricColumn: 'Métrique',
  notFoundTitle: 'Joueur introuvable',
  notFoundDescription: 'Ce joueur n\'existe pas ou n\'a aucune donnée accessible.',
  errorTitle: 'Erreur',
  errorDescription: 'Impossible de récupérer la comparaison.',
  detailsTitle: 'Face-à-face détaillé',
  partialWarning: (gamertag) => `Les données de ${gamertag} sont partielles — certaines métriques peuvent être absentes.`,
  close: 'Fermer',
  localBadge: 'Local',
  stats: {
    matches: 'Matchs',
    winRate: 'Victoires',
    kda: 'KDA',
    kdr: 'K/D',
    accuracy: 'Précision',
    killsPerGame: 'Kills / partie',
    currentCsr: 'CSR actuel',
  },
  metrics: {
    win_rate: 'Taux de victoire',
    kda: 'KDA',
    kdr: 'K/D',
    kills_per_game: 'Kills / partie',
    deaths_per_game: 'Morts / partie',
    assists_per_game: 'Assists / partie',
    accuracy: 'Précision',
    damage_per_game: 'Dégâts / partie',
    matches: 'Matchs joués',
    csr_current: 'CSR actuel',
    csr_best: 'Meilleur CSR',
    career_rank: 'Rang carrière',
  },
}

const EN_TEXT: CompareText = {
  intlLocale: 'en-GB',
  drawerTitle: 'Head-to-head with a player',
  formLabel: 'Player gamertag to face',
  placeholder: 'E.g. HaloPlayer123',
  submit: 'Head-to-head',
  emptyPrompt: 'Enter a gamertag to start the head-to-head.',
  loading: 'Loading comparison…',
  metricColumn: 'Metric',
  notFoundTitle: 'Player not found',
  notFoundDescription: 'This player does not exist or no accessible data was found.',
  errorTitle: 'Error',
  errorDescription: 'Unable to load the comparison.',
  detailsTitle: 'Head-to-head breakdown',
  partialWarning: (gamertag) => `${gamertag} has partial data — some metrics may be missing.`,
  close: 'Close',
  localBadge: 'Local',
  stats: {
    matches: 'Matches',
    winRate: 'Win rate',
    kda: 'KDA',
    kdr: 'K/D',
    accuracy: 'Accuracy',
    killsPerGame: 'Kills / match',
    currentCsr: 'Current CSR',
  },
  metrics: {
    win_rate: 'Win rate',
    kda: 'KDA',
    kdr: 'K/D',
    kills_per_game: 'Kills / match',
    deaths_per_game: 'Deaths / match',
    assists_per_game: 'Assists / match',
    accuracy: 'Accuracy',
    damage_per_game: 'Damage / match',
    matches: 'Matches played',
    csr_current: 'Current CSR',
    csr_best: 'Best CSR',
    career_rank: 'Career rank',
  },
}

const TEXT: Record<CompareLocale, CompareText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizeCompareLocale(locale?: string | null): CompareLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getCompareText(locale?: string | null): CompareText {
  return TEXT[normalizeCompareLocale(locale)]
}
