/**
 * assistsI18n.ts — libellés des assistances échangées (page Relations : tableau, cartes
 * Binôme et Noyau dur ; vue match : historique des rencontres).
 *
 * Vit dans `_shared` parce que deux features l'affichent (palmares, match-view) ; parité
 * FR/EN par typage `Record<Locale, AssistsText>`.
 *
 * Bornes des tranches (0-25 / 25-50 / plus de 50 %) : miroir de
 * `domain.AssistTierLowMaxPct` / `AssistTierMidMaxPct` côté Go.
 */
import type { Locale } from '@/lib/i18n/locale'

export type AssistTier = 'low' | 'mid' | 'high'

export interface AssistsText {
  /** Infobulle d'en-tête de colonne des tableaux. Le libellé de colonne lui-même vient du
   *  champ `assists` des mappings du titre (useFieldLabel), pas de ce dictionnaire. */
  columnTooltip: string
  /** Infobulle d'une cellule de tableau. */
  cellGiven: (count: string) => string
  cellReceived: (count: string) => string
  cellCoverage: (measured: string, together: string) => string
  /** Carte Binôme : têtes de la barre papillon. */
  receivedHead: string
  givenHead: string
  /** Légende sous les barres de la carte Noyau dur. */
  legendReceived: string
  legendGiven: string
  /** Infobulle d'un segment de barre. */
  segment: (count: string, tier: AssistTier) => string
  /** Infobulle d'un fidèle sans match mesuré. */
  notMeasured: string
}

const TIER_FR: Record<AssistTier, string> = {
  low: '0 à 25 % des dégâts',
  mid: '25 à 50 % des dégâts',
  high: 'plus de 50 % des dégâts',
}

const TIER_EN: Record<AssistTier, string> = {
  low: '0 to 25% of the damage',
  mid: '25 to 50% of the damage',
  high: 'more than 50% of the damage',
}

export const ASSISTS_TEXT: Record<Locale, AssistsText> = {
  fr: {
    columnTooltip:
      "Assistances données (à gauche) et reçues (à droite), sur les matchs joués dans la même équipe dont le film a été analysé.",
    cellGiven: (count) => `Tu l'as assisté ${count} fois`,
    cellReceived: (count) => `Il t'a assisté ${count} fois`,
    cellCoverage: (measured, together) => `Mesuré sur ${measured} de vos ${together} matchs ensemble`,
    receivedHead: "Il t'a assisté",
    givenHead: "Tu l'as assisté",
    legendReceived: '◀ te sert',
    legendGiven: 'tu le sers ▶',
    segment: (count, tier) => `${count} frags assistés · ${TIER_FR[tier]}`,
    notMeasured: 'Aucun match ensemble avec film analysé',
  },
  en: {
    columnTooltip:
      'Assists given (left) and received (right), over matches played on the same team whose film was analyzed.',
    cellGiven: (count) => `You assisted them ${count} times`,
    cellReceived: (count) => `They assisted you ${count} times`,
    cellCoverage: (measured, together) => `Measured over ${measured} of your ${together} matches together`,
    receivedHead: 'They assisted you',
    givenHead: 'You assisted them',
    legendReceived: '◀ serves you',
    legendGiven: 'you serve them ▶',
    segment: (count, tier) => `${count} kills assisted · ${TIER_EN[tier]}`,
    notMeasured: 'No match together with an analyzed film',
  },
}
