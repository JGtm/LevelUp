/**
 * SquadIsolementNuageCard — « Isolement et couverture » (onglet Synergies, item 7.7).
 *
 * Un point par (joueur, session) : en abscisse la part de morts sans coéquipier visible à
 * portée du radar, en ordonnée le taux d'échange de la MÊME session (`squadEchange.logic`
 * mesure déjà ce taux pour tout le camp ; ici c'est la même mesure, par joueur et par
 * session — cf. `domain.SquadIsolementPoint` côté contrat). Taille du point = morts
 * examinées, couleur = joueur (`getSquadPlayerColors`, mêmes tokens que le reste de la
 * page). Deux lignes pointillées tracent les MÉDIANES du nuage et découpent quatre
 * quadrants nommés — libellés repris à l'identique de la maquette
 * echange-escouade.html.
 *
 * PAS LE WRAPPER `<ScatterChart>` GÉNÉRIQUE : ce nuage a besoin d'un encodage PAR POINT
 * (taille, opacité, infobulle) que `ChartSeries<ChartPointScatter>` ne porte pas — le
 * wrapper partagé n'expose qu'un `symbolSize` UNIFORME par série. Même pattern que
 * `FirstBloodLanes` (composer `<ChartCard>` directement avec un `buildOption` custom)
 * plutôt que de modifier un wrapper partagé et ses tests pour un seul consommateur.
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
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { intlLocale } from '@/lib/formatters'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type { SquadEchangeJoueur, SquadNuageIsolement, SquadIsolementPoint } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadPlayerColors } from './colors'
import {
  medianesNuage,
  PLANCHER_MORTS_SESSION,
  pointAttenue,
  pointMedianJoueur,
  quadrantDuPoint,
  tailleDuPoint,
  tailleMedianeDuPoint,
  type MedianesNuage,
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

  const points = useMemo(() => nuage.points ?? [], [nuage.points])
  const vide = points.length === 0

  const playerColors = useMemo(() => {
    const [main, ...teammates] = joueurs
    return getSquadPlayerColors(main?.gamertag ?? '', teammates.map((j) => j.gamertag))
  }, [joueurs])

  const series = useMemo(() => seriesParJoueur(points, joueurs), [points, joueurs])
  const medianes = useMemo(() => medianesNuage(points), [points])

  const buildOption = useMemo(
    () => (s: ChartSeries<SquadIsolementPoint>[]) =>
      buildNuageOption(s, { playerColors, medianes, t, pctFmt }),
    [playerColors, medianes, t, pctFmt],
  )

  // Planchers et définition en infobulle ⓘ plutôt qu'en pied de carte (retour utilisateur
  // 2026-09-09) : deux paragraphes de texte gris sous le nuage, lus une fois puis jamais,
  // qui poussaient le graphe suivant hors de l'écran.
  const help = (
    <span className="space-y-1.5">
      <span className="block">{t.floor(PLANCHER_MORTS_SESSION, nuage.plancher_echantillon_faible)}</span>
      <span className="block">{t.definition}</span>
    </span>
  )

  return (
    <SectionCard
      title={t.sectionTitle}
      label={t.sectionLabel}
      titleAdornment={(label) => (
        <span className="flex items-center gap-1.5">
          {label}
          <InfoTooltip content={help} />
        </span>
      )}
    >
      <div className="px-3 py-2" data-testid="squad-isolement-nuage">
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <ChartCard series={series} buildOption={buildOption} height={340} />
        )}
      </div>
    </SectionCard>
  )
}

/** Une série ECharts PAR JOUEUR du roster — la couleur et le nom viennent de la série,
 *  pas du point (cohérent avec le reste de la page : un joueur = une couleur stable). */
function seriesParJoueur(
  points: SquadIsolementPoint[],
  joueurs: SquadEchangeJoueur[],
): ChartSeries<SquadIsolementPoint>[] {
  return joueurs
    .map((j) => ({
      key: j.xuid,
      meta: { gamertag: j.gamertag },
      datapoints: points.filter((p) => p.xuid === j.xuid),
    }))
    .filter((s) => s.datapoints.length > 0)
}

