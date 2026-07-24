/**
 * i18n du réglage des pistes audio des médias (bouton engrenage + modale).
 * Hand-written FR/EN (Record<Locale, T>) comme i18n-modals.ts — pas de manifest TOML.
 */

import type { Locale } from '@/lib/i18n/locale'

export interface MediaAudioConfigText {
  gearAriaLabel: string
  title: string
  intro: string
  modeAuto: string
  modeAutoHint: string
  modeManual: string
  modeManualHint: string
  tracksLabel: string
  tracksHint: string
  trackLabel: (n: number) => string
  roleGame: string
  roleVoice: string
  roleOther: string
  addTrack: string
  removeTrack: string
  emptyManual: string
  cancel: string
  save: string
  saving: string
  saveError: string
  closeAriaLabel: string
}

const FR: MediaAudioConfigText = {
  gearAriaLabel: 'Réglage des pistes audio',
  title: 'Pistes audio des médias',
  intro:
    "Déclare le rôle de tes pistes audio pour les prochains transcodages. L'ordre des pistes de ta configuration d'enregistrement est stable, ce réglage s'applique à tous tes futurs médias.",
  modeAuto: 'Automatique',
  modeAutoHint: "L'analyse détecte seule le jeu et les voix.",
  modeManual: 'Manuel',
  modeManualHint: 'Tu déclares le rôle de chaque piste, dans leur ordre.',
  tracksLabel: 'Pistes',
  tracksHint: 'Piste 1 = première piste audio du fichier.',
  trackLabel: (n) => `Piste ${n}`,
  roleGame: 'Jeu',
  roleVoice: 'Voix',
  roleOther: 'Autres pistes',
  addTrack: 'Ajouter une piste',
  removeTrack: 'Retirer',
  emptyManual: 'Ajoute au moins une piste.',
  cancel: 'Annuler',
  save: 'Enregistrer',
  saving: 'Enregistrement…',
  saveError: "Échec de l'enregistrement",
  closeAriaLabel: 'Fermer',
}

const EN: MediaAudioConfigText = {
  gearAriaLabel: 'Audio tracks settings',
  title: 'Media audio tracks',
  intro:
    'Declare the role of your audio tracks for upcoming transcodes. Your recording track order is stable, so this setting applies to all your future media.',
  modeAuto: 'Automatic',
  modeAutoHint: 'Analysis detects game and voices on its own.',
  modeManual: 'Manual',
  modeManualHint: 'You declare the role of each track, in order.',
  tracksLabel: 'Tracks',
  tracksHint: 'Track 1 = first audio track of the file.',
  trackLabel: (n) => `Track ${n}`,
  roleGame: 'Game',
  roleVoice: 'Voice',
  roleOther: 'Other tracks',
  addTrack: 'Add a track',
  removeTrack: 'Remove',
  emptyManual: 'Add at least one track.',
  cancel: 'Cancel',
  save: 'Save',
  saving: 'Saving…',
  saveError: 'Save failed',
  closeAriaLabel: 'Close',
}

const TEXT: Record<Locale, MediaAudioConfigText> = { fr: FR, en: EN }

export function getMediaAudioConfigText(locale?: string | null): MediaAudioConfigText {
  return TEXT[locale === 'en' ? 'en' : 'fr']
}
