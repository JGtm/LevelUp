/**
 * SessionRangeCard — « Portée des engagements » de la colonne de session (lot O, D22-4).
 *
 * UN BÂTON PAR MATCH DE LA SESSION, dans l'ordre chronologique, étiqueté « #N · carte »
 * comme les autres graphes par match. Sa hauteur est MON ÉCART à la médiane du lobby du
 * match : la seule échelle qui neutralise la carte et le mode (D22-4 amende la §4 de la
 * maquette, qui proposait des bâtons p10→p90 par classe d'arme en absolu). Le zéro est la
 * ligne du lobby ; trois bandes de fond portent les rôles, leurs seuils sont les TIERS des
 * bâtons pleins de la session, pas des mètres en dur.
 *
 * ELLE SUIT LA SECTION COORDINATION plutôt que d'y entrer : le contrat ne sert PAS de
 * miroir `compare_coordination`, alors qu'il sert `compare_range_profiles`. Fondues en une
 * seule clé de section, les deux cartes seraient remplacées par un placeholder dans la
 * colonne comparée alors que la portée, elle, a ses données. Deux clés, deux rangées.
 *
 * PAS DE DÉTAIL PAR CLASSE D'ARME : `MatchRangeBlock` ne sert qu'une médiane par (match,
 * joueur), sans ventilation par arme ni par classe — il n'y a rien à replier.
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
import { tokenCssVar } from '@/lib/accessibility'
import type { MatchRangeBlock } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildSessionRangeOption } from './charts/sessionRangeChart'
import { COORDINATION_TEXT } from './coordinationI18n'
import {
  batonsPortee,
  medianeSession,
  seuilsSession,
  PLANCHER_MESURE,
} from './sessionRange.logic'

export function SessionRangeCard({
  block,
  meLabel,
  compact = false,
  yDomain,
}: {
  /** Le bloc `range_profiles` de la réponse — absent (vieux serveur) : rien ne se rend. */
  block: MatchRangeBlock | null | undefined
  /** Libellé du joueur de la page (slug de route) — départage un profil multi-joueurs. */
  meLabel: string
  /** Colonne divisée : même graphe, plus court. */
  compact?: boolean
  /** Bornes d'axe Y partagées A/B (mode comparaison) — figées par la page. */
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

  const { batons, categories } = useMemo(() => batonsPortee(block, meLabel), [block, meLabel])
  const seuils = useMemo(() => seuilsSession(batons), [batons])
  const mediane = useMemo(() => medianeSession(batons), [batons])

  const chartSeries = useMemo(
    () => (batons.length > 0 ? [{ key: 'session-portee', datapoints: batons }] : []),
    [batons],
  )

  const fmtM = useMemo(() => (v: number) => `${numFmt.format(v)} m`, [numFmt])
  const fmtSigne = useMemo(
    () => (v: number) => `${v > 0 ? '+' : ''}${numFmt.format(v)} m`,
    [numFmt],
  )

  const buildOption = useMemo(
    () => () =>
      buildSessionRangeOption(batons, {
        categories,
        seuils,
        medianeM: mediane,
        yDomain,
        libelles: {
          yAxis: t.rangeAxis,
          lobbyLine: t.rangeLobbyLine,
          medianLine: t.rangeSessionMedian,
          bandes: { front: t.roleFront, polyvalent: t.roleVersatile, sniper: t.roleSniper },
          tooltip: (b) =>
            t.rangeTipFmt(fmtM(b.medianeM), fmtM(b.lobbyM), fmtSigne(b.ecartM), b.mesures),
        },
      }),
    [batons, categories, seuils, mediane, yDomain, t, fmtM, fmtSigne],
  )

  if (block == null) return null
  const vide = batons.length === 0

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
              height={compact ? 260 : 320}
              frameless
            />
            {/* Les trois encodages qu'ECharts ne nomme pas : les deux verdicts et le creux. */}
            <div
              className="flex flex-wrap items-center gap-4 text-2xs text-muted-foreground"
              data-testid="session-portee-legende"
            >
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block h-2.5 w-4"
                  style={{ backgroundColor: tokenCssVar('divergent-pos') }}
                />
                {t.rangeAbove}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block h-2.5 w-4"
                  style={{ backgroundColor: tokenCssVar('divergent-neg') }}
                />
                {t.rangeBelow}
              </span>
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-2.5 w-4 border border-dashed border-muted-foreground" />
                {t.rangeLowSample(PLANCHER_MESURE)}
              </span>
            </div>
            <p className="text-2xs text-muted-foreground" data-testid="session-portee-couverture">
              {t.rangeCoverageFmt(block.kills_measured, block.kills_total)}
            </p>
          </>
        )}
      </div>
    </SectionCard>
  )
}
