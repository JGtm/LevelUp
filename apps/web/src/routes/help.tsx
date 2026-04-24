/**
 * Route /help — page d'aide (notes de version + glossaire).
 */
import { createFileRoute } from '@tanstack/react-router'
import { HelpPage } from '@/features/help/HelpPage'

const HELP_TABS = ['glossary', 'release-notes'] as const
export type HelpTab = (typeof HELP_TABS)[number]

export const Route = createFileRoute('/help')({
  component: HelpPage,
  validateSearch: (search: Record<string, unknown>): { tab: HelpTab } => ({
    tab: HELP_TABS.includes(search.tab as HelpTab) ? (search.tab as HelpTab) : 'glossary',
  }),
})
