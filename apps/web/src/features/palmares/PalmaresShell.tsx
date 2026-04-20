import type { ReactNode } from 'react'

import { Link } from '@tanstack/react-router'

import { PageHeader } from '@/components/shell/PageHeader'
import { useAppShellStore } from '@/stores/appShellStore'

import { getPalmaresText, normalizePalmaresLocale, type PalmaresTab } from './i18n'

const TAB_ORDER: PalmaresTab[] = ['leaderboard', 'relations', 'compare', 'season-pass']

export function PalmaresShell({
  playerSlug,
  activeTab,
  children,
}: {
  playerSlug: string
  activeTab: PalmaresTab
  children: ReactNode
}) {
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)

  return (
    <div className="flex flex-col gap-6 p-6">
      <PageHeader title={text.page.title} subtitle={text.page.subtitle} inset={false} />

      <div className="flex flex-wrap gap-2 rounded-3xl border border-border bg-card/90 p-2 shadow-sm">
        {TAB_ORDER.map((tab) => {
          const href =
            tab === 'leaderboard'
              ? '/players/$playerSlug/palmares'
              : `/players/$playerSlug/palmares/${tab}`
          const isActive = activeTab === tab
          return (
            <Link
              key={tab}
              to={href}
              params={{ playerSlug }}
              className={[
                'rounded-2xl px-4 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              ].join(' ')}
            >
              {text.tabs[tab]}
            </Link>
          )
        })}
      </div>

      {children}
    </div>
  )
}
