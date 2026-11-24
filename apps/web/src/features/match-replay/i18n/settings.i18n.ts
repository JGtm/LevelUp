/**
 * settings.i18n.ts — LES TEXTES DU PANNEAU « VUE » (rubriques, préréglages souples, Mes vues, accès rapide, pied du
 * panneau) : le contrat (`SettingsText`) et ses deux langues. Les libellés et infobulles des réglages eux-mêmes
 * restent dans leurs dictionnaires d'origine (`layerAim`, `lives`, `relief`…) : `settingsText.ts` les range sous
 * les identifiants de la table des réglages. Servi par `REPLAY_TEXT[locale].settings` ; la parité FR/EN est tenue
 * par le typage (`satisfies SettingsText` dans chaque langue).
 */
import type { MatchModeKey } from '../settings/settingsAvailability'
import type { PresetId } from '../settings/viewPresets'
import type { RubricId, SubGroupId } from '../settings/settingsTable'

export interface SettingsText {
  /** Les rubriques : leur nom (onglet, titre) et la phrase qui dit ce qu'elles contiennent. */
  rubrics: Record<RubricId, { label: string; hint: string }>
  rubricsAria: string
  /** Les sous-titres d'une rubrique. */
  subs: Record<SubGroupId, string>
  /** « N sur M » d'une rubrique : combien de ses interrupteurs affichés sont allumés. */
  countFmt: (on: number, total: number) => string
  /** Le nom accessible de l'interrupteur maître d'une rubrique. */
  masterAriaFmt: (rubric: string) => string
  /** Les préréglages : leur nom et ce qu'ils montrent. */
  presetsTitle: string
  presets: Record<PresetId, { label: string; hint: string }>
  /** Le suffixe d'une vue dont un réglage a changé (« Combat · modifié »), et combien ont changé. */
  modified: string
  changedFmt: (n: number) => string
  revertFmt: (name: string) => string
  saveMine: string
  updateView: string
  startingPoint: string
  myViewState: string
  /** L'enregistrement d'une vue : son champ de nom, ses gestes, et la limite. */
  viewNameLabel: string
  save: string
  cancel: string
  viewLimit: string
  /** « Mes vues » : le gestionnaire (en-tête, vide, une vue, la vue d'ouverture, le mode). */
  myViewsFmt: (count: number, max: number) => string
  noViews: string
  renameAria: string
  defaultViewHint: string
  defaultViewAriaFmt: (name: string) => string
  deleteHint: string
  deleteAriaFmt: (name: string) => string
  modeLabel: string
  modeNone: string
  modes: Record<MatchModeKey, string>
  modeAriaFmt: (name: string) => string
  optional: string
  openingLabel: string
  openingAria: string
  openingLast: string
  openingNote: string
  /** L'accès rapide : épingler, retirer, le groupe, la raison d'un réglage grisé. */
  pinAddFmt: (label: string) => string
  pinRemoveFmt: (label: string) => string
  pinAddHint: string
  pinRemoveHint: string
  quickAria: string
  quickOnFmt: (label: string) => string
  quickOffFmt: (label: string) => string
  quickNoMatch: string
  quickNoFrame: string
  /** L'infobulle d'un réglage épinglé grisé : son nom, puis la raison. */
  quickDisabledFmt: (label: string, reason: string) => string
  /** Les petites phrases de la rubrique « Fil ». */
  feedFilterLabel: string
  feedStandardNote: string
  /** Les libellés que les dictionnaires d'origine ne portaient pas. */
  volumeLabel: string
  colorsHint: string
  backgroundHint: string
  /** Le résultat d'un geste, dit sous le panneau (région d'état). */
  statusSavedFmt: (name: string) => string
  statusUpdatedFmt: (name: string) => string
  statusDeletedFmt: (name: string) => string
  statusRevertedFmt: (name: string) => string
  statusPinFull: string
  statusReset: string
  /** Le pied du panneau. */
  readingTitle: string
  reset: string
  resetAsk: string
  resetConfirm: string
  /** Le bouton « Vue » : la vue active, et son état modifié. */
  buttonModifiedAria: string
}

