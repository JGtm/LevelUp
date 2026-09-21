/**
 * coordinationI18n — LE DICTIONNAIRE FR/EN des cartes « Riposte », « Appui reçu » et
 * « Portée des engagements » de la colonne de session (lot O, D22-1 / D22-4 / D22-6).
 *
 * Dictionnaire LOCAL typé `Record<Locale, T>`, comme `_shared/usage/usageI18n.ts` : ces
 * cartes montent les formes du bloc « usages » (jauges, bande, légende) et parlent donc la
 * même langue qu'elles — pas celle du manifeste `session.toml`, qui sert le chrome de la
 * page (titres de section, tableau des matchs, comparaison).
 *
 * D22-VERBOSITÉ (LOI) : AUCUNE phrase de lecteur. Les libellés sont factuels et tactiques
 * (« Je suis couvert », « Ma part des appuis ») ; toute l'explication tient dans l'infobulle
 * (i) du titre de carte, en TROIS phrases au plus. Ne pas re-déverser de méthode sous les
 * graphes : c'est exactement ce que D22 retire du reste de l'app.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface CoordinationText {
  // ─── Carte Riposte ───────────────────────────────────────────────────────────
  cardRiposte: string
  gaugeCovered: string
  gaugeIRiposte: string
  bandRiposte: string
  delaiMedian: string
  /** Un délai déjà formaté en secondes, p. ex. « 4,2 s ». */
  delaiFmt: (secondes: string) => string
  infoRiposte1: (fenetreSecondes: string) => string
  infoRiposte2: string
  infoRiposte3: string

  // ─── Carte Appui reçu ────────────────────────────────────────────────────────
  cardAppui: string
  gaugePrepared: string
  gaugeAssistShare: string
  bandAppui: string
  infoAppui1: string
  infoAppui2: string
  infoAppui3: string

  // ─── Communs aux deux cartes ─────────────────────────────────────────────────
  lowSample: string
  /** Réserve de couverture, en pied de carte : « 7 matchs mesurés sur 9 ». */
  coverageMatchesFmt: (mesures: number, total: number) => string
  /** Infobulle d'une case de bande : n° de match, part, parité. */
  bandTipFmt: (index: number, part: string, parite: string) => string
  /** Infobulle d'une case non mesurée. */
  bandTipUnmeasured: (index: number) => string
  /** Infobulle d'un rail de jauge : valeur puis comptes bruts. */
  gaugeTipFmt: (valeur: string, brut: number, n: number) => string
  /** Le compte nu à droite d'une bande : « 5/8 ». */
  bandCountFmt: (audessus: number, total: number) => string

  // ─── Carte Portée des engagements ────────────────────────────────────────────
  cardRange: string
  rangeAxis: string
  rangeAbove: string
  rangeBelow: string
  rangeLobbyLine: string
  rangeSessionMedian: string
  roleFront: string
  roleVersatile: string
  roleSniper: string
  rangeTipFmt: (mediane: string, lobby: string, ecart: string, frags: number) => string
  rangeCoverageFmt: (mesures: number, total: number) => string
  rangeLowSample: (seuil: number) => string
  infoRange1: string
  infoRange2: string
  infoRange3: (seuil: number) => string
  rangeEmpty: string
}

