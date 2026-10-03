/**
 * TimeseriesRangeRolesCard — « Rôles de portée » sur l'onglet Résumé (a.1, D23-a du
 * 2026-09-22).
 *
 * À QUELLE DISTANCE JE JOUE, MATCH APRÈS MATCH, ET CE QUE ÇA A CHANGÉ. Un point par match :
 * en abscisse le match, du plus ancien au plus récent ; en ordonnée l'écart de ma médiane de
 * frag à celle de TOUS les joueurs du lobby — la seule échelle qui neutralise la carte et le
 * mode (une arène tourne autour de 11 m, un BTB autour de 24 m : en mètres bruts, la courbe
 * ne dessine que la playlist de la soirée).
 *
 * ELLE NE REMPLACE PAS « PORTÉE PAR ARME », elle répond à l'autre question. La carte
 * existante dit avec quoi je tire et à quelle distance chaque arme porte ; celle-ci dit quel
 * joueur je suis devenu sur la fenêtre — un rôle, une dérive, des écarts. Elle se pose juste
 * après elle, dans la même famille de sujet, sous la même capability produit `weapon_range`.
 *
 * LE COMPOSANT EST CELUI DE L'ESCOUADE, à une seule série (`squadRangeRolesChart` +
 * `squadRangeRoles.logic` : `PLANCHER_MESURE`, `FENETRE_ROLE`, `roleDeEcart`). Même axe,
 * mêmes bandes, mêmes étiquettes « #N · carte » — c'est voulu : les deux pages doivent se
 * lire l'une après l'autre sans réapprentissage. Import cross-feature durable
 * (`timeseries=>squad`, déjà déclaré). Seule la légende des séries tombe : un nuage à un
 * joueur n'a personne à nommer.
 *
 * D22-VERBOSITÉ (LOI) : graphe et légendes seulement. La lecture tient dans l'infobulle du
 * titre, en trois phrases ; sous le graphe, rien d'autre que la légende et la couverture.
 */
import { useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { resolveToken, tokenCssVar } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { MatchRangeBlock } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildSquadRangeRolesOption } from '@/features/squad/charts/squadRangeRolesChart'
import {
  categoriesMatchs,
  FENETRE_ROLE,
  ordonnerProfils,
  PLANCHER_MESURE,
  seriesPortee,
  seuilsRoles,
  TAILLE_POINT_MIN,
} from '@/features/squad/squadRangeRoles.logic'

import { getTimeseriesRangeRolesText } from './timeseriesRangeRolesStrings'

export interface TimeseriesRangeRolesCardProps {
  bloc: MatchRangeBlock | undefined
}

export function TimeseriesRangeRolesCard({ bloc }: TimeseriesRangeRolesCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getTimeseriesRangeRolesText(locale)
  const numLoc = intlLocale(locale)

  const numFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )

  const profils = useMemo(() => ordonnerProfils(bloc?.profiles ?? []), [bloc?.profiles])
  const categories = useMemo(() => categoriesMatchs(profils), [profils])
  // Le scope Timeseries ne sert QUE le joueur consulté dans `players` (lot U). La coupe à
  // une série est une ceinture : un jour où le producteur servirait le lobby entier, la
  // carte resterait solo au lieu de virer au nuage d'inconnus.
  const series = useMemo(() => seriesPortee(profils).slice(0, 1), [profils])
  const seuils = useMemo(() => seuilsRoles(series), [series])

  const encre = resolveToken('chart-series-1')

  // Les extrêmes RÉELS de `measured` sur la fenêtre : la taille des points s'y projette.
  const mesures = series.flatMap((s) => s.points).map((p) => p.mesures)
  const mesuresMin = mesures.length > 0 ? Math.min(...mesures) : 0
  const mesuresMax = mesures.length > 0 ? Math.max(...mesures) : 0

  const chartSeries = useMemo(
    () => (series.length > 0 ? [{ key: 'timeseries-portee', datapoints: series }] : []),
    [series],
  )

  const buildOption = useMemo(
    () => () =>
      buildSquadRangeRolesOption(series, {
        categories,
        seuils,
        couleurs: {},
        couleurDefaut: encre,
        legende: false,
        mesuresMin,
        mesuresMax,
        libelles: {
          xAxis: t.xAxis,
          yAxis: t.yAxis,
          lobbyLine: t.lobbyLine,
          bandes: t.bandes,
          tooltipMedian: t.tooltipMedian,
          tooltipDelta: t.tooltipDelta,
          tooltipMeasured: t.tooltipMeasured,
        },
        fmtM: (v: number) => numFmt.format(v),
      }),
    [series, categories, seuils, encre, mesuresMin, mesuresMax, t, numFmt],
  )

  const vide = series.length === 0

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      titleAdornment={titleWithInfo(<TooltipParagraphs items={[t.help(PLANCHER_MESURE)]} />)}
    >
      <div className="space-y-2 px-3 py-2" data-testid="timeseries-portee-roles">
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <>
            <ChartCard series={chartSeries} buildOption={buildOption} height={340} frameless />
            {/* Les trois encodages qu'ECharts ne sait pas nommer : le point, le point creux
                et la tendance. Les bandes, elles, portent leur nom dans le graphe. */}
            <div
              className="flex flex-wrap items-center gap-4 text-2xs text-muted-foreground"
              data-testid="timeseries-portee-legende"
            >
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full"
                  style={{
                    width: TAILLE_POINT_MIN,
                    height: TAILLE_POINT_MIN,
                    backgroundColor: tokenCssVar('chart-series-1'),
                  }}
                />
                {t.legendMatch}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full border border-dashed border-muted-foreground"
                  style={{ width: TAILLE_POINT_MIN, height: TAILLE_POINT_MIN }}
                />
                {t.legendLowSample(PLANCHER_MESURE)}
              </span>
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-px w-5 bg-muted-foreground" />
                {t.legendTrend(FENETRE_ROLE)}
              </span>
            </div>
            <p className="text-2xs text-muted-foreground" data-testid="timeseries-portee-couverture">
              {t.coverage(bloc?.kills_measured ?? 0, bloc?.kills_total ?? 0)}
            </p>
          </>
        )}
      </div>
    </SectionCard>
  )
}
