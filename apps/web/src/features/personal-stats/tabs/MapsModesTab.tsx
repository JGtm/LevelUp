/**
 * MapsModesTab — onglet Cartes & Modes (placeholder).
 *
 * Les libellés map / mode / playlist sont pré-résolus côté Go (champs DTO
 * map_ui / mode_ui / playlist_ui). Pas de résolution côté front.
 */
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { getPersonalStatsText } from '../i18n'

export function MapsModesTab() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPersonalStatsText(locale)
  return (
    <EmptyStateCard
      title={t.empty.placeholderTitle}
      description={t.empty.placeholderDescription}
    />
  )
}
