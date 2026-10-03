/**
 * coordinationModel — LES PROJECTIONS PURES des cartes « Riposte » et « Appui reçu » de la
 * colonne de session (lot O, D22-1 / D22-6).
 *
 * Deux grandeurs par carte, une seule forme : la JAUGE À PARITÉ (`UsageGaugeGrid`) doublée
 * de la BANDE DE RÉGULARITÉ match par match (`UsageRegularityBand`) — les composants du
 * bloc « usages », réutilisés tels quels, aucun graphe neuf.
 *
 * NORMALISATION (le point de D22-1/6, à ne pas relâcher) :
 *  - « je riposte » se rapporte aux MORTS DE MON CAMP, jamais aux miennes ni à celles du
 *    lobby : un match où le camp meurt peu gonflerait sinon ma part ;
 *  - « on me prépare » se rapporte à MES FRAGS, « ma part des appuis » aux APPUIS DU CAMP :
 *    deux dénominateurs différents, que mélanger donnerait un nombre sans sens ;
 *  - la PARITÉ est `parity_pct` = 1/n avec n l'effectif du camp DU MATCH (R1), jamais 1/4.
 *
 * UNE CASE SANS DÉNOMINATEUR RESTE GRISE (`unmeasured`) — non mesuré n'est pas zéro : un
 * match sans mort de camp, ou sans appui dans le camp, ne vaut pas 0 %.
 *
 * Pur : aucun React, aucune couleur en dur, aucune lecture de store.
 */
import type { UsageGaugeModel, UsageGaugeRowModel } from '@/features/_shared/usage/usageGaugeModel'
import type { UsageBandCell } from '@/features/_shared/usage/usageRegularityBandModel'
import { formatUsagePct } from '@/features/_shared/usage/usageFormat'
import type { Couverture, CoordinationBlock, CoordinationMatchPoint } from '@/lib/api/types'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type { Locale } from '@/lib/i18n/locale'

import type { CoordinationText } from './coordinationI18n'

/** Sous ±ε points de la parité, une case est « à la parité » (ni bonus ni déficit). */
const BAND_EPSILON_PT = 1

/**
 * Une jauge de couverture : sa valeur, son repère, ses textes.
 *
 * `repere` est LE TRAIT de la jauge (`parityPct`), et il dit DEUX choses selon la grandeur :
 * la PARITÉ 1/n pour « je riposte » et « ma part des appuis », l'HABITUEL (la même mesure
 * sur la période de référence, `habituel_pct`, lot S) pour « je suis couvert » et « on me
 * prépare », qui ne se comparent à aucune part équitable. Le trait est le même ; ce qui
 * change est ce que l'infobulle en dit — `usuel` nomme le repère quand c'est un habituel.
 *
 * `null` reste possible des deux côtés (scope FFA, référence tautologique ou non mesurée) :
 * une jauge sans repère vaut mieux qu'un repère inventé.
 */
interface Repere {
  /** Position du trait sur le rail, en points de pourcentage. `null` = pas de trait. */
  pct: number | null
  /** Ce trait est un HABITUEL (et non la parité) : l'infobulle le nomme. */
  usuel?: boolean
}

function gaugeFromCouverture(
  key: string,
  couverture: Couverture | null | undefined,
  repere: Repere,
  t: CoordinationText,
  locale: Locale,
): UsageGaugeModel {
  // `n` nul = aucun dénominateur : la jauge n'a pas de valeur (jamais un 0 % inventé).
  const mesure = couverture != null && couverture.n > 0
  const valuePct = mesure ? couverture.taux * 100 : null
  const valueText = formatUsagePct(valuePct, locale)
  let tooltip = mesure
    ? withLowSampleNote(
        t.gaugeTipFmt(valueText, couverture.brut, couverture.n),
        couverture.echantillon_faible,
        t.lowSample,
      )
    : valueText
  if (repere.usuel === true && repere.pct != null) {
    tooltip = t.gaugeTipUsualFmt(tooltip, formatUsagePct(repere.pct, locale))
  }
  return {
    key,
    valuePct,
    parityPct: repere.pct,
    valueText: mesure
      ? withLowSampleNote(valueText, couverture.echantillon_faible, t.lowSample)
      : valueText,
    honestyText: mesure ? String(couverture.brut) : '—',
    tooltip,
    teammatesRatePct: null,
    opponentsRatePct: null,
  }
}

