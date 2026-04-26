/**
 * Route /settings — page des paramètres.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SettingsPage } from '@/features/settings/SettingsPage'

const SETTINGS_TABS = ['general', 'sync', 'analyse', 'accessibility', 'notifications', 'lab', 'users'] as const
export type SettingsTab = (typeof SETTINGS_TABS)[number]

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
  validateSearch: (search: Record<string, unknown>): { tab: SettingsTab } => ({
    tab: SETTINGS_TABS.includes(search.tab as SettingsTab)
      ? (search.tab as SettingsTab)
      : 'general',
  }),
})
