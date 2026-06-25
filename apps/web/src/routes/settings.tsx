/**
 * Route /settings — page des paramètres.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { resolveSettingsTab, type SettingsTab } from '@/features/settings/tabs'

export type { SettingsTab }

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
  validateSearch: (search: Record<string, unknown>): { tab: SettingsTab } => ({
    tab: resolveSettingsTab(typeof search.tab === 'string' ? search.tab : null),
  }),
})
