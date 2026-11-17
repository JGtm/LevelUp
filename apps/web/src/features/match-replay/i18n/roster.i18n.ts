/**
 * roster.i18n.ts — LES TEXTES DES JOUEURS DANS LA COLONNE ET LA RANGÉE (lignes, tuiles, sélecteur de
 * la fiche) : le contrat (`RosterText`) et ses deux langues.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de
 * taille du dépôt. Servi par `REPLAY_TEXT[locale].roster` ; la parité FR/EN reste tenue par le typage
 * (`satisfies RosterText` dans chaque langue, `satisfies ReplayText` au-dessus).
 */

export interface RosterText {
  /** Le sélecteur de la fiche ouverte : passer à la fiche d'un autre joueur sans revenir au fil. */
  changePlayer: string
  /** Le bouton de caméra d'un joueur : la carte se cadre sur lui à chaque image. */
  cameraFollowFmt: (name: string) => string
}

export const FR_ROSTER = {
  changePlayer: 'Changer de joueur',
  cameraFollowFmt: (name) => `Caméra : suivre ${name}`,
} satisfies RosterText

export const EN_ROSTER = {
  changePlayer: 'Change player',
  cameraFollowFmt: (name) => `Camera: follow ${name}`,
} satisfies RosterText