interface BuildOpts {
  playerColors: Record<string, string>
  medianes: MedianesNuage | null
  t: SquadIsolementText
  pctFmt: Intl.NumberFormat
}

/** Un point de la donnée ECharts : la valeur [x,y] en POURCENTS (0..100, lisible sur les
 *  axes), le style par point, et le point BRUT pour l'infobulle. */
interface EchartScatterDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: Record<string, unknown>
  raw: SquadIsolementPoint
}

/** Le GROS point médian d'un joueur (D4, lot C3) : une agrégation, pas un match — son
 *  tooltip et son étiquette sont distincts d'un point de session (`EchartScatterDatum`). */
interface EchartMedianDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: Record<string, unknown>
  label: Record<string, unknown>
  medianRaw: { gamertag: string; mortsExaminees: number; isolement: number; couverture: number }
}

/** Style d'un point de session : cercle plein à l'échantillon suffisant, cercle POINTILLÉ
 *  (bordure en tirets, pas de remplissage) à l'échantillon faible — décision C3, remplace
 *  l'ancienne opacité réduite (`opaciteDuPoint`, retirée, plus fiable pour le contraste). */
function itemStyleDuPoint(point: SquadIsolementPoint, color: string): Record<string, unknown> {
  if (pointAttenue(point)) {
    return { color: 'transparent', borderColor: color, borderWidth: 1.5, borderType: 'dashed' }
  }
  return { color }
}

