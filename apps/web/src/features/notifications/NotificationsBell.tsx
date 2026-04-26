/**
 * Cloche cliquable dans la NavL1 + dropdown des notifs récentes.
 *
 * Pattern : réutilise la structure de SettingsSplitButton (NavL1.tsx) — ref +
 * mousedown listener pour click-outside, panneau positionné absolute right-0,
 * tokens design system (bg-popover, border-border, z-50).
 */
import { useEffect, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { getNotificationsText } from './i18n'
import { useNotificationsList, useUnreadCount } from './queries'
import { useDismiss, useMarkAllRead, useMarkRead, useMarkUnread } from './mutations'
import { NotificationItem } from './NotificationItem'
import type { Notification } from './types'

export interface NotificationsBellProps {
  playerSlug: string
}

const DROPDOWN_LIMIT = 12

export function NotificationsBell({ playerSlug }: NotificationsBellProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getNotificationsText(locale)
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const { data: countData } = useUnreadCount(playerSlug, !!playerSlug)
  const unreadCount = countData?.count ?? 0

  const { data: list, isLoading, isError } = useNotificationsList(
    playerSlug,
    { limit: DROPDOWN_LIMIT },
    { enabled: open && !!playerSlug, refetchInterval: open ? 60_000 : undefined },
  )

  const markRead = useMarkRead({ playerSlug })
  const markUnread = useMarkUnread({ playerSlug })
  const dismiss = useDismiss({ playerSlug })
  const markAllRead = useMarkAllRead({ playerSlug })

  // Click-outside
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  // Esc pour fermer
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])

  const items = list?.items ?? []
  const unread = items.filter((n) => n.read_at == null)
  const older = items.filter((n) => n.read_at != null)

  const ariaLabel =
    unreadCount > 0
      ? t.bellAriaLabelWithCount.replace('{count}', String(unreadCount))
      : t.bellAriaLabelEmpty

  return (
    <div ref={ref} className="relative ml-1">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={ariaLabel}
        aria-haspopup="menu"
        aria-expanded={open}
        className="relative flex items-center rounded-md px-2 py-1.5 text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
      >
        <BellIcon />
        {unreadCount > 0 && (
          <span
            className="absolute -right-0.5 -top-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-medium leading-none text-destructive-foreground"
            aria-hidden="true"
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div
          role="menu"
          aria-label={t.dropdownTitle}
          className="absolute right-0 top-full z-50 mt-1 max-h-[70vh] w-[22rem] overflow-hidden rounded-md border border-border bg-popover shadow-lg"
        >
          <BellHeader
            title={t.dropdownTitle}
            disabled={unreadCount === 0 || markAllRead.isPending}
            onMarkAll={() => markAllRead.mutate(undefined)}
            label={t.dropdownMarkAllRead}
          />
          <div className="max-h-[calc(70vh-6rem)] overflow-y-auto">
            <BellBody
              items={items}
              unread={unread}
              older={older}
              isLoading={isLoading}
              isError={isError}
              t={t}
              playerSlug={playerSlug}
              onMarkRead={(id) => markRead.mutate([id])}
              onMarkUnread={(id) => markUnread.mutate(id)}
              onDismiss={(id) => dismiss.mutate(id)}
              onAfterClick={() => setOpen(false)}
            />
          </div>
          <BellFooter playerSlug={playerSlug} label={t.dropdownViewAll} onNavigate={() => setOpen(false)} />
        </div>
      )}
    </div>
  )
}

function BellHeader(props: {
  title: string
  disabled: boolean
  label: string
  onMarkAll: () => void
}) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border px-3 py-2">
      <span className="text-sm font-medium text-popover-foreground">{props.title}</span>
      <button
        type="button"
        onClick={props.onMarkAll}
        disabled={props.disabled}
        className="text-xs text-popover-foreground/70 hover:text-popover-foreground disabled:cursor-not-allowed disabled:opacity-40"
      >
        {props.label}
      </button>
    </div>
  )
}

function BellBody(props: {
  items: Notification[]
  unread: Notification[]
  older: Notification[]
  isLoading: boolean
  isError: boolean
  t: ReturnType<typeof getNotificationsText>
  playerSlug: string
  onMarkRead: (id: number) => void
  onMarkUnread: (id: number) => void
  onDismiss: (id: number) => void
  onAfterClick: () => void
}) {
  const { items, unread, older, isLoading, isError, t } = props
  if (isLoading) {
    return <div className="px-3 py-6 text-center text-sm text-muted-foreground">…</div>
  }
  if (isError) {
    return (
      <div className="px-3 py-6 text-center text-sm text-destructive">
        {t.dropdownErrorLoading}
      </div>
    )
  }
  if (items.length === 0) {
    return (
      <div className="px-3 py-8 text-center text-sm text-muted-foreground">{t.dropdownEmpty}</div>
    )
  }
  return (
    <>
      {unread.length > 0 && (
        <SectionLabel label={t.dropdownUnread} />
      )}
      {unread.map((n) => (
        <NotificationItem
          key={n.id}
          notif={n}
          playerSlug={props.playerSlug}
          onMarkRead={props.onMarkRead}
          onMarkUnread={props.onMarkUnread}
          onDismiss={props.onDismiss}
          onAfterClick={props.onAfterClick}
        />
      ))}
      {older.length > 0 && (
        <>
          <div role="separator" className="my-1 h-px bg-border" />
          <SectionLabel label={t.dropdownOlder} />
        </>
      )}
      {older.map((n) => (
        <NotificationItem
          key={n.id}
          notif={n}
          playerSlug={props.playerSlug}
          onMarkRead={props.onMarkRead}
          onMarkUnread={props.onMarkUnread}
          onDismiss={props.onDismiss}
          onAfterClick={props.onAfterClick}
        />
      ))}
    </>
  )
}

function SectionLabel({ label }: { label: string }) {
  return (
    <div className="px-3 pt-1.5 pb-0.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
      {label}
    </div>
  )
}

function BellFooter(props: { playerSlug: string; label: string; onNavigate: () => void }) {
  return (
    <div className="border-t border-border bg-popover">
      <Link
        to="/players/$playerSlug/notifications"
        params={{ playerSlug: props.playerSlug }}
        onClick={props.onNavigate}
        className="block w-full px-3 py-2 text-center text-xs text-popover-foreground/80 hover:bg-accent hover:text-accent-foreground"
      >
        {props.label}
      </Link>
    </div>
  )
}

function BellIcon() {
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
        d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zm-3 14a3 3 0 006 0H7z"
        clipRule="evenodd"
      />
    </svg>
  )
}