export const FR_SETTINGS = {
  rubrics: {
    joueurs: { label: 'Joueurs', hint: 'Ce que montre chaque joueur : regard, trajet, effets, vies.' },
    carte: { label: 'Carte', hint: 'Le terrain : fond, relief, étage, orientation, zones, chaleur.' },
    objets: { label: 'Objets', hint: 'Armes, véhicules, équipements posés et objectifs du mode.' },
    fil: { label: 'Fil', hint: 'Le fil des événements du match.' },
    son: { label: 'Son', hint: 'Sons, volume et catégories.' },
  },
  rubricsAria: 'Rubriques',
  subs: { vies: 'Vies', categories: 'Catégories' },
  countFmt: (on, total) => `${on} sur ${total}`,
  masterAriaFmt: (rubric) => `Tout activer ou désactiver : ${rubric}`,
  presetsTitle: 'Préréglages',
  presets: {
    complet: { label: 'Complet', hint: 'Tout ce qui se dessine ou se dit, sauf les traces de vies et la chaleur' },
    combat: { label: 'Combat', hint: 'Regard, tirs, véhicules, objectifs, morts et tueurs' },
    deplacements: { label: 'Déplacements', hint: 'Trajets, apparitions, zones et chaleur' },
    objectif: { label: 'Objectif', hint: 'Trajets, zones et objectifs' },
  },
  modified: 'modifié',
  changedFmt: (n) => (n === 1 ? '1 réglage changé' : `${n} réglages changés`),
  revertFmt: (name) => `Revenir à ${name}`,
  saveMine: 'Enregistrer comme ma vue',
  updateView: 'Mettre à jour cette vue',
  startingPoint: 'Un préréglage est un point de départ : changez ce que vous voulez.',
  myViewState: 'Ma vue enregistrée',
  viewNameLabel: 'Nom de la vue',
  save: 'Enregistrer',
  cancel: 'Annuler',
  viewLimit: '3 vues au plus. Supprimez ou mettez à jour une vue de « Mes vues » d’abord.',
  myViewsFmt: (count, max) => `Mes vues (${count} sur ${max})`,
  noViews: 'Aucune vue enregistrée. Réglez le panneau, puis « Enregistrer comme ma vue ». Vous pouvez en garder trois.',
  renameAria: 'Nom de la vue (renommer)',
  defaultViewHint: 'Vue d’ouverture par défaut',
  defaultViewAriaFmt: (name) => `Vue d’ouverture par défaut : ${name}`,
  deleteHint: 'Supprimer',
  deleteAriaFmt: (name) => `Supprimer la vue ${name}`,
  modeLabel: 'Utiliser pour les matchs',
  modeNone: 'Aucun',
  modes: { flag: 'à drapeau', skull: 'à crâne', vip: 'à VIP', bomb: 'à bombe', zones: 'à zones', open: 'sans objectif' },
  modeAriaFmt: (name) => `Rattacher « ${name} » à un mode de jeu`,
  optional: 'facultatif',
  openingLabel: 'À l’ouverture d’un rejeu',
  openingAria: 'Vue appliquée à l’ouverture d’un rejeu',
  openingLast: 'Dernière vue utilisée',
  openingNote:
    'Une vue rattachée à un mode passe avant la vue d’ouverture, pour ce mode seulement. Sans choix, vos réglages restent tels que vous les avez laissés.',
  pinAddFmt: (label) => `Épingler « ${label} » à l’accès rapide`,
  pinRemoveFmt: (label) => `Retirer « ${label} » de l’accès rapide`,
  pinAddHint: 'Épingler à l’accès rapide (5 au plus)',
  pinRemoveHint: 'Retirer de l’accès rapide',
  quickAria: 'Accès rapide aux réglages',
  quickOnFmt: (label) => `${label} : activé`,
  quickOffFmt: (label) => `${label} : désactivé`,
  quickNoMatch: 'Sans objet sur ce match',
  quickNoFrame: 'Disponible au Cinéma et au Plein cadre',
  quickDisabledFmt: (label, reason) => `${label} : ${reason}`,
  feedFilterLabel: 'Filtre du fil',
  feedStandardNote: 'Le fil posé sur le plan se règle au Cinéma et au Plein cadre.',
  volumeLabel: 'Volume',
  colorsHint: 'Couleur du camp, ou couleur stable par joueur.',
  backgroundHint: 'L’image sous les joueurs. Retenue pour toutes les cartes.',
  statusSavedFmt: (name) => `Vue « ${name} » enregistrée.`,
  statusUpdatedFmt: (name) => `Vue « ${name} » mise à jour.`,
  statusDeletedFmt: (name) => `Vue « ${name} » supprimée.`,
  statusRevertedFmt: (name) => `Vue rétablie : ${name}`,
  statusPinFull: '5 réglages au plus dans l’accès rapide : retirez-en un d’abord.',
  statusReset: 'Réglages d’usine rétablis. Vos vues et vos épingles sont conservées.',
  readingTitle: 'Lecture',
  reset: 'Réinitialiser',
  resetAsk: 'Remettre les réglages d’usine ?',
  resetConfirm: 'Confirmer',
  buttonModifiedAria: 'modifié',
} satisfies SettingsText

