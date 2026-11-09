import { useEffect } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { useAssetDrawerStore } from './assetDrawer.store'
import { AssetPanelContent } from './AssetPanelContent'

/**
 * Panneau coulissant des référentiels du site : mini-onglet fixe sur le bord droit, voile, panneau
 * qui se superpose à la page. L'application de bureau lui préfère le volet ancré (`AssetDockedPanel`).
 */
export function AssetDrawer() {
  const locale = useAppShellStore((s) => s.locale)
  const { isOpen, close, toggle } = useAssetDrawerStore()

  const t = (key: keyof typeof assetDrawerManifest) => formatMessage(assetDrawerManifest, key, locale)

  // Phase 3 — fermeture au clavier Escape
  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [isOpen, close])

  return (
    <>
      {/* Tab de déclenchement — fixe sur le bord droit, caché sur mobile */}
      <button
        type="button"
        onClick={toggle}
        aria-label={t(isOpen ? 'asset_drawer.toggle.close' : 'asset_drawer.toggle.open')}
        aria-expanded={isOpen}
        className={`fixed right-0 top-1/2 z-40 -translate-y-1/2 translate-x-0 cursor-pointer select-none items-center justify-center rounded-l border border-r-0 border-border bg-popover px-1.5 py-3 text-2xs font-medium uppercase tracking-widest text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground ${isOpen ? 'hidden' : 'hidden sm:flex'}`}
        style={{ writingMode: 'vertical-rl' }}
      >
        {t('asset_drawer.mini_tab')}
      </button>

      {isOpen && <div className="fixed inset-0 z-[49]" onClick={close} aria-hidden="true" />}

      {/* Panneau drawer — mobile : sheet pleine hauteur ; desktop (sm:) : carte
          flottante centrée verticalement (rendu desktop inchangé). */}
      <div
        role="complementary"
        aria-label={t('asset_drawer.toggle.open')}
        aria-hidden={!isOpen}
        className={`fixed right-0 top-0 z-50 flex h-full w-[88vw] max-w-sm flex-col rounded-l-lg border border-r-0 border-border bg-popover shadow-xl transition-transform duration-200 ease-out sm:top-1/2 sm:h-[min(600px,80vh)] sm:w-[290px] sm:max-w-none sm:-translate-y-1/2 ${isOpen ? 'translate-x-0' : 'translate-x-full'}`}
      >
        <AssetPanelContent onClose={close} />
      </div>
    </>
  )
}
