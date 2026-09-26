/**
 * squadIsolementNuageOption — la construction de l'option ECharts du nuage « Frags non
 * ripostés ». Logique PURE (aucun React) : `SquadIsolementNuageCard` ne fait plus que le
 * rendu et la légende DOM.
 *
 * POURQUOI CE FICHIER (2026-09-22) : la carte dépassait le seuil des 500 lignes de
 * CLAUDE.md. La frontière retenue est celle qui se teste : tout ce qui décide de la figure
 * (séries, styles, axes, repères, infobulle) vit ici et se vérifie sans monter un composant.
 *
 * ENCRE PLEINE, ET LA FORME DIT L'ÉTAT (décision utilisateur du 2026-09-22). Aucune opacité
 * inférieure à 1 dans cette option : ni sur les points, ni sur les repères médians, ni sur
 * les deux bandes nommées — le rendu atténué rendait la figure terne. Ce que l'opacité
 * codait est repris par la TAILLE (un petit point pour une mort, un gros pour la médiane
 * d'un joueur) et par la SUPERPOSITION (les repères sont émis en dernier, du plus gros au
 * plus petit). Une mort dont le tueur est tombé hors fenêtre ne se dessine plus évidée mais
 * en LOSANGE, dans la même encre et à la même taille : la forme porte l'état, la couleur
 * reste celle du joueur — le seul encodage de couleur du graphe.
 *
 * PAS LE WRAPPER `<ScatterChart>` GÉNÉRIQUE : ce nuage a besoin d'un encodage PAR POINT
 * (forme, taille, infobulle) que `ChartSeries<ChartPointScatter>` ne porte pas.
 */
import type { EChartsCoreOption } from 'echarts/core'

import type { ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SquadIsolementMort, SquadIsolementRepere } from '@/lib/api/types'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'

import {
  delaiSecondes,
  etatMort,
  graduationsDelai,
  positionMort,
  positionRepere,
  ordreDessinReperes,
  repereAttenue,
  REPERE_PORTEE_RADAR,
  TAILLE_MORT,
  tailleRepere,
  type EchellesNuage,
} from './squadIsolement.logic'
import type { SquadIsolementText } from './squadIsolementStrings'

/**
 * LA FORME DIT L'ÉTAT (décision du 2026-09-22). Un disque plein : le tueur est tombé dans
 * la fenêtre, c'est une riposte. Un LOSANGE plein, même encre et même taille : il est tombé
 * après — le point reste pleinement visible, c'est sa silhouette qui tranche. Le contour
 * évidé d'avant se lisait comme un point effacé.
 */
export const SYMBOLE_MORT = 'circle'
export const SYMBOLE_MORT_HORS_FENETRE = 'diamond'

export interface BuildNuageOpts {
  playerColors: Record<string, string>
  echelles: EchellesNuage
  repereParXUID: Map<string, SquadIsolementRepere>
  t: SquadIsolementText
  pctFmt: Intl.NumberFormat
  numFmt: Intl.NumberFormat
  /** Extrêmes RÉELS du nombre de morts par joueur — l'échelle de taille des gros points. */
  mortsMin: number
  mortsMax: number
  /** Format des graduations de l'axe log : 0,2 · 0,5 · 1 … en séparateur de la locale. */
  secFmt: Intl.NumberFormat
  /** La fenêtre de riposte du CONTRAT (`fenetre_ms`), en secondes — jamais un 5 en dur. */
  fenetreS: number
}

/** Un point de la donnée ECharts : la position [x, y] bandes comprises, la FORME qui dit
 *  l'état, et la mort BRUTE pour l'infobulle. */
export interface EchartMortDatum {
  value: [number, number]
  symbol: string
  symbolSize: number
  itemStyle: Record<string, unknown>
  raw: SquadIsolementMort
}

/** Le GROS point médian d'un joueur : une agrégation, pas une mort — son infobulle est
 *  distincte de celle d'un petit point (`EchartMortDatum`). */
export interface EchartRepereDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: Record<string, unknown>
  label: Record<string, unknown>
  repereRaw: SquadIsolementRepere
}

/** Une série ECharts PAR JOUEUR du roster — la couleur et le nom viennent de la série,
 *  pas du point (cohérent avec le reste de la page : un joueur = une couleur stable). */
