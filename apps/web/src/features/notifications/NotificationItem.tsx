/**
 * Une ligne de notification (utilisée dans le dropdown et la page dédiée).
 *
 * Pas de gestion d'état interne : le parent contrôle ce qui se passe au clic
 * via les callbacks (onClick, onMarkRead, onMarkUnread, onDismiss). Permet une
 * réutilisation propre entre le dropdown compact et la timeline expanded.
 */
import { useNavigate } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useTitleSlug } from '@/lib/title-routing'
import { CategoryIcon } from './icons'
import { resolveTitle, resolveBody } from './format'
import { resolveTarget } from './navigation'
import { formatRelative } from './relativeTime'
import { getNotificationsText } from './i18n'
import type { Notification } from './types'

export interface NotificationItemProps {
  notif: Notification
  playerSlug: string
  onMarkRead: (id: number) => void
  onMarkUnread: (id: number) => void
  onDismiss: (id: number) => void
  onAfterClick?: () => void // pour fermer le dropdown
}

export function NotificationItem(props: NotificationItemProps) {
  const { notif, playerSlug, onMarkRead, onMarkUnread, onDismiss, onAfterClick } = props
  const locale = useAppShellStore((s) => s.locale)
  const t = getNotificationsText(locale)
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()
  const isUnread = notif.read_at == null
  const title = resolveTitle(notif, locale)
  const body = resolveBody(notif, locale)
  const rel = formatRelative(notif.created_at, locale)
  const severityClass = SEVERITY_CLASS[notif.severity] ?? 'text-popover-foreground'

  function handleClick(e: React.MouseEvent) {
    if (e.button !== 0) return // ignore middle/right click
    if (e.metaKey || e.ctrlKey) return // open in new tab
    e.preventDefault()
    if (isUnread) onMarkRead(notif.id)
    onAfterClick?.()
    const target = resolveTarget(notif, playerSlug, titleSlug)
    if (target) {
      navigate({
        to: target.to,
        params: target.params,
        search: target.search,
      } as Parameters<typeof navigate>[0])
    }
  }

  return (
    <div
      role="menuitem"
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') handleClick(e as unknown as React.MouseEvent)
      }}
      className={[
        'group relative flex cursor-pointer items-start gap-2 px-3 py-2 transition-colors hover:bg-accent hover:text-accent-foreground',
        isUnread ? 'bg-accent/30' : '',
      ].join(' ')}
    >
      <span className={`mt-0.5 shrink-0 ${severityClass}`} aria-hidden="true">
        <CategoryIcon category={notif.category} />
      </span>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium">{title}</span>
          {isUnread && (
            <span
              className="ml-auto inline-block h-2 w-2 shrink-0 rounded-full bg-sidebar-primary"
              aria-label="Non lue"
            />
          )}
        </div>
        {body && (
          <p className="line-clamp-2 text-xs text-muted-foreground">{body}</p>
        )}
        <p className="mt-0.5 text-3xs text-muted-foreground">{rel}</p>
      </div>

      <div className="invisible absolute right-2 top-2 flex gap-1 group-hover:visible">
        {isUnread ? (
          <ItemActionButton
            label={t.actionMarkAsRead}
            onClick={(e) => {
              e.stopPropagation()
              onMarkRead(notif.id)
            }}
            iconPath="M16.7 4.2a.75.75 0 01.1 1l-8 10.5a.75.75 0 01-1.1.1L3.2 11.3a.75.75 0 011-1L8 14l7.5-9.8a.75.75 0 011.1-.1z"
          />
        ) : (
          <ItemActionButton
            label={t.actionMarkAsUnread}
            onClick={(e) => {
              e.stopPropagation()
              onMarkUnread(notif.id)
            }}
            iconPath="M3 9a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
          />
        )}
        <ItemActionButton
          label={t.actionDismiss}
          onClick={(e) => {
            e.stopPropagation()
            onDismiss(notif.id)
          }}
          iconPath="M5.3 5.3a1 1 0 011.4 0L10 8.6l3.3-3.3a1 1 0 111.4 1.4L11.4 10l3.3 3.3a1 1 0 01-1.4 1.4L10 11.4l-3.3 3.3a1 1 0 01-1.4-1.4L8.6 10 5.3 6.7a1 1 0 010-1.4z"
        />
      </div>
    </div>
  )
}

const SEVERITY_CLASS: Record<Notification['severity'], string> = {
  info: 'text-popover-foreground/70',
  success: 'text-success',
  warn: 'text-warning',
  error: 'text-destructive',
}

function ItemActionButton(props: {
  label: string
  onClick: (e: React.MouseEvent) => void
  iconPath: string
}) {
  return (
    <button
      type="button"
      onClick={props.onClick}
      title={props.label}
      aria-label={props.label}
      className="rounded p-1 text-popover-foreground/70 hover:bg-popover hover:text-popover-foreground"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-3 w-3"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d={props.iconPath} />
      </svg>
    </button>
  )
}
