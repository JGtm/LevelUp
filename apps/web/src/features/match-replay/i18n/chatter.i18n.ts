/**
 * chatter.i18n.ts — LES TEXTES DES RÉPLIQUES DES SPARTANS : les réglages du panneau de la vue (bulles, icône de voix, fil,
 * familles, fréquence, voix), la ligne du fil et son nom accessible. Le contrat (`ChatterText`) et ses deux langues, comme
 * `lives.i18n.ts` : un sous-dictionnaire à part parce que `fr.ts` et `en.ts` sont au plafond de taille du dépôt.
 *
 * LES RÉPLIQUES ELLES-MÊMES ne sont pas ici : leur texte est celui du jeu, versé avec leur voix
 * (`static/sounds/halo_infinite/repliques/repliques.json`). Servi par `REPLAY_TEXT[locale].chatter` ; la parité FR/EN est tenue
 * par le typage (`satisfies ChatterText` dans chaque langue).
 */
import type { ChatterFamily, ChatterFrequency } from '../model/chatterLines'

export interface ChatterText {
  /** « Répliques en bulles » : une bulle de texte sur le joueur qui parle. */
  layerBubbles: string
  layerBubblesHint: string
  /** « Icône de voix » : un haut-parleur sur le joueur qui parle. */
  layerIcon: string
  layerIconHint: string
  /** « Répliques dans le fil » : les répliques rejoignent le fil du match. */
  layerFeed: string
  layerFeedHint: string
  /** Les cinq familles : leur nom et leur explication (la même phrase pour les cinq). */
  family: Record<ChatterFamily, string>
  familyHint: string
  /** La fréquence : son libellé, son explication, ses trois options et l'explication de chacune. */
  frequencyLabel: string
  frequencyHint: string
  frequency: Record<ChatterFrequency, string>
  frequencyOptionHint: Record<ChatterFrequency, string>
  /** « Voix des répliques » : la voix se joue quand le son du rejeu est activé. */
  layerVoice: string
  layerVoiceHint: string
  /** Une réplique entre guillemets, comme la langue les écrit. */
  quote: (text: string) => string
  /** Le nom accessible d'une ligne du fil : le joueur, sa famille, ce qu'il dit. */
  feedLineAriaFmt: (player: string, family: string, text: string) => string
  /** La réplique a une voix dans la langue de l'interface (infobulle du pictogramme du fil). */
  hasVoice: string
}

export const FR_CHATTER = {
  layerBubbles: 'Répliques en bulles',
  layerBubblesHint: 'Une bulle de texte sur le joueur qui parle, 2,6 secondes, en fondu.',
  layerIcon: 'Icône de voix',
  layerIconHint: 'Une icône de haut-parleur sur le joueur qui parle, quand sa réplique a une voix.',
  layerFeed: 'Répliques dans le fil',
  layerFeedHint: 'Les répliques parlées rejoignent le fil du match.',
  family: {
    enemy: 'Ennemi repéré',
    shields: 'Boucliers',
    squadDown: 'Équipier tombé',
    praise: 'Félicitations',
    other: 'Autres',
  },
  familyHint: 'Famille de répliques. S’applique aux bulles, aux icônes, au fil et aux voix.',
  frequencyLabel: 'Fréquence',
  frequencyHint:
    'Les répliques qui reviennent le plus souvent noient les autres : on peut les écarter. S’applique aux bulles, aux icônes, au fil et aux voix.',
  frequency: { all: 'Toutes', noFrequent: 'Sans les plus fréquentes', rare: 'Rares seulement' },
  frequencyOptionHint: {
    all: 'Toutes les répliques parlées du match',
    noFrequent: 'Écarte les phrases dites quatre fois ou plus dans le match',
    rare: 'Ne garde que les phrases dites une ou deux fois dans le match',
  },
  layerVoice: 'Voix des répliques',
  layerVoiceHint: 'Joue la réplique à voix haute quand le son est activé ; sinon seule l’icône s’affiche.',
  quote: (text) => `«\u00a0${text}\u00a0»`,
  feedLineAriaFmt: (player, family, text) => `${player}, ${family} : ${text}`,
  hasVoice: 'Cette réplique a une voix',
} satisfies ChatterText

export const EN_CHATTER = {
  layerBubbles: 'Spoken lines in bubbles',
  layerBubblesHint: 'A text bubble on the player who speaks, 2.6 seconds, fading in and out.',
  layerIcon: 'Voice icon',
  layerIconHint: 'A speaker icon on the player who speaks, when the line has a voice.',
  layerFeed: 'Spoken lines in the feed',
  layerFeedHint: 'The spoken lines join the match feed.',
  family: {
    enemy: 'Enemy spotted',
    shields: 'Shields',
    squadDown: 'Squadmate down',
    praise: 'Praise',
    other: 'Others',
  },
  familyHint: 'A family of lines. Applies to bubbles, icons, the feed and voices.',
  frequencyLabel: 'Frequency',
  frequencyHint:
    'The lines that come back most often drown the others: you can leave them out. Applies to bubbles, icons, the feed and voices.',
  frequency: { all: 'All', noFrequent: 'Without the most frequent', rare: 'Rare only' },
  frequencyOptionHint: {
    all: 'Every spoken line of the match',
    noFrequent: 'Leaves out the phrases said four times or more in the match',
    rare: 'Keeps only the phrases said once or twice in the match',
  },
  layerVoice: 'Voice of the lines',
  layerVoiceHint: 'Plays the line aloud when sound is on; otherwise only the icon shows.',
  quote: (text) => `“${text}”`,
  feedLineAriaFmt: (player, family, text) => `${player}, ${family}: ${text}`,
  hasVoice: 'This line has a voice',
} satisfies ChatterText
