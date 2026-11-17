/**
 * Textes de la génération à la demande d'un rejeu (bloc de la page du rejeu et bouton de l'en-tête de la page du
 * match) — dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage).
 * Mots : « générer », jamais « cuire » ; titres factuels, sans possessif ni pronom.
 */
import type { Locale } from '@/lib/i18n/locale'

import type { ReplayGenerationMessageKey } from './logic'

export interface ReplayGenerationText {
  /** Boutons. */
  generate: string
  retry: string
  open: string
  /** Erreurs de la demande elle-même (le POST), distinctes de l'état du match. */
  queueFull: string
  requestFailed: string
  /** La phrase de chaque état ; `queued` : gabarit {position}. */
  messages: Record<ReplayGenerationMessageKey, string>
}

export const REPLAY_GENERATION_TEXT: Record<Locale, ReplayGenerationText> = {
  fr: {
    generate: 'Générer le rejeu',
    retry: 'Réessayer',
    open: 'Ouvrir le rejeu',
    queueFull: 'La file est pleine : réessayer dans quelques minutes.',
    requestFailed: 'La demande de génération a échoué.',
    messages: {
      generatable: 'Le film du match sera téléchargé puis analysé sur ce poste.',
      queued: 'En file d’attente : position {position}.',
      queuedNoPosition: 'En file d’attente.',
      waitingForGame: 'En attente de la fin de la partie.',
      paused: 'File en pause.',
      downloading: 'Téléchargement du film…',
      building: 'Génération du rejeu en cours…',
      failed: 'La génération a échoué.',
      failedFinal: 'Ce match ne peut pas être généré : la carte ou le film n’est pas reconnu.',
      filmUnavailable: 'Le film de ce match n’est plus disponible.',
      modeUnmanaged: 'Ce mode de jeu n’a pas de rejeu.',
      matchUnknown: 'Match introuvable dans les données de ce profil.',
      noToken: 'Aucun compte Xbox relié à ce profil.',
      ready: 'Le rejeu est prêt.',
    },
  },
  en: {
    generate: 'Generate replay',
    retry: 'Try again',
    open: 'Open replay',
    queueFull: 'The queue is full: try again in a few minutes.',
    requestFailed: 'The generation request failed.',
    messages: {
      generatable: 'The match film will be downloaded then analysed on this computer.',
      queued: 'Queued: position {position}.',
      queuedNoPosition: 'Queued.',
      waitingForGame: 'Waiting for the game to end.',
      paused: 'Queue paused.',
      downloading: 'Downloading the film…',
      building: 'Generating the replay…',
      failed: 'Generation failed.',
      failedFinal: 'This match cannot be generated: the map or the film is not recognised.',
      filmUnavailable: 'The film of this match is no longer available.',
      modeUnmanaged: 'This game mode has no replay.',
      matchUnknown: 'Match not found in the data of this profile.',
      noToken: 'No Xbox account linked to this profile.',
      ready: 'The replay is ready.',
    },
  },
}

/** La phrase d'un état, le gabarit `queued` rempli par la position servie. */
export function generationMessage(t: ReplayGenerationText, key: ReplayGenerationMessageKey, position: number): string {
  return t.messages[key].replace('{position}', String(position))
}
