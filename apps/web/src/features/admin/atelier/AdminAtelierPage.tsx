/**
 * AdminAtelierPage — onglet « Atelier » de l'Admin.
 *
 * Réhabilitation Lab (2026-06-18) : l'ancien /lab (outil opérateur enfoui dans
 * les Paramètres) est rapatrié dans l'Admin, où vit déjà la console opérateur.
 * L'Atelier héberge l'explorateur de ressources Waypoint (snapshots / assets /
 * médailles) — socle pour la maintenance et la préparation de nouveaux titres.
 * Le diff « Contrats API » (scaffolding de migration) est retiré ; les
 * diagnostics de parité rejoignent l'onglet « Qualité données ».
 *
 * Réutilise les panneaux Lab existants (ResourcesPanel + i18n local). Le gating
 * est assuré par AdminLayout (RequireAdmin côté serveur, redirection côté front).
 */
import { useDeferredValue, useEffect, useMemo, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { useAppShellStore } from '@/stores/appShellStore'

import { LabIntroNotice, LabSelectedToolNotice } from '@/features/lab/LabHelp'
import { getLabText, normalizeLabLocale } from '@/features/lab/i18n'
import { useLabResources } from '@/features/lab/queries'
import { ResourcesPanel } from '@/features/lab/ResourcesPanel'

const RESOURCE_LIMIT = 12

export function AdminAtelierPage() {
  const currentTitleSlug = useAppShellStore((state) => state.currentTitleSlug)
  const locale = normalizeLabLocale(useAppShellStore((state) => state.locale))
  const text = useMemo(() => getLabText(locale), [locale])

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

  const resourcesQuery = useLabResources(resourceParams, true)

  useEffect(() => {
    const first = resourcesQuery.data?.snapshots?.[0]
    if (first && !selectedSnapshotKey) setSelectedSnapshotKey(first.resource_key)
  }, [resourcesQuery.data?.snapshots, selectedSnapshotKey])

  useEffect(() => {
    const first = resourcesQuery.data?.assets.items?.[0]
    if (first && !selectedAssetID) setSelectedAssetID(first.asset_id)
  }, [resourcesQuery.data?.assets.items, selectedAssetID])

  useEffect(() => {
    const first = resourcesQuery.data?.medals.items?.[0]
    if (first && selectedMedalID == null) setSelectedMedalID(first.medal_id)
  }, [resourcesQuery.data?.medals.items, selectedMedalID])

  return (
    <div className="space-y-6">
      <div className="flex justify-end">
        <Badge variant="outline">
          {text.page.currentTitleBadge}: {currentTitleSlug}
        </Badge>
      </div>

      <LabIntroNotice locale={locale} />
      <LabSelectedToolNotice tab="resources" locale={locale} />

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
    </div>
  )
}
