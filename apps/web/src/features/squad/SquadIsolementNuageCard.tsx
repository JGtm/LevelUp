/**
 * SquadIsolementNuageCard — « Frags non ripostés » (section Coordination).
 *
 * UN PETIT POINT PAR MORT, UN GROS POINT PAR JOUEUR (contrat refondu le 2026-09-19,
 * PLAN_AJUSTEMENTS_PRE_V75 décision 4). En abscisse la distance au coéquipier visible le
 * plus proche RAPPORTÉE à la portée du radar du match (1,0 = à la portée, repère vertical
 * tracé) ; en ordonnée le délai avant la riposte, en secondes. Deux bandes nommées portent
 * ce qui n'a pas de valeur : « hors de vue » à droite (aucun coéquipier visible), « jamais
 * ripostée » en haut. Le gros point d'un joueur est la médiane de ses deux axes, sa taille
 * dit combien de morts.
 *
 * PAS LE WRAPPER `<ScatterChart>` GÉNÉRIQUE : ce nuage a besoin d'un encodage PAR POINT
 * (taille, opacité, infobulle) que `ChartSeries<ChartPointScatter>` ne porte pas — le
 * wrapper partagé n'expose qu'un `symbolSize` UNIFORME par série. Même pattern que
 * `FirstBloodLanes` (composer `<ChartCard>` directement avec un `buildOption` custom).
 *
 * La carte n'est pas montée quand la section est absente du contrat (cf.
 * SquadSynergiesPage) : une section omise n'est pas une section à zéro.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
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
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { intlLocale } from '@/lib/formatters'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type {
  SquadEchangeJoueur,
  SquadIsolementMort,
  SquadIsolementRepere,
  SquadNuageIsolement,
} from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadPlayerColors } from './colors'
import {
  delaiSecondes,
  echellesNuage,
  OPACITE_MORT,
  positionMort,
  positionRepere,
  ordreDessinReperes,
  repereAttenue,
  REPERE_PORTEE_RADAR,
  TAILLE_MORT,
  tailleRepere,
  type EchellesNuage,
} from './squadIsolement.logic'
import { getSquadIsolementText, type SquadIsolementText } from './squadIsolementStrings'

export interface SquadIsolementNuageCardProps {
  nuage: SquadNuageIsolement
  joueurs: SquadEchangeJoueur[]
}

export function SquadIsolementNuageCard({ nuage, joueurs }: SquadIsolementNuageCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadIsolementText(locale)
  const numLoc = intlLocale(locale)

  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [numLoc],
  )
  const numFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )

  const morts = useMemo(() => nuage.morts ?? [], [nuage.morts])
  const reperes = useMemo(() => nuage.reperes ?? [], [nuage.reperes])
  const vide = morts.length === 0

  const playerColors = useMemo(() => {
    const [main, ...teammates] = joueurs
    return getSquadPlayerColors(main?.gamertag ?? '', teammates.map((j) => j.gamertag))
  }, [joueurs])

  const series = useMemo(() => seriesParJoueur(morts, joueurs), [morts, joueurs])
  const echelles = useMemo(() => echellesNuage(morts), [morts])
  const repereParXUID = useMemo(() => {
    const map = new Map<string, SquadIsolementRepere>()
    for (const r of reperes) map.set(r.xuid, r)
    return map
  }, [reperes])

  // Les extrêmes RÉELS du roster : la taille des gros points s'y projette, et la légende
  // de taille les nomme. Une échelle absolue ne dirait plus rien au-delà de son plafond.
  const volumes = reperes.map((r) => r.nb_morts).filter((n) => n > 0)
  const mortsMin = volumes.length > 0 ? Math.min(...volumes) : 0
  const mortsMax = volumes.length > 0 ? Math.max(...volumes) : 0

  const buildOption = useMemo(
    () => (s: ChartSeries<SquadIsolementMort>[]) =>
      buildNuageOption(s, {
        playerColors,
        echelles,
        repereParXUID,
        t,
        pctFmt,
        numFmt,
        mortsMin,
        mortsMax,
      }),
    [playerColors, echelles, repereParXUID, t, pctFmt, numFmt, mortsMin, mortsMax],
  )

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      titleAdornment={titleWithInfo(<TooltipParagraphs items={[t.help, t.figure]} />)}
    >
      <div className="space-y-2 px-3 py-2" data-testid="squad-isolement-nuage">
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <>
            <ChartCard series={series} buildOption={buildOption} height={380} frameless />
            {/* LÉGENDE DE TAILLE, en DOM : ECharts n'en a pas pour un encodage de taille.
                Elle porte les VRAIES valeurs extrêmes du roster — un encodage qu'on ne
                nomme pas ne se lit pas. */}
            <div
              className="flex flex-wrap items-center gap-4 text-2xs text-muted-foreground"
              data-testid="squad-isolement-legende-taille"
            >
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full bg-muted-foreground/50"
                  style={{ width: TAILLE_MIN_LEGENDE, height: TAILLE_MIN_LEGENDE }}
                />
                {t.legendDeaths(mortsMin)}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full bg-muted-foreground/50"
                  style={{ width: TAILLE_MAX_LEGENDE, height: TAILLE_MAX_LEGENDE }}
                />
                {t.legendDeaths(mortsMax)}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full bg-muted-foreground/50"
                  style={{ width: TAILLE_MORT, height: TAILLE_MORT }}
                />
                {t.legendDeath}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full border border-dashed border-muted-foreground"
                  style={{ width: TAILLE_MIN_LEGENDE, height: TAILLE_MIN_LEGENDE }}
                />
                {t.legendLowSample(nuage.plancher_echantillon_faible)}
              </span>
            </div>
          </>
        )}
      </div>
    </SectionCard>
  )
}

