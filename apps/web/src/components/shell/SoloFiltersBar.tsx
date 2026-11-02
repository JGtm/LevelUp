/**
 * SoloFiltersBar — bandeau des filtres analytiques des pages Stats (FilterOmnibar + rail période et
 * session), collant sous la ligne d'onglets de l'en-tête de page.
 *
 * Seules les pages qui montrent les filtres solo le rendent (règle `routeShowsSoloFilters`, source
 * unique partagée avec la résolution des filtres du layout joueur). Les onglets de section
 * ne sont plus ici : ils sont dans l'en-tête de page (`SectionHeader`, `PageHeader`). La section
 * Escouade gère sa propre barre de filtres dans SquadLayout.
 *
 * Les filtres (Session, Période, cascade) sont gérés par FilterOmnibar via des pills.
 */
import { FilterOmnibar } from './FilterOmnibar'
import { STICKY_BELOW_TABS_CLASS } from '@/components/foundations/pageHeaderLogic'
import { PeriodSessionRail } from './PeriodSessionRail'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRouterState } from '@tanstack/react-router'
import { routeShowsSoloFilters } from './shellNavigation'
import { playerRelativePath } from '@/lib/title-routing'

export function SoloFiltersBar() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const locale = useAppShellStore((s) => s.locale)
  if (!routeShowsSoloFilters(pathname)) return null

  // match_context='solo' : Timeseries/History ne concernent que les matchs solo.
  // Page Sessions : on active le sélecteur de contexte (Solo/Escouade/Mixte) —
  // les autres pages stats restent figées sur 'solo' (aucune régression).
  const isSessionsPage = /^\/stats\/sessions/.test(playerRelativePath(pathname) ?? '')
  return (
    <div
      className={`sticky ${STICKY_BELOW_TABS_CLASS} z-30 shrink-0 bg-background px-6`}
      role="navigation"
      aria-label={formatMessage(commonManifest, 'common.shell.nav_analytics_aria', locale)}
    >
      <FilterOmnibar matchContext="solo" contextSelectable={isSessionsPage} />
      <PeriodSessionRail />
    </div>
  )
}
