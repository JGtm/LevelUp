/**
 * i18n des modals média (MediaMatchPicker, CoverFlowModal).
 * Séparé du i18n.ts principal pour respecter SRP.
 */

import type { Locale } from '@/lib/i18n/locale'

export interface MatchPickerText {
  title: string
  /** Variante quand le média n'a PAS encore de match associé (hasCurrentMatch=false). */
  titleAssociate: string
  capturePrefix: string
  closeAriaLabel: string
  windowLabel: string
  matchesFound: (count: number) => string
  loading: string
  error: string
  noMatchesFound: string
  lobbyUnavailable: string
  currentBadge: string
  confirmTitle: string
  /** Variante « Confirmer l'association ? » (hasCurrentMatch=false). */
  confirmTitleAssociate: string
  cancel: string
  confirm: string
  applying: string
  deltaUnder1Min: string
  deltaMinFormat: (m: number) => string
  spectators: string
  /** Libellé d'équipe — reçoit le nom officiel Halo (Eagle…) ou le numéro. */
  teamLabel: (n: string | number) => string
}

export interface CoverFlowText {
  reassociateButton: string
  reassociateTitle: string
  associateButton: string
  associateTitle: string
  viewMatchButton: string
  viewMatchTitle: string
  chainButton: string
  enableChaining: string
  disableChaining: string
  closeAriaLabel: string
  audioGame: string
  audioVoice: string
  audioGroupLabel: string
  enterFullscreen: string
  exitFullscreen: string
  /** Suppression définitive d'un média (item 3.1) — propriétaire ou admin. */
  deleteButton: string
  deleteTitle: string
  /** Infobulle du bouton sur le bureau : l'original part à la Corbeille. */
  deleteTitleTrash: string
  deleteConfirmTitle: string
  /** Corps de la confirmation : dit explicitement que c'est irréversible. */
  deleteConfirmBody: string
  /** Corps de la confirmation sur le bureau : l'original part à la Corbeille de Windows. */
  deleteConfirmBodyTrash: string
  deleteConfirmLabel: string
  /** Bouton de confirmation sur le bureau : l'original part à la Corbeille. */
  deleteConfirmLabelTrash: string
  deleteCancelLabel: string
  deleteSuccess: string
  /** Toast quand l'original est parti à la Corbeille du système. */
  deleteSuccessTrash: string
  /** Toast sur le bureau quand l'original est supprimé définitivement (disque sans Corbeille). */
  deleteSuccessPermanent: string
  deleteError: string
}

/** Fenêtre « Fichiers introuvables », bandeau du dossier et ouverture de l'explorateur. */
export interface MissingMediaText {
  toolbarAriaLabel: string
  title: string
  loading: string
  error: string
  empty: string
  capturePrefix: string
  findMedia: string
  forget: string
  back: string
  candidatesTitle: string
  candidatesEmpty: string
  confirmMergeTitle: string
  confirmMergeDetail: string
  confirmForgetTitle: string
  confirmForgetDetail: string
  cancel: string
  confirm: string
  applying: string
  mergeSuccess: string
  mergeError: string
  forgetSuccess: string
  forgetError: string
  rootUnavailable: string
  bulkPending: (count: number) => string
  bulkConfirm: string
  bulkConfirmError: string
  revealButton: string
  revealTitle: string
  revealError: string
  closeAriaLabel: string
  /** Écart de date de capture, en minutes (0 = moins d'une minute). */
  deltaFormat: (minutes: number) => string
}

export interface MediaModalsText {
  matchPicker: MatchPickerText
  coverFlow: CoverFlowText
  missing: MissingMediaText
}

