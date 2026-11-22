/**
 * Textes de l'encadré « Synchronisation » (Réglages › Données & Médias) — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, SyncNowText>`). Un refus du serveur arrive avec un `code` fixe ; le texte est écrit
 * ici, dans la langue de l'interface, et jamais lu dans le message de l'erreur.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface SyncNowText {
  title: string
  intro: string
  button: string
  buttonRunning: string
  toastStarted: string
  toastBusy: string
  toastFailed: string
}

const FR: SyncNowText = {
  title: 'Synchronisation',
  intro:
    "Les nouveaux matchs sont récupérés automatiquement. Le bouton lance tout de suite la même synchronisation, rattrapage de l'historique compris.",
  button: 'Synchroniser maintenant',
  buttonRunning: 'Synchronisation en cours…',
  toastStarted: 'Synchronisation lancée.',
  toastBusy: 'Une synchronisation est déjà en cours.',
  toastFailed: "La synchronisation n'a pas pu être lancée.",
}

const EN: SyncNowText = {
  title: 'Sync',
  intro:
    'New matches are fetched automatically. The button runs the same sync right away, history catch-up included.',
  button: 'Sync now',
  buttonRunning: 'Syncing…',
  toastStarted: 'Sync started.',
  toastBusy: 'A sync is already running.',
  toastFailed: 'The sync could not be started.',
}

const TEXT: Record<Locale, SyncNowText> = { fr: FR, en: EN }

export function getSyncNowText(locale: Locale): SyncNowText {
  return TEXT[locale]
}
