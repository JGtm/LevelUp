/**
 * metricLabel — libellé lisible d'une clé de métrique. Source UNIQUE pour les
 * features Ascension (clés canoniques snake_case) et Prestige (clés `Field*`).
 *
 * Deux familles de clés convergent ici :
 *  - canoniques snake_case (kda, accuracy, matches_played, …) — records,
 *    milestones, patterns ;
 *  - `Field*` (FieldKDA, FieldWinRate, …) — métriques de défi Prestige, alias
 *    vers la clé canonique correspondante.
 *
 * Clé inconnue → libellé humanisé (jamais la clé brute, jamais un jeton `Field*`).
 * Centralisation (règle n°6 CLAUDE.md) : aucun composant ne doit mapper une clé
 * de métrique en dur ni afficher `Field*`. Garde-rails : metricLabel.test.ts
 * (humanisation) + features/metric-key-guardrail.test.ts (aucun `Field*` en JSX).
 */
import type { Locale } from '@/lib/i18n/locale'

const METRIC_LABEL_FR: Record<string, string> = {
  performance_score: 'Score de performance',
  kda: 'KDA',
  kdr: 'K/D',
  kpm: 'Tueries / minute',
  accuracy: 'Précision',
  pspm: 'Score perso / minute',
  matches_played: 'Matchs joués',
  wins: 'Victoires',
  kills: 'Éliminations',
  // headshots : clé du catalogue milestones (config/titles/*/milestones/catalog.toml,
  // metric = "headshots") — historique, conservée telle quelle.
  // headshot_kills : clé canonique réelle (games/canonical/fields.go FieldHeadshotKills
  // = "headshot_kills", fields.toml). C'est CETTE forme que reçoit metricLabel() quand
  // le contrat fil transporte la clé canonique brute plutôt qu'un alias `Field*` —
  // sans entrée dédiée, elle retombait sur humanizeMetricKey (jamais traduite en FR).
  headshots: 'Tirs à la tête',
  headshot_kills: 'Tirs à la tête',
  assists: 'Assistances',
  damage_dealt: 'Dégâts infligés',
  personal_score: 'Score personnel',
  win_rate: 'Taux de victoire',
  accuracy_threshold_days: 'Jours réguliers',
  combat_precision_matches: 'Matchs de précision',
  combat_endurance_matches: "Matchs d'endurance",
  combat_excellence_matches: "Matchs d'excellence",
}

const METRIC_LABEL_EN: Record<string, string> = {
  performance_score: 'Performance score',
  kda: 'KDA',
  kdr: 'K/D',
  kpm: 'Kills per minute',
  accuracy: 'Accuracy',
  pspm: 'Personal score per minute',
  matches_played: 'Matches played',
  wins: 'Wins',
  kills: 'Kills',
  headshots: 'Headshots',
  headshot_kills: 'Headshots',
  assists: 'Assists',
  damage_dealt: 'Damage dealt',
  personal_score: 'Personal score',
  win_rate: 'Win rate',
  accuracy_threshold_days: 'Consistent days',
  combat_precision_matches: 'Precision matches',
  combat_endurance_matches: 'Endurance matches',
  combat_excellence_matches: 'Excellence matches',
}

/**
 * Alias des clés Prestige `Field*` vers leur clé canonique snake_case.
 * Confiné ici (hors JSX) : la valeur `Field*` reste le contrat fil envoyé au
 * backend, jamais un libellé affiché.
 */
const FIELD_KEY_ALIASES: Record<string, string> = {
  FieldKDA: 'kda',
  FieldKDR: 'kdr',
  FieldAccuracy: 'accuracy',
  FieldHeadshotKills: 'headshot_kills',
  FieldDamageDealt: 'damage_dealt',
  FieldPersonalScore: 'personal_score',
  FieldWinRate: 'win_rate',
}

/**
 * Options de métrique proposées à la création d'un défi Prestige (mode libre).
 * Les valeurs `Field*` vivent ici (module .ts, hors JSX rendu) pour ne pas
 * exposer d'identifiant technique dans les composants — garde-rail respecté.
 */
export const PRESTIGE_METRIC_OPTIONS: readonly string[] = [
  'FieldKDA',
  'FieldKDR',
  'FieldAccuracy',
  'FieldHeadshotKills',
  'FieldDamageDealt',
  'FieldPersonalScore',
  'FieldWinRate',
]

/** Retourne le libellé localisé d'une clé de métrique (canonique ou `Field*`). */
export function metricLabel(key: string, locale: Locale): string {
  const canonical = FIELD_KEY_ALIASES[key] ?? key
  const map = locale === 'en' ? METRIC_LABEL_EN : METRIC_LABEL_FR
  return map[canonical] ?? humanizeMetricKey(key)
}

/**
 * humanizeMetricKey produit un libellé lisible depuis une clé inconnue : retire
 * le préfixe `Field`, sépare camelCase et snake_case, capitalise l'initiale.
 * Ne renvoie jamais la clé brute telle quelle ni un jeton `Field*`.
 */
function humanizeMetricKey(key: string): string {
  const spaced = key
    .replace(/^Field/, '')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/_/g, ' ')
    .trim()
  if (spaced === '') return key
  return spaced.charAt(0).toUpperCase() + spaced.slice(1).toLowerCase()
}
