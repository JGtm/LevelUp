/**
 * SessionRangeCard — « Portée des engagements » de la colonne de session (lot W, D23-4).
 *
 * LE NUAGE DE LA PÉRIODE À UN SEUL JOUEUR, SESSION EN SURBRILLANCE. L'axe X porte les matchs
 * de ma période de référence (`range_reference`, lot U), du plus ancien au plus récent ; en
 * ordonnée, mon écart à la médiane du lobby de chaque match — la seule échelle qui neutralise
 * la carte et le mode. Les matchs de la session affichée sont surlignés et peints à l'encre
 * du joueur ; le reste de la période reste en gris. C'est le concept de l'Escouade (lot R) à
 * une seule série : SITUER UN POINT DANS SA POPULATION.
 *
 * POURQUOI PAS LES BÂTONS DU LOT O : leurs bandes de rôle étaient les tiers de la SESSION,
 * donc chaque soirée redéfinissait ses rôles et deux soirées n'étaient pas comparables. Ici
 * les bandes sont les SEUILS SERVIS de la période (`role_low_m` / `role_high_m`) — les mêmes
 * que l'Escouade, jamais recalculés côté web ; absents, il n'y a pas de bandes.
 *
 * ELLE SUIT LA SECTION COORDINATION plutôt que d'y entrer : le contrat ne sert PAS de
 * miroir `compare_coordination`, alors qu'il sert `compare_range_profiles`. Fondues en une
 * seule clé de section, les deux cartes seraient remplacées par un placeholder dans la
 * colonne comparée alors que la portée, elle, a ses données. Deux clés, deux rangées.
 *
 * EN COMPARAISON, LES DEUX COLONNES PARTAGENT LE MÊME NUAGE (un seul `range_reference` : la
 * référence dépend du filtre, pas de la session) et chacune y surligne SA session — c'est
 * exactement la lecture demandée : deux soirées situées dans la même population.
 *
 * D22-VERBOSITÉ (LOI) : graphe, légende et couverture. La lecture tient dans l'infobulle
 * (i) du titre, trois phrases.
 */
import { useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { buildSquadRangeRolesOption } from '@/features/squad/charts/squadRangeRolesChart'
import { getSquadPlayerColors } from '@/features/squad/colors'
import { resolveToken, tokenCssVar } from '@/lib/accessibility'
import type { MatchRangeBlock, RangeReferenceBlock } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { COORDINATION_TEXT } from './coordinationI18n'
import {
  FENETRE_ROLE,
  PLANCHER_MESURE,
  TAILLE_POINT_MIN,
  nuagePortee,
  pointDeLaSession,
} from './sessionRange.logic'

export function SessionRangeCard({
  block,
  reference,
  meLabel,
  compact = false,
  yDomain,
}: {
  /** Le bloc `range_profiles` de la réponse — les matchs de CETTE session (surbrillance). */
  block: MatchRangeBlock | null | undefined
  /**
   * La période de référence du joueur (`range_reference`) — l'axe du nuage. Absente
   * (vieux serveur, ou période tautologique) : repli sur les seuls matchs de la session,
   * sans bandes ni surbrillance.
   */
  reference?: RangeReferenceBlock | null
  /** Libellé du joueur de la page (slug de route) — départage un profil multi-joueurs. */
  meLabel: string
  /** Colonne divisée : même graphe, plus court. */
  compact?: boolean
  /** Bornes d'axe Y plancher partagées A/B (mode comparaison) — figées par la page. */
  yDomain?: [number, number]
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = COORDINATION_TEXT[locale]
  const numFmt = useMemo(
    () =>
      new Intl.NumberFormat(intlLocale(locale), {
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [locale],
  )

  const nuage = useMemo(() => nuagePortee(reference, block, meLabel), [reference, block, meLabel])
  const serie = nuage.serie

  // L'encre du joueur consulté — la même source que la pill et les graphes de l'Escouade.
  const couleurs = useMemo(
    () => (serie ? getSquadPlayerColors(serie.gamertag, []) : {}),
    [serie],
  )
  const encreSession = serie ? (couleurs[serie.gamertag] ?? resolveToken('info')) : ''
  // Les matchs hors session n'ont pas d'identité à porter : ils sont la population, pas un
  // camp — `zone-neutral` est achromatique dans toutes les palettes.
  const encrePeriode = resolveToken('zone-neutral')

  const chartSeries = useMemo(
    () => (serie ? [{ key: 'session-portee-periode', datapoints: serie.points }] : []),
    [serie],
  )

  const buildOption = useMemo(
    () => () => {
      if (!serie) return {}
      const mesures = serie.points.map((p) => p.mesures)
      return buildSquadRangeRolesOption([serie], {
        categories: nuage.categories,
        seuils: nuage.seuils,
        couleurs,
        mesuresMin: mesures.length > 0 ? Math.min(...mesures) : 0,
        mesuresMax: mesures.length > 0 ? Math.max(...mesures) : 0,
        yDomain,
        masquerLegende: true,
        encrePoint: (p) => (pointDeLaSession(nuage, p) ? encreSession : encrePeriode),
        surbrillance: nuage.surbrillance
          ? { ...nuage.surbrillance, label: t.rangeThisSession, couleur: encreSession }
          : undefined,
        libelles: {
          xAxis: nuage.periode ? t.rangeXAxisPeriod : t.rangeXAxisSession,
          yAxis: t.rangeAxis,
          lobbyLine: t.rangeLobbyLine,
          bandes: { front: t.roleFront, polyvalent: t.roleVersatile, sniper: t.roleSniper },
          tooltipMedian: t.rangeTipMedian,
          tooltipDelta: t.rangeTipDelta,
          tooltipMeasured: t.rangeTipMeasured,
        },
        fmtM: (v: number) => numFmt.format(v),
      })
    },
    [serie, nuage, couleurs, encreSession, encrePeriode, yDomain, t, numFmt],
  )

  if (block == null) return null
  const vide = serie == null || serie.points.length === 0

  return (
    <SectionCard
      title={t.cardRange}
      label={t.cardRange}
      titleAdornment={titleWithInfo(
        <TooltipParagraphs
          items={[t.infoRange1, t.infoRange2, t.infoRange3(PLANCHER_MESURE)]}
        />,
      )}
    >
      <div className="space-y-2 px-3 py-2" data-testid="session-portee">
        {vide ? (
          <EmptyStateNotice title={t.cardRange} description={t.rangeEmpty} />
        ) : (
          <>
            <ChartCard
              series={chartSeries}
              buildOption={buildOption}
              height={compact ? 300 : 360}
              frameless
            />
            {/* Les quatre encodages qu'ECharts ne nomme pas : les deux appartenances, la
                tendance et le point creux. */}
            <div
              className="flex flex-wrap items-center gap-4 text-2xs text-muted-foreground"
              data-testid="session-portee-legende"
            >
              {nuage.periode && (
                <span className="flex items-center gap-1.5">
                  <span
                    className="inline-block rounded-full"
                    style={{
                      width: TAILLE_POINT_MIN,
                      height: TAILLE_POINT_MIN,
                      backgroundColor: tokenCssVar('zone-neutral'),
                    }}
                  />
                  {t.rangeLegendPeriod}
                </span>
              )}
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full"
                  style={{
                    width: TAILLE_POINT_MIN,
                    height: TAILLE_POINT_MIN,
                    backgroundColor: tokenCssVar('squad-player-1'),
                  }}
                />
                {t.rangeLegendSession}
              </span>
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-px w-5 bg-muted-foreground" />
                {t.rangeLegendTrend(FENETRE_ROLE)}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full border border-dashed border-muted-foreground"
                  style={{ width: TAILLE_POINT_MIN, height: TAILLE_POINT_MIN }}
                />
                {t.rangeLowSample(PLANCHER_MESURE)}
              </span>
            </div>
            <p className="text-2xs text-muted-foreground" data-testid="session-portee-couverture">
              {nuage.periode && reference
                ? t.coverageMatchesFmt(reference.matches_measured, reference.matches_total)
                : t.rangeCoverageFmt(block.kills_measured, block.kills_total)}
            </p>
          </>
        )}
      </div>
    </SectionCard>
  )
}
