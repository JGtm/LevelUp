/**
 * SummaryTab — onglet Résumé (placeholder).
 *
 * Le contenu sera ajouté progressivement (KPIs détaillés, breakdowns,
 * highlights, etc.) en réutilisant les patterns Home (tuiles match avec
 * map_ui / mode_ui / playlist_ui pré-résolus côté Go) et match-view
 * (gamertag + is_bot pré-résolus côté Go).
 */
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { getPersonalStatsText } from '../i18n'

export function SummaryTab() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPersonalStatsText(locale)
  return (
    <EmptyStateCard
      title={t.empty.placeholderTitle}
      description={t.empty.placeholderDescription}
    />
  )
}
