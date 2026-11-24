/**
 * playerPanel.i18n.ts — LES TEXTES DU PANNEAU DE JOUEUR (lot 4 du plan « maquettes du rejeu »)
 * et ceux des actions des tuiles : le contrat (`PlayerPanelText`) et ses deux langues. Un clic sur une fiche ouvre
 * la fiche du joueur : une fenêtre au Cinéma et au Plein cadre, la colonne entière au Standard. Suivre et l'œil
 * (Masquer / Afficher) sont sur les tuiles et les lignes ; « Seul » est dans la fiche.
 *
 * UN SOUS-DICTIONNAIRE À PART : `fr.ts` et `en.ts` sont au plafond de taille du dépôt, et ces textes
 * forment un tout (le panneau). Ils sont servis par `ReplayText.playerPanel`, donc par
 * `REPLAY_TEXT[locale].playerPanel` : la parité FR/EN reste tenue par le typage (`satisfies
 * PlayerPanelText` dans chaque langue, `satisfies ReplayText` au-dessus).
 */

/** Les actions d'une tuile ou d'une ligne : Suivre, l'œil, et « Tout afficher » en tête de la colonne. */
interface PlayerTileText {
  /** Le titre de l'infobulle de chaque geste ; il sert aussi d'intitulé de la liste « Caméra : suivre » du menu de caméra. */
  follow: string
  unfollow: string
  hide: string
  show: string
  /** Le nom accessible de chaque bouton, avec le nom du joueur. */
  followLabelFmt: (name: string) => string
  unfollowLabelFmt: (name: string) => string
  hideLabelFmt: (name: string) => string
  showLabelFmt: (name: string) => string
  /** La phrase de l'infobulle, sous son titre. */
  followHintFmt: (name: string) => string
  unfollowHint: string
  hideHintFmt: (name: string) => string
  showHintFmt: (name: string) => string
  /** La mention écrite sur la tuile ou la ligne d'un joueur masqué. */
  hiddenMention: string
  /** « Tout afficher », en tête de la colonne et de la rangée des équipes quand un joueur est masqué. */
  showAll: string
}

/** « Seul » de la fiche : son libellé et son infobulle, au repos et actif. */
interface PlayerSoloText {
  label: string
  labelActive: string
  hint: string
  hintActive: string
}

export interface PlayerPanelText {
  /** La croix de la fiche en fenêtre (Cinéma, Plein cadre) : son nom accessible et son infobulle. */
  close: string
  /** Le nom accessible du panneau d'un joueur. */
  labelFmt: (name: string) => string
  /** Le nom accessible d'une fiche devenue bouton : ce que le clic ouvre. */
  cardOpenFmt: (name: string) => string
  /** Les actions des tuiles et des lignes. */
  tile: PlayerTileText
  /** « Seul », dans la fiche. */
  solo: PlayerSoloText
  /** « Prendre le point de vue », dans la fiche : le geste du menu de la frise, posé sur le joueur ouvert. */
  viewpoint: { label: string; hint: string }
  /** La fiche du cadre Standard : le retour à la colonne des équipes (et du fil), le titre de l'extrait du fil et son lien. */
  backToTeams: string
  extractTitle: string
  fullFeed: string
}

export const FR_PLAYER_PANEL = {
  close: 'Fermer',
  labelFmt: (name) => `Fiche de ${name}`,
  cardOpenFmt: (name) => `Ouvrir la fiche de ${name}`,
  tile: {
    follow: 'Caméra : suivre',
    unfollow: 'Ne plus suivre',
    hide: 'Masquer',
    show: 'Afficher',
    followLabelFmt: (name) => `Suivre ${name} (caméra)`,
    unfollowLabelFmt: (name) => `Ne plus suivre ${name} (caméra)`,
    hideLabelFmt: (name) => `Masquer ${name}`,
    showLabelFmt: (name) => `Afficher ${name}`,
    followHintFmt: (name) => `La caméra cadre la carte sur ${name}.`,
    unfollowHint: 'La carte revient en vue entière.',
    hideHintFmt: (name) => `Retire ${name} de la carte : pion, traînée, nom.`,
    showHintFmt: (name) => `${name} réapparaît sur la carte.`,
    hiddenMention: 'masqué',
    showAll: 'Tout afficher',
  },
  solo: {
    label: 'Seul',
    labelActive: 'Seul · tout réafficher',
    hint: "Masque tous les autres joueurs. L'œil d'une tuile en réaffiche un.",
    hintActive: 'Réaffiche tous les joueurs.',
  },
  viewpoint: {
    label: 'Prendre le point de vue',
    hint: 'La carte, la frise et les camps se lisent par les yeux de ce joueur. La lecture ne bouge pas.',
  },
  backToTeams: 'Équipes',
  extractTitle: 'Éliminations',
  fullFeed: 'Fil complet',
} satisfies PlayerPanelText

export const EN_PLAYER_PANEL = {
  close: 'Close',
  labelFmt: (name) => `${name}'s card`,
  cardOpenFmt: (name) => `Open ${name}'s card`,
  tile: {
    follow: 'Camera: follow',
    unfollow: 'Stop following',
    hide: 'Hide',
    show: 'Show',
    followLabelFmt: (name) => `Follow ${name} (camera)`,
    unfollowLabelFmt: (name) => `Stop following ${name} (camera)`,
    hideLabelFmt: (name) => `Hide ${name}`,
    showLabelFmt: (name) => `Show ${name}`,
    followHintFmt: (name) => `The camera frames the map on ${name}.`,
    unfollowHint: 'The map goes back to the full view.',
    hideHintFmt: (name) => `Removes ${name} from the map: marker, trail, name.`,
    showHintFmt: (name) => `${name} reappears on the map.`,
    hiddenMention: 'hidden',
    showAll: 'Show all',
  },
  solo: {
    label: 'Solo',
    labelActive: 'Solo · show all',
    hint: "Hides every other player. A tile's eye shows one again.",
    hintActive: 'Shows every player again.',
  },
  viewpoint: {
    label: 'Take the viewpoint',
    hint: 'The map, the timeline and the camps read through this player’s eyes. Playback does not move.',
  },
  backToTeams: 'Teams',
  extractTitle: 'Kills',
  fullFeed: 'Full feed',
} satisfies PlayerPanelText
