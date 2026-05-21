/**
 * SessionPill — pill privé pour FilterOmnibar permettant de sélectionner
 * une session unique (ou "Toutes les sessions").
 *
 * Affiche le `match_count` de chaque session. Recherche inline par label.
 */
import { useMemo, useState } from 'react'
import type { SessionOption } from '@/lib/api/types'
import { useDismissable } from './_hooks'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export interface SessionPillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  currentLabel: string | null
  allSessions: SessionOption[]
  pickedId: string | null
  onPick: (id: string | null) => void
}

export function SessionPill({
  open,
  onToggle,
  onClose,
  currentLabel,
  allSessions,
  pickedId,
  onPick,
}: SessionPillProps) {
  const ref = useDismissable(open, onClose)
  const [search, setSearch] = useState('')
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Masquer les sessions retournant 0 matchs avec les filtres actifs (sauf
  // celle déjà sélectionnée — pour ne pas la faire disparaître brutalement).
  const visibleSessions = useMemo(
    () => allSessions.filter((s) => s.match_count_filtered > 0 || s.session_id === pickedId),
    [allSessions, pickedId],
  )

  const filtered = useMemo(() => {
    if (!search.trim()) return visibleSessions
    const q = search.toLowerCase()
    return visibleSessions.filter((s) => s.label.toLowerCase().includes(q))
  }, [visibleSessions, search])

  const triggerLabel = currentLabel ? `Session : ${currentLabel}` : t('common.filters.session_all')

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={[
          'flex max-w-[18rem] items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          pickedId
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span className="truncate">{triggerLabel}</span>
        <span className="text-2xs opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="listbox"
          aria-label="Sessions"
          className="absolute left-0 top-full z-40 mt-1 flex w-80 flex-col rounded-md border border-border bg-background shadow-lg"
        >
          <div className="border-b border-border p-2">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('common.filters.session_search_placeholder')}
              className="w-full rounded border border-input bg-background px-2 py-1 text-xs focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
              autoFocus
            />
          </div>
          <div className="max-h-72 overflow-y-auto py-1">
            <button
              type="button"
              onClick={() => onPick(null)}
              className={[
                'flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs transition-colors hover:bg-muted',
                pickedId === null ? 'bg-primary/10 text-primary' : 'text-foreground',
              ].join(' ')}
            >
              <span className="font-medium">{t('common.filters.session_all')}</span>
              <span className="text-2xs text-muted-foreground">{visibleSessions.length}</span>
            </button>
            {filtered.length === 0 ? (
              <p className="px-3 py-4 text-center text-xs text-muted-foreground">
                {t('common.filters.session_no_match')}
              </p>
            ) : (
              filtered.map((s) => {
                const isPicked = s.session_id === pickedId
                return (
                  <button
                    key={s.session_id}
                    type="button"
                    onClick={() => onPick(s.session_id)}
                    className={[
                      'flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs transition-colors hover:bg-muted',
                      isPicked ? 'bg-primary/10 text-primary' : 'text-foreground',
                    ].join(' ')}
                  >
                    <span className="truncate">
                      {s.label}
                      {s.is_squad && (
                        <span className="ml-1 text-2xs text-muted-foreground">· escouade</span>
                      )}
                    </span>
                    <span className="shrink-0 text-2xs text-muted-foreground">
                      {s.match_count_filtered} match{s.match_count_filtered > 1 ? 's' : ''}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </div>
      )}
    </div>
  )
}
