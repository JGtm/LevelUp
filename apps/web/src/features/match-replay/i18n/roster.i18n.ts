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
  /** La mention à côté du nom du joueur suivi (le joueur regardé), sur sa ligne. */
  followed: string
  /** L'en-tête des colonnes de la ligne d'un joueur : texte court et nom entier (infobulle). */
  colWeapon: string
  colWeaponTitle: string
  colGrenades: string
  colGrenadesTitle: string
  colEquipment: string
  colEquipmentTitle: string
  colKda: string
  colKdaTitle: string
}

export const FR_ROSTER = {
  changePlayer: 'Changer de joueur',
  cameraFollowFmt: (name) => `Caméra : suivre ${name}`,
  followed: 'suivi',
  colWeapon: 'Arme',
  colWeaponTitle: 'Arme en main',
  colGrenades: 'Gr.',
  colGrenadesTitle: 'Grenades',
  colEquipment: 'Éq.',
  colEquipmentTitle: 'Équipement',
  colKda: 'F / M / A',
  colKdaTitle: 'Frags / Morts / Assistances',
} satisfies RosterText

export const EN_ROSTER = {
  changePlayer: 'Change player',
  cameraFollowFmt: (name) => `Camera: follow ${name}`,
  followed: 'followed',
  colWeapon: 'Weapon',
  colWeaponTitle: 'Weapon in hand',
  colGrenades: 'Gr.',
  colGrenadesTitle: 'Grenades',
  colEquipment: 'Eq.',
  colEquipmentTitle: 'Equipment',
  colKda: 'K / D / A',
  colKdaTitle: 'Kills / Deaths / Assists',
} satisfies RosterText
