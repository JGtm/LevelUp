/**
 * Strings i18n du module coach (ADR 0020 Phase 10).
 * Convention identique aux autres features : FR + EN inline.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface CoachStrings {
  proposalsTitle: string
  proposalsEmpty: string
  proposalsOptInHint: string
  proposalsLoadError: string

  // Actions
  accept: string
  dismiss: string
  accepting: string
  dismissing: string

  // Détails carte
  origin: string
  originCatalog: string
  originSynthesized: string
  strength: string
  signal: string
  kindChallenge: string
  kindArc: string
  suggestedTier: string

  // Toast / feedback
  acceptedSuccess: string
  dismissedSuccess: string
  acceptError: string
  dismissError: string
}

export const coachStringsFR: CoachStrings = {
  proposalsTitle: 'Suggestions du coach',
  proposalsEmpty: 'Aucune suggestion pour le moment. Reviens après quelques matchs.',
  proposalsOptInHint:
    "Active 'Coach proactif' dans les paramètres pour recevoir des suggestions d'objectifs et d'arcs Prestige calibrés sur les tendances récentes.",
  proposalsLoadError: 'Impossible de charger les suggestions.',

  accept: 'Accepter',
  dismiss: 'Ignorer',
  accepting: 'Création...',
  dismissing: 'Suppression...',

  origin: 'Origine',
  originCatalog: 'Catalogue',
  originSynthesized: 'Synthétisé',
  strength: 'Force du signal',
  signal: 'Signal',
  kindChallenge: 'Défi',
  kindArc: 'Arc',
  suggestedTier: 'Palier suggéré',

  acceptedSuccess: 'Suggestion acceptée — défi créé dans Prestige.',
  dismissedSuccess: 'Suggestion ignorée.',
  acceptError: "Erreur lors de l'acceptation.",
  dismissError: "Erreur lors de l'ignoration.",
}

export const coachStringsEN: CoachStrings = {
  proposalsTitle: 'Coach suggestions',
  proposalsEmpty: 'No suggestions right now. Come back after a few matches.',
  proposalsOptInHint:
    "Turn on 'Proactive coach' in settings to receive suggestions for objectives and Prestige arcs calibrated on recent trends.",
  proposalsLoadError: 'Failed to load suggestions.',

  accept: 'Accept',
  dismiss: 'Dismiss',
  accepting: 'Creating...',
  dismissing: 'Removing...',

  origin: 'Origin',
  originCatalog: 'Catalog',
  originSynthesized: 'Synthesized',
  strength: 'Signal strength',
  signal: 'Signal',
  kindChallenge: 'Challenge',
  kindArc: 'Arc',
  suggestedTier: 'Suggested tier',

  acceptedSuccess: 'Suggestion accepted — challenge created in Prestige.',
  dismissedSuccess: 'Suggestion dismissed.',
  acceptError: 'Error while accepting.',
  dismissError: 'Error while dismissing.',
}

export function getCoachStrings(locale: Locale): CoachStrings {
  return locale === 'fr' ? coachStringsFR : coachStringsEN
}
