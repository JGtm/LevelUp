import { useNavigate, useRouterState } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { getHelpText, normalizeHelpLocale, type HelpTab } from './i18n'
import { GlossaryTab } from './GlossaryTab'
import { ReleaseNotesTab } from './ReleaseNotesTab'

const TABS: HelpTab[] = ['glossary', 'release-notes']

export function HelpPage() {
  const navigate = useNavigate()
  const routerState = useRouterState()
  const locale = normalizeHelpLocale(useAppShellStore((s) => s.locale))
  const text = getHelpText(locale)

  const rawTab = new URLSearchParams(routerState.location.search).get('tab')
  const activeTab: HelpTab =
    TABS.includes(rawTab as HelpTab) ? (rawTab as HelpTab) : 'glossary'

  function setActiveTab(tab: HelpTab) {
    navigate({ to: '/help', search: { tab }, replace: true }).catch(() => {})
  }

  const tabLabels: Record<HelpTab, string> = {
    glossary: text.tabs.glossary,
    'release-notes': text.tabs.releaseNotes,
  }

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      {/* Onglets */}
      <div
        role="tablist"
        aria-label={text.page.title}
        className="flex gap-1 border-b border-border"
      >
        {TABS.map((tab) => (
          <button
            key={tab}
            role="tab"
            type="button"
            aria-selected={activeTab === tab}
            onClick={() => setActiveTab(tab)}
            className={[
              'px-4 py-2 text-sm font-medium transition-colors whitespace-nowrap',
              activeTab === tab
                ? 'border-b-2 border-sidebar-primary text-foreground'
                : 'text-muted-foreground hover:text-foreground',
            ].join(' ')}
          >
            {tabLabels[tab]}
          </button>
        ))}
      </div>

      {/* Contenu */}
      {activeTab === 'glossary' && <GlossaryTab text={text} />}
      {activeTab === 'release-notes' && (
        <ReleaseNotesTab
          locale={locale}
          loadingMessage={text.releaseNotes.loading}
          errorMessage={text.releaseNotes.error}
        />
      )}
    </div>
  )
}
