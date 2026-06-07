/**
 * i18n du module Prestige (dictionnaire local FR/EN, même pattern que
 * features/ascension/i18n.ts). Les composants Prestige étaient historiquement
 * français-only (strings hardcodées) ; ce module centralise les libellés pour
 * les rendre i18n-compatibles et lever les warnings `no-hardcoded-strings`.
 *
 * Règle : la version FR doit être en français PROPRE (pas de franglais). Les
 * emprunts anglais (baseline, deadline, preset, headshot…) sont traduits.
 */
import type { ManifestLocale } from '@/lib/i18n/format'

export interface PrestigeText {
  // ── Objectif (ObjectiveRow) ──
  objectiveSquadBadge: string
  objectivePPTooltip: string
  objectiveBadgeAlt: string
  // ── Arc (ArcSummary) ──
  arcCompleted: string
  arcPreset: string
  arcTotalPPTooltip: string
  /** Petit tag accolé au bonus de complétion d'arc (ex: "+150 PP bonus"). */
  arcBonusBadge: string
  arcBonusPPTooltip: string
  // ── Formulaire de création (CreateChallengeForm) ──
  formNewChallenge: string
  formCancel: string
  formModeHybrid: string
  formModeAuto: string
  formFieldMetric: string
  metricAccuracy: string
  metricHeadshots: string
  metricDamage: string
  metricPersonalScore: string
  metricWinRate: string
  formFieldTarget: string
  formFieldWindow: string
  windowSessions: string
  windowRollingDays: string
  windowDeadline: string
  formFieldDate: string
  formFieldValue: string
  formFieldCadence: string
  cadenceFree: string
  cadenceDaily: string
  cadenceWeekly: string
  cadenceMonthly: string
  formFieldLabel: string
  formLabelPlaceholder: string
  formLoadingSuggestions: string
  formNoTemplates: string
  formHybridHint: string
  formFieldAdjustedTarget: string
  formGenerating: string
  formAutoHint: string
  /** Interpolé : `{target}` remplacé par le palier Héroïque. */
  formAcceptHeroic: string
  formCreating: string
  formCreate: string
  // ── Formulaire de création d'arc (CreateArcForm) ──
  arcFormNew: string
  arcFormTitle: string
  arcFormTitlePlaceholder: string
  arcFormDescription: string
  arcFormDescriptionPlaceholder: string
  arcFormCreate: string
  // ── MomentCard ──
  momentAchieved: string
  momentBaseline: string
  momentDelta: string
  momentMatches: string
}

const FR: PrestigeText = {
  objectiveSquadBadge: 'Escouade',
  objectivePPTooltip: 'Points de Prestige gagnés à la complétion',
  objectiveBadgeAlt: 'Objectif',
  arcCompleted: 'Accompli',
  arcPreset: 'Préréglage',
  arcTotalPPTooltip: "Points de Prestige cumulés des objectifs de l'arc",
  arcBonusBadge: 'bonus',
  arcBonusPPTooltip:
    "Bonus de complétion de l'arc, crédité une fois toutes les étapes terminées — en plus des PP de chaque objectif.",
  formNewChallenge: 'Nouveau défi',
  formCancel: 'Annuler',
  formModeHybrid: 'Hybride',
  formModeAuto: 'Automatique',
  formFieldMetric: 'Métrique',
  metricAccuracy: 'Précision',
  metricHeadshots: 'Tirs à la tête',
  metricDamage: 'Dégâts infligés',
  metricPersonalScore: 'Score perso',
  metricWinRate: 'Taux de victoire (%)',
  formFieldTarget: 'Cible',
  formFieldWindow: 'Fenêtre',
  windowSessions: 'Sessions',
  windowRollingDays: 'Jours glissants',
  windowDeadline: 'Échéance',
  formFieldDate: 'Date (AAAA-MM-JJ)',
  formFieldValue: 'Valeur',
  formFieldCadence: 'Cadence',
  cadenceFree: 'Libre',
  cadenceDaily: 'Quotidien',
  cadenceWeekly: 'Hebdomadaire',
  cadenceMonthly: 'Mensuel',
  formFieldLabel: 'Libellé (optionnel)',
  formLabelPlaceholder: 'Ex. Slayer Niv. 2',
  formLoadingSuggestions: 'Chargement des suggestions…',
  formNoTemplates: 'Aucun modèle disponible. Bascule en mode libre pour créer un défi personnalisé.',
  formHybridHint: 'Choisis un modèle ; ajuste la cible si tu veux.',
  formFieldAdjustedTarget: 'Cible ajustée',
  formGenerating: 'Génération des propositions…',
  formAutoHint: '3 défis tirés du catalogue, à accepter directement.',
  formAcceptHeroic: 'Accepter (Héroïque : {target})',
  formCreating: 'Création…',
  formCreate: 'Créer le défi',
  arcFormNew: 'Nouvel arc',
  arcFormTitle: 'Titre',
  arcFormTitlePlaceholder: 'Ex. Ascension du Spartan',
  arcFormDescription: 'Description (optionnelle)',
  arcFormDescriptionPlaceholder: 'Ex. Enchaîne les objectifs pour gravir les paliers.',
  arcFormCreate: "Créer l'arc",
  momentAchieved: 'Atteint',
  momentBaseline: 'Référence',
  momentDelta: 'Évolution',
  momentMatches: 'matchs',
}

