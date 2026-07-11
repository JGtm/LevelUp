/**
 * AdminLogsPage — viewer des logs JSON slog par module : filtres en URL-state
 * (module/level/contains/limit partageables), auto-refresh opt-in 5 s,
 * expansion du détail JSON par ligne. Flux (pas de tri) — du plus récent au
 * plus ancien, comme un `tail -f` figé.
 *
 * A3 (DC-8) : rendu comme SECTION de l'onglet Système (le triage vit dans
 * l'onglet Détections) — l'URL-state est porté par /admin/system.
 */
import { useEffect, useState } from 'react'
import { useNavigate, useSearch } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminLogEntry } from '@/lib/api/types'
import { useLogModules, useLogTail } from './queries'
import { flattenLogFields, logEntryDetail, logEntryText, logLevelStatus } from './logDisplay'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { StatusBadge } from '../components/StatusBadge'
import { SectionHeader } from '../components/SectionHeader'

const LEVELS = ['debug', 'info', 'warn', 'error'] as const
const LIMITS = ['50', '200', '500'] as const

export function AdminLogsPage() {
  const search = useSearch({ from: '/admin/system' })
  const navigate = useNavigate({ from: '/admin/system' })
  const tA = useAdminT()

  const [autoRefresh, setAutoRefresh] = useState(false)
  // Saisie locale debouncée 300 ms → URL (q=) → query.
  const [containsInput, setContainsInput] = useState(search.q ?? '')
  useEffect(() => {
    const handle = setTimeout(() => {
      const next = containsInput.trim()
      if ((search.q ?? '') !== next) {
        void navigate({
          search: (prev) => ({ ...prev, q: next || undefined }),
          replace: true,
        })
      }
    }, 300)
    return () => clearTimeout(handle)
  }, [containsInput, navigate, search.q])

  const modules = useLogModules()
  const limit = Number(search.n ?? '200')
  const tail = useLogTail(
    {
      module: search.module,
      level: search.level ?? '',
      contains: search.q ?? '',
      limit,
    },
    autoRefresh,
  )

  function setParam(patch: Record<string, string | undefined>) {
    void navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })
  }

  return (
    <div className="space-y-6">
      <SectionHeader title={tA('admin.system.section_logs')} />

      <div className="space-y-4">
      {/* Barre de filtres */}
      <div className="flex flex-wrap items-center gap-3">
        <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {tA('admin.logs.module')}
          <select
            value={search.module}
            onChange={(e) => setParam({ module: e.target.value })}
            className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
          >
            {(modules.data?.modules ?? [{ module: search.module, size_bytes: 0, modified_at: '' }]).map(
              (m) => (
                <option key={m.module} value={m.module}>
                  {m.module}
                </option>
              ),
            )}
          </select>
        </label>

        <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {tA('admin.logs.level')}
          <select
            value={search.level ?? ''}
            onChange={(e) => setParam({ level: e.target.value || undefined })}
            className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
          >
            <option value="">{tA('admin.logs.level_all')}</option>
            {LEVELS.map((l) => (
              <option key={l} value={l}>
                {l.toUpperCase()}
              </option>
            ))}
          </select>
        </label>

        <input
          type="text"
          value={containsInput}
          onChange={(e) => setContainsInput(e.target.value)}
          placeholder={tA('admin.logs.contains_placeholder')}
          className="w-64 rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
        />

        <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {tA('admin.logs.limit')}
          <select
            value={search.n ?? '200'}
            onChange={(e) => setParam({ n: e.target.value })}
            className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
          >
            {LIMITS.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>

        <div className="ml-auto flex items-center gap-2">
          <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            {tA('admin.logs.auto_refresh')}
          </label>
          <Button size="sm" variant="outline" onClick={() => void tail.refetch()} disabled={tail.isFetching}>
            {tA('admin.logs.refresh')}
          </Button>
        </div>
      </div>

      {/* Liste */}
      {tail.isError ? (
        <p className="text-sm text-destructive">{tA('admin.logs.unavailable')}</p>
      ) : !tail.data || (tail.data.entries?.length ?? 0) === 0 ? (
        <EmptyStateNotice title={tA('admin.logs.empty_title')} description={tA('admin.logs.empty_desc')} />
      ) : (
        <>
          <LogEntriesList entries={tail.data.entries ?? []} />
          {tail.data.truncated && (
            <p className="text-xs" style={{ color: tokenCssVar('warning') }}>
              {tA('admin.logs.truncated')}
            </p>
          )}
        </>
      )}
      </div>
    </div>
  )
}

function LogEntriesList({ entries }: { entries: AdminLogEntry[] }) {
  const locale = useAdminLocale()
  const tA = useAdminT()
  const [expanded, setExpanded] = useState<number | null>(null)

  return (
    <div className="rounded-md border">
      {entries.map((entry, i) => {
        const chips = flattenLogFields(entry.fields, 40)
        const isOpen = expanded === i
        return (
          <div key={`${entry.time ?? ''}-${i}`} className="border-b last:border-b-0">
            <button
              type="button"
              onClick={() => setExpanded(isOpen ? null : i)}
              className="flex w-full flex-wrap items-baseline gap-x-3 gap-y-0.5 px-3 py-1.5 text-left text-xs hover:bg-muted/30"
              title={tA('admin.logs.raw')}
            >
              <span
                className="w-20 flex-none font-mono text-muted-foreground"
                title={adminAbsoluteTime(entry.time, locale)}
              >
                {adminRelativeTime(entry.time, locale)}
              </span>
              <StatusBadge status={logLevelStatus(entry.level)} label={entry.level.toUpperCase()} />
              <span className="min-w-0 flex-1 break-words text-foreground">{logEntryText(entry)}</span>
              {entry.err && entry.msg && (
                <span className="break-all font-mono" style={{ color: tokenCssVar('destructive') }}>
                  {entry.err}
                </span>
              )}
              {entry.source && <span className="font-mono text-muted-foreground/70">{entry.source}</span>}
              {chips.slice(0, 4).map((c) => (
                <span key={c} className="font-mono text-muted-foreground/70">
                  {c}
                </span>
              ))}
            </button>
            {isOpen && (
              <pre className="overflow-x-auto border-t bg-muted/40 px-3 py-2 font-mono text-[11px] leading-relaxed text-muted-foreground">
                {logEntryDetail(entry)}
              </pre>
            )}
          </div>
        )
      })}
    </div>
  )
}
