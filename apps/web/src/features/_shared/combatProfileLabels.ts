/**
 * Labels FR/EN des descripteurs du profil de combat (3 axes × 5 bandes).
 *
 * Source unique consommée par Synthesis, Escouade (et référencée par le glossaire).
 * Vocabulaire et bornes calibrés sur la distribution des world leaders
 * (cf. .ai/PLAN_COMBAT_PROFILE_RECALIBRATION.md). Bornes backend dans
 * apps/go-api/internal/analysis/combat_yield.go.
 *
 * Localisé : chaque fonction prend la locale active (ManifestLocale) — un utilisateur
 * EN ne doit plus voir les libellés FR (bug « section Profil de combat hardcodée FR »).
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import type {
  CombatStyleOffensive,
  CombatStyleDefensive,
  CombatStyleActivity,
} from '@/lib/api/types'

// Offensif (conversion dégâts→kill) — du plus dispersé au plus létal.
const COMBAT_STYLE_OFFENSIVE_LABELS: Record<ManifestLocale, Record<CombatStyleOffensive, string>> = {
  fr: {
    disperse: 'Dispersé',
    irregulier: 'Irrégulier',
    equilibre: 'Équilibré',
    precis: 'Précis',
    chirurgical: 'Chirurgical',
  },
  en: {
    disperse: 'Scattered',
    irregulier: 'Irregular',
    equilibre: 'Balanced',
    precis: 'Precise',
    chirurgical: 'Surgical',
  },
}

// Défensif (survie) — du plus fragile au plus encaissant.
const COMBAT_STYLE_DEFENSIVE_LABELS: Record<ManifestLocale, Record<CombatStyleDefensive, string>> = {
  fr: {
    fragile: 'Fragile',
    expose: 'Exposé',
    solide: 'Solide',
    resistant: 'Résistant',
    inebranlable: 'Inébranlable',
  },
  en: {
    fragile: 'Fragile',
    expose: 'Exposed',
    solide: 'Solid',
    resistant: 'Resilient',
    inebranlable: 'Unshakable',
  },
}

// Activité (engagement absolu = pace joueur / pace lobby) — du passif à l'agressif.
const COMBAT_STYLE_ACTIVITY_LABELS: Record<ManifestLocale, Record<CombatStyleActivity, string>> = {
  fr: {
    passif: 'Passif',
    discret: 'Discret',
    mesure: 'Mesuré',
    actif: 'Actif',
    agressif: 'Agressif',
  },
  en: {
    passif: 'Passive',
    discret: 'Discreet',
    mesure: 'Measured',
    actif: 'Active',
    agressif: 'Aggressive',
  },
}

export function offensiveLabel(s: CombatStyleOffensive | null | undefined, locale: ManifestLocale): string | null {
  return s ? (COMBAT_STYLE_OFFENSIVE_LABELS[locale][s] ?? s) : null
}
export function defensiveLabel(s: CombatStyleDefensive | null | undefined, locale: ManifestLocale): string | null {
  return s ? (COMBAT_STYLE_DEFENSIVE_LABELS[locale][s] ?? s) : null
}
export function activityLabel(s: CombatStyleActivity | null | undefined, locale: ManifestLocale): string | null {
  return s ? (COMBAT_STYLE_ACTIVITY_LABELS[locale][s] ?? s) : null
}
