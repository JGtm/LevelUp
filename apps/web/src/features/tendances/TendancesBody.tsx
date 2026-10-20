/**
 * TendancesBody — le corps de la page Tendances sous la barre de commandes : l'invite
 * « choisis une escouade », le chargement, l'erreur avec « Réessayer », ou les blocs de la
 * page (vides ou non) une fois la réponse reçue.
 */
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import type { Horizon, Step, TendancesView } from './tendances.logic'
import { TendancesCalendar } from './TendancesCalendar'
import { TendancesEvolution } from './TendancesEvolution'
import { TendancesHorizonBar } from './TendancesHorizonBar'
import { TendancesMatrix } from './TendancesMatrix'
import { TendancesMix } from './TendancesMix'
import { TendancesMedals, TendancesWinLoss } from './TendancesOutcomes'

interface TendancesContentProps {
  locale: Locale
  view: TendancesView
  data: TrendsPageResponse
  horizon: Horizon
  step: Step
  onHorizonChange: (horizon: Horizon) => void
  onStepChange: (step: Step) => void
}

/** Les blocs de la page une fois la réponse reçue, selon la vue. */
function TendancesContent({
  locale,
  view,
  data,
  horizon,
  step,
  onHorizonChange,
  onStepChange,
}: TendancesContentProps) {
  const t = getTendancesText(locale)
  if ((data.indicators ?? []).length === 0) {
    return <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
  }
  return (
    <>
      <TendancesMatrix locale={locale} data={data} />
      <TendancesHorizonBar locale={locale} horizon={horizon} onChange={onHorizonChange} />
      <TendancesEvolution
        locale={locale}
        data={data}
        horizon={horizon}
        step={step}
        view={view}
        onStepChange={onStepChange}
      />
      {view === 'solo' && (
        <>
          <TendancesCalendar locale={locale} data={data} horizon={horizon} />
          <TendancesWinLoss locale={locale} data={data} horizon={horizon} />
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <TendancesMedals locale={locale} data={data} horizon={horizon} />
            <TendancesMix locale={locale} data={data} horizon={horizon} step={step} />
          </div>
        </>
      )}
    </>
  )
}

export interface TendancesBodyProps extends Omit<TendancesContentProps, 'data'> {
  /** Vue Escouade sans coéquipier choisi : aucune requête, une invite. */
  squadWithoutTeammate: boolean
  data: TrendsPageResponse | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
}

export function TendancesBody({
  squadWithoutTeammate,
  isLoading,
  isError,
  onRetry,
  data,
  ...content
}: TendancesBodyProps) {
  const t = getTendancesText(content.locale)
  if (squadWithoutTeammate) {
    return <EmptyStateNotice title={t.squadEmptyTitle} description={t.squadEmptyDescription} />
  }
  if (isLoading) {
    return (
      <div className="flex justify-center p-6" data-testid="tendances-loading">
        <Spinner label={t.loading} />
      </div>
    )
  }
  if (isError || !data) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <p className="font-medium text-destructive">{t.error}</p>
          <button onClick={onRetry} className="mt-2 text-sm text-primary underline">
            {t.retry}
          </button>
        </CardContent>
      </Card>
    )
  }
  return <TendancesContent {...content} data={data} />
}
