/**
 * CareerHubPage — hub Carrière : container unique avec tabs Progression / Citations.
 * Sprint 55 (Volet B) : route canonique /players/$playerSlug/career?tab=progression|citations.
 * Remplace CareerPage qui assemblait progression + analytics (top matchs, encounters).
 */
import { useParams, useSearch, useNavigate } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { CareerProgressionTab } from './CareerProgressionTab'
import { CareerCitationsTab } from './CareerCitationsTab'

type CareerTab = 'progression' | 'citations'

const TABS: { value: CareerTab; label: string }[] = [
  { value: 'progression', label: 'Progression' },
  { value: 'citations', label: 'Citations' },
]

export function CareerHubPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { tab } = useSearch({ from: '/players/$playerSlug/career' }) as { tab?: CareerTab }
  const navigate = useNavigate()
  const activeTab: CareerTab = tab ?? 'progression'

  function handleTabChange(value: CareerTab) {
    void navigate({
      to: '/players/$playerSlug/career',
      params: { playerSlug },
      search: { tab: value },
      replace: true,
    })
  }

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Carrière"
        subtitle="Capital long terme — progression de rang et maîtrise"
      />

      {/* Tabs deep-linkables */}
      <div className="border-b border-border bg-background px-6">
        <nav className="-mb-px flex gap-1" aria-label="Onglets Carrière">
          {TABS.map(({ value, label }) => (
            <button
              key={value}
              onClick={() => handleTabChange(value)}
              aria-selected={activeTab === value}
              className={[
                'inline-flex items-center px-4 py-3 text-sm font-medium transition-colors',
                activeTab === value
                  ? 'border-b-2 border-primary text-primary'
                  : 'border-b-2 border-transparent text-muted-foreground hover:border-border hover:text-foreground',
              ].join(' ')}
            >
              {label}
            </button>
          ))}
        </nav>
      </div>

      {/* Contenu de l'onglet actif */}
      {activeTab === 'progression' && <CareerProgressionTab />}
      {activeTab === 'citations' && <CareerCitationsTab />}
    </div>
  )
}
