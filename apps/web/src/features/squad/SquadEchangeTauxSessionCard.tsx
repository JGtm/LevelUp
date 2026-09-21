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
 * SOUS LE PLANCHER DE TENDANCE, ON NE TRACE PAS DE COURBE, MAIS ON NE SE TAIT PLUS
 * (décision 5 du 2026-09-19) : la ou les soirées retenues se lisent FACE A L'HABITUEL, que
 * le contrat sert déjà (`echange.habituel`). Filtrer sur une soirée est l'usage NOMINAL de
 * la page : l'ancien état vide y laissait la carte muette.
 */
import { useMemo } from 'react'

import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { intlLocale } from '@/lib/formatters'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  comparaisonSessions,
  PLANCHER_SESSIONS_TENDANCE,
  tauxSessionSeries,
} from './squadEchange.logic'
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

  const lignes = useMemo(
    () => comparaisonSessions(echange, t.sessionRateSelection),
    [echange, t.sessionRateSelection],
  )

  const assezDeSessions = sessions.length >= PLANCHER_SESSIONS_TENDANCE

  return (
    <SectionCard
      title={t.sessionRateTitle}
      label={t.sessionRateLabel}
      // Ce que la courbe dénombre est passé en infobulle ⓘ du titre (2026-09-21) : c'est de
      // la méthode, elle ne se relit pas à chaque visite au-dessus du graphe.
      titleAdornment={(label) => (
        <span className="flex items-center gap-1.5">
          {label}
          <InfoTooltip content={t.sessionRateFigure(secondes)} />
        </span>
      )}
    >
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-taux-session">
        {!assezDeSessions || !bornes ? (
          <div className="space-y-2" data-testid="squad-echange-taux-session-vs-habituel">
            <p className="border-l-2 border-info pl-3 text-sm text-foreground">
              {t.sessionRateFewSay({ n: sessions.length, floor: PLANCHER_SESSIONS_TENDANCE })}
            </p>
            <dl className="divide-y divide-border rounded border border-border">
              {lignes.map((l) => (
                <div key={l.label} className="flex items-baseline justify-between px-3 py-2">
                  <dt className="text-sm text-foreground">{l.label}</dt>
                  <dd className="text-sm font-semibold tabular-nums text-foreground">
                    {withLowSampleNote(pctFmt.format(l.taux), l.echantillonFaible, t.lowSample)}
                  </dd>
                </div>
              ))}
              <div className="flex items-baseline justify-between bg-muted/40 px-3 py-2">
                <dt className="text-sm text-muted-foreground">
                  {t.sessionRateUsual}{' '}
                  <span className="text-2xs">{t.sessionRateUsualSub(echange.matchs_habituel)}</span>
                </dt>
                <dd className="text-sm font-semibold tabular-nums text-muted-foreground">
                  {pctFmt.format(echange.habituel.taux)}
                </dd>
              </div>
            </dl>
          </div>
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
              frameless
            />
          </>
        )}
      </div>
    </SectionCard>
  )
}