/** Une série ECharts PAR JOUEUR du roster — la couleur et le nom viennent de la série,
 *  pas du point (cohérent avec le reste de la page : un joueur = une couleur stable). */
function seriesParJoueur(
  morts: SquadIsolementMort[],
  joueurs: SquadEchangeJoueur[],
): ChartSeries<SquadIsolementMort>[] {
  return joueurs
    .map((j) => ({
      key: j.xuid,
      meta: { gamertag: j.gamertag },
      datapoints: morts.filter((m) => m.xuid === j.xuid),
    }))
    .filter((s) => s.datapoints.length > 0)
}

interface BuildOpts {
  playerColors: Record<string, string>
  echelles: EchellesNuage
  repereParXUID: Map<string, SquadIsolementRepere>
  t: SquadIsolementText
  pctFmt: Intl.NumberFormat
  numFmt: Intl.NumberFormat
  /** Extrêmes RÉELS du nombre de morts par joueur — l'échelle de taille des gros points. */
  mortsMin: number
  mortsMax: number
}

/** Diamètres (px) des pastilles de la légende de taille — les deux bouts de l'échelle. */
const TAILLE_MIN_LEGENDE = 12
const TAILLE_MAX_LEGENDE = 22

/** Un point de la donnée ECharts : la position [x, y] bandes comprises, et la mort BRUTE
 *  pour l'infobulle. */
interface EchartMortDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: Record<string, unknown>
  raw: SquadIsolementMort
}

/** Le GROS point médian d'un joueur : une agrégation, pas une mort — son infobulle est
 *  distincte de celle d'un petit point (`EchartMortDatum`). */
interface EchartRepereDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: Record<string, unknown>
  label: Record<string, unknown>
  repereRaw: SquadIsolementRepere
}