export function seriesParJoueur(
  morts: SquadIsolementMort[],
  joueurs: { xuid: string; gamertag: string }[],
): ChartSeries<SquadIsolementMort>[] {
  return joueurs
    .map((j) => ({
      key: j.xuid,
      meta: { gamertag: j.gamertag },
      datapoints: morts.filter((m) => m.xuid === j.xuid),
    }))
    .filter((s) => s.datapoints.length > 0)
}

function gamertagDeSerie(s: ChartSeries<SquadIsolementMort>): string {
  return (s.meta as { gamertag?: string } | undefined)?.gamertag ?? s.key
}

/** Les deux repères posés UNE SEULE FOIS, sur la première série : ce sont des overlays du
 *  graphe entier, pas d'une série. ECharts n'accepte qu'un `markLine` par série, donc la
 *  portée du radar (vertical) et la fenêtre de riposte (horizontal) le partagent — d'où un
 *  formatter PAR ENTRÉE, une chaîne fixe écraserait le texte de l'autre. */
function markLineOverlays(t: SquadIsolementText, fenetreS: number): Record<string, unknown> {
  const warningColor = resolveToken('warning')
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed', color: warningColor },
    label: {
      show: true,
      formatter: (pt: { name?: string }) => pt.name ?? '',
      color: warningColor,
      fontSize: 10,
      position: 'insideEndTop',
    },
    data: [
      { xAxis: REPERE_PORTEE_RADAR, name: t.radarLine },
      {
        yAxis: fenetreS,
        name: t.windowLine(fenetreS),
        // `rotate: 0` : sans lui l'étiquette suit l'inclinaison de la ligne (le piège
        // mesuré sur `HistogramChart`).
        label: { position: 'insideEndTop', rotate: 0 },
      },
    ],
  }
}

/**
 * Les deux bandes nommées — « hors de vue » à droite, « jamais ripostée » en haut.
 *
 * LE BAS DES BANDES EST LE PLANCHER DE L'AXE, PLUS ZÉRO : un axe log n'a pas de zéro, et
 * une aire ancrée à 0 ne se dessinerait pas du tout. ENCRE PLEINE (décision du
 * 2026-09-22) : l'ancien `opacity: 0.08` ne codait rien d'autre qu'une atténuation, et les
 * bandes restent DERRIÈRE le nuage (un `markArea` a `z: 0`, une série `z: 2`).
 */
function markAreaBandes(
  t: SquadIsolementText,
  echelles: EchellesNuage,
  fondBande: string,
  couleurLabel: string,
): Record<string, unknown> {
  return {
    silent: true,
    label: { show: true, fontSize: 10, color: couleurLabel },
    data: [
      [
        {
          coord: [echelles.distance.bandeDebut, echelles.delai.min],
          name: t.bandOutOfSight,
          itemStyle: { color: fondBande },
          label: { position: 'insideTop' },
        },
        { coord: [echelles.distance.max, echelles.delai.max] },
      ],
      [
        {
          coord: [0, echelles.delai.bandeDebut],
          name: t.bandNever,
          itemStyle: { color: fondBande },
          label: { position: 'insideTopLeft' },
        },
        { coord: [echelles.distance.bandeDebut, echelles.delai.max] },
      ],
    ],
  }
}

/** Le gros point médian d'un joueur : une seconde série ECharts, MÊME nom (partage l'entrée
 *  de légende — toggler l'un cache l'autre) et MÊME couleur que la série des morts, mais
 *  nettement plus grande et étiquetée du gamertag. `z` la pose au-dessus du nuage. */
