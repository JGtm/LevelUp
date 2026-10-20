/**
 * TendancesHorizonBar — la barre « Horizon » : UN seul horizon (7, 30, 90 ou 365 jours) pour
 * tous les blocs situés sous la matrice, chacun comparé à la période d'avant de même durée.
 *
 * Collante : gabarit de `features/ascension/campaign/CampaignTracker` (`sticky top-2 z-10`,
 * carte bordée). L'état vit dans `TendancesTab` ; la barre ne fait que l'afficher.
 */
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { HORIZONS, type Horizon } from './tendances.logic'
import { TendancesSegmented } from './TendancesSegmented'

export interface TendancesHorizonBarProps {
  locale: Locale
  horizon: Horizon
  onChange: (horizon: Horizon) => void
}

export function TendancesHorizonBar({ locale, horizon, onChange }: TendancesHorizonBarProps) {
  const t = getTendancesText(locale)
  const options = HORIZONS.map((days) => ({ value: days, label: t.horizonShort(days) }))
  return (
    <section
      className="sticky top-2 z-10 flex flex-wrap items-center gap-3 rounded-lg border border-border bg-card px-4 py-2 shadow-sm"
      data-testid="tendances-horizon-bar"
    >
      <h2 className="flex items-center gap-1.5 text-sm font-semibold uppercase text-muted-foreground">
        {t.horizonLabel}
        <InfoTooltip content={t.horizonInfo} />
      </h2>
      <TendancesSegmented
        options={options}
        value={horizon}
        onChange={onChange}
        ariaLabel={t.horizonAria}
      />
    </section>
  )
}
