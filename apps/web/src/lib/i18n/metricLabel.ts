/**
 * metricLabel — résolution du libellé lisible d'une clé de métrique.
 *
 * SOURCE UNIQUE (v7.3 lot 2, item 3.3) : les libellés viennent des manifests
 * TOML servis par `/api/v1/titles/{slug}/field-mappings` (config/titles/{slug}/
 * mappings/fields.toml). Les deux dictionnaires FR/EN qui vivaient ici ont été
 * SUPPRIMÉS — ils doublonnaient le TOML et divergeaient (« Éliminations » ici,
 * « Frags » dans le registre ; « Matchs joués » ici, « Matchs » là-bas).
 *
 * Ce module ne conserve que ce qui n'est PAS un libellé :
 *  - `FIELD_KEY_ALIASES` : clé transmise par le backend → FieldKey canonique.
 *    Deux familles d'alias, aucune valeur affichable :
 *      · jetons Prestige `Field*` (contrat fil des défis) ;
 *      · clés courtes historiques du détecteur de records (`kpm`, `pspm`) et
 *        clés de catalogue milestones (`headshots`, `matches_played`).
 *  - `humanizeMetricKey` : UNIQUE repli d'affichage quand le TOML ne déclare
 *    pas la clé (clé brute humanisée, jamais un dictionnaire).
 *
 * Garde-rails : metricLabel.test.ts, features/metric-key-guardrail.test.ts
 * (aucun `Field*` en JSX) et lib/i18n/no-field-label-dictionary.test.ts
 * (interdit de reconstituer un dictionnaire de libellés de field-keys).
 */
import { useFieldLabel } from '@/lib/i18n/fieldMappings'

/**
 * Alias des clés reçues du backend vers leur FieldKey canonique (celle déclarée
 * dans les TOML). Confiné ici, hors JSX : ces valeurs sont des identifiants
 * techniques, jamais un libellé affiché.
 *
 * Les clés courtes `kpm`/`pspm` sont émises par le détecteur de records
 * (internal/progression/records/extractors.go) ; `headshots` et
 * `matches_played` proviennent des catalogues de jalons
 * (config/titles/{slug}/milestones/catalog.toml). Aligner ces noms côté backend
 * est un chantier distinct : tant qu'ils circulent, l'alias les rattache au
 * registre canonique plutôt qu'à un libellé en dur.
 */
const FIELD_KEY_ALIASES: Record<string, string> = {
  FieldKDA: 'kda',
  FieldKDR: 'kdr',
  FieldAccuracy: 'accuracy',
  FieldHeadshotKills: 'headshot_kills',
  FieldDamageDealt: 'damage_dealt',
  FieldPersonalScore: 'personal_score',
  FieldWinRate: 'win_rate',
  kpm: 'kills_per_min',
  pspm: 'personal_score_per_min',
  headshots: 'headshot_kills',
  matches_played: 'match_count',
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

/** Résout la FieldKey canonique d'une clé de métrique (alias appliqué). */
export function canonicalMetricKey(key: string): string {
  return FIELD_KEY_ALIASES[key] ?? key
}

/**
 * humanizeMetricKey produit un libellé lisible depuis une clé non déclarée dans
 * les TOML : retire le préfixe `Field`, sépare camelCase et snake_case,
 * capitalise l'initiale. Ne renvoie jamais la clé brute telle quelle ni un
 * jeton `Field*`. C'est le SEUL repli d'affichage autorisé.
 */
export function humanizeMetricKey(key: string): string {
  const spaced = key
    .replace(/^Field/, '')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/_/g, ' ')
    .trim()
  if (spaced === '') return key
  return spaced.charAt(0).toUpperCase() + spaced.slice(1).toLowerCase()
}

/**
 * useMetricLabel — libellé localisé d'une clé de métrique, depuis les TOML.
 *
 * Hook React : à appeler dans le corps d'un composant, jamais dans une boucle
 * ni après un retour conditionnel (règles des hooks). Pour une liste, extraire
 * un sous-composant par élément.
 *
 * `useFieldLabel` renvoie la clé elle-même quand le mapping est absent (ou pas
 * encore chargé) : dans ce cas seulement, on humanise — en repartant de la clé
 * D'ORIGINE pour que `FieldMysteryStat` donne « Mystery stat » et non le nom
 * canonique de l'alias.
 */
export function useMetricLabel(key: string): string {
  const canonical = canonicalMetricKey(key)
  const label = useFieldLabel(canonical)
  return label === canonical ? humanizeMetricKey(key) : label
}