function serieRepere(
  repere: SquadIsolementRepere,
  gamertag: string,
  color: string,
  opts: BuildNuageOpts,
  couleurSeparation: string,
  couleurLabel: string,
): Record<string, unknown> {
  // ENCRE PLEINE MÊME POUR UN ÉCHANTILLON FAIBLE : le repère n'est plus évidé, c'est sa
  // bordure TIRETÉE (dans l'encre de séparation, jamais une teinte neuve) qui porte la
  // réserve — la même grammaire que la pastille tiretée de la légende, et l'infobulle le
  // dit en toutes lettres.
  const itemStyle = repereAttenue(repere)
    ? { color, borderColor: couleurSeparation, borderWidth: 2, borderType: 'dashed' }
    : { color, borderColor: couleurSeparation, borderWidth: 2 }
  return {
    type: 'scatter',
    name: gamertag,
    z: 5,
    data: [
      {
        value: positionRepere(repere, opts.echelles),
        symbolSize: tailleRepere(repere.nb_morts, opts.mortsMin, opts.mortsMax),
        itemStyle,
        label: {
          show: true,
          formatter: gamertag,
          position: 'top',
          color: couleurLabel,
          fontSize: 10,
          fontWeight: 600,
        },
        repereRaw: repere,
      } satisfies EchartRepereDatum,
    ],
  }
}

/**
 * Les séries ECharts du nuage, dans leur ORDRE DE DESSIN : d'abord un nuage de morts par
 * joueur, puis les repères médians, du plus gros au plus petit.
 */
function seriesDuNuage(
  series: ChartSeries<SquadIsolementMort>[],
  opts: BuildNuageOpts,
  tc: ReturnType<typeof getEChartsThemeColors>,
): Record<string, unknown>[] {
  const { playerColors, echelles, repereParXUID, t, fenetreS } = opts
  // DEUX PASSES, ET C'EST L'ORDRE DE DESSIN QUI L'IMPOSE (retour utilisateur du
  // 2026-09-21). Les repères médians sont émis APRÈS tous les nuages de morts, et entre eux
  // par TAILLE DÉCROISSANTE (`ordreDessinReperes`) : ECharts dessine la dernière série
  // au-dessus, donc le plus petit repère finit au premier plan au lieu d'être recouvert par
  // celui d'un coéquipier plus exposé. C'est CETTE superposition, et la taille, qui portent
  // la hiérarchie depuis que toute opacité est à 1.
  const reperesSeries: Record<string, unknown>[] = []
  const nuageSeries = series.flatMap((s, idx) => {
    const gamertag = gamertagDeSerie(s)
    const color = playerColors[gamertag] ?? resolveToken('info')
    // LA FORME DIT L'ÉTAT, LA COULEUR DIT LE JOUEUR. Disque plein = le tueur est tombé dans
    // la fenêtre (une riposte) ; LOSANGE plein = il est tombé après, donc sans riposte —
    // même encre, même taille, silhouette différente. Aucune teinte neuve, aucune
    // atténuation : la couleur de ce graphe appartient au joueur et à lui seul.
    const data: EchartMortDatum[] = s.datapoints.map((m) => ({
      value: positionMort(m, echelles),
      symbol: etatMort(m) === 'horsFenetre' ? SYMBOLE_MORT_HORS_FENETRE : SYMBOLE_MORT,
      symbolSize: TAILLE_MORT,
      itemStyle: { color },
      raw: m,
    }))
    const serie: Record<string, unknown> = {
      type: 'scatter',
      name: gamertag,
      data,
      itemStyle: { color },
    }
    if (idx === 0) {
      serie.markLine = markLineOverlays(t, fenetreS)
      serie.markArea = markAreaBandes(t, echelles, tc.axisLine, tc.axisLabel)
    }
    const repere = repereParXUID.get(s.key)
    if (repere) {
      reperesSeries.push(serieRepere(repere, gamertag, color, opts, tc.card, tc.axisLabel))
    }
    return [serie]
  })
  // Le tri se fait sur la MESURE (`nb_morts`), pas sur le diamètre calculé : c'est la
  // grandeur que `tailleRepere` traduit, et elle est monotone.
  const reperesTries = ordreDessinReperes(
    reperesSeries.map((r) => (r.data as EchartRepereDatum[])[0].repereRaw),
  )
  const parXuid = new Map(
    reperesSeries.map((r) => [(r.data as EchartRepereDatum[])[0].repereRaw.xuid, r]),
  )
  return [
    ...nuageSeries,
    ...reperesTries.map((r) => parXuid.get(r.xuid)).filter((r) => r != null),
  ]
}