const FR: MediaModalsText = {
  matchPicker: {
    title: 'Réassocier ce média',
    titleAssociate: 'Associer ce média',
    capturePrefix: 'Capture :',
    closeAriaLabel: 'Fermer',
    windowLabel: 'Fenêtre :',
    matchesFound: (count) => `${count} match${count > 1 ? 's' : ''} trouvé${count > 1 ? 's' : ''}`,
    loading: 'Chargement…',
    error: 'Erreur de chargement',
    noMatchesFound: 'Aucun match trouvé dans cette fenêtre. Élargis la recherche.',
    lobbyUnavailable: 'Lobby indisponible',
    currentBadge: 'actuel',
    confirmTitle: 'Confirmer la réassociation ?',
    confirmTitleAssociate: "Confirmer l'association ?",
    cancel: 'Annuler',
    confirm: 'Confirmer',
    applying: 'Application…',
    deltaUnder1Min: '< 1 min',
    deltaMinFormat: (m) => `±${m} min`,
    spectators: 'Spectateurs',
    teamLabel: (n) => `Équipe ${n}`,
  },
  coverFlow: {
    reassociateButton: 'Réassocier',
    reassociateTitle: 'Réassocier ce média à un autre match',
    associateButton: 'Associer',
    associateTitle: 'Associer ce média à un match',
    viewMatchButton: 'Voir le match →',
    viewMatchTitle: 'Ouvrir la page du match associé',
    chainButton: 'Enchaîner',
    enableChaining: 'Activer enchaînement',
    disableChaining: 'Désactiver enchaînement',
    closeAriaLabel: 'Fermer',
    audioGame: 'Jeu',
    audioVoice: 'Voix',
    audioGroupLabel: 'Pistes audio',
    enterFullscreen: 'Plein écran',
    exitFullscreen: 'Quitter le plein écran',
    deleteButton: 'Supprimer',
    deleteTitle: 'Supprimer définitivement ce média',
    deleteTitleTrash: 'Envoyer à la Corbeille',
    deleteConfirmTitle: 'Supprimer ce média ?',
    deleteConfirmBody:
      'Le fichier sera effacé du serveur et ne pourra pas être récupéré. '
      + 'Le média disparaîtra de la galerie et des pages de match.',
    deleteConfirmBodyTrash:
      "Le fichier d'origine sera envoyé à la Corbeille de Windows, d'où il pourra être restauré. "
      + 'Sa miniature et sa version convertie seront effacées. '
      + 'Sur un disque sans Corbeille (clé USB, disque réseau), le fichier sera supprimé définitivement.',
    deleteConfirmLabel: 'Supprimer définitivement',
    deleteConfirmLabelTrash: 'Envoyer à la Corbeille',
    deleteCancelLabel: 'Annuler',
    deleteSuccess: 'Média supprimé.',
    deleteSuccessTrash: 'Média envoyé à la Corbeille.',
    deleteSuccessPermanent: 'Média supprimé définitivement.',
    deleteError: 'Impossible de supprimer ce média.',
  },
  missing: {
    toolbarAriaLabel: 'Afficher les fichiers introuvables',
    title: 'Fichiers introuvables',
    loading: 'Chargement…',
    error: 'Erreur de chargement',
    empty: 'Aucun fichier introuvable.',
    capturePrefix: 'Capture :',
    findMedia: 'Retrouver dans la galerie',
    forget: 'Oublier',
    back: 'Retour',
    candidatesTitle: 'Médias présents dans la galerie',
    candidatesEmpty: 'Aucun média de même nature dans la galerie.',
    confirmMergeTitle: 'Fondre dans le média choisi ?',
    confirmMergeDetail:
      'Les marques « aimé » et le rattachement à un match passent au média choisi. '
      + "Le fichier d'origine n'est pas touché.",
    confirmForgetTitle: 'Oublier ce fichier ?',
    confirmForgetDetail:
      "Le média quitte la galerie, avec sa miniature et sa version convertie. Le fichier d'origine n'est pas touché.",
    cancel: 'Annuler',
    confirm: 'Confirmer',
    applying: 'Application…',
    mergeSuccess: 'Média fondu dans la fiche choisie.',
    mergeError: 'Impossible de fondre ce média.',
    forgetSuccess: 'Fichier oublié.',
    forgetError: "Impossible d'oublier ce fichier.",
    rootUnavailable:
      "Le dossier des captures est inaccessible : aucun média n'est marqué comme introuvable tant qu'il ne répond pas.",
    bulkPending: (count) =>
      `${count} média${count > 1 ? 's' : ''} ${count > 1 ? "n'ont" : "n'a"} plus de fichier dans le dossier des captures. `
      + 'Rien n\'est masqué tant que ce retrait n\'est pas confirmé.',
    bulkConfirm: 'Confirmer le retrait',
    bulkConfirmError: 'Impossible de confirmer le retrait.',
    revealButton: "Afficher dans l'explorateur",
    revealTitle: "Ouvrir l'explorateur de fichiers sur ce média",
    revealError: "Impossible d'ouvrir l'explorateur.",
    closeAriaLabel: 'Fermer',
    deltaFormat: (minutes) => (minutes < 1 ? '< 1 min' : `±${minutes} min`),
  },
}