function buildNuageOption(
  series: ChartSeries<SquadIsolementMort>[],
  opts: BuildOpts,
): EChartsCoreOption {
  const { playerColors, echelles, repereParXUID, t, pctFmt, numFmt, mortsMin, mortsMax } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const warningColor = resolveToken('warning')

  // DEUX PASSES, ET C'EST L'ORDRE DE DESSIN QUI L'IMPOSE (retour utilisateur du
  // 2026-09-21). Les repères médians sont émis APRÈS tous les nuages de morts, et entre eux
  // par TAILLE DÉCROISSANTE (`ordreDessinReperes`) : ECharts dessine la dernière série
  // au-dessus, donc le plus petit repère finit au premier plan au lieu d'être recouvert par
  // celui d'un coéquipier plus exposé.
  const reperesSeries: Record<string, unknown>[] = []
  const nuageSeries = series.flatMap((s, idx) => {
    const gamertag = (s.meta as { gamertag?: string } | undefined)?.gamertag ?? s.key
    const color = playerColors[gamertag] ?? resolveToken('info')
    const data: EchartMortDatum[] = s.datapoints.map((m) => ({
      value: positionMort(m, echelles),
      symbolSize: TAILLE_MORT,
      itemStyle: { color, opacity: OPACITE_MORT },
      raw: m,
    }))
    const serie: Record<string, unknown> = {
      type: 'scatter',
      name: gamertag,
      data,
      itemStyle: { color },
    }
    // Le repère de portée du radar et les deux bandes nommées sont posés UNE SEULE FOIS,
    // sur la première série : ce sont des overlays du graphe entier, pas d'une série.
    if (idx === 0) {
      serie.markLine = {
        silent: true,
        symbol: 'none',
        lineStyle: { type: 'dashed', color: warningColor },
        label: {
          show: true,
          formatter: t.radarLine,
          color: warningColor,
          fontSize: 10,
          position: 'insideEndTop',
        },
        data: [{ xAxis: REPERE_PORTEE_RADAR }],
      }
      serie.markArea = {
        silent: true,
        label: { show: true, fontSize: 10, color: tc.axisLabel },
        data: [
          [
            {
              coord: [echelles.distance.bandeDebut, 0],
              name: t.bandOutOfSight,
              itemStyle: { color: tc.axisLine, opacity: 0.08 },
              label: { position: 'insideTop' },
            },
            { coord: [echelles.distance.max, echelles.delai.max] },
          ],
          [
            {
              coord: [0, echelles.delai.bandeDebut],
              name: t.bandNever,
              itemStyle: { color: tc.axisLine, opacity: 0.08 },
              label: { position: 'insideTopLeft' },
            },
            { coord: [echelles.distance.bandeDebut, echelles.delai.max] },
          ],
        ],
      }
    }
    // Gros point médian du joueur : une seconde série ECharts, MÊME nom (partage l'entrée
    // de légende — toggler l'un cache l'autre) et MÊME couleur que la série des morts,
    // mais nettement plus grande et étiquetée du gamertag. `z` la pose au-dessus du nuage.
    const repere = repereParXUID.get(s.key)
    const repereSerie: Record<string, unknown> | null = repere
      ? {
          type: 'scatter',
          name: gamertag,
          z: 5,
          data: [
            {
              value: positionRepere(repere, echelles),
              symbolSize: tailleRepere(repere.nb_morts, mortsMin, mortsMax),
              itemStyle: repereAttenue(repere)
                ? { color: 'transparent', borderColor: color, borderWidth: 2, borderType: 'dashed' }
                : { color, borderColor: tc.card, borderWidth: 2 },
              label: {
                show: true,
                formatter: gamertag,
                position: 'top',
                color: tc.axisLabel,
                fontSize: 10,
                fontWeight: 600,
              },
              repereRaw: repere,
            } satisfies EchartRepereDatum,
          ],
        }
      : null
    if (repereSerie) reperesSeries.push(repereSerie)
    return [serie]
  })
  // Le tri se fait sur la MESURE (`nb_morts`), pas sur le diamètre calculé : c'est la
  // grandeur que `tailleRepere` traduit, et elle est monotone.
  const reperesTries = ordreDessinReperes(
    reperesSeries.map((r) => (r.data as EchartRepereDatum[])[0].repereRaw),
  )
  const parXuid = new Map(reperesSeries.map((r) => [(r.data as EchartRepereDatum[])[0].repereRaw.xuid, r]))
  const echartsSeries = [
    ...nuageSeries,
    ...reperesTries.map((r) => parXuid.get(r.xuid)).filter((r) => r != null),
  ]

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
          const gamertag = (s.meta as { gamertag?: string } | undefined)?.gamertag ?? s.key
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
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: echelles.delai.max,
      name: t.yAxis,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value} s' },
    },
    series: echartsSeries,
  }
}

/** L'infobulle d'un point : une mort, ou le repère médian d'un joueur. */
function formatTooltip(
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
  const delai = delaiSecondes(m)
  const delay = delai == null ? t.tooltipNever : t.tooltipRiposted(numFmt.format(delai))
  const couverture =
    m.distance_ratio == null
      ? t.tooltipDeathOutOfSight
      : t.tooltipDeathCoverage(numFmt.format(m.distance_ratio))
  return [escapeHtml(m.gamertag ?? ''), couverture, delay].join('<br/>')
}
