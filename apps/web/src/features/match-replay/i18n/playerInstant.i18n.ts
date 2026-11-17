/**
 * playerInstant.i18n.ts — LES TEXTES DE LA FICHE DU JOUEUR « À L'INSTANT » : ce que le joueur a fait
 * du début du match jusqu'à l'instant lu, à côté des totaux du match. Le contrat (`PlayerInstantText`)
 * et ses deux langues.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de
 * taille du dépôt. Servi par `REPLAY_TEXT[locale].playerInstant` ; la parité FR/EN reste tenue par le
 * typage (`satisfies PlayerInstantText` dans chaque langue, `satisfies ReplayText` au-dessus).
 *
 * LES NOMS DE CLASSE D'ARME NE SONT PAS ICI : ils viennent du manifeste de l'application
 * (`frags.class.*`), le même que la page Synthèse.
 */

export interface PlayerInstantText {
  /** Le titre de l'état du joueur à l'instant lu, et l'en-tête de la colonne du même nom. */
  now: string
  /** Le tableau des éliminations, par classe d'arme puis par arme. */
  killsByWeapon: string
  /** En-tête de colonne : la part de chaque arme dans les éliminations du joueur à l'instant. */
  colShare: string
  /** En-tête de colonne : le total de fin de match. */
  colMatch: string
  /** Le tableau de l'équipement, par famille. */
  equipment: string
  /** Les trois colonnes de l'équipement : jamais « ramassé ». */
  equipmentTaken: string
  equipmentUsed: string
  equipmentLost: string
  medals: string
  /** La section des gestes du joueur (sprint, sauts). */
  gestures: string
  sprintTotal: string
  jumps: string
  matchTotals: string
  /** Les deux états d'une arme dans le bloc « À l'instant » : dégainée, et rangée. */
  weaponInHand: string
  weaponStowed: string
  /** Les charges restantes de l'équipement porté (« 1 charge », « 2 charges »). */
  equipmentCharges: (n: number) => string
}

export const FR_PLAYER_INSTANT = {
  now: "À l'instant",
  killsByWeapon: 'Éliminations par arme',
  colShare: 'Part',
  colMatch: 'Match',
  equipment: 'Équipement',
  equipmentTaken: 'Pris',
  equipmentUsed: 'Utilisé',
  equipmentLost: 'Perdu',
  medals: 'Médailles',
  gestures: 'Gestes',
  sprintTotal: 'Sprint, temps cumulé',
  jumps: 'Sauts',
  matchTotals: 'Totaux du match',
  weaponInHand: 'en main',
  weaponStowed: 'en réserve',
  equipmentCharges: (n) => (n === 1 ? '1 charge' : `${n} charges`),
} satisfies PlayerInstantText

export const EN_PLAYER_INSTANT = {
  now: 'Right now',
  killsByWeapon: 'Kills by weapon',
  colShare: 'Share',
  colMatch: 'Match',
  equipment: 'Equipment',
  equipmentTaken: 'Taken',
  equipmentUsed: 'Used',
  equipmentLost: 'Lost',
  medals: 'Medals',
  gestures: 'Moves',
  sprintTotal: 'Sprint, total time',
  jumps: 'Jumps',
  matchTotals: 'Match totals',
  weaponInHand: 'in hand',
  weaponStowed: 'in reserve',
  equipmentCharges: (n) => (n === 1 ? '1 charge' : `${n} charges`),
} satisfies PlayerInstantText
