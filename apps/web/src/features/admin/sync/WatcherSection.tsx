/**
 * WatcherSection — état du watcher présence Xbox (sync live) sur la page
 * Sync & Jobs : daemon / RTA / token + état FSM par joueur surveillé.
 * Réutilise useWatcherStatus (polling 10 s, features/settings).
 */
import { useWatcherStatus } from '@/features/settings/watcher-queries'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { WatcherPlayerStatus } from '@/lib/api/types'
import { StatusBadge } from '../components/StatusBadge'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'
import { watcherLivenessStatus, type AdminStatus } from '../statusDisplay'
import { SectionHeader } from '../components/SectionHeader'

/** État FSM watcher → statut de badge (Watching/Syncing actifs, Cooling warn). */
function watcherStateStatus(state: string): AdminStatus {
  switch (state) {
    case 'Watching':
      return 'running'
    case 'Syncing':
      return 'running'
    case 'Cooling':
      return 'warning'
    default:
      return 'idle'
  }
}

export function WatcherSection() {
  const { data, isError } = useWatcherStatus()
  const tA = useAdminT()
  const locale = useAdminLocale()

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.watcher.section')} />
      {isError || !data ? (
        <EmptyStateNotice title={tA('admin.watcher.disabled')} description="" />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge
              status={data.daemon_running ? 'running' : 'idle'}
              label={`${tA('admin.watcher.daemon')} ${data.daemon_running ? 'ON' : 'OFF'}`}
            />
            <StatusBadge
              status={data.rta_connected ? 'ok' : data.daemon_running ? 'error' : 'idle'}
              label={tA('admin.watcher.rta')}
            />
            <StatusBadge
              status={data.token_valid ? 'ok' : 'error'}
              label={`${tA('admin.watcher.token')}${data.token_gamertag ? ` (${data.token_gamertag})` : ''}`}
              title={data.token_expires_at}
            />
            {data.daemon_running && (
              <StatusBadge
                status={watcherLivenessStatus(data.last_event_at, data.daemon_running)}
                label={`${tA('admin.watcher.activity')} · ${
                  data.last_event_at
                    ? adminRelativeTime(data.last_event_at, locale)
                    : tA('admin.watcher.no_event_yet')
                }`}
                title={data.last_event_at ? adminAbsoluteTime(data.last_event_at, locale) : undefined}
              />
            )}
          </div>
          {!data.daemon_running ? (
            <p className="text-xs text-muted-foreground">{tA('admin.watcher.disabled')}</p>
          ) : !data.players?.length ? (
            <p className="text-xs text-muted-foreground">{tA('admin.watcher.no_players')}</p>
          ) : (
            <WatcherPlayersTable players={data.players} tA={tA} locale={locale} />
          )}
        </>
      )}
    </section>
  )
}

function WatcherPlayersTable({
  players,
  tA,
  locale,
}: {
  players: WatcherPlayerStatus[]
  tA: TAdmin
  locale: 'fr' | 'en'
}) {
  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.sync.col_player')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.watcher.col_state')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.watcher.col_presence')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.watcher.col_since')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.watcher.col_last_event')}</th>
          </tr>
        </thead>
        <tbody>
          {players.map((p) => (
            <tr key={p.xuid || p.gamertag} className="border-b last:border-b-0 hover:bg-muted/30">
              <td className="px-3 py-2 font-medium text-foreground">
                {p.gamertag}
                {p.subscribe_error && (
                  <div className="text-xs font-normal" style={{ color: tokenCssVar('destructive') }} title={p.subscribe_error}>
                    {p.subscribe_error}
                  </div>
                )}
              </td>
              <td className="px-3 py-2">
                <StatusBadge status={watcherStateStatus(p.state)} label={p.state} title={p.cooldown_left} />
              </td>
              <td className="px-3 py-2 text-xs text-muted-foreground">
                {p.presence_state || '—'}
                {p.in_game ? ' · in game' : ''}
              </td>
              <td
                className="px-3 py-2 text-xs text-muted-foreground"
                title={adminAbsoluteTime(p.state_since, locale)}
              >
                {adminRelativeTime(p.state_since, locale)}
              </td>
              <td
                className="px-3 py-2 text-xs text-muted-foreground"
                title={p.last_event_at ? adminAbsoluteTime(p.last_event_at, locale) : undefined}
              >
                {p.last_event_at ? adminRelativeTime(p.last_event_at, locale) : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
