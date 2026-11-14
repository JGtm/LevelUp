/**
 * assistFlowStrings — les textes du diagramme de flux « assistant -> finisseur », UNE seule
 * source pour « Appui » (Escouade) et « Assistances » (page Match) : à question identique, mots
 * identiques. Parité FR / EN garantie par le typage `Record<Locale, …>`.
 *
 * VOCABULAIRE (décision du 2026-11-14) : « assistant » (celui qui prépare le frag) et
 * « finisseur » (celui qui le conclut) ; en anglais « assister » et « finisher ». La statistique
 * comptée reste « assistance ». PHRASES TOUJOURS À LA VOIX ACTIVE, sujet = l'assistant : jamais
 * « X a été assisté par ».
 */
import type { Locale } from '@/lib/i18n/locale'

import type { AssistFlowTexts } from './assistFlowOption'

export interface AssistFlowText extends AssistFlowTexts {
  /** En-tête de la colonne de gauche. */
  assistantHeader: string
  /** En-tête de la colonne de droite. */
  finisherHeader: string
}

const plural = (n: number, one: string, many: string) => (n > 1 ? many : one)

const FR: AssistFlowText = {
  assistantHeader: 'Assistant : prépare le frag',
  finisherHeader: 'Finisseur : conclut le frag',
  link: (assistant, finisher, count) => `${assistant} a assisté ${finisher} sur ${count} ${plural(count, 'frag', 'frags')}`,
  outdamaged: (n) => `dont ${n} où l’assistant a fait plus de dégâts que le finisseur`,
  avgShare: (pct) => `part moyenne ${pct} %`,
  given: (name, n) => `${name} : ${n} ${plural(n, 'assistance donnée', 'assistances données')}`,
  concluded: (name, n) => `${name} : ${n} ${plural(n, 'frag conclu', 'frags conclus')} sur une assistance`,
}

const EN: AssistFlowText = {
  assistantHeader: 'Assister: sets up the kill',
  finisherHeader: 'Finisher: completes the kill',
  link: (assistant, finisher, count) => `${assistant} assisted ${finisher} on ${count} ${plural(count, 'kill', 'kills')}`,
  outdamaged: (n) => `including ${n} where the assister dealt more damage than the finisher`,
  avgShare: (pct) => `avg share ${pct}%`,
  given: (name, n) => `${name}: ${n} ${plural(n, 'assist', 'assists')} given`,
  concluded: (name, n) => `${name}: ${n} ${plural(n, 'kill', 'kills')} finished on an assist`,
}

export const ASSIST_FLOW_TEXT: Record<Locale, AssistFlowText> = { fr: FR, en: EN }