function buildNuageOption(
  series: ChartSeries<SquadIsolementPoint>[],
  opts: BuildOpts,
): EChartsCoreOption {
  const { playerColors, medianes, t, pctFmt } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const warningColor = resolveToken('warning')

  const echartsSeries = series.flatMap((s, idx) => {
    const gamertag = (s.meta as { gamertag?: string } | undefined)?.gamertag ?? s.key
    const color = playerColors[gamertag] ?? resolveToken('info')
    const data: EchartScatterDatum[] = s.datapoints.map((p) => ({
      value: [p.part_isolee.taux * 100, p.couverture.taux * 100],
      symbolSize: tailleDuPoint(p.morts_examinees),
      itemStyle: itemStyleDuPoint(p, color),
      raw: p,
    }))
    const serie: Record<string, unknown> = {
      type: 'scatter',
      name: gamertag,
      data,
      itemStyle: { color },
    }
    // Les deux médianes + le quadrant d'alerte sont posés UNE SEULE FOIS, sur la
    // première série : ce sont des overlays du graphe entier, pas d'une série.
    if (idx === 0 && medianes) {
      const mx = medianes.isolement * 100
      const my = medianes.couverture * 100
      serie.markLine = {
        silent: true,
        symbol: 'none',
        lineStyle: { type: 'dashed', color: tc.axisLine },
        label: { show: false },
        data: [{ xAxis: mx }, { yAxis: my }],
      }
      serie.markArea = {
        silent: true,
        label: { show: true, fontSize: 10, color: tc.axisLabel },
        data: [
          [
            { coord: [0, my], name: t.quadrant('procheCouvert'),
              itemStyle: { color: 'transparent' }, label: { position: 'insideTopLeft' } },
            { coord: [mx, 100] },
          ],
          [
            { coord: [mx, my], name: t.quadrant('loinCouvert'),
              itemStyle: { color: 'transparent' }, label: { position: 'insideTopRight' } },
            { coord: [100, 100] },
          ],
          [
            { coord: [0, 0], name: t.quadrant('procheSeul'),
              itemStyle: { color: 'transparent' }, label: { position: 'insideBottomLeft' } },
            { coord: [mx, my] },
          ],
          [
            { coord: [mx, 0], name: t.quadrant('loinSansSecours'),
              itemStyle: { color: warningColor, opacity: 0.08 },
              label: { position: 'insideBottomRight', color: warningColor, fontWeight: 600 } },
            { coord: [100, my] },
          ],
        ],
      }
    }
    // Gros point médian du joueur (D4) : une seconde série ECharts, MÊME nom (partage
    // l'entrée de légende — toggler l'un cache l'autre) et MÊME couleur que la série de
    // session, mais nettement plus grande et étiquetée du gamertag. `z` la pose au-dessus
    // du nuage de points pour qu'elle ne se fasse pas recouvrir par une session voisine.
    const medianAgg = pointMedianJoueur(s.datapoints)
    const medianSerie: Record<string, unknown> | null = medianAgg
      ? {
          type: 'scatter',
          name: gamertag,
          z: 5,
          data: [
            {
              value: [medianAgg.isolement * 100, medianAgg.couverture * 100],
              symbolSize: tailleMedianeDuPoint(medianAgg.mortsExaminees),
              itemStyle: { color, borderColor: tc.card, borderWidth: 2 },
              label: {
                show: true,
                formatter: gamertag,
                position: 'top',
                color: tc.axisLabel,
                fontSize: 10,
                fontWeight: 600,
              },
              medianRaw: { gamertag, ...medianAgg },
            } satisfies EchartMedianDatum,
          ],
        }
      : null
    return medianSerie ? [serie, medianSerie] : [serie]
  })

  return {
    backgroundColor: CHART_BG,
    // `bottom: 78` : il faut la place du nom d'axe X (nameGap 32) ET de la légende, qui se
    // pose au ras du bas comme sur tous les autres graphes.
    grid: { top: 24, bottom: 78, left: 56, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown) => {
        const data = (params as { data?: EchartScatterDatum | EchartMedianDatum }).data
        if (!data) return ''
        if ('medianRaw' in data) {
          const mr = data.medianRaw
          return t.tooltipMedian({
            gamertag: escapeHtml(mr.gamertag),
            n: mr.mortsExaminees,
            isoRate: pctFmt.format(mr.isolement),
            covRate: pctFmt.format(mr.couverture),
          })
        }
        const p = data.raw
        if (!p) return ''
        const base = t.tooltip({
          gamertag: escapeHtml((p.gamertag ?? '') as string),
          session: escapeHtml(p.session_label ?? ''),
          isoRate: pctFmt.format(p.part_isolee.taux),
          isoBrut: p.morts_isolees,
          isoN: p.morts_examinees,
          covRate: pctFmt.format(p.couverture.taux),
          covBrut: p.couverture.brut,
          covN: p.couverture.n,
        })
        // `quadrantDuPoint` rebranché (lot C3) : le tooltip d'un point NOMME le quadrant
        // auquel il appartient, en plus des quatre libellés déjà affichés dans les coins
        // (markArea ci-dessus). Rien à nommer sans médianes calculées (nuage vide — cas
        // déjà exclu plus haut, mais `medianes` reste nullable au niveau du type).
        const withQuadrant = medianes ? `${base}<br/>${t.pointQuadrant(quadrantDuPoint(p, medianes))}` : base
        // L'opacité réduite n'est plus utilisée pour l'échantillon faible (remplacée par un
        // cercle pointillé, décision C3) : le tooltip reste la seule mention textuelle
        // fiable (contraste, daltonisme), par la forme unique du dépôt (`withLowSampleNote`,
        // séparateur HTML : le tooltip ECharts est du HTML).
        return withLowSampleNote(withQuadrant, pointAttenue(p), t.lowSample, '<br/>')
      },
    },
    // Socle de légende COMMUN à tous les graphes de l'app (`getLegendBase` : en pied,
    // pastille et texte au même gabarit). Elle portait jusqu'ici sa propre mise en forme,
    // et se lisait donc autrement que partout ailleurs (retour utilisateur 2026-09-09).
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
      max: 100,
      name: t.xAxis,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value}%' },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      name: t.yAxis,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value}%' },
    },
    series: echartsSeries,
  }
}

