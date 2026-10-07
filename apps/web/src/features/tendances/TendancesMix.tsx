/**
 * TendancesMix — la carte « Matchs par type de partie » : un bâton empilé par intervalle du
 * pas courant sur l'horizon, un composant par type de partie.
 *
 * Seules les briques existantes : `SectionCard` + `titleWithInfo` pour la carte (le wrapper
 * `BarStackedChart` n'accepte qu'un titre texte), le wrapper sans cadre dont la légende est
 * en bas, l'infobulle sans les zéros et l'état vide du cadre.
 */
import { useMemo } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { gameTypeLabel } from './labels'
import type { Horizon, Step } from './tendances.logic'
import { buildMixChart } from './tendancesMix.logic'

/** Hauteur du graphique, en pixels (celle des graphiques d'« Évolution »). */
const CHART_HEIGHT = 280

export interface TendancesMixProps {
  locale: Locale
  data: TrendsPageResponse
  horizon: Horizon
  step: Step
}

export function TendancesMix({ locale, data, horizon, step }: TendancesMixProps) {
  const t = getTendancesText(locale)
  const chart = useMemo(
    () =>
      buildMixChart({
        data,
        horizon,
        step,
        locale,
        labelOf: (key) => gameTypeLabel(key, locale),
      }),
    [data, horizon, step, locale],
  )
  return (
    <SectionCard title={t.mixTitle} titleAdornment={titleWithInfo(t.mixInfo)}>
      {/* LE GRAPHE REMPLIT LE BLOC (`fluid`) : la cellule voisine (« Médailles par match »)
          fixe souvent la hauteur de la rangée ; sans cela, le graphe restait collé en haut d'un
          bloc étiré, sa légende loin du bas. CHART_HEIGHT devient le minimum. */}
      <div className="flex min-h-0 flex-1 flex-col" data-testid="tendances-mix">
        <BarStackedChart
          series={chart.series}
          height={CHART_HEIGHT}
          frameless
          fluid
          componentColors={chart.componentColors}
          componentOrder={chart.componentOrder}
          tooltipHideZero
          emptyMessage={t.mixEmptyMessage}
        />
      </div>
    </SectionCard>
  )
}