const EN: PrestigeText = {
  objectiveSquadBadge: 'Squad',
  objectivePPTooltip: 'Prestige Points earned on completion',
  objectiveBadgeAlt: 'Objective',
  arcCompleted: 'Completed',
  arcPreset: 'Preset',
  arcTotalPPTooltip: "Total Prestige Points from the arc's objectives",
  arcBonusBadge: 'bonus',
  arcBonusPPTooltip:
    "Arc completion bonus, credited once all steps are done — on top of each objective's PP.",
  formNewChallenge: 'New challenge',
  formCancel: 'Cancel',
  formModeHybrid: 'Hybrid',
  formModeAuto: 'Automatic',
  formFieldMetric: 'Metric',
  metricAccuracy: 'Accuracy',
  metricHeadshots: 'Headshots',
  metricDamage: 'Damage dealt',
  metricPersonalScore: 'Personal score',
  metricWinRate: 'Win rate (%)',
  formFieldTarget: 'Target',
  formFieldWindow: 'Window',
  windowSessions: 'Sessions',
  windowRollingDays: 'Rolling days',
  windowDeadline: 'Deadline',
  formFieldDate: 'Date (YYYY-MM-DD)',
  formFieldValue: 'Value',
  formFieldCadence: 'Cadence',
  cadenceFree: 'Free',
  cadenceDaily: 'Daily',
  cadenceWeekly: 'Weekly',
  cadenceMonthly: 'Monthly',
  formFieldLabel: 'Label (optional)',
  formLabelPlaceholder: 'e.g. Slayer Lv.2',
  formLoadingSuggestions: 'Loading suggestions…',
  formNoTemplates: 'No template available. Switch to free mode to create a custom challenge.',
  formHybridHint: 'Pick a template; adjust the target if you want.',
  formFieldAdjustedTarget: 'Adjusted target',
  formGenerating: 'Generating proposals…',
  formAutoHint: '3 challenges drawn from the catalog, accept directly.',
  formAcceptHeroic: 'Accept (Heroic: {target})',
  formCreating: 'Creating…',
  formCreate: 'Create challenge',
  arcFormNew: 'New arc',
  arcFormTitle: 'Title',
  arcFormTitlePlaceholder: 'e.g. Spartan Ascension',
  arcFormDescription: 'Description (optional)',
  arcFormDescriptionPlaceholder: 'e.g. Chain objectives to climb the tiers.',
  arcFormCreate: 'Create arc',
  momentAchieved: 'Achieved',
  momentBaseline: 'Baseline',
  momentDelta: 'Delta',
  momentMatches: 'matches',
}

export function getPrestigeText(locale: ManifestLocale): PrestigeText {
  return locale === 'en' ? EN : FR
}
