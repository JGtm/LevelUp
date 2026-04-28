import { useEffect } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { useAssetDrawerStore } from './assetDrawer.store'
import { useAssetMaps, useAssetWeapons } from './useAssetDrawer'
import { AssetSearch } from './AssetSearch'
import { AssetGrid } from './AssetGrid'

export function AssetDrawer() {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)

  const { isOpen, activeTab, search, close, toggle, setTab, setSearch } =
    useAssetDrawerStore()

  const t = (key: keyof typeof assetDrawerManifest) => formatMessage(assetDrawerManifest, key, locale)

  const mapsQuery = useAssetMaps(titleSlug, activeTab === 'maps' ? search : '')
  const weaponsQuery = useAssetWeapons(titleSlug, activeTab === 'weapons' ? search : '')

  // Phase 3 — fermeture au clavier Escape
  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [isOpen, close])

  const currentQuery = activeTab === 'maps' ? mapsQuery : weaponsQuery
  const items = currentQuery.data ?? []
  const emptyKey = activeTab === 'maps' ? 'asset_drawer.empty.maps' : 'asset_drawer.empty.weapons'

  return (
    <>
      {/* Tab de déclenchement — fixe sur le bord droit, caché sur mobile */}
      <button
        type="button"
        onClick={toggle}
        aria-label={t(isOpen ? 'asset_drawer.toggle.close' : 'asset_drawer.toggle.open')}
        aria-expanded={isOpen}
        className="fixed right-0 top-1/2 z-40 hidden -translate-y-1/2 translate-x-0 cursor-pointer select-none items-center justify-center rounded-l border border-r-0 border-border bg-popover px-1.5 py-4 text-[10px] font-medium uppercase tracking-widest text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground sm:flex"
        style={{ writingMode: 'vertical-rl' }}
      >
        {activeTab === 'maps' ? t('asset_drawer.tab.maps') : t('asset_drawer.tab.weapons')}
      </button>

      {/* Panneau drawer */}
      <div
        role="complementary"
        aria-label={t('asset_drawer.toggle.open')}
        aria-hidden={!isOpen}
        className="fixed right-0 top-0 z-50 hidden h-full w-[360px] flex-col border-l border-border bg-popover shadow-xl transition-transform duration-200 ease-out sm:flex"
        style={{ transform: isOpen ? 'translateX(0)' : 'translateX(100%)' }}
      >
        {/* Header : onglets + bouton fermer */}
        <div className="flex shrink-0 items-center justify-between border-b border-border px-3 py-2">
          <div className="flex gap-1" role="tablist">
            <TabButton
              active={activeTab === 'maps'}
              onClick={() => setTab('maps')}
              label={t('asset_drawer.tab.maps')}
            />
            <TabButton
              active={activeTab === 'weapons'}
              onClick={() => setTab('weapons')}
              label={t('asset_drawer.tab.weapons')}
            />
          </div>
          <button
            type="button"
            onClick={close}
            aria-label={t('asset_drawer.toggle.close')}
            className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          >
            <CloseIcon />
          </button>
        </div>

        {/* Search */}
        <div className="shrink-0 border-b border-border px-3 py-2">
          <AssetSearch
            value={search}
            placeholder={t('asset_drawer.search.placeholder')}
            onChange={setSearch}
          />
        </div>

        {/* Grid scrollable */}
        <div className="flex-1 overflow-y-auto">
          <AssetGrid
            items={items}
            locale={locale}
            emptyMessage={t(emptyKey)}
            isLoading={currentQuery.isLoading}
            isError={currentQuery.isError}
            errorMessage={t('asset_drawer.empty.error')}
            loadingMessage={t('asset_drawer.empty.loading')}
          />
        </div>
      </div>
    </>
  )
}

function TabButton({
  active,
  label,
  onClick,
}: {
  active: boolean
  label: string
  onClick: () => void
}) {
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
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className="h-4 w-4"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
        clipRule="evenodd"
      />
    </svg>
  )
}
