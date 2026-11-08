/**
 * Textes de la carte du paquet de ressources (Paramètres, onglet Application) — dictionnaire de feature :
 * chaque langue de `config/locales.toml` figure ici (parité par typage).
 */
import type { Locale } from '@/lib/i18n/locale'

export interface ResourcePackText {
  title: string
  intro: string
  upToDate: string
  mismatch: string
  waitingGame: string
  /** Gabarit : {percent}. */
  downloading: string
  verifying: string
  ready: string
  retry: string
  progressLabel: string
  /** Texte par code d'erreur du serveur ; `unknown` pour un code inconnu. */
  errors: Record<'download' | 'signature' | 'checksum' | 'unpack' | 'noKey' | 'refused' | 'unknown', string>
  retryFailed: string
}

export const RESOURCE_PACK_TEXT: Record<Locale, ResourcePackText> = {
  fr: {
    title: 'Ressources',
    intro:
      'Images, sons et données de référence sont livrés à part du moteur : une mise à jour courante ne les retélécharge que s’ils ont changé.',
    upToDate: 'Les ressources sont à jour.',
    mismatch: 'Les ressources installées ne correspondent pas à cette version.',
    waitingGame: 'Téléchargement des ressources en attente de la fin de la partie.',
    downloading: 'Téléchargement des ressources : {percent} %.',
    verifying: 'Vérification des ressources…',
    ready: 'Les ressources sont prêtes : elles seront appliquées au prochain démarrage de LevelUp.',
    retry: 'Réessayer',
    progressLabel: 'Progression du téléchargement des ressources',
    errors: {
      download: 'Le téléchargement des ressources a échoué. Vérifiez votre connexion.',
      signature: 'La signature des ressources téléchargées est invalide : elles ont été refusées.',
      checksum: 'Les ressources téléchargées sont corrompues : elles ont été refusées.',
      unpack: 'Les ressources téléchargées n’ont pas pu être préparées.',
      noKey: 'La clé de vérification des mises à jour est absente : les ressources ne peuvent pas être téléchargées.',
      refused: 'La description des ressources à télécharger est invalide.',
      unknown: 'La préparation des ressources a échoué.',
    },
    retryFailed: 'Le nouvel essai n’a pas pu être lancé.',
  },
  en: {
    title: 'Resources',
    intro:
      'Images, sounds and reference data ship apart from the engine: an ordinary update only downloads them again if they changed.',
    upToDate: 'Resources are up to date.',
    mismatch: 'The installed resources do not match this version.',
    waitingGame: 'Resource download is waiting for the match to end.',
    downloading: 'Downloading resources: {percent}%.',
    verifying: 'Checking resources…',
    ready: 'Resources are ready: they will be applied the next time LevelUp starts.',
    retry: 'Try again',
    progressLabel: 'Resource download progress',
    errors: {
      download: 'Downloading the resources failed. Check your connection.',
      signature: 'The signature of the downloaded resources is invalid: they were rejected.',
      checksum: 'The downloaded resources are corrupted: they were rejected.',
      unpack: 'The downloaded resources could not be prepared.',
      noKey: 'The update verification key is missing: resources cannot be downloaded.',
      refused: 'The description of the resources to download is invalid.',
      unknown: 'Preparing the resources failed.',
    },
    retryFailed: 'The new attempt could not be started.',
  },
}

/** La clé de `errors` qui répond à un code d'erreur du serveur. */
export function resourcePackErrorKey(code: string | undefined): keyof ResourcePackText['errors'] {
  switch (code) {
    case 'resources_download_failed':
      return 'download'
    case 'resources_signature_invalid':
      return 'signature'
    case 'resources_checksum_mismatch':
      return 'checksum'
    case 'resources_unpack_failed':
      return 'unpack'
    case 'resources_no_key':
      return 'noKey'
    case 'resources_refused':
      return 'refused'
    default:
      return 'unknown'
  }
}
