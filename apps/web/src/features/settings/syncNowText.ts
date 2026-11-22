/**
 * Textes de l'encadré « Synchronisation » (Réglages › Données & Médias) — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, SyncNowText>`). Un refus du serveur arrive avec un `code` fixe ; le texte est écrit
 * ici, dans la langue de l'interface, et jamais lu dans le message de l'erreur. Le libellé de l'état
 * de l'historique et celui d'une passe arrivent résolus du serveur, dans la langue de la requête.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface SyncNowText {
  title: string
  intro: string
  button: string
  buttonRunning: string
  /** Gabarit : {remaining} (délai restant, « 1 min 40 » ou « 40 s »). */
  buttonCooldown: string
  /** Ligne sous le bouton pendant un cycle. */
  statusRunning: string
  /** Gabarit : {last} (heure de fin du dernier cycle), {next} (heure à partir de laquelle un lancement est permis). */
  statusCooldown: string
  toastStarted: string
  toastBusy: string
  toastFailed: string
  loading: string
  loadFailed: string
  emptyTitle: string
  emptyDescription: string
  rowHistory: string
  rowMatches: string
  rowOldest: string
  rowLastPass: string
  rowNextPass: string
  noMatches: string
  /** Gabarit : {status} (libellé résolu par le serveur), {date}. */
  lastPassValue: string
  /** Gabarit : {count} (matchs ajoutés par la passe). */
  lastPassAdded: string
  lastPassNone: string
  /** Gabarit : {date}. */
  nextPassAt: string
  nextPassRunning: string
  nextPassManualOnly: string
  nextPassUnknown: string
}

const FR: SyncNowText = {
  title: 'Synchronisation',
  intro:
    "Les nouveaux matchs sont récupérés automatiquement. Le bouton lance tout de suite la même synchronisation, rattrapage de l'historique compris.",
  button: 'Synchroniser maintenant',
  buttonRunning: 'Synchronisation en cours…',
  buttonCooldown: 'Disponible dans {remaining}',
  statusRunning: 'Synchronisation en cours.',
  statusCooldown: 'Dernière synchronisation terminée à {last}. Prochain lancement possible à {next}.',
  toastStarted: 'Synchronisation lancée.',
  toastBusy: 'Une synchronisation est déjà en cours.',
  toastFailed: "La synchronisation n'a pas pu être lancée.",
  loading: "Lecture de l'état de l'historique…",
  loadFailed: "L'état de l'historique n'a pas pu être lu.",
  emptyTitle: 'Aucun titre suivi',
  emptyDescription: "Dès qu'un titre est suivi pour ce joueur, l'état de son historique apparaît ici.",
  rowHistory: 'Historique',
  rowMatches: 'Matchs enregistrés',
  rowOldest: 'Plus ancien match',
  rowLastPass: 'Dernière passe',
  rowNextPass: 'Prochaine passe',
  noMatches: 'Aucun match',
  lastPassValue: '{status}, le {date}',
  lastPassAdded: 'Matchs ajoutés : {count}',
  lastPassNone: 'Aucune passe depuis le démarrage',
  nextPassAt: 'Vers {date}',
  nextPassRunning: 'Synchronisation en cours',
  nextPassManualOnly: 'Synchronisation automatique désactivée',
  nextPassUnknown: 'Pas encore planifiée',
}

const EN: SyncNowText = {
  title: 'Sync',
  intro:
    'New matches are fetched automatically. The button runs the same sync right away, history catch-up included.',
  button: 'Sync now',
  buttonRunning: 'Sync in progress…',
  buttonCooldown: 'Available in {remaining}',
  statusRunning: 'Sync in progress.',
  statusCooldown: 'Last sync finished at {last}. Next run available at {next}.',
  toastStarted: 'Sync started.',
  toastBusy: 'A sync is already running.',
  toastFailed: 'The sync could not be started.',
  loading: 'Reading the history state…',
  loadFailed: 'The history state could not be read.',
  emptyTitle: 'No followed title',
  emptyDescription: 'Once a title is followed for this player, its history state shows up here.',
  rowHistory: 'History',
  rowMatches: 'Recorded matches',
  rowOldest: 'Oldest match',
  rowLastPass: 'Last pass',
  rowNextPass: 'Next pass',
  noMatches: 'No match',
  lastPassValue: '{status}, on {date}',
  lastPassAdded: 'Matches added: {count}',
  lastPassNone: 'No pass since startup',
  nextPassAt: 'Around {date}',
  nextPassRunning: 'Sync in progress',
  nextPassManualOnly: 'Automatic sync is turned off',
  nextPassUnknown: 'Not scheduled yet',
}

const TEXT: Record<Locale, SyncNowText> = { fr: FR, en: EN }

export function getSyncNowText(locale: Locale): SyncNowText {
  return TEXT[locale]
}
