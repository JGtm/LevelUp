/**
 * LabPage — outils admin (resources / contracts / diagnostics).
 *
 * P8.4 (revue 2026-04-29) : panels extraits dans des fichiers dédiés
 * (ResourcesPanel, ContractsPanel, DiagnosticsPanel) ; helpers + UI atoms
 * dans `_labShared.tsx`. Ce fichier ne porte plus que l'orchestrateur.
 */
import { startTransition, useDeferredValue, useEffect, useMemo, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'

import { LabIntroNotice, LabSelectedToolNotice } from './LabHelp'
import { getLabText, normalizeLabLocale, type LabTab } from './i18n'
import { useLabContracts, useLabDiagnostics, useLabResources } from './queries'
import { ResourcesPanel } from './ResourcesPanel'
import { ContractsPanel } from './ContractsPanel'
import { DiagnosticsPanel } from './DiagnosticsPanel'

const TAB_VALUES: LabTab[] = ['resources', 'contracts', 'diagnostics']

const RESOURCE_LIMIT = 12

export function LabPage() {
  const capabilities = useAppShellStore((state) => state.capabilities)
  const currentTitleSlug = useAppShellStore((state) => state.currentTitleSlug)
  const locale = normalizeLabLocale(useAppShellStore((state) => state.locale))
  const canManageInstance = capabilities?.can_manage_instance ?? false
  const text = useMemo(() => getLabText(locale), [locale])

  const [activeTab, setActiveTab] = useState<LabTab>('resources')
  const [assetSearch, setAssetSearch] = useState('')
  const [medalSearch, setMedalSearch] = useState('')
  const [selectedSnapshotKey, setSelectedSnapshotKey] = useState('')
  const [selectedAssetID, setSelectedAssetID] = useState('')
  const [selectedMedalID, setSelectedMedalID] = useState<number | null>(null)

  const deferredAssetSearch = useDeferredValue(assetSearch)
  const deferredMedalSearch = useDeferredValue(medalSearch)

  const resourceParams = useMemo(
    () => ({
      snapshotKey: selectedSnapshotKey || undefined,
      assetID: selectedAssetID || undefined,
      assetSearch: deferredAssetSearch || undefined,
      medalID: selectedMedalID,
      medalSearch: deferredMedalSearch || undefined,
      limit: RESOURCE_LIMIT,
    }),
    [deferredAssetSearch, deferredMedalSearch, selectedAssetID, selectedMedalID, selectedSnapshotKey],
  )

  const resourcesQuery = useLabResources(resourceParams, canManageInstance)
  const contractsQuery = useLabContracts(canManageInstance && activeTab === 'contracts')
  const diagnosticsQuery = useLabDiagnostics(canManageInstance && activeTab === 'diagnostics')

  useEffect(() => {
    const first = resourcesQuery.data?.snapshots?.[0]
    if (first && !selectedSnapshotKey) {
      setSelectedSnapshotKey(first.resource_key)
    }
  }, [resourcesQuery.data?.snapshots, selectedSnapshotKey])

  useEffect(() => {
    const first = resourcesQuery.data?.assets.items?.[0]
    if (first && !selectedAssetID) {
      setSelectedAssetID(first.asset_id)
    }
  }, [resourcesQuery.data?.assets.items, selectedAssetID])

  useEffect(() => {
    const first = resourcesQuery.data?.medals.items?.[0]
    if (first && selectedMedalID == null) {
      setSelectedMedalID(first.medal_id)
    }
  }, [resourcesQuery.data?.medals.items, selectedMedalID])

  if (!canManageInstance) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={text.page.accessDeniedTitle}
          description={text.page.accessDeniedDescription}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-end">
        <Badge variant="outline">{text.page.currentTitleBadge}: {currentTitleSlug}</Badge>
      </div>

      {/* Onglets : même grammaire que Admin/Paramètres (soulignement bas), pas une
          barre de nav « pilule ». État client (useState), pas de sous-routes. */}
      <div className="border-b border-border">
        <nav className="-mb-px flex gap-0 overflow-x-auto">
          {TAB_VALUES.map((tab) => {
            const active = activeTab === tab
            return (
              <button
                key={tab}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => startTransition(() => setActiveTab(tab))}
                className={`flex items-center gap-1.5 whitespace-nowrap border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
                  active
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
              >
                {text.tabs[tab]}
              </button>
            )
          })}
        </nav>
      </div>

      <LabIntroNotice locale={locale} />
      <LabSelectedToolNotice tab={activeTab} locale={locale} />

      {activeTab === 'resources' ? (
        <ResourcesPanel
          data={resourcesQuery.data}
          isLoading={resourcesQuery.isLoading}
          isError={resourcesQuery.isError}
          onRetry={() => void resourcesQuery.refetch()}
          locale={locale}
          text={text}
          assetSearch={assetSearch}
          setAssetSearch={setAssetSearch}
          medalSearch={medalSearch}
          setMedalSearch={setMedalSearch}
          selectedSnapshotKey={selectedSnapshotKey}
          setSelectedSnapshotKey={setSelectedSnapshotKey}
          selectedAssetID={selectedAssetID}
          setSelectedAssetID={setSelectedAssetID}
          selectedMedalID={selectedMedalID}
          setSelectedMedalID={setSelectedMedalID}
        />
      ) : null}

      {activeTab === 'contracts' ? (
        <ContractsPanel
          data={contractsQuery.data}
          isLoading={contractsQuery.isLoading}
          isError={contractsQuery.isError}
          onRetry={() => void contractsQuery.refetch()}
          locale={locale}
          text={text}
        />
      ) : null}

      {activeTab === 'diagnostics' ? (
        <DiagnosticsPanel
          data={diagnosticsQuery.data}
          isLoading={diagnosticsQuery.isLoading}
          isError={diagnosticsQuery.isError}
          onRetry={() => void diagnosticsQuery.refetch()}
          locale={locale}
          text={text}
        />
      ) : null}
    </div>
  )
}