/** Les deux lignes de la carte « Riposte » : « je suis couvert », puis « je riposte ». */
export function buildRiposteGaugeRows(
  block: CoordinationBlock,
  t: CoordinationText,
  locale: Locale,
): UsageGaugeRowModel[] {
  const parity = block.riposte.parity_pct ?? null
  const usual = block.riposte.habituel_pct ?? null
  return [
    {
      key: 'riposte',
      label: t.cardRiposte,
      gauges: [
        // « Je suis couvert » ne se compare à aucune parité : son repère est l'HABITUEL
        // de la période de référence, quand le contrat le sert (lot S).
        gaugeFromCouverture('covered', block.riposte.je_suis_couvert, { pct: usual, usuel: true }, t, locale),
        gaugeFromCouverture('mine', block.riposte.je_riposte, { pct: parity }, t, locale),
      ],
    },
  ]
}

/** Les deux lignes de la carte « Appui reçu » : « on me prépare », « ma part des appuis ». */
export function buildAppuiGaugeRows(
  block: CoordinationBlock,
  t: CoordinationText,
  locale: Locale,
): UsageGaugeRowModel[] {
  const parity = block.appui.parity_pct ?? null
  const usual = block.appui.habituel_pct ?? null
  return [
    {
      key: 'appui',
      label: t.cardAppui,
      gauges: [
        // « On me prépare » : même règle que « je suis couvert » — le repère est l'habituel.
        gaugeFromCouverture('prepared', block.appui.on_me_prepare, { pct: usual, usuel: true }, t, locale),
        gaugeFromCouverture('share', block.appui.ma_part_des_appuis, { pct: parity }, t, locale),
      ],
    },
  ]
}

/** Ce qu'une case de bande lit sur un match : sa part, sa parité, son dénominateur. */
interface BandReader {
  share: (p: CoordinationMatchPoint) => number | null | undefined
  /** Le dénominateur du match : nul → case grise, quelle que soit la part. */
  denominator: (p: CoordinationMatchPoint) => number
}

const RIPOSTE_BAND: BandReader = {
  share: (p) => p.riposte_share_pct,
  denominator: (p) => p.team_deaths,
}

const APPUI_BAND: BandReader = {
  share: (p) => p.assist_share_of_team_pct,
  denominator: (p) => p.team_assists,
}

function buildBand(
  perMatch: readonly CoordinationMatchPoint[],
  reader: BandReader,
  t: CoordinationText,
  locale: Locale,
): UsageBandCell[] {
  return perMatch.map((p, i) => {
    const share = reader.share(p)
    const parity = p.parity_pct
    if (reader.denominator(p) <= 0 || share == null || parity == null) {
      return { matchId: p.match_id, tone: 'unmeasured', tooltip: t.bandTipUnmeasured(i + 1) }
    }
    const delta = share - parity
    return {
      matchId: p.match_id,
      tone: delta > BAND_EPSILON_PT ? 'above' : delta < -BAND_EPSILON_PT ? 'below' : 'near',
      tooltip: t.bandTipFmt(i + 1, formatUsagePct(share, locale), formatUsagePct(parity, locale)),
    }
  })
}

export function buildRiposteBand(
  perMatch: readonly CoordinationMatchPoint[],
  t: CoordinationText,
  locale: Locale,
): UsageBandCell[] {
  return buildBand(perMatch, RIPOSTE_BAND, t, locale)
}

export function buildAppuiBand(
  perMatch: readonly CoordinationMatchPoint[],
  t: CoordinationText,
  locale: Locale,
): UsageBandCell[] {
  return buildBand(perMatch, APPUI_BAND, t, locale)
}

/**
 * Le COMPTE NU à droite d'une bande (« 5/8 ») : les cases au-dessus de la parité sur les
 * cases MESURÉES. Aucune case mesurée → pas de compte (`null`), jamais « 0/0 ».
 */
export function bandCaption(cells: readonly UsageBandCell[], t: CoordinationText): string | null {
  const measured = cells.filter((c) => c.tone !== 'unmeasured')
  if (measured.length === 0) return null
  return t.bandCountFmt(measured.filter((c) => c.tone === 'above').length, measured.length)
}

/** Le délai médian de riposte, en secondes, déjà formaté. `null` quand non mesuré. */
export function formatDelaiMedian(
  block: CoordinationBlock,
  t: CoordinationText,
  locale: Locale,
): string | null {
  const ms = block.riposte.delai_median_ms
  if (ms == null || ms <= 0) return null
  const s = (ms / 1000).toFixed(1)
  return t.delaiFmt(locale === 'fr' ? s.replace('.', ',') : s)
}

/** La fenêtre de riposte en secondes, pour l'infobulle (« 5 »). */
export function fenetreSeconds(block: CoordinationBlock, locale: Locale): string {
  const s = (block.fenetre_ms / 1000).toFixed(1).replace(/[.,]0$/, '')
  return locale === 'fr' ? s.replace('.', ',') : s
}
