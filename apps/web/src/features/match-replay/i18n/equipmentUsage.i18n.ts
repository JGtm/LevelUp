/**
 * equipmentUsage.i18n.ts — LES TEXTES DU BLOC « Usage d'équipement, par joueur » (Vue match,
 * onglet « Armes et terrain » et Chronologie) : le contrat (`EquipmentUsageText`) et ses deux langues.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de
 * taille du dépôt. Servis par `REPLAY_TEXT[locale].equipmentUsage` ; la parité FR/EN reste tenue par le
 * typage (`satisfies EquipmentUsageText` dans chaque langue, `satisfies ReplayText` au-dessus).
 */

/**
 * « Usage d'équipements, par joueur » (Vue match, onglet « Armes et terrain ») : la grille compte,
 * sur tout le match, ce que le rejeu ne montre qu'image par image.
 *
 * `infoByPlayer` dit ce que la grille compte ; le RÉPULSEUR n'a pas de colonne (une colonne vide
 * se lirait « zéro usage »), et aucun texte ne dit ce que la grille ne compte pas (aucun inconnu à
 * l'écran, garde `lib/i18n/noUnknownMentions.guard.test.ts`).
 *
 * LES NOMS DE FAMILLE NE SONT PAS ICI, et c'est voulu : ils vivent déjà dans `placementFamily`
 * (règles de rendu) et `padEquipmentFamily` (socles de bonus). Une troisième table de noms
 * divergerait au premier ajout du manifeste.
 */
export interface EquipmentUsageText {
  title: string
  /** Titre de la carte : la grille des joueurs, une colonne par famille d'équipement. */
  viewByPlayer: string
  /** Aide du titre : ce que la grille compte et sur quel périmètre (une ou deux phrases). */
  infoByPlayer: string
  /** Légende : les trois issues d'une pile, puis les tractions de grappin. */
  legendUsed: string
  legendKept: string
  legendDropped: string
  legendGrapple: string
  /** Infobulle d'une barre de la grille : joueur, grandeur, valeur DEJA ecrite. */
  gridTipFmt: (player: string, column: string, value: string) => string
  /** En-tête de la colonne des tractions de grappin (la seule ACTIVATION de capacité attribuée). */
  groupGrapple: string
  /**
   * LES TROIS ISSUES D'UNE PILE, formatées pour l'infobulle d'UN segment — pas pour un
   * en-tête de colonne (celui-ci reste `equipmentFamilyLabel`). Le total de la cellule
   * s'écrit « N objets pris », JAMAIS « ramassés » : `outcomeTotalTakenFmt`.
   */
  outcomeUsedFmt: (count: number) => string
  outcomeKeptFmt: (count: number) => string
  outcomeDroppedFmt: (count: number) => string
  outcomeTotalTakenFmt: (count: number) => string
  /**
   * LE BADGE « TEMPS FORT » (`features/match-view/equipmentKillBadges.ts`) : « N frags
   * sous camouflage » / « N frags sous surbouclier », le nombre RÉEL du meilleur épisode du
   * match — jamais arrondi, jamais une somme sur le joueur. Vit ici (et pas dans
   * `match-view/i18n`) pour la même raison que le reste du vocabulaire d'équipement : les
   * noms de famille appartiennent au dictionnaire du rejeu, `match-view` l'importe (même sens
   * que `MatchEquipmentUsageSection`, réutilisé tel quel par `MatchViewTabChronology`).
   */
  killBadgeFmt: Record<'camo' | 'overshield', (kills: number) => string>
  /** Infobulle du badge : la réserve de l'état actif (source non distinguée, bornes
   * à la précision de la retransmission, camo seul sous le seuil de mesure en lecture large). */
  killBadgeHint: string
}

export const FR_EQUIPMENT_USAGE = {
  title: "Usages d'équipement",
  viewByPlayer: "Usage d'équipements, par joueur",
  infoByPlayer:
    "Objets pris par chaque joueur du match, une pile par issue : servi (activé pour un bonus, posé pour le mur, charge consommée pour le reste), gardé sans servir, lâché en mourant ; le grappin compte ses tractions. Chaque colonne a sa propre échelle.",
  legendUsed: 'Servi',
  legendKept: 'Gardé sans servir',
  legendDropped: 'Lâché',
  legendGrapple: 'Tractions de grappin',
  gridTipFmt: (player, column, value) => `${player} — ${column} : ${value}`,
  groupGrapple: 'Grappin',
  outcomeUsedFmt: (count) => `Servi : ${count}`,
  outcomeKeptFmt: (count) => `Gardé sans servir : ${count}`,
  outcomeDroppedFmt: (count) => `Lâché : ${count}`,
  outcomeTotalTakenFmt: (count) => `${count} objet${count > 1 ? 's' : ''} pris`,
  killBadgeFmt: {
    camo: (kills) => `${kills} frags sous camouflage`,
    overshield: (kills) => `${kills} frags sous surbouclier`,
  },
  killBadgeHint:
    "Le meilleur épisode du match pour cette famille. Même réserve que les états actifs : la source de l'épisode n'est pas distinguée (ramassage ou capacité déclenchée), ses bornes sont à la précision de la retransmission près, et le camo seul reste sous le seuil de mesure en lecture large.",
} satisfies EquipmentUsageText

export const EN_EQUIPMENT_USAGE = {
  title: 'Equipment usage',
  viewByPlayer: 'Equipment use, by player',
  infoByPlayer:
    'Objects taken by each player of the match, one stack per outcome: used (activated for a power-up, placed for the drop wall, charge spent for the rest), kept without use, dropped on death; the grappleshot counts its pulls. Each column has its own scale.',
  legendUsed: 'Used',
  legendKept: 'Kept without use',
  legendDropped: 'Dropped',
  legendGrapple: 'Grapple pulls',
  gridTipFmt: (player, column, value) => `${player} — ${column}: ${value}`,
  groupGrapple: 'Grappleshot',
  outcomeUsedFmt: (count) => `Used: ${count}`,
  outcomeKeptFmt: (count) => `Kept without use: ${count}`,
  outcomeDroppedFmt: (count) => `Dropped: ${count}`,
  outcomeTotalTakenFmt: (count) => `${count} object${count > 1 ? 's' : ''} taken`,
  killBadgeFmt: {
    camo: (kills) => `${kills} kills under camo`,
    overshield: (kills) => `${kills} kills under overshield`,
  },
  killBadgeHint:
    "The best episode of the match for this family. Same reserve as active states: the episode's source is not told apart (picked up or triggered), its bounds are at broadcast precision, and camo alone stays under the measurement threshold in broad reading.",
} satisfies EquipmentUsageText
