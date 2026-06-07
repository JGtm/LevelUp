/**
 * i18n du module Prestige (dictionnaire local FR/EN, même pattern que
 * features/ascension/i18n.ts). Les composants Prestige étaient historiquement
 * français-only (strings hardcodées) ; ce module centralise les libellés pour
 * les rendre i18n-compatibles et lever les warnings `no-hardcoded-strings`.
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
}

const FR: PrestigeText = {
  objectiveSquadBadge: 'Escouade',
  objectivePPTooltip: 'Points de Prestige gagnés à la complétion',
  objectiveBadgeAlt: 'Objectif',
  arcCompleted: 'Accompli',
  arcPreset: 'Preset',
  arcTotalPPTooltip: "Points de Prestige cumulés des objectifs de l'arc",
}

const EN: PrestigeText = {
  objectiveSquadBadge: 'Squad',
  objectivePPTooltip: 'Prestige Points earned on completion',
  objectiveBadgeAlt: 'Objective',
  arcCompleted: 'Completed',
  arcPreset: 'Preset',
  arcTotalPPTooltip: "Total Prestige Points from the arc's objectives",
}

export function getPrestigeText(locale: ManifestLocale): PrestigeText {
  return locale === 'en' ? EN : FR
}
