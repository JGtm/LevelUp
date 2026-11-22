/**
 * Textes de la carte « Synchronisation » des réglages (Données & Médias) — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, SyncCardText>`). Le refus d'un lancement arrive avec un code fixe de l'API ; le texte
 * est écrit ici, dans la langue de l'interface, jamais lu dans le message de l'erreur.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface SyncCardText {
  title: string
  /** Gabarit : {player} (pseudo du joueur affiché). */
  intro: string
  start: string
  running: string
  /** Gabarit : {minutes} (minutes restantes avant qu'un lancement manuel soit de nouveau accepté). */
  cooldown: string
  paused: string
  historyComplete: string
  historyCatchingUp: string
  historyPausedIncomplete: string
  statusUnavailable: string
  toastStarted: string
  toastCooldown: string
  toastAlreadyActive: string
  toastFailed: string
}

const FR: SyncCardText = {
  title: 'Synchronisation',
  intro: 'Récupère les nouveaux matchs de {player}.',
  start: 'Synchroniser maintenant',
  running: 'Synchronisation en cours…',
  cooldown: 'Une nouvelle synchronisation sera possible dans environ {minutes} min.',
  paused:
    'La synchronisation de ce jeu est en pause pour ce joueur : elle se réactive dans Paramètres › Jeux.',
  historyComplete: "Historique complet : le premier match disponible a été atteint.",
  historyCatchingUp:
    "Rattrapage de l'historique en cours : chaque synchronisation récupère jusqu'à 200 matchs plus anciens, jusqu'au premier match.",
  historyPausedIncomplete:
    "Historique incomplet : le rattrapage reprendra quand la synchronisation de ce jeu sera réactivée.",
  statusUnavailable: "L'état de la synchronisation est indisponible pour l'instant.",
  toastStarted: 'Synchronisation lancée.',
  toastCooldown: "Une synchronisation vient d'être lancée : réessaie dans quelques minutes.",
  toastAlreadyActive: 'Une synchronisation de ce joueur est déjà en cours.',
  toastFailed: "La synchronisation n'a pas pu démarrer.",
}

const EN: SyncCardText = {
  title: 'Synchronisation',
  intro: 'Retrieves the new matches of {player}.',
  start: 'Synchronise now',
  running: 'Synchronisation running…',
  cooldown: 'A new synchronisation will be possible in about {minutes} min.',
  paused: 'Sync for this game is paused for this player: turn it back on in Settings › Games.',
  historyComplete: 'History complete: the first available match has been reached.',
  historyCatchingUp:
    'Catching up on the history: each synchronisation retrieves up to 200 older matches, until the first match.',
  historyPausedIncomplete:
    'History incomplete: the catch-up will resume when sync for this game is turned back on.',
  statusUnavailable: 'The synchronisation status is unavailable for now.',
  toastStarted: 'Synchronisation started.',
  toastCooldown: 'A synchronisation was just started: try again in a few minutes.',
  toastAlreadyActive: 'A synchronisation of this player is already running.',
  toastFailed: 'The synchronisation could not be started.',
}

const SYNC_CARD_TEXT: Record<Locale, SyncCardText> = { fr: FR, en: EN }

export function getSyncCardText(locale: Locale): SyncCardText {
  return SYNC_CARD_TEXT[locale]
}
