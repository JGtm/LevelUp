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
  OPACITE_SESSION,
  plafondAxe,
  PLANCHER_MORTS_SESSION,
  pointAttenue,
  pointMedianJoueur,
  quadrantDuPoint,
  tailleMedianeEchelle,
  TAILLE_SESSION,
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

  // Les extremes REELS du nuage : la taille des gros points s'y projette, et la legende de
  // taille les nomme. Une echelle absolue ne dirait plus rien au-dela de son plafond.
  const totaux = useMemo(
    () => series.map((s) => pointMedianJoueur(s.datapoints)?.mortsExaminees ?? 0).filter((n) => n > 0),
    [series],
  )
  const mortsMin = totaux.length > 0 ? Math.min(...totaux) : 0
  const mortsMax = totaux.length > 0 ? Math.max(...totaux) : 0

  // La phrase du haut compare les DEUX EXTREMES d'isolement du roster : le plus expose et
  // le moins expose. Sur un seul joueur, il n'y a rien a opposer.
  const contraste = useMemo(() => contrasteIsolement(series), [series])

  // Les axes s'ajustent aux donnees (bas ancre a zero) : bornes a 100 % en dur, la moitie
  // du canvas restait vide et les trois reperes se chevauchaient.
  const maxX = useMemo(() => plafondAxe(points.map((p) => p.part_isolee.taux)), [points])
  const maxY = useMemo(() => plafondAxe(points.map((p) => p.couverture.taux)), [points])

  const buildOption = useMemo(
    () => (s: ChartSeries<SquadIsolementPoint>[]) =>
      buildNuageOption(s, { playerColors, medianes, t, pctFmt, mortsMin, mortsMax, maxX, maxY }),
    [playerColors, medianes, t, pctFmt, mortsMin, mortsMax, maxX, maxY],
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

  const footer = (
    <div className="space-y-1 border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">{t.footRadar}</p>
      <p className="text-xs text-muted-foreground">
        {t.footDenominator(nuage.plancher_echantillon_faible)}
      </p>
    </div>
  )

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      footer={footer}
      titleAdornment={(label) => (
        <span className="flex items-center gap-1.5">
          {label}
          <InfoTooltip content={help} />
        </span>
      )}
    >
      <div className="space-y-2 px-3 py-2" data-testid="squad-isolement-nuage">
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <>
            {/* La ligne narrative vit AU-DESSUS du graphe, jamais en dessous. */}
            {contraste && (
              <p className="border-l-2 border-info pl-3 text-sm text-foreground">
                {t.say({
                  loin: contraste.loin.gamertag,
                  loinIso: pctFmt.format(contraste.loin.isolement),
                  loinCouv: pctFmt.format(contraste.loin.couverture),
                  proche: contraste.proche.gamertag,
                  procheIso: pctFmt.format(contraste.proche.isolement),
                  procheCouv: pctFmt.format(contraste.proche.couverture),
                })}
              </p>
            )}
            <p className="text-xs text-muted-foreground">{t.figure}</p>
            <ChartCard series={series} buildOption={buildOption} height={380} />
            {/* LEGENDE DE TAILLE, en DOM : ECharts n'en a pas pour un encodage de taille.
                Elle porte les VRAIES valeurs extremes du roster — un encodage qu'on ne
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
                  style={{ width: TAILLE_SESSION, height: TAILLE_SESSION }}
                />
                {t.legendSession}
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
  /** Extremes REELS des morts examinees par joueur — l'echelle de taille des gros points. */
  mortsMin: number
  mortsMax: number
  /** Plafonds des axes en pourcents (le bas reste a zero). */
  maxX: number
  maxY: number
}

/** Diametres (px) des pastilles de la legende de taille — les deux bouts de l'echelle. */
const TAILLE_MIN_LEGENDE = 12
const TAILLE_MAX_LEGENDE = 22

/** Le joueur le PLUS et le MOINS expose hors radar — les deux bouts de la phrase du haut. */
interface ContrasteIsolement {
  loin: { gamertag: string; isolement: number; couverture: number }
  proche: { gamertag: string; isolement: number; couverture: number }
}

/**
 * contrasteIsolement designe les deux extremes d'isolement du roster.
 *
 * `null` sous deux joueurs agregeables : une phrase qui compare a besoin de deux termes, et
 * un roster d'un seul joueur n'oppose rien.
 */
function contrasteIsolement(
  series: ChartSeries<SquadIsolementPoint>[],
): ContrasteIsolement | null {
  const agreges = series
    .map((s) => {
      const agg = pointMedianJoueur(s.datapoints)
      if (!agg) return null
      const gamertag = (s.meta as { gamertag?: string } | undefined)?.gamertag ?? s.key
      return { gamertag, isolement: agg.isolement, couverture: agg.couverture }
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
  if (agreges.length < 2) return null
  const trie = [...agreges].sort((a, b) => b.isolement - a.isolement)
  return { loin: trie[0], proche: trie[trie.length - 1] }
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
    return {
      color: 'transparent',
      borderColor: color,
      borderWidth: 1.5,
      borderType: 'dashed',
      opacity: OPACITE_SESSION,
    }
  }
  return { color, opacity: OPACITE_SESSION }
}

function buildNuageOption(
  series: ChartSeries<SquadIsolementPoint>[],
  opts: BuildOpts,
): EChartsCoreOption {
  const { playerColors, medianes, t, pctFmt, mortsMin, mortsMax, maxX, maxY } = opts
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
      // TAILLE FIXE, opacite 0,45 : les sessions disent la DISPERSION, jamais le volume.
      // Un seul encodage de taille par graphe, et c'est celui du gros point par joueur.
      symbolSize: TAILLE_SESSION,
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
            { coord: [mx, maxY] },
          ],
          [
            { coord: [mx, my], name: t.quadrant('loinCouvert'),
              itemStyle: { color: 'transparent' }, label: { position: 'insideTopRight' } },
            { coord: [maxX, maxY] },
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
            { coord: [maxX, my] },
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
              symbolSize: tailleMedianeEchelle(medianAgg.mortsExaminees, mortsMin, mortsMax),
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
      max: maxX,
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
      max: maxY,
      name: t.yAxis,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value}%' },
    },
    series: echartsSeries,
  }
}