export const EN_SETTINGS = {
  rubrics: {
    joueurs: { label: 'Players', hint: 'What each player shows: gaze, path, effects, lives.' },
    carte: { label: 'Map', hint: 'The terrain: background, relief, floor, orientation, zones, heat.' },
    objets: { label: 'Objects', hint: 'Weapons, vehicles, placed equipment and mode objectives.' },
    fil: { label: 'Feed', hint: 'The match event feed.' },
    son: { label: 'Sound', hint: 'Sounds, volume and categories.' },
  },
  rubricsAria: 'Sections',
  subs: { vies: 'Lives', categories: 'Categories' },
  countFmt: (on, total) => `${on} of ${total}`,
  masterAriaFmt: (rubric) => `Turn all on or off: ${rubric}`,
  presetsTitle: 'Presets',
  presets: {
    complet: { label: 'Full', hint: 'Everything that is drawn or said, except life traces and heat' },
    combat: { label: 'Combat', hint: 'Aim, shots, vehicles, objectives, deaths and killers' },
    deplacements: { label: 'Movement', hint: 'Paths, spawns, zones and heat' },
    objectif: { label: 'Objective', hint: 'Paths, zones and objectives' },
  },
  modified: 'modified',
  changedFmt: (n) => (n === 1 ? '1 setting changed' : `${n} settings changed`),
  revertFmt: (name) => `Back to ${name}`,
  saveMine: 'Save as my view',
  updateView: 'Update this view',
  startingPoint: 'A preset is a starting point: change whatever you like.',
  myViewState: 'My saved view',
  viewNameLabel: 'View name',
  save: 'Save',
  cancel: 'Cancel',
  viewLimit: '3 views at most. Delete or update a view in “My views” first.',
  myViewsFmt: (count, max) => `My views (${count} of ${max})`,
  noViews: 'No saved view. Adjust the panel, then “Save as my view”. You can keep three.',
  renameAria: 'View name (rename)',
  defaultViewHint: 'Default opening view',
  defaultViewAriaFmt: (name) => `Default opening view: ${name}`,
  deleteHint: 'Delete',
  deleteAriaFmt: (name) => `Delete the view ${name}`,
  modeLabel: 'Use for matches',
  modeNone: 'None',
  modes: { flag: 'with flags', skull: 'with a skull', vip: 'with a VIP', bomb: 'with a bomb', zones: 'with zones', open: 'without objectives' },
  modeAriaFmt: (name) => `Attach “${name}” to a game mode`,
  optional: 'optional',
  openingLabel: 'When a replay opens',
  openingAria: 'View applied when a replay opens',
  openingLast: 'Last view used',
  openingNote:
    'A view attached to a mode comes before the opening view, for that mode only. With no choice, your settings stay as you left them.',
  pinAddFmt: (label) => `Pin “${label}” to quick access`,
  pinRemoveFmt: (label) => `Remove “${label}” from quick access`,
  pinAddHint: 'Pin to quick access (5 at most)',
  pinRemoveHint: 'Remove from quick access',
  quickAria: 'Quick access to settings',
  quickOnFmt: (label) => `${label}: on`,
  quickOffFmt: (label) => `${label}: off`,
  quickNoMatch: 'Not applicable to this match',
  quickNoFrame: 'Available in Cinema and Full frame',
  quickDisabledFmt: (label, reason) => `${label}: ${reason}`,
  feedFilterLabel: 'Feed filter',
  feedStandardNote: 'The feed laid over the map is set in Cinema and Full frame.',
  volumeLabel: 'Volume',
  colorsHint: 'The team’s colour, or a steady colour per player.',
  backgroundHint: 'The image under the players. Kept for every map.',
  statusSavedFmt: (name) => `View “${name}” saved.`,
  statusUpdatedFmt: (name) => `View “${name}” updated.`,
  statusDeletedFmt: (name) => `View “${name}” deleted.`,
  statusRevertedFmt: (name) => `View restored: ${name}`,
  statusPinFull: '5 settings at most in quick access: remove one first.',
  statusReset: 'Factory settings restored. Your views and pins are kept.',
  readingTitle: 'Playback',
  reset: 'Reset',
  resetAsk: 'Restore the factory settings?',
  resetConfirm: 'Confirm',
  buttonModifiedAria: 'modified',
} satisfies SettingsText
