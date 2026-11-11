/**
 * Textes de la régénération des rejeux (bandeau de la page de rejeu et carte des Paramètres) — dictionnaire
 * de feature : chaque langue de `config/locales.toml` figure ici (parité par typage).
 */
import type { Locale } from '@/lib/i18n/locale'

export interface ReplayRegenText {
  /** Gabarit : {count}. */
  staleOne: string
  staleMany: string
  hint: string
  regenerate: string
  later: string
  pause: string
  resume: string
  checking: string
  upToDate: string
  preparing: string
  /** Gabarits : {done}, {total}. */
  running: string
  paused: string
  waitingGame: string
  /** Fin de passe : gabarit {count}. */
  compressing: string
  /** Gabarit : {count}. */
  compressedOne: string
  compressedMany: string
  compressFailedOne: string
  compressFailedMany: string
  /** Gabarit : {count}. */
  failedOne: string
  failedMany: string
  commandFailed: string
  /** Carte des Paramètres. */
  cardTitle: string
  cardIntro: string
  progressLabel: string
  /** Bouton secondaire de la carte : refait le décompte des rejeux en retard. */
  recount: string
}

export const REPLAY_REGEN_TEXT: Record<Locale, ReplayRegenText> = {
  fr: {
    staleOne: '1 rejeu à régénérer après la mise à jour.',
    staleMany: '{count} rejeux à régénérer après la mise à jour.',
    hint: 'Ils restent lisibles. La régénération les met à niveau, un film à la fois, en arrière-plan ; elle attend la fin de la partie.',
    regenerate: 'Régénérer',
    later: 'Plus tard',
    pause: 'Pause',
    resume: 'Reprendre',
    checking: 'Vérification des rejeux…',
    upToDate: 'Tous les rejeux sont à jour.',
    preparing: 'Préparation de la liste des rejeux…',
    running: 'Régénération des rejeux : {done} sur {total}.',
    paused: 'Régénération en pause : {done} sur {total}.',
    waitingGame: 'En attente de la fin de la partie : {done} sur {total}.',
    compressing: 'Compression des rejeux sur le disque : {count} faits.',
    compressedOne: '1 rejeu compressé sur le disque.',
    compressedMany: '{count} rejeux compressés sur le disque.',
    compressFailedOne: '1 rejeu n’a pas pu être compressé ; il reste lisible.',
    compressFailedMany: '{count} rejeux n’ont pas pu être compressés ; ils restent lisibles.',
    failedOne: '1 rejeu n’a pas pu être régénéré.',
    failedMany: '{count} rejeux n’ont pas pu être régénérés.',
    commandFailed: 'La commande n’a pas pu être envoyée.',
    cardTitle: 'Rejeux',
    cardIntro:
      'Après une mise à jour, les rejeux déjà construits peuvent être refaits avec la nouvelle version. Rien ne démarre sans votre accord.',
    progressLabel: 'Progression de la régénération',
    recount: 'Recompter',
  },
  en: {
    staleOne: '1 replay to regenerate after the update.',
    staleMany: '{count} replays to regenerate after the update.',
    hint: 'They stay readable. Regeneration brings them up to date, one film at a time, in the background; it waits for the game to end.',
    regenerate: 'Regenerate',
    later: 'Later',
    pause: 'Pause',
    resume: 'Resume',
    checking: 'Checking replays…',
    upToDate: 'All replays are up to date.',
    preparing: 'Preparing the replay list…',
    running: 'Regenerating replays: {done} of {total}.',
    paused: 'Regeneration paused: {done} of {total}.',
    waitingGame: 'Waiting for the game to end: {done} of {total}.',
    compressing: 'Compressing replays on disk: {count} done.',
    compressedOne: '1 replay compressed on disk.',
    compressedMany: '{count} replays compressed on disk.',
    compressFailedOne: '1 replay could not be compressed; it stays readable.',
    compressFailedMany: '{count} replays could not be compressed; they stay readable.',
    failedOne: '1 replay could not be regenerated.',
    failedMany: '{count} replays could not be regenerated.',
    commandFailed: 'The command could not be sent.',
    cardTitle: 'Replays',
    cardIntro:
      'After an update, replays already built can be redone with the new version. Nothing starts without your consent.',
    progressLabel: 'Regeneration progress',
    recount: 'Recount',
  },
}
