/**
 * feedLines.i18n.ts — LES TEXTES DES LIGNES DU FIL qui ne sont pas une action nommée du joueur :
 * l'infobulle d'une ligne d'élimination (assistance, victime porteuse du drapeau, médailles en trop),
 * les assistances d'une capture, la fin de manche, le crâne d'Oddball et la colline. Le contrat
 * (`FeedLinesText`) et ses deux langues.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `lives.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de taille
 * du dépôt. Servis par `REPLAY_TEXT[locale].feedLines` ; la parité FR/EN est tenue par le typage
 * (`satisfies FeedLinesText` dans chaque langue). Les phrases d'action suivent celles des autres
 * lignes du fil (`feedObjectiveAction`) : passé composé, après le nom, aucun possessif.
 */
export interface FeedLinesText {
  /** Infobulle d'une ligne de frag dont la victime portait le drapeau. */
  carrierBadgeHint: string
  /** Infobulle d'une ligne de frag assistée : l'assistant nommé et sa part de dégâts, quand elle est connue. */
  assistedBy: (name: string, sharePct: number | null) => string
  /** Les médailles d'une ligne au-delà des deux pictogrammes : « +N », puis ce que l'infobulle liste. */
  medalsMore: (n: number) => string
  medalsMoreHint: (names: string) => string
  /** Les pictogrammes d'une ligne d'élimination : tir à la tête, frag parfait (infobulle et nom accessible). */
  headshotMark: string
  perfectMark: string
  /** Infobulle d'une capture qui porte des assistances : `names` sont les joueurs crédités. */
  captureAssistedBy: (names: string) => string
  /** La fin d'une manche : `n` est son rang d'affichage, à partir de 1. */
  roundEnd: (n: number) => string
  /** Le crâne d'Oddball : après le nom du porteur. */
  skullTaken: string
  skullLost: string
  /** La colline : après le nom du camp qui la tient. */
  hillTaken: string
}

export const FR_FEED_LINES = {
  carrierBadgeHint: 'La victime portait le drapeau',
  assistedBy: (name, sharePct) => `Assistance : ${name}${sharePct == null ? '' : ` (${sharePct} %)`}`,
  medalsMore: (n) => `+${n}`,
  medalsMoreHint: (names) => `Autres médailles : ${names}`,
  headshotMark: 'Tir à la tête',
  perfectMark: 'Frag parfait',
  captureAssistedBy: (names) => `Assistances : ${names}`,
  roundEnd: (n) => `Fin de la manche ${n}`,
  skullTaken: 'a pris le crâne',
  skullLost: 'a perdu le crâne',
  hillTaken: 'a pris la colline',
} satisfies FeedLinesText

export const EN_FEED_LINES = {
  carrierBadgeHint: 'The victim was carrying the flag',
  assistedBy: (name, sharePct) => `Assist: ${name}${sharePct == null ? '' : ` (${sharePct}%)`}`,
  medalsMore: (n) => `+${n}`,
  medalsMoreHint: (names) => `More medals: ${names}`,
  headshotMark: 'Headshot',
  perfectMark: 'Perfect kill',
  captureAssistedBy: (names) => `Assists: ${names}`,
  roundEnd: (n) => `Round ${n} over`,
  skullTaken: 'picked up the skull',
  skullLost: 'lost the skull',
  hillTaken: 'took the hill',
} satisfies FeedLinesText
