/**
 * Textes de la mise en commun du jeton — FR et EN.
 *
 * Dictionnaire de feature (même couche que `features/legal/i18n.ts`) : le texte explique
 * un choix de consentement, il doit rester identique entre la bannière de première
 * connexion et la carte des réglages, et suivre la politique de confidentialité
 * (`features/legal/i18n.ts`, section « Mise en commun des jetons ») — toute évolution de
 * l'un se répercute sur l'autre dans le même commit.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface PoolSharingText {
  bannerTitle: string
  bannerBody: string
  accept: string
  decline: string
  learnMore: string
  cardTitle: string
  cardToggle: string
  stateGranted: string
  stateDeclined: string
  stateUndecided: string
  toastGranted: string
  toastDeclined: string
  toastFailed: string
}

const FR: PoolSharingText = {
  bannerTitle: 'Mise en commun des jetons',
  bannerBody:
    "Pour accélérer la synchro de tout le monde, les jetons des joueurs qui l'acceptent sont mis en commun : une requête de synchro peut passer par le jeton de l'un d'eux. Il ne sert qu'à lire des statistiques publiques, jamais tes données privées, et n'est montré à personne. Si tu refuses, ta synchro n'utilise que ton jeton : elle reste possible, mais plus lente, limitée par le débit de Microsoft. Tu peux changer d'avis à tout moment dans les réglages.",
  accept: 'Mettre mon jeton à disposition',
  decline: 'Garder mon jeton pour moi',
  learnMore: 'En savoir plus',
  cardTitle: 'Mise en commun du jeton',
  cardToggle: 'Mettre mon jeton à disposition du pool de synchro',
  stateGranted:
    "Ton jeton sert à synchroniser les parties des autres joueurs, et leurs jetons servent aux tiennes.",
  stateDeclined:
    "Ton jeton ne sert qu'à ta propre synchro, qui est plus lente : elle est limitée par le débit de Microsoft.",
  stateUndecided:
    "Tu n'as pas encore choisi : en attendant, ton jeton ne sert qu'à ta propre synchro.",
  toastGranted: 'Ton jeton est mis à disposition du pool.',
  toastDeclined: 'Ton jeton est retiré du pool.',
  toastFailed: "Le choix n'a pas pu être enregistré.",
}

const EN: PoolSharingText = {
  bannerTitle: 'Token pooling',
  bannerBody:
    "To speed up everyone's sync, the tokens of players who agree are pooled: a sync request may go through one of their tokens. It is only used to read public statistics, never your private data, and is shown to nobody. If you decline, your sync only uses your own token: it still works, but slower, limited by Microsoft's rate limit. You can change your mind at any time in the settings.",
  accept: 'Make my token available',
  decline: 'Keep my token to myself',
  learnMore: 'Learn more',
  cardTitle: 'Token pooling',
  cardToggle: 'Make my token available to the sync pool',
  stateGranted: "Your token syncs other players' matches, and theirs sync yours.",
  stateDeclined:
    "Your token only serves your own sync, which is slower: it is limited by Microsoft's rate limit.",
  stateUndecided: 'You have not chosen yet: until you do, your token only serves your own sync.',
  toastGranted: 'Your token is now available to the pool.',
  toastDeclined: 'Your token is withdrawn from the pool.',
  toastFailed: 'The choice could not be saved.',
}

const TEXT: Record<Locale, PoolSharingText> = { fr: FR, en: EN }

export function getPoolSharingText(locale: Locale): PoolSharingText {
  return TEXT[locale]
}
