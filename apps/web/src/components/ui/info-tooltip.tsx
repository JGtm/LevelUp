import { useState, useRef, useEffect, type ReactNode } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

interface InfoTooltipProps {
  /** Texte ou contenu affiché dans le tooltip */
  content: ReactNode
  /** Taille de l'icône ⓘ en classes Tailwind (défaut: "w-4 h-4") */
  iconClass?: string
}

/**
 * Icône ⓘ qui affiche un tooltip explicatif au hover/focus.
 * Pure CSS+state, aucune dépendance externe.
 */
export function InfoTooltip({ content, iconClass = 'w-4 h-4' }: InfoTooltipProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  return (
    <div className="relative inline-flex items-center" ref={ref}>
      <button
        type="button"
        className={`${iconClass} inline-flex items-center justify-center rounded-full border border-input text-muted-foreground hover:text-foreground hover:border-border text-2xs font-bold leading-none cursor-help transition-colors`}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onClick={() => setOpen((v) => !v)}
        aria-label={t('common.tooltip.more_info_aria')}
      >
        i
      </button>
      {open && (
        <div
          role="tooltip"
          className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50 w-64 rounded-lg border border-border bg-background p-3 text-xs font-normal normal-case tracking-normal text-foreground shadow-lg"
        >
          {content}
          <div className="absolute top-full left-1/2 -translate-x-1/2 -mt-px border-4 border-transparent border-t-background" />
        </div>
      )}
    </div>
  )
}
