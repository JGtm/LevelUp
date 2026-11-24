/**
 * hillSequence.i18n.ts — LES TEXTES DE LA SUITE DES COLLINES du Roi de la colline : le titre de la pastille posée sur
 * la carte, la mention de la prochaine colline, et la phrase que lit un lecteur d'écran.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `lives.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de taille du dépôt. Servis
 * par `REPLAY_TEXT[locale].hillSequence` ; la parité FR/EN est tenue par le typage (`satisfies HillSequenceText`).
 */
export interface HillSequenceText {
  /** Le mot qui ouvre la pastille : « Collines ». */
  title: string
  /** La prochaine colline, quand le cycle est établi : « Prochaine B ». */
  nextFmt: (label: string) => string
  /** La phrase du lecteur d'écran : les collines jouées dans l'ordre, la courante, la prochaine. */
  ariaFmt: (played: string, current: string | null, next: string | null) => string
}

export const FR_HILL_SEQUENCE = {
  title: 'Collines',
  nextFmt: (label) => `Prochaine ${label}`,
  ariaFmt: (played, current, next) =>
    `Collines jouées : ${played}${current ? `. Colline en cours : ${current}` : ''}${next ? `. Prochaine : ${next}` : ''}`,
} satisfies HillSequenceText

export const EN_HILL_SEQUENCE = {
  title: 'Hills',
  nextFmt: (label) => `Next ${label}`,
  ariaFmt: (played, current, next) =>
    `Hills played: ${played}${current ? `. Current hill: ${current}` : ''}${next ? `. Next: ${next}` : ''}`,
} satisfies HillSequenceText
