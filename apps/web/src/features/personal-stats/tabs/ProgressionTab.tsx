/**
 * ProgressionTab — onglet Progression (placeholder).
 */
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { getPersonalStatsText } from '../i18n'

export function ProgressionTab() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPersonalStatsText(locale)
  return (
    <EmptyStateCard
      title={t.empty.placeholderTitle}
      description={t.empty.placeholderDescription}
    />
  )
}
