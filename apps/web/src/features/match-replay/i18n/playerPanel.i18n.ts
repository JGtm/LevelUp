/**
 * playerPanel.i18n.ts — LES TEXTES DU PANNEAU DE JOUEUR (lot 4 du plan « maquettes du rejeu ») :
 * le contrat (`PlayerPanelText`) et ses deux langues. Un clic sur une fiche ouvre un panneau à la
 * place du fil ; « Retour au fil » le referme.
 *
 * UN SOUS-DICTIONNAIRE À PART : `fr.ts` et `en.ts` sont au plafond de taille du dépôt, et ces textes
 * forment un tout (le panneau). Ils sont servis par `ReplayText.playerPanel`, donc par
 * `REPLAY_TEXT[locale].playerPanel` : la parité FR/EN reste tenue par le typage (`satisfies
 * PlayerPanelText` dans chaque langue, `satisfies ReplayText` au-dessus).
 *
 * LES TITRES DISENT « SUR LE MATCH » : précision, dégâts, détail du score et armes sont des TOTAUX
 * DE FIN DE MATCH lus sur la feuille — le film ne publie pas la précision en cours de partie. Les
 * compteurs frags / morts / assistances de la tête, eux, sont ceux de l'instant, comme la fiche.
 */
import type { PanelRowKey, PanelSectionId, PlayerPanelAction } from '../model/playerPanelLogic'

export interface PlayerPanelText {
  /** « Retour au fil » : referme le panneau et rend sa place au fil, filtre et défilement compris. */
  back: string
  /** Le nom accessible du panneau d'un joueur. */
  labelFmt: (name: string) => string
  /** Le nom accessible d'une fiche devenue bouton : ce que le clic ouvre. */
  cardOpenFmt: (name: string) => string
  /**
   * Ce qui date les compteurs de la tête : l'horloge du match (compteurs de l'INSTANT, comme la
   * fiche), ou « totaux du match » quand le film ne les publie pas pour ce joueur et que la fiche
   * retombe sur la feuille.
   */
  atFmt: (clock: string) => string
  totals: string
  /** Les trois actions sur la carte, et leur infobulle (Suivre dit que la main l'arrête). */
  action: Record<PlayerPanelAction, string>
  actionHint: Record<PlayerPanelAction, string>
  /** Les titres des listes : toujours « sur le match ». */
  section: Record<PanelSectionId, string>
  /** Le libellé de chaque ligne ; les lignes d'objectif suivent les clés de la feuille. */
  row: Record<PanelRowKey, string>
  /** Joueur sans ligne de feuille : le panneau le dit au lieu d'afficher des zéros. */
  noSheet: string
  noSheetHint: string
  /** Une arme de la feuille hors référentiel : son numéro, jamais le nom d'une autre. */
  weaponFmt: (weaponId: number) => string
}

export const FR_PLAYER_PANEL = {
  back: 'Retour au fil',
  labelFmt: (name) => `Fiche de ${name}`,
  cardOpenFmt: (name) => `Ouvrir la fiche de ${name}`,
  atFmt: (clock) => `à ${clock}`,
  totals: 'totaux du match',
  action: { follow: 'Suivre', solo: 'Seul', hide: 'Masquer' },
  actionHint: {
    follow: "Cadre la carte sur ce joueur, image par image ; un zoom ou un déplacement à la main l'arrête",
    solo: "N'affiche que ce joueur sur la carte",
    hide: 'Retire ce joueur de la carte',
  },
  section: {
    precision: 'Précision sur le match',
    damage: 'Dégâts sur le match',
    score: 'Détail du score sur le match',
    objective: 'Objectif sur le match',
    weapons: 'Éliminations par arme sur le match',
  },
  row: {
    precision: 'Précision',
    shotsHit: 'Touches',
    shotsFired: 'Tirs',
    damageDealt: 'Infligés',
    damageTaken: 'Subis',
    score: 'Score',
    headshotKills: 'Éliminations à la tête',
    powerWeaponKills: 'Éliminations aux armes de puissance',
    meleeKills: 'Éliminations au corps à corps',
    grenadeKills: 'Éliminations à la grenade',
    flag_captures: 'Captures de drapeau',
    flag_returns: 'Retours de drapeau',
    flag_grabs: 'Drapeaux saisis',
    flag_carriers_killed: 'Porteurs de drapeau éliminés',
    flag_capture_assists: 'Assistances de capture',
    zone_captures: 'Captures de zone',
    zone_offensive_kills: 'Éliminations en attaque de zone',
    zone_defensive_kills: 'Éliminations en défense de zone',
    bomb_grabs: 'Bombes saisies',
    bomb_arms: 'Bombes armées',
    bomb_detonations: 'Bombes explosées',
    bomb_carriers_killed: 'Porteurs de bombe éliminés',
    skull_grabs: 'Crânes saisis',
    skull_carriers_killed: 'Porteurs de crâne éliminés',
  },
  noSheet: 'Pas de statistiques de match pour ce joueur',
  noSheetHint: "La feuille de score du match n'a pas de ligne pour lui : rien n'est affiché plutôt qu'un zéro.",
  weaponFmt: (weaponId) => `Arme n° ${weaponId}`,
} satisfies PlayerPanelText

export const EN_PLAYER_PANEL = {
  back: 'Back to the feed',
  labelFmt: (name) => `${name}'s card`,
  cardOpenFmt: (name) => `Open ${name}'s card`,
  atFmt: (clock) => `at ${clock}`,
  totals: 'match totals',
  action: { follow: 'Follow', solo: 'Solo', hide: 'Hide' },
  actionHint: {
    follow: 'Frames the map on this player, frame by frame; zooming or panning by hand stops it',
    solo: 'Shows only this player on the map',
    hide: 'Removes this player from the map',
  },
  section: {
    precision: 'Accuracy over the match',
    damage: 'Damage over the match',
    score: 'Score breakdown over the match',
    objective: 'Objective over the match',
    weapons: 'Kills by weapon over the match',
  },
  row: {
    precision: 'Accuracy',
    shotsHit: 'Shots hit',
    shotsFired: 'Shots fired',
    damageDealt: 'Dealt',
    damageTaken: 'Taken',
    score: 'Score',
    headshotKills: 'Headshot kills',
    powerWeaponKills: 'Power weapon kills',
    meleeKills: 'Melee kills',
    grenadeKills: 'Grenade kills',
    flag_captures: 'Flag captures',
    flag_returns: 'Flag returns',
    flag_grabs: 'Flag grabs',
    flag_carriers_killed: 'Flag carriers killed',
    flag_capture_assists: 'Capture assists',
    zone_captures: 'Zone captures',
    zone_offensive_kills: 'Zone offensive kills',
    zone_defensive_kills: 'Zone defensive kills',
    bomb_grabs: 'Bomb grabs',
    bomb_arms: 'Bombs armed',
    bomb_detonations: 'Bomb detonations',
    bomb_carriers_killed: 'Bomb carriers killed',
    skull_grabs: 'Skull grabs',
    skull_carriers_killed: 'Skull carriers killed',
  },
  noSheet: 'No match statistics for this player',
  noSheetHint: 'The match scoreboard has no row for them: nothing is shown rather than a zero.',
  weaponFmt: (weaponId) => `Weapon #${weaponId}`,
} satisfies PlayerPanelText
