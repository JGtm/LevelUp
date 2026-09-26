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
  /**
   * L'infobulle d'un rail QUI PORTE LE REPÈRE D'HABITUEL : l'infobulle de base, puis le
   * taux de la période de référence. C'est le seul endroit où le repère est NOMMÉ — le
   * trait, lui, se rend comme celui de la parité (D22-verbosité : rien sous la forme).
   */
  gaugeTipUsualFmt: (tip: string, habituel: string) => string
  /** Le compte nu à droite d'une bande : « 5/8 ». */
  bandCountFmt: (audessus: number, total: number) => string

  // ─── Carte Portée des engagements ────────────────────────────────────────────
  cardRange: string
  rangeAxis: string
  /** Le nom de l'axe X — selon que le nuage porte la periode ou la seule session (repli). */
  rangeXAxisPeriod: string
  rangeXAxisSession: string
  rangeLobbyLine: string
  /** L'etiquette du fond qui marque les matchs de la session dans la periode. */
  rangeThisSession: string
  roleFront: string
  roleVersatile: string
  roleSniper: string
  rangeLegendPeriod: string
  rangeLegendSession: string
  rangeLegendTrend: (fenetre: number) => string
  rangeTipMedian: (mediane: string) => string
  rangeTipDelta: (ecart: string) => string
  rangeTipMeasured: (frags: number) => string
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
    gaugeTipUsualFmt: (tip, habituel) => `${tip} · habituel ${habituel}`,
    bandCountFmt: (a, t) => `${a}/${t}`,

    cardRange: 'Portée des engagements',
    rangeAxis: 'Écart au lobby (m)',
    rangeXAxisPeriod: 'Matchs de la période, du plus ancien au plus récent',
    rangeXAxisSession: 'Matchs de la session, du plus ancien au plus récent',
    rangeLobbyLine: 'Médiane du lobby',
    rangeThisSession: 'Cette session',
    roleFront: 'Ligne de front',
    roleVersatile: 'Polyvalent',
    roleSniper: 'Tireur d’élite',
    rangeLegendPeriod: 'Période de référence',
    rangeLegendSession: 'Matchs de la session',
    rangeLegendTrend: (f) => `Tendance, fenêtre ${f} matchs`,
    rangeTipMedian: (med) => `Médiane ${med} m`,
    rangeTipDelta: (ecart) => `Écart au lobby ${ecart} m`,
    rangeTipMeasured: (frags) => `${frags} frags mesurés`,
    rangeCoverageFmt: (m, t) => `${m} frags mesurés sur ${t}`,
    rangeLowSample: (s) => `point creux : moins de ${s} frags mesurés`,
    infoRange1:
      'Chaque point est un match de ma période : l’écart entre ma médiane de frag et celle du lobby.',
    infoRange2:
      'Les bandes sont mes rôles sur la période ; le fond marque les matchs de cette session.',
    infoRange3: (s) => `Sous ${s} frags mesurés le point reste creux : la médiane est du bruit.`,
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
    gaugeTipUsualFmt: (tip, habituel) => `${tip} · usual ${habituel}`,
    bandCountFmt: (a, t) => `${a}/${t}`,

    cardRange: 'Engagement range',
    rangeAxis: 'Gap to lobby (m)',
    rangeXAxisPeriod: 'Matches of the period, oldest to most recent',
    rangeXAxisSession: 'Matches of the session, oldest to most recent',
    rangeLobbyLine: 'Lobby median',
    rangeThisSession: 'This session',
    roleFront: 'Front line',
    roleVersatile: 'All-rounder',
    roleSniper: 'Marksman',
    rangeLegendPeriod: 'Reference period',
    rangeLegendSession: 'Session matches',
    rangeLegendTrend: (f) => `Trend, ${f}-match window`,
    rangeTipMedian: (med) => `Median ${med} m`,
    rangeTipDelta: (ecart) => `Gap to lobby ${ecart} m`,
    rangeTipMeasured: (frags) => `${frags} measured kills`,
    rangeCoverageFmt: (m, t) => `${m} of ${t} kills measured`,
    rangeLowSample: (s) => `hollow dot: fewer than ${s} measured kills`,
    infoRange1:
      'Each dot is one match of my period: the gap between my kill median and the lobby median.',
    infoRange2: 'The bands are my roles over the period; the shading marks this session’s matches.',
    infoRange3: (s) => `Below ${s} measured kills the dot stays hollow: the median is noise.`,
    rangeEmpty: 'No measured kill across the matches of this session.',
  },
}
