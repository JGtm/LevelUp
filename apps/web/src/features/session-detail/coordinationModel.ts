/**
 * coordinationModel — LES PROJECTIONS PURES de la carte « Appui reçu » de la colonne de session
 * (lot O, D22-6).
 *
 * Deux grandeurs, une seule forme : la JAUGE À PARITÉ (`UsageGaugeGrid`) doublée de la BANDE DE
 * RÉGULARITÉ match par match (`UsageRegularityBand`) — composants partagés, aucun graphe neuf.
 *
 * NORMALISATION (le point de D22-6, à ne pas relâcher) :
 *  - « frags appuyés » se rapporte aux FRAGS DU JOUEUR, « part des appuis de l’équipe » aux APPUIS DE L’ÉQUIPE :
 *    deux dénominateurs différents, que mélanger donnerait un nombre sans sens ;
 *  - la PARITÉ est `parity_pct` = 1/n avec n l'effectif du camp DU MATCH (R1), jamais 1/4.
 *
 * UNE CASE SANS DÉNOMINATEUR RESTE GRISE (ton `unmeasured`) : un match sans appui dans le
 * camp ne vaut pas 0 %. Son infobulle le dit tel quel (« aucun appui d'équipe »).
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
 * la PARITÉ 1/n pour « part des appuis de l’équipe », l'HABITUEL (la même mesure sur la période de
 * référence, `habituel_pct`, lot S) pour « on me prépare », qui ne se compare à aucune part
 * équitable. Le trait est le même ; ce qui
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
    tooltip,
  }
}

/** Les deux lignes de la carte « Appui reçu » : « frags appuyés », « part des appuis de l’équipe ». */
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
        // « Frags appuyés » ne se compare à aucune parité : son repère est l'HABITUEL de la
        // période de référence, quand le contrat le sert (lot S).
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
      return { matchId: p.match_id, tone: 'unmeasured', tooltip: t.bandTipNoTeamAssist(i + 1) }
    }
    const delta = share - parity
    return {
      matchId: p.match_id,
      tone: delta > BAND_EPSILON_PT ? 'above' : delta < -BAND_EPSILON_PT ? 'below' : 'near',
      tooltip: t.bandTipFmt(i + 1, formatUsagePct(share, locale), formatUsagePct(parity, locale)),
    }
  })
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
