import type { Locale } from '@/lib/i18n/locale'

export interface CompareText {
  intlLocale: string
  pageTitle: string
  backToExplorer: string
  searchPlaceholder: string
  emptyPrompt: string
  loading: string
  vs: string
  notFoundTitle: string
  notFoundDescription: string
  errorTitle: string
  errorDescription: string
  partialWarning: (gamertag: string) => string
  ariaWinner: (name: string) => string
  ariaEqual: string
  sampleSize: (n: number) => string
  notAvailable: string
  ariaNotAvailable: string
  catCombat: string
  catPrecision: string
  catBilan: string
  metrics: Record<string, string>
}

const FR_TEXT: CompareText = {
  intlLocale: 'fr-FR',
  pageTitle: 'Face-à-face',
  backToExplorer: 'Retour à l\'Explorer',
  searchPlaceholder: 'Rechercher un joueur…',
  emptyPrompt: 'Recherchez un joueur pour lancer le face-à-face.',
  loading: 'Chargement de la comparaison…',
  vs: 'vs',
  notFoundTitle: 'Joueur introuvable',
  notFoundDescription: 'Ce joueur n\'existe pas ou n\'a aucune donnée accessible.',
  errorTitle: 'Erreur',
  errorDescription: 'Impossible de récupérer la comparaison.',
  partialWarning: (gamertag) => `Les données de ${gamertag} sont partielles — certaines métriques peuvent être absentes.`,
  ariaWinner: (name) => `${name} domine cette métrique`,
  ariaEqual: 'Égalité',
  sampleSize: (n) => `(sur ${n} matchs)`,
  notAvailable: 'N/A',
  ariaNotAvailable: 'Donnée non disponible',
  catCombat: 'Combat',
  catPrecision: 'Précision & Survie',
  catBilan: 'Bilan & Rang',
  metrics: {
    win_rate: 'Taux de victoire',
    kda: 'KDA',
    kdr: 'K/D',
    kills_per_game: 'Frags/match',
    deaths_per_game: 'Morts/match',
    assists_per_game: 'Assistances/match',
    accuracy: 'Précision',
    damage_per_game: 'Dégâts/match',
    matches: 'Matchs joués',
    career_rank: 'Rang carrière',
    csr: 'CSR (saison actuelle)',
    csr_alltime: 'CSR (record)',
    rendement: 'Rendement',
    resistance: 'Résistance',
    perfect_kills_per_game: 'Tirs parfaits/match',
    max_killing_spree: 'Folie meurtrière max',
    avg_life_secs: 'Durée de vie moy./match',
    headshot_kills_per_game: 'Tirs à la tête/match',
    perf_ath: 'Perf. record',
    lusr_ath: 'LUSR record',
  },
}

const EN_TEXT: CompareText = {
  intlLocale: 'en-GB',
  pageTitle: 'Head-to-head',
  backToExplorer: 'Back to Explorer',
  searchPlaceholder: 'Search a player…',
  emptyPrompt: 'Search a player to start the head-to-head.',
  loading: 'Loading comparison…',
  vs: 'vs',
  notFoundTitle: 'Player not found',
  notFoundDescription: 'This player does not exist or no accessible data was found.',
  errorTitle: 'Error',
  errorDescription: 'Unable to load the comparison.',
  partialWarning: (gamertag) => `${gamertag} has partial data — some metrics may be missing.`,
  ariaWinner: (name) => `${name} leads`,
  ariaEqual: 'Tied',
  sampleSize: (n) => `(${n} matches)`,
  notAvailable: 'N/A',
  ariaNotAvailable: 'Data not available',
  catCombat: 'Combat',
  catPrecision: 'Precision & Survival',
  catBilan: 'Stats & Rank',
  metrics: {
    win_rate: 'Win rate',
    kda: 'KDA',
    kdr: 'K/D',
    kills_per_game: 'Kills/game',
    deaths_per_game: 'Deaths/game',
    assists_per_game: 'Assists/game',
    accuracy: 'Accuracy',
    damage_per_game: 'Damage/game',
    matches: 'Matches played',
    career_rank: 'Career rank',
    csr: 'CSR (current season)',
    csr_alltime: 'CSR (all-time)',
    rendement: 'Efficiency',
    resistance: 'Toughness',
    perfect_kills_per_game: 'Perfect kills/game',
    max_killing_spree: 'Max killing spree',
    avg_life_secs: 'Avg. life/game',
    headshot_kills_per_game: 'Headshots/game',
    perf_ath: 'Perf. all-time',
    lusr_ath: 'LUSR all-time',
  },
}

const TEXT: Record<Locale, CompareText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizeCompareLocale(locale?: string | null): Locale {
  return locale === 'en' ? 'en' : 'fr'
}

const METRIC_TO_FIELD_KEY: Record<string, string> = {
  win_rate: 'win_rate',
  kda: 'kda',
  kdr: 'kdr',
  accuracy: 'accuracy',
  matches: 'total_matches_played',
}

export function getCompareText(
  locale?: string | null,
  fieldMappings?: { fields: Record<string, { label: string }> },
): CompareText {
  const base = TEXT[normalizeCompareLocale(locale)]
  if (!fieldMappings) return base
  const merged: CompareText = {
    ...base,
    metrics: { ...base.metrics },
  }
  for (const [metricKey, fieldKey] of Object.entries(METRIC_TO_FIELD_KEY)) {
    const canonical = fieldMappings.fields?.[fieldKey]?.label
    if (canonical) merged.metrics[metricKey] = canonical
  }
  return merged
}
