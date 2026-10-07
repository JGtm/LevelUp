/**
 * coordinationI18n — LE DICTIONNAIRE FR/EN des cartes « Appui reçu » et « Portée des
 * engagements » de la colonne de session (lot O, D22-4 / D22-6).
 *
 * Dictionnaire LOCAL typé `Record<Locale, T>`, comme `_shared/usage/usageI18n.ts` : ces
 * cartes montent les formes du bloc « usages » (jauges, bande, légende) et parlent donc la
 * même langue qu'elles — pas celle du manifeste `session.toml`, qui sert le chrome de la
 * page (titres de section, tableau des matchs, comparaison).
 *
 * D22-VERBOSITÉ (LOI) : AUCUNE phrase de lecteur. Les libellés sont factuels et tactiques
 * (« Frags appuyés », « Part des appuis de l’équipe ») ; toute l'explication tient dans l'infobulle
 * (i) du titre de carte, en TROIS phrases au plus. Ne pas re-déverser de méthode sous les
 * graphes : c'est exactement ce que D22 retire du reste de l'app.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface CoordinationText {
  // ─── Carte Appui reçu ────────────────────────────────────────────────────────
  cardAppui: string
  gaugePrepared: string
  gaugeAssistShare: string
  bandAppui: string
  infoAppui1: string
  infoAppui2: string
  infoAppui3: string

  // ─── Pied et infobulles de la carte ──────────────────────────────────────────
  lowSample: string
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
  rangeLowSample: (seuil: number) => string
  infoRange1: string
  infoRange2: string
  infoRange3: (seuil: number) => string
  rangeEmpty: string
}

export const COORDINATION_TEXT: Record<Locale, CoordinationText> = {
  fr: {
    cardAppui: 'Appui reçu',
    gaugePrepared: 'Frags appuyés',
    gaugeAssistShare: 'Part des appuis de l’équipe',
    bandAppui: 'Part des appuis de l’équipe, par match',
    infoAppui1:
      'Frags appuyés : frags du joueur ayant reçu une assistance, sur ses frags ; part des appuis de l’équipe : ' +
      'assistances reçues par le joueur, sur toutes celles de l’équipe.',
    infoAppui2:
      'Un appui dont l’auteur n’est pas résolu par le film n’entre dans aucun des deux dénominateurs.',
    infoAppui3: 'Parité : 1/n, n étant l’effectif de l’équipe sur le match.',

    lowSample: 'échantillon faible',
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
    rangeLowSample: (s) => `point creux : moins de ${s} frags mesurés`,
    infoRange1:
      'Chaque point est un match de la période : écart entre la médiane de frag du joueur et celle du lobby.',
    infoRange2:
      'Bandes : rôles du joueur sur la période ; fond : matchs de cette session.',
    infoRange3: (s) => `Sous ${s} frags mesurés le point reste creux : la médiane est du bruit.`,
    rangeEmpty: 'Aucun frag mesuré sur les matchs de cette session.',
  },
  en: {
    cardAppui: 'Support received',
    gaugePrepared: 'Assisted kills',
    gaugeAssistShare: 'Share of the team’s assists',
    bandAppui: 'Share of the team’s assists, by match',
    infoAppui1:
      'Assisted kills: the player’s kills that received an assist, over the player’s kills; share of the team’s ' +
      'assists: assists received by the player, over all of the team’s.',
    infoAppui2: 'An assist whose author the film cannot resolve enters neither denominator.',
    infoAppui3: 'Parity: 1/n, n being the team’s headcount on the match.',

    lowSample: 'low sample',
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
    rangeLowSample: (s) => `hollow dot: fewer than ${s} measured kills`,
    infoRange1:
      'Each dot is one match of the period: the gap between the player’s kill median and the lobby median.',
    infoRange2: 'Bands: the player’s roles over the period; shading: this session’s matches.',
    infoRange3: (s) => `Below ${s} measured kills the dot stays hollow: the median is noise.`,
    rangeEmpty: 'No measured kill across the matches of this session.',
  },
}
