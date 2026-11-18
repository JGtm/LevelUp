/**
 * playerPanel.i18n.ts — LES TEXTES DU PANNEAU DE JOUEUR (lot 4 du plan « maquettes du rejeu ») :
 * le contrat (`PlayerPanelText`) et ses deux langues. Un clic sur une fiche ouvre la fiche du joueur :
 * une fenêtre au Cinéma et au Plein cadre, la colonne entière au Standard.
 *
 * UN SOUS-DICTIONNAIRE À PART : `fr.ts` et `en.ts` sont au plafond de taille du dépôt, et ces textes
 * forment un tout (le panneau). Ils sont servis par `ReplayText.playerPanel`, donc par
 * `REPLAY_TEXT[locale].playerPanel` : la parité FR/EN reste tenue par le typage (`satisfies
 * PlayerPanelText` dans chaque langue, `satisfies ReplayText` au-dessus).
 */
import type { PlayerPanelAction } from '../model/playerPanelLogic'

export interface PlayerPanelText {
  /** La croix de la fiche en fenêtre (Cinéma, Plein cadre) : son nom accessible et son infobulle. */
  close: string
  /** Le nom accessible du panneau d'un joueur. */
  labelFmt: (name: string) => string
  /** Le nom accessible d'une fiche devenue bouton : ce que le clic ouvre. */
  cardOpenFmt: (name: string) => string
  /** Les trois actions sur la carte, et leur infobulle (Suivre dit que la main l'arrête). */
  action: Record<PlayerPanelAction, string>
  actionHint: Record<PlayerPanelAction, string>
  /** La fiche du cadre Standard : le retour à la colonne des équipes (et du fil), le titre de l'extrait du fil et son lien. */
  backToTeams: string
  extractTitle: string
  fullFeed: string
}

export const FR_PLAYER_PANEL = {
  close: 'Fermer',
  labelFmt: (name) => `Fiche de ${name}`,
  cardOpenFmt: (name) => `Ouvrir la fiche de ${name}`,
  action: { follow: 'Suivre', solo: 'Seul', hide: 'Masquer' },
  actionHint: {
    follow: "Cadre la carte sur ce joueur, image par image ; un zoom ou un déplacement à la main l'arrête",
    solo: "N'affiche que ce joueur sur la carte",
    hide: 'Retire ce joueur de la carte',
  },
  backToTeams: 'Équipes',
  extractTitle: 'Éliminations',
  fullFeed: 'Fil complet',
} satisfies PlayerPanelText

export const EN_PLAYER_PANEL = {
  close: 'Close',
  labelFmt: (name) => `${name}'s card`,
  cardOpenFmt: (name) => `Open ${name}'s card`,
  action: { follow: 'Follow', solo: 'Solo', hide: 'Hide' },
  actionHint: {
    follow: 'Frames the map on this player, frame by frame; zooming or panning by hand stops it',
    solo: 'Shows only this player on the map',
    hide: 'Removes this player from the map',
  },
  backToTeams: 'Teams',
  extractTitle: 'Kills',
  fullFeed: 'Full feed',
} satisfies PlayerPanelText