const EN: MediaModalsText = {
  matchPicker: {
    title: 'Reassociate this media',
    titleAssociate: 'Associate this media',
    capturePrefix: 'Capture:',
    closeAriaLabel: 'Close',
    windowLabel: 'Window:',
    matchesFound: (count) => `${count} match${count > 1 ? 'es' : ''} found`,
    loading: 'Loading…',
    error: 'Loading error',
    noMatchesFound: 'No match found in this window. Broaden the search.',
    lobbyUnavailable: 'Lobby unavailable',
    currentBadge: 'current',
    confirmTitle: 'Confirm reassociation?',
    confirmTitleAssociate: 'Confirm association?',
    cancel: 'Cancel',
    confirm: 'Confirm',
    applying: 'Applying…',
    deltaUnder1Min: '< 1 min',
    deltaMinFormat: (m) => `±${m} min`,
    spectators: 'Spectators',
    teamLabel: (n) => `Team ${n}`,
  },
  coverFlow: {
    reassociateButton: 'Reassociate',
    reassociateTitle: 'Reassociate this media to another match',
    associateButton: 'Associate',
    associateTitle: 'Associate this media to a match',
    viewMatchButton: 'View match →',
    viewMatchTitle: 'Open the associated match page',
    chainButton: 'Chain',
    enableChaining: 'Enable chaining',
    disableChaining: 'Disable chaining',
    closeAriaLabel: 'Close',
    audioGame: 'Game',
    audioVoice: 'Voice',
    audioGroupLabel: 'Audio tracks',
    enterFullscreen: 'Fullscreen',
    exitFullscreen: 'Exit fullscreen',
    deleteButton: 'Delete',
    deleteTitle: 'Permanently delete this media',
    deleteTitleTrash: 'Move to Recycle Bin',
    deleteConfirmTitle: 'Delete this media?',
    deleteConfirmBody:
      'The file will be erased from the server and cannot be recovered. '
      + 'The media will disappear from the gallery and from match pages.',
    deleteConfirmBodyTrash:
      'The original file will be sent to the Windows Recycle Bin, from where it can be restored. '
      + 'Its thumbnail and converted version will be erased. '
      + 'On a drive without a Recycle Bin (USB stick, network drive), the file will be permanently deleted.',
    deleteConfirmLabel: 'Delete permanently',
    deleteConfirmLabelTrash: 'Move to Recycle Bin',
    deleteCancelLabel: 'Cancel',
    deleteSuccess: 'Media deleted.',
    deleteSuccessTrash: 'Media sent to the Recycle Bin.',
    deleteSuccessPermanent: 'Media permanently deleted.',
    deleteError: 'Could not delete this media.',
  },
  missing: {
    toolbarAriaLabel: 'Show missing files',
    title: 'Missing files',
    loading: 'Loading…',
    error: 'Loading error',
    empty: 'No missing file.',
    capturePrefix: 'Capture:',
    findMedia: 'Find in the gallery',
    forget: 'Forget',
    back: 'Back',
    candidatesTitle: 'Media in the gallery',
    candidatesEmpty: 'No media of the same kind in the gallery.',
    confirmMergeTitle: 'Merge into the chosen media?',
    confirmMergeDetail:
      'Likes and the match link move to the chosen media. The original file is not touched.',
    confirmForgetTitle: 'Forget this file?',
    confirmForgetDetail:
      'The media leaves the gallery, with its thumbnail and converted version. The original file is not touched.',
    cancel: 'Cancel',
    confirm: 'Confirm',
    applying: 'Applying…',
    mergeSuccess: 'Media merged into the chosen entry.',
    mergeError: 'Could not merge this media.',
    forgetSuccess: 'File forgotten.',
    forgetError: 'Could not forget this file.',
    rootUnavailable:
      'The captures folder is unreachable: no media is marked as missing while it does not respond.',
    bulkPending: (count) =>
      `${count} media ${count > 1 ? 'have' : 'has'} no file left in the captures folder. `
      + 'Nothing is hidden until this removal is confirmed.',
    bulkConfirm: 'Confirm the removal',
    bulkConfirmError: 'Could not confirm the removal.',
    revealButton: 'Show in file explorer',
    revealTitle: 'Open the file explorer on this media',
    revealError: 'Could not open the file explorer.',
    closeAriaLabel: 'Close',
    deltaFormat: (minutes) => (minutes < 1 ? '< 1 min' : `±${minutes} min`),
  },
}

const TEXT: Record<Locale, MediaModalsText> = { fr: FR, en: EN }

export function getMediaModalsText(locale: Locale): MediaModalsText {
  return TEXT[locale]
}
