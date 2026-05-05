/**
 * CompareDrawer — tiroir latéral de comparaison joueur vs joueur.
 * Sprint 54-C.
 *
 * Usage :
 *   <CompareDrawer playerSlug={slug} open={open} onClose={() => setOpen(false)} />
 */
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'

import { CompareSurface } from './CompareSurface'
import { getCompareText, normalizeCompareLocale } from './i18n'

interface CompareDrawerProps {
  playerSlug: string
  open: boolean
  onClose: () => void
}

/** Tiroir principal de comparaison. */
export function CompareDrawer({ playerSlug, open, onClose }: CompareDrawerProps) {
  const locale = normalizeCompareLocale(useAppShellStore((state) => state.locale))
  const { data: fieldMappings } = useFieldMappings()
  const text = getCompareText(locale, fieldMappings)

  if (!open) return null

  return (
    <>
      <div
        className="fixed inset-0 bg-background/40 z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      <aside className="fixed right-0 top-0 h-full w-full max-w-lg bg-background shadow-xl z-50 flex flex-col overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">{text.drawerTitle}</h2>
          <button
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground text-xl font-bold"
            aria-label={text.close}
          >
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          <CompareSurface playerSlug={playerSlug} />
        </div>
      </aside>
    </>
  )
}
