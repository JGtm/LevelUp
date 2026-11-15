/**
 * Contenu des référentiels (onglets Cartes / Armes / Médailles, recherche, grille), partagé par le
 * panneau coulissant du site (`AssetDrawer`) et le volet ancré de l'application de bureau
 * (`AssetDockedPanel`) : un seul endroit pour les requêtes, la liste et ses états.
 *
 * `searchFirst` : le volet ancré place le champ de recherche en tête, au-dessus des onglets ; le
 * panneau du site garde ses onglets en tête. Tiroir fermé : aucune requête (le panneau du site est
 * monté sur toutes les pages).
 */
import type { Ref } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { PinIcon, PinOffIcon } from '@/components/icons/lucide'
import { useAssetDrawerStore, type AssetTab } from './assetDrawer.store'
import { useAssetMaps, useAssetWeapons, useAssetMedals } from './useAssetDrawer'
import { dedupeAssetsById } from './assetDrawerLogic'
import { AssetSearch } from './AssetSearch'
import { AssetGrid } from './AssetGrid'

type ManifestKey = keyof typeof assetDrawerManifest

const TABS: { tab: AssetTab; label: ManifestKey; empty: ManifestKey }[] = [
  { tab: 'maps', label: 'asset_drawer.tab.maps', empty: 'asset_drawer.empty.maps' },
  { tab: 'weapons', label: 'asset_drawer.tab.weapons', empty: 'asset_drawer.empty.weapons' },
  { tab: 'medals', label: 'asset_drawer.tab.medals', empty: 'asset_drawer.empty.medals' },
]

interface AssetPanelContentProps {
  /** Bouton de fermeture, placé à droite des onglets (panneau du site) ou de l'en-tête (volet ancré). */
  onClose: () => void
  /** Champ de recherche au-dessus des onglets. */
  searchFirst?: boolean
  /** Focus programmatique du champ de recherche (volet ancré). */
  searchRef?: Ref<HTMLInputElement>
  /** Volet ancré : bouton d'épinglage, placé avant la fermeture. Sans `onTogglePin`, pas de bouton. */
  pinned?: boolean
  onTogglePin?: () => void
}

export function AssetPanelContent({
  onClose,
  searchFirst = false,
  searchRef,
  pinned = false,
  onTogglePin,
}: AssetPanelContentProps) {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { isOpen, activeTab, search, setTab, setSearch } = useAssetDrawerStore()

  const t = (key: ManifestKey) => formatMessage(assetDrawerManifest, key, locale)

  // Fermé : aucune requête. Ouvert : les trois catalogues, la recherche sur l'onglet actif.
  const mapsQuery = useAssetMaps(titleSlug, activeTab === 'maps' ? search : '', isOpen)
  const weaponsQuery = useAssetWeapons(titleSlug, activeTab === 'weapons' ? search : '', isOpen)
  const medalsQuery = useAssetMedals(titleSlug, activeTab === 'medals' ? search : '', isOpen)
  const currentQuery = activeTab === 'maps' ? mapsQuery : activeTab === 'weapons' ? weaponsQuery : medalsQuery
  // dedupeAssetsById — dernier rempart avant rendu (cf. assetDrawerLogic.ts) : un catalogue avec deux
  // libellés pour le même `id` fait crier React sur `key={asset.id}` dans `AssetGrid`.
  const items = dedupeAssetsById(currentQuery.data ?? [])
  const emptyKey = TABS.find((x) => x.tab === activeTab)?.empty ?? 'asset_drawer.empty.maps'

  const tabs = (
    <div className="flex shrink-0 items-center justify-between border-b border-border px-3 py-2">
      <div className="flex gap-1" role="tablist">
        {TABS.map(({ tab, label }) => (
          <TabButton key={tab} active={activeTab === tab} onClick={() => setTab(tab)} label={t(label)} />
        ))}
      </div>
      <div className="flex items-center gap-1">
        {onTogglePin && (
          <button
            type="button"
            onClick={onTogglePin}
            aria-pressed={pinned}
            aria-label={t('asset_drawer.pin.toggle')}
            title={t('asset_drawer.pin.toggle')}
            className={`rounded p-1 hover:bg-accent hover:text-accent-foreground ${pinned ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
          >
            {pinned ? <PinOffIcon /> : <PinIcon />}
          </button>
        )}
        <button
          type="button"
          onClick={onClose}
          aria-label={t('asset_drawer.toggle.close')}
          className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
        >
          <CloseIcon />
        </button>
      </div>
    </div>
  )
  const searchField = (
    <div className="shrink-0 border-b border-border px-3 py-2">
      <AssetSearch
        value={search}
        placeholder={t('asset_drawer.search.placeholder')}
        onChange={setSearch}
        inputRef={searchRef}
      />
    </div>
  )

  return (
    <>
      {searchFirst ? searchField : tabs}
      {searchFirst ? tabs : searchField}
      <div className="flex-1 overflow-y-auto">
        <AssetGrid
          items={items}
          kind={activeTab}
          emptyMessage={t(emptyKey)}
          isLoading={currentQuery.isLoading}
          isError={currentQuery.isError}
          errorMessage={t('asset_drawer.empty.error')}
          loadingMessage={t('asset_drawer.empty.loading')}
        />
      </div>
    </>
  )
}

function TabButton({ active, label, onClick }: { active: boolean; label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={`rounded px-3 py-1 text-sm font-medium transition-colors ${
        active
          ? 'bg-accent text-accent-foreground'
          : 'text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground'
      }`}
    >
      {label}
    </button>
  )
}

function CloseIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
        clipRule="evenodd"
      />
    </svg>
  )
}
