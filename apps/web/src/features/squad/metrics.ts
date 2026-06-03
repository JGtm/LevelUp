/**
 * metrics.ts — Listes de FieldKeys canoniques utilisées par les surfaces
 * de la feature Escouade (KPI cards, bar synergies, radar contributions,
 * overlay HS/PK).
 *
 * Frontière multi-titres : chaque entrée pointe vers une FieldKey canonique
 * (cf. apps/go-api/internal/games/canonical/fields.go) que `useFieldLabel`
 * résout via le fields.toml du titre courant. Si la key est absente du titre
 * (ex. synthetic_title_b qui n'a pas `win_rate` ni `kdr`), la métrique est
 * silencieusement omise par le rendu (graceful degradation).
 *
 * Ne pas hardcoder de fallback FR ici — le fallback est la key elle-même,
 * géré par useFieldLabel.
 */

import type { TeammateKPIs } from '@/lib/api/types'

/**
 * Format d'affichage d'une métrique. Détermine le suffixe (%, /match, ratio)
 * et la précision de rendu.
 */
export type MetricFormat = 'integer' | 'percent' | 'ratio' | 'per_game'

/**
 * Une métrique d'escouade : pointe vers une FieldKey canonique pour son label,
 * extrait sa valeur depuis un TeammateKPIs et déclare son format d'affichage.
 *
 * - `key` : FieldKey canonique (cf. fields.go) — résolu via useFieldLabel.
 * - `extract` : projection depuis le DTO TeammateKPIs vers un nombre nullable.
 * - `format` : pour `per_game` le label affiché est composé : `${label}${units.perGame}`.
 */
export interface SquadMetric {
  key: string
  extract: (k: TeammateKPIs) => number | null
  format: MetricFormat
}

// ─── KPI cards (en-tête de page) ──────────────────────────────────────────────
//
// 4 métriques par défaut sur HI : matchs, win rate, K/D, kills/match.
// Sur synthetic_title_b (fields.toml minimaliste), seules les keys présentes
// produisent une card — le rendu skip les autres sans crash.

export const SQUAD_KPI_METRICS: readonly SquadMetric[] = [
  { key: 'total_matches_played', extract: (k) => k.match_count, format: 'integer' },
  { key: 'win_rate', extract: (k) => k.win_rate * 100, format: 'percent' },
  { key: 'kdr', extract: (k) => k.kd_ratio, format: 'ratio' },
  { key: 'kills', extract: (k) => k.kills_per_game, format: 'per_game' },
] as const

// ─── Bar chart Synergies ──────────────────────────────────────────────────────

export const SQUAD_SYNERGY_METRICS: readonly SquadMetric[] = [
  { key: 'win_rate', extract: (k) => k.win_rate * 100, format: 'percent' },
  { key: 'kdr', extract: (k) => k.kd_ratio, format: 'ratio' },
  { key: 'kills', extract: (k) => k.kills_per_game, format: 'per_game' },
  { key: 'assists', extract: (k) => k.assists_per_game, format: 'per_game' },
] as const

// ─── Radar Contributions ──────────────────────────────────────────────────────
//
// Valeurs normalisées 0-100 pour la lisibilité du radar.
// L'extracteur renvoie une valeur déjà normalisée par rapport à un plafond
// arbitraire choisi pour Halo Infinite (3 pour KDR, 20 pour KPG, 10 pour APG,
// 1 pour accuracy ratio). Pour un autre titre, ces plafonds peuvent être
// inadaptés : à terme, exposer un `ceiling` dans le SquadMetric pour le rendre
// title-aware (TODO multi-title Phase >).

const norm = (v: number | null, max: number): number =>
  v != null ? Math.min(100, (v / max) * 100) : 0

export const SQUAD_RADAR_METRICS: readonly SquadMetric[] = [
  { key: 'win_rate', extract: (k) => k.win_rate * 100, format: 'percent' },
  { key: 'kdr', extract: (k) => norm(k.kd_ratio, 3), format: 'ratio' },
  { key: 'kills', extract: (k) => norm(k.kills_per_game, 20), format: 'per_game' },
  { key: 'assists', extract: (k) => norm(k.assists_per_game, 10), format: 'per_game' },
  // norm(accuracy, 1) renvoie déjà un % 0-100 (norm fait *100 en interne).
  // L'ancien code SquadContributionsPage faisait `norm(...) * 100` qui
  // produisait 4500 pour accuracy=0.45 — bug visible une fois passé en test.
  { key: 'accuracy', extract: (k) => norm(k.accuracy, 1), format: 'percent' },
] as const

// ─── Overlay HS / PK ──────────────────────────────────────────────────────────
//
// Métriques dérivées de FieldKeys canoniques (`headshot_kills`, et un futur
// `perfect_kills` non encore au registre canonical). Pour l'instant on déclare
// 'headshot_kills' (présent dans le fields.toml HI) et un fallback explicite
// pour perfect_kills (key réservée mais pas encore enregistrée canonical).

export const SQUAD_HSPK_METRICS = {
  hs: {
    key: 'headshot_kills',
    extract: (k: TeammateKPIs) => k.headshot_kills_per_game ?? 0,
    format: 'per_game' as MetricFormat,
  },
  pk: {
    key: 'perfect_kills',
    extract: (k: TeammateKPIs) => k.perfect_kills_per_game ?? 0,
    format: 'per_game' as MetricFormat,
  },
} as const satisfies Record<'hs' | 'pk', SquadMetric>
