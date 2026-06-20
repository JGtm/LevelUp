/**
 * Labels FR des descripteurs du profil de combat (3 axes × 5 bandes).
 *
 * Source unique consommée par Synthesis, Escouade (et référencée par le glossaire).
 * Vocabulaire et bornes calibrés sur la distribution des world leaders
 * (cf. .ai/PLAN_COMBAT_PROFILE_RECALIBRATION.md). Bornes backend dans
 * apps/go-api/internal/analysis/combat_yield.go.
 */
import type {
  CombatStyleOffensive,
  CombatStyleDefensive,
  CombatStyleActivity,
} from '@/lib/api/types'

// Offensif (conversion dégâts→kill) — du plus dispersé au plus létal.
const COMBAT_STYLE_OFFENSIVE_LABELS: Record<CombatStyleOffensive, string> = {
  disperse: 'Dispersé',
  irregulier: 'Irrégulier',
  equilibre: 'Équilibré',
  precis: 'Précis',
  chirurgical: 'Chirurgical',
}

// Défensif (survie) — du plus fragile au plus encaissant.
const COMBAT_STYLE_DEFENSIVE_LABELS: Record<CombatStyleDefensive, string> = {
  fragile: 'Fragile',
  expose: 'Exposé',
  solide: 'Solide',
  resistant: 'Résistant',
  inebranlable: 'Inébranlable',
}

// Activité (engagement absolu = pace joueur / pace lobby) — du passif à l'agressif.
const COMBAT_STYLE_ACTIVITY_LABELS: Record<CombatStyleActivity, string> = {
  passif: 'Passif',
  discret: 'Discret',
  mesure: 'Mesuré',
  actif: 'Actif',
  agressif: 'Agressif',
}

export function offensiveLabel(s: CombatStyleOffensive | null | undefined): string | null {
  return s ? (COMBAT_STYLE_OFFENSIVE_LABELS[s] ?? s) : null
}
export function defensiveLabel(s: CombatStyleDefensive | null | undefined): string | null {
  return s ? (COMBAT_STYLE_DEFENSIVE_LABELS[s] ?? s) : null
}
export function activityLabel(s: CombatStyleActivity | null | undefined): string | null {
  return s ? (COMBAT_STYLE_ACTIVITY_LABELS[s] ?? s) : null
}
