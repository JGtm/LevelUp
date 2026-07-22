/**
 * Page dédiée — /players/$playerSlug/notifications
 *
 * Liste paginée (cursor before_id), filtres par catégorie, "non lues uniquement",
 * timeline groupée par jour, bulk actions (sélection multiple + actions groupées).
 */
import { useMemo, useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { getNotificationsText } from './i18n'
import { useNotificationsList } from './queries'
import { useDismiss, useMarkAllRead, useMarkRead, useMarkUnread } from './mutations'
import { NotificationItem } from './NotificationItem'
import type { Notification, NotificationCategory } from './types'
import { ALL_CATEGORIES } from './types'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const PAGE_LIMIT = 50

export function NotificationsPage() {
  const params = useParams({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/notifications' })
  const playerSlug = params.playerSlug
  const locale = useAppShellStore((s) => s.locale)
  const t = getNotificationsText(locale)
  const tc = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [unreadOnly, setUnreadOnly] = useState(false)
  const [category, setCategory] = useState<NotificationCategory | undefined>(undefined)
  const [pages, setPages] = useState<Notification[]>([])
  const [cursor, setCursor] = useState<number | undefined>(undefined)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(() => new Set())

  const { data, isLoading, isError, refetch } = useNotificationsList(
    playerSlug,
    { unread_only: unreadOnly, category, limit: PAGE_LIMIT, before_id: cursor },
    { enabled: !!playerSlug, refetchInterval: 60_000 },
  )

  const markRead = useMarkRead({ playerSlug })
  const markUnread = useMarkUnread({ playerSlug })
  const dismiss = useDismiss({ playerSlug })
  const markAllRead = useMarkAllRead({ playerSlug })

  // Concat pour pagination cursor (basique, sans duplication-check)
  const items = cursor === undefined ? data?.items ?? [] : pages.concat(data?.items ?? [])
  const hasMore = data?.next_cursor != null

  function loadMore() {
    if (data?.next_cursor != null) {
      setPages(items)
      setCursor(data.next_cursor)
    }
  }

  function resetFilters() {
    setPages([])
    setCursor(undefined)
  }

  function onFilterChange<K extends 'unread' | 'category'>(kind: K, val: unknown) {
    if (kind === 'unread') setUnreadOnly(val as boolean)
    else setCategory(val as NotificationCategory | undefined)
    resetFilters()
  }

  return (
    <div className="px-4 py-6">
      <header className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold">{t.pageTitle}</h1>
        <button
          type="button"
          onClick={() => markAllRead.mutate(undefined)}
          disabled={markAllRead.isPending}
          className="rounded-md border border-border bg-popover px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
        >
          {t.dropdownMarkAllRead}
        </button>
      </header>

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={unreadOnly}
            onChange={(e) => onFilterChange('unread', e.target.checked)}
          />
          {t.pageFilterUnread}
        </label>
        <select
          value={category ?? ''}
          onChange={(e) =>
            onFilterChange(
              'category',
              e.target.value === '' ? undefined : (e.target.value as NotificationCategory),
            )
          }
          className="rounded-md border border-border bg-popover px-2 py-1 text-sm text-popover-foreground"
        >
          <option value="">{t.pageFilterAll}</option>
          {ALL_CATEGORIES.map((c) => (
            <option key={c} value={c}>
              {t.categoryLabel[c]}
            </option>
          ))}
        </select>
        <button
          type="button"
          onClick={() => refetch()}
          className="ml-auto rounded-md border border-border bg-popover px-2 py-1 text-xs text-popover-foreground hover:bg-accent"
        >
          ↻
        </button>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {t.dropdownErrorLoading}
        </div>
      )}

      {isLoading && cursor === undefined ? (
        <div className="py-8 text-center text-sm text-muted-foreground">…</div>
      ) : items.length === 0 ? (
        <div className="py-12 text-center text-sm text-muted-foreground">{t.pageEmpty}</div>
      ) : (
        <Timeline
          items={items}
          playerSlug={playerSlug}
          t={t}
          selectedIds={selectedIds}
          onToggleSelect={(id) => {
            setSelectedIds((prev) => {
              const next = new Set(prev)
              if (next.has(id)) next.delete(id)
              else next.add(id)
              return next
            })
          }}
          onMarkRead={(id) => markRead.mutate([id])}
          onMarkUnread={(id) => markUnread.mutate(id)}
          onDismiss={(id) => dismiss.mutate(id)}
        />
      )}

      {/* Bulk action bar — affichée seulement si au moins 1 sélection */}
      {selectedIds.size > 0 && (
        <div
          role="region"
          aria-label={tc('common.notifications.bulk_actions_aria')}
          className="sticky bottom-4 mt-4 flex flex-wrap items-center gap-2 rounded-md border border-border bg-popover p-3 shadow-lg"
        >
          <span className="text-sm text-popover-foreground">
            {selectedIds.size} {tc('common.notifications.selected_suffix')}
          </span>
          <button
            type="button"
            onClick={() => {
              markRead.mutate(Array.from(selectedIds))
              setSelectedIds(new Set())
            }}
            className="rounded-md border border-border bg-accent px-3 py-1 text-sm text-accent-foreground hover:bg-accent/80"
          >
            {t.pageBulkMarkRead.replace('{count}', String(selectedIds.size))}
          </button>
          <button
            type="button"
            onClick={() => {
              for (const id of selectedIds) dismiss.mutate(id)
              setSelectedIds(new Set())
            }}
            className="rounded-md border border-border bg-popover px-3 py-1 text-sm text-popover-foreground hover:bg-accent"
          >
            {t.pageBulkDismiss.replace('{count}', String(selectedIds.size))}
          </button>
          <button
            type="button"
            onClick={() => setSelectedIds(new Set())}
            className="ml-auto text-xs text-muted-foreground hover:text-popover-foreground"
            aria-label={tc('common.notifications.deselect_all_aria')}
          >
            ×
          </button>
        </div>
      )}

      {hasMore && (
        <div className="mt-6 text-center">
          <button
            type="button"
            onClick={loadMore}
            className="rounded-md border border-border bg-popover px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent"
          >
            {t.pageLoadMore}
          </button>
        </div>
      )}
    </div>
  )
}

function Timeline(props: {
  items: Notification[]
  playerSlug: string
  t: ReturnType<typeof getNotificationsText>
  selectedIds: Set<number>
  onToggleSelect: (id: number) => void
  onMarkRead: (id: number) => void
  onMarkUnread: (id: number) => void
  onDismiss: (id: number) => void
}) {
  const groups = useMemo(() => groupByDay(props.items, props.t), [props.items, props.t])
  return (
    <div className="space-y-6">
      {groups.map((g) => (
        <section key={g.label}>
          <h2 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {g.label}
          </h2>
          <div className="divide-y divide-border rounded-md border border-border bg-popover">
            {g.items.map((n) => (
              <div key={n.id} className="flex items-stretch">
                <label
                  className="flex shrink-0 items-center justify-center px-3 hover:bg-accent/30"
                  aria-label={`Sélectionner ${n.id}`}
                >
                  <input
                    type="checkbox"
                    checked={props.selectedIds.has(n.id)}
                    onChange={() => props.onToggleSelect(n.id)}
                  />
                </label>
                <div className="min-w-0 flex-1">
                  <NotificationItem
                    notif={n}
                    playerSlug={props.playerSlug}
                    onMarkRead={props.onMarkRead}
                    onMarkUnread={props.onMarkUnread}
                    onDismiss={props.onDismiss}
                  />
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}

interface DayGroup {
  label: string
  items: Notification[]
}

function groupByDay(items: Notification[], t: ReturnType<typeof getNotificationsText>): DayGroup[] {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const yesterday = today - 86_400_000
  const weekStart = today - 6 * 86_400_000
  const buckets: Record<string, Notification[]> = {
    today: [],
    yesterday: [],
    week: [],
    older: [],
  }
  for (const n of items) {
    const ts = new Date(n.created_at).getTime()
    if (ts >= today) buckets.today.push(n)
    else if (ts >= yesterday) buckets.yesterday.push(n)
    else if (ts >= weekStart) buckets.week.push(n)
    else buckets.older.push(n)
  }
  const out: DayGroup[] = []
  if (buckets.today.length > 0) out.push({ label: t.pageGroupToday, items: buckets.today })
  if (buckets.yesterday.length > 0)
    out.push({ label: t.pageGroupYesterday, items: buckets.yesterday })
  if (buckets.week.length > 0) out.push({ label: t.pageGroupThisWeek, items: buckets.week })
  if (buckets.older.length > 0) out.push({ label: t.pageGroupOlder, items: buckets.older })
  return out
}
