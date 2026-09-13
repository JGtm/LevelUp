/**
 * SquadEchangeTauxSessionCard — « Taux d'échange par session » (onglet Synergies).
 *
 * LA MÊME MESURE QUE « LE COMPTE », DÉCOUPÉE PAR SOIRÉE. Le serveur la sert déjà ainsi
 * (`SquadEchange.taux_par_session`, `teammates_squad_echange.tauxParSession`) : aucun taux
 * n'est recalculé ici, et une session dont aucun match n'est mesuré n'a PAS de point — elle
 * n'a pas un taux nul, elle n'a pas de taux.
 *
 * UNE SEULE SÉRIE, DONC PAS DE LÉGENDE : le titre la nomme. Le DERNIER point est appuyé et
 * étiqueté, les autres non — une valeur sur chaque point ferait d'une courbe un tableau.
 *
 * SOUS LE SEUIL DE SIGNIFICATIVITÉ, ON NE MONTRE PAS UNE COURBE, ON DIT POURQUOI. Deux ou
 * trois soirées ne font pas une évolution : la carte rend alors son état vide nommé, la
 * même doctrine que le « Constat du moment » (qui, lui, ne se rend pas du tout — la
 * différence est qu'une carte de suivi absente se lirait comme une donnée manquante).
 */
import { useMemo } from 'react'

import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { PLANCHER_SESSIONS_TENDANCE, tauxSessionSeries } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeTauxSessionCardProps {
  echange: SquadEchange
}

export function SquadEchangeTauxSessionCard({ echange }: SquadEchangeTauxSessionCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadEchangeText(locale)
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

  const secondes = echange.fenetre_ms / 1000
  const sessions = useMemo(() => echange.taux_par_session ?? [], [echange.taux_par_session])
  const series = useMemo(() => tauxSessionSeries(echange), [echange])

  const bornes = useMemo(() => {
    if (sessions.length === 0) return null
    const taux = sessions.map((p) => p.couverture.taux)
    return { bas: Math.min(...taux), haut: Math.max(...taux) }
  }, [sessions])

  const dernier = sessions.length > 0 ? sessions[sessions.length - 1] : null

  const footer = (
    <div className="border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">{t.sessionRateFoot}</p>
    </div>
  )

  const assezDeSessions = sessions.length >= PLANCHER_SESSIONS_TENDANCE

  return (
    <SectionCard title={t.sessionRateTitle} label={t.sessionRateLabel} footer={footer}>
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-taux-session">
        {!assezDeSessions || !bornes ? (
          <EmptyStateNotice
            title={t.sessionRateEmptyTitle}
            description={t.sessionRateEmptyDescription(PLANCHER_SESSIONS_TENDANCE)}
          />
        ) : (
          <>
            <p className="border-l-2 border-info pl-3 text-sm text-foreground">
              {t.sessionRateSay({
                bas: pctFmt.format(bornes.bas),
                haut: pctFmt.format(bornes.haut),
                taux: pctFmt.format(echange.couverture.taux),
                n: sessions.length,
              })}
            </p>
            <p className="text-xs text-muted-foreground">{t.sessionRateFigure(secondes)}</p>
            <TimeseriesLineChart
              series={series}
              xAxisType="category"
              outcomeMarkers={false}
              showLegend={false}
              height={280}
              yAxisLabel={t.sessionRateYAxis}
              yAxisLabelFormatter="{value} %"
              xAxisLabelRotate={-30}
              lastPointLabel={dernier ? pctFmt.format(dernier.couverture.taux) : undefined}
            />
          </>
        )}
      </div>
    </SectionCard>
  )
}