export function buildNuageOption(
  series: ChartSeries<SquadIsolementMort>[],
  opts: BuildNuageOpts,
): EChartsCoreOption {
  const { playerColors, echelles, t, pctFmt, numFmt, secFmt } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const graduations = graduationsDelai(echelles.delai)

  return {
    backgroundColor: CHART_BG,
    // `bottom: 78` : il faut la place du nom d'axe X (nameGap 32) ET de la légende, qui se
    // pose au ras du bas comme sur tous les autres graphes.
    grid: { top: 24, bottom: 78, left: 56, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown) =>
        formatTooltip((params as { data?: EchartMortDatum | EchartRepereDatum }).data, {
          t,
          pctFmt,
          numFmt,
        }),
    },
    // Socle de légende COMMUN à tous les graphes de l'app (`getLegendBase` : en pied,
    // pastille et texte au même gabarit).
    legend: {
      ...getLegendBase(tc),
      data: legendEntries(
        series.map((s) => {
          const gamertag = gamertagDeSerie(s)
          return { name: gamertag, color: playerColors[gamertag] ?? resolveToken('info') }
        }),
      ),
    },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: echelles.distance.max,
      name: t.xAxis,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value}×' },
    },
    // AXE DES DÉLAIS EN LOG (décision utilisateur du 2026-09-22) : la zone qui porte le
    // sens — les ripostes sous la fenêtre de 5 s — occupait le douzième bas d'un axe
    // linéaire monté jusqu'à la minute. Il court du plancher d'affichage au plafond du
    // contrat, plus la bande réservée. `customValues` impose les durées qu'un lecteur
    // reconnaît : livré à lui-même, un log ne gradue qu'aux puissances de dix.
    yAxis: {
      ...axis,
      type: 'log',
      logBase: 10,
      min: echelles.delai.min,
      max: echelles.delai.max,
      name: t.yAxis,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: {
        ...axis.axisLabel,
        customValues: graduations,
        // `{value}` ne suffit plus : un log rendrait « 0.2 » brut, sans séparateur
        // décimal de la locale. Le « s » est le symbole SI, le même dans les deux langues
        // (et l'unité est déjà nommée par le titre d'axe).
        formatter: (v: number) => `${secFmt.format(v)} s`,
      },
      splitLine: { ...axis.splitLine, customValues: graduations },
    },
    series: seriesDuNuage(series, opts, tc),
  }
}

/** L'infobulle d'un point : une mort, ou le repère médian d'un joueur. */
export function formatTooltip(
  data: EchartMortDatum | EchartRepereDatum | undefined,
  opts: { t: SquadIsolementText; pctFmt: Intl.NumberFormat; numFmt: Intl.NumberFormat },
): string {
  const { t, pctFmt, numFmt } = opts
  if (!data) return ''
  // LE RETOUR À LA LIGNE EST POSÉ ICI, jamais dans le message : un `<br>` isolé dans une
  // chaîne ICU fait échouer le parseur, qui rend alors le gabarit brut à l'écran.
  if ('repereRaw' in data) {
    const r = data.repereRaw
    const base = [
      t.tooltipRepereHead({ gamertag: escapeHtml(r.gamertag), n: r.nb_morts }),
      t.tooltipRepereIso(pctFmt.format(r.part_isolee.taux)),
      t.tooltipRepereCov(pctFmt.format(r.couverture.taux)),
    ].join('<br/>')
    return withLowSampleNote(base, repereAttenue(r), t.lowSample, '<br/>')
  }
  const m = data.raw
  if (!m) return ''
  // TROIS ÉTATS, TROIS PHRASES. Un tueur tombé à 60 s porte bien un délai, et l'infobulle
  // le dit — mais elle tranche aussi : hors fenêtre, ce n'est pas une riposte.
  const delai = delaiSecondes(m)
  const etat = etatMort(m)
  let delay: string
  if (etat === 'jamais' || delai == null) {
    delay = t.tooltipNever
  } else if (etat === 'horsFenetre') {
    delay = t.tooltipOutOfWindow(numFmt.format(delai))
  } else {
    delay = t.tooltipRiposted(numFmt.format(delai))
  }
  const couverture =
    m.distance_ratio == null
      ? t.tooltipDeathOutOfSight
      : t.tooltipDeathCoverage(numFmt.format(m.distance_ratio))
  return [escapeHtml(m.gamertag ?? ''), couverture, delay].join('<br/>')
}