export const COORDINATION_TEXT: Record<Locale, CoordinationText> = {
  fr: {
    cardRiposte: 'Riposte',
    gaugeCovered: 'Je suis couvert',
    gaugeIRiposte: 'Je riposte',
    bandRiposte: 'Je riposte, match par match',
    delaiMedian: 'Délai médian de riposte',
    delaiFmt: (s) => `${s} s`,
    infoRiposte1: (f) =>
      `Une mort est ripostée quand un coéquipier abat le tueur dans les ${f} s qui suivent.`,
    infoRiposte2:
      'Le dénominateur de « je riposte » est le nombre de morts de mon camp, pas mes morts.',
    infoRiposte3: 'La parité vaut 1/n, n étant l’effectif de mon camp sur le match.',

    cardAppui: 'Appui reçu',
    gaugePrepared: 'On me prépare',
    gaugeAssistShare: 'Ma part des appuis',
    bandAppui: 'Ma part des appuis, match par match',
    infoAppui1:
      '« On me prépare » se rapporte à mes frags ; « ma part des appuis » aux appuis distribués dans mon camp.',
    infoAppui2:
      'Un appui dont l’auteur n’est pas résolu par le film n’entre dans aucun des deux dénominateurs.',
    infoAppui3: 'La parité vaut 1/n, n étant l’effectif de mon camp sur le match.',

    lowSample: 'échantillon faible',
    coverageMatchesFmt: (m, t) => `${m} matchs mesurés sur ${t}`,
    bandTipFmt: (i, part, parite) => `Match #${i} · ${part} (parité ${parite})`,
    bandTipUnmeasured: (i) => `Match #${i} · non mesuré`,
    gaugeTipFmt: (v, brut, n) => `${v} · ${brut} sur ${n}`,
    bandCountFmt: (a, t) => `${a}/${t}`,

    cardRange: 'Portée des engagements',
    rangeAxis: 'Écart au lobby (m)',
    rangeAbove: 'Plus loin que le lobby',
    rangeBelow: 'Plus près que le lobby',
    rangeLobbyLine: 'Médiane du lobby',
    rangeSessionMedian: 'Médiane de session',
    roleFront: 'Ligne de front',
    roleVersatile: 'Polyvalent',
    roleSniper: 'Tireur d’élite',
    rangeTipFmt: (med, lobby, ecart, frags) =>
      `Médiane ${med} · lobby ${lobby} · écart ${ecart} · ${frags} frags mesurés`,
    rangeCoverageFmt: (m, t) => `${m} frags mesurés sur ${t}`,
    rangeLowSample: (s) => `bâton creux : moins de ${s} frags mesurés`,
    infoRange1: 'Chaque bâton est l’écart entre ma médiane de frag et celle du lobby du match.',
    infoRange2: 'L’écart neutralise la carte et le mode : quand un lobby joue court, moi aussi.',
    infoRange3: (s) => `Sous ${s} frags mesurés le bâton reste creux : la médiane est du bruit.`,
    rangeEmpty: 'Aucun frag mesuré sur les matchs de cette session.',
  },
  en: {
    cardRiposte: 'Payback',
    gaugeCovered: 'I am covered',
    gaugeIRiposte: 'I pay back',
    bandRiposte: 'I pay back, match by match',
    delaiMedian: 'Median payback delay',
    delaiFmt: (s) => `${s}s`,
    infoRiposte1: (f) => `A death is paid back when a teammate kills the killer within ${f}s.`,
    infoRiposte2: 'The denominator of "I pay back" is my team’s deaths, not my own.',
    infoRiposte3: 'Parity is 1/n, n being my team size on that match.',

    cardAppui: 'Support received',
    gaugePrepared: 'Set up for me',
    gaugeAssistShare: 'My share of assists',
    bandAppui: 'My share of assists, match by match',
    infoAppui1:
      '"Set up for me" is measured against my kills; "my share of assists" against the assists dealt inside my team.',
    infoAppui2: 'An assist whose author the film cannot resolve enters neither denominator.',
    infoAppui3: 'Parity is 1/n, n being my team size on that match.',

    lowSample: 'low sample',
    coverageMatchesFmt: (m, t) => `${m} of ${t} matches measured`,
    bandTipFmt: (i, part, parite) => `Match #${i} · ${part} (parity ${parite})`,
    bandTipUnmeasured: (i) => `Match #${i} · not measured`,
    gaugeTipFmt: (v, brut, n) => `${v} · ${brut} of ${n}`,
    bandCountFmt: (a, t) => `${a}/${t}`,

    cardRange: 'Engagement range',
    rangeAxis: 'Gap to lobby (m)',
    rangeAbove: 'Farther than the lobby',
    rangeBelow: 'Closer than the lobby',
    rangeLobbyLine: 'Lobby median',
    rangeSessionMedian: 'Session median',
    roleFront: 'Front line',
    roleVersatile: 'All-rounder',
    roleSniper: 'Marksman',
    rangeTipFmt: (med, lobby, ecart, frags) =>
      `Median ${med} · lobby ${lobby} · gap ${ecart} · ${frags} measured kills`,
    rangeCoverageFmt: (m, t) => `${m} of ${t} kills measured`,
    rangeLowSample: (s) => `hollow bar: fewer than ${s} measured kills`,
    infoRange1: 'Each bar is the gap between my kill median and the lobby median of that match.',
    infoRange2: 'The gap neutralises map and mode: when a lobby plays close, so do I.',
    infoRange3: (s) => `Below ${s} measured kills the bar stays hollow: the median is noise.`,
    rangeEmpty: 'No measured kill across the matches of this session.',
  },
}
