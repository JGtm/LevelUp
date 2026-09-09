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
  opaciteDuPoint,
  PLANCHER_MORTS_SESSION,
  pointAttenue,
  tailleDuPoint,
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
 *  axes), la taille/opacité par point, et le point BRUT pour l'infobulle. */
interface EchartScatterDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: { opacity: number }
  raw: SquadIsolementPoint
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

  const echartsSeries = series.map((s, idx) => {
    const gamertag = (s.meta as { gamertag?: string } | undefined)?.gamertag ?? s.key
    const color = playerColors[gamertag] ?? resolveToken('info')
    const data: EchartScatterDatum[] = s.datapoints.map((p) => ({
      value: [p.part_isolee.taux * 100, p.couverture.taux * 100],
      symbolSize: tailleDuPoint(p.morts_examinees),
      itemStyle: { opacity: opaciteDuPoint(p) },
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
    return serie
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
        const p = (params as { data?: EchartScatterDatum }).data?.raw
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
        // L'opacité réduite (opaciteDuPoint) n'est pas un signal fiable à elle
        // seule (contraste, daltonisme) : le tooltip nomme explicitement la
        // réserve d'échantillon faible, par la forme unique du dépôt (`withLowSampleNote`,
        // séparateur HTML : le tooltip ECharts est du HTML).
        return withLowSampleNote(base, pointAttenue(p), t.lowSample, '<br/>')
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

