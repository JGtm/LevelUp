import { humanizeMetricKey } from '@/lib/i18n/metricLabel'
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
  // N'entrent ici que les métriques SANS FieldKey canonique équivalent (cf.
  // METRIC_TO_FIELD_KEY plus bas) : toutes les autres viennent du registre.
  metrics: {
    kills_per_game: 'Frags/match',
    deaths_per_game: 'Morts/match',
    assists_per_game: 'Assistances/match',
    career_rank: 'Rang carrière',
    csr: 'CSR (saison actuelle)',
    csr_alltime: 'CSR (record)',
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
  // Voir la note du dictionnaire FR : parité stricte des clés.
  metrics: {
    kills_per_game: 'Kills/game',
    deaths_per_game: 'Deaths/game',
    assists_per_game: 'Assists/game',
    career_rank: 'Career rank',
    csr: 'CSR (current season)',
    csr_alltime: 'CSR (all-time)',
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

/**
 * Métrique du contrat compare → FieldKey canonique du registre de titre
 * (`config/titles/{slug}/mappings/fields.toml`, servi par /field-mappings).
 *
 * Ces clés N'ONT PAS de libellé local : le registre est la source unique
 * (doctrine v7.3 lot 2 — cf. lib/i18n/metricLabel.ts). Repli si le registre ne
 * déclare pas la clé (titre partiel, réseau) : `humanizeMetricKey`, jamais un
 * dictionnaire TS.
 *
 * Les métriques absentes de cette table (kills_per_game, deaths_per_game,
 * assists_per_game, career_rank, csr, csr_alltime, perf_ath, lusr_ath) n'ont
 * pas d'équivalent canonique — elles restent dans le dictionnaire de feature.
 */
export const METRIC_TO_FIELD_KEY: Record<string, string> = {
  win_rate: 'win_rate',
  kda: 'kda',
  kdr: 'kdr',
  accuracy: 'accuracy',
  matches: 'total_matches_played',
  damage_per_game: 'avg_damage_dealt',
  damage_taken_per_game: 'avg_damage_taken',
  rendement: 'offensive_conversion',
  resistance: 'defensive_resistance',
  perfect_kills_per_game: 'perfect_kills_per_match',
  // MAX(max_killing_spree) sur la période (compare_repo.go) → la clé « valeur
  // de match », pas la moyenne `avg_max_killing_spree`.
  max_killing_spree: 'max_killing_spree',
  avg_life_secs: 'avg_life_seconds',
  headshot_kills_per_game: 'headshots_per_match',
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

/**
 * Libellé affichable d'une ligne de comparaison.
 *
 * Ordre : registre canonique (fusionné dans `metrics` par getCompareText) →
 * dictionnaire de feature → clé humanisée. Le contrat backend ne transporte
 * plus de libellé (`label_fr` retiré le 2026-08-04).
 */
export function resolveMetricLabel(text: CompareText, metric: string): string {
  return text.metrics[metric] ?? humanizeMetricKey(metric)
}
