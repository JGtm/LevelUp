/**
 * WatcherCard — carte de configuration du watcher de présence Xbox RTA.
 *
 * Gère : toggle on/off, statut token, auth Device Code Flow, sélecteur joueurs, statut RTA.
 */
import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SettingsText } from '@/features/settings/i18n'
import {
  useWatcherStatus,
  useStartWatcherAuth,
  useWatcherAuthPoll,
  useUpdateWatcherSubscriptions,
} from '@/features/settings/watcher-queries'

interface WatcherCardProps {
  enabled: boolean
  onToggle: (v: boolean) => void
  t: SettingsText
}

function Toggle({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!value)}
      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
        value ? 'bg-primary' : 'bg-muted'
      }`}
    >
      <span
        className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-background shadow ring-0 transition-transform ${
          value ? 'translate-x-4' : 'translate-x-0'
        }`}
      />
    </button>
  )
}

function TokenStatus({ t, onStartAuth }: { t: SettingsText; onStartAuth: () => void }) {
  const { data, isLoading } = useWatcherStatus()

  // Pendant le premier chargement, afficher un placeholder discret
  if (isLoading && !data) {
    return <span className="text-xs text-muted-foreground">{t.watcherTokenMissing}</span>
  }

  // Sans données (erreur 403, réseau KO, etc.) : afficher "Connecter" par défaut
  if (!data) {
    return (
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground">🔴 {t.watcherTokenMissing}</span>
        <Button variant="outline" size="sm" onClick={onStartAuth}>{t.watcherAuthButton}</Button>
      </div>
    )
  }

  if (data.token_valid && data.token_expires_at) {
    const date = new Date(data.token_expires_at).toLocaleDateString()
    return (
      <p className="text-xs text-success">
        ✅ {t.watcherTokenValid.replace('{date}', date).replace('{gamertag}', data.token_gamertag ?? '')}
      </p>
    )
  }

  if (data.token_expires_at && !data.token_valid) {
    return (
      <div className="flex items-center gap-2">
        <span className="text-xs text-warning">⚠️ {t.watcherTokenExpired}</span>
        <Button variant="outline" size="sm" onClick={onStartAuth}>{t.watcherAuthReconnect}</Button>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-muted-foreground">🔴 {t.watcherTokenMissing}</span>
      <Button variant="outline" size="sm" onClick={onStartAuth}>{t.watcherAuthButton}</Button>
    </div>
  )
}

function AuthFlow({
  attemptId,
  userCode,
  verificationUrl,
  expiresIn,
  t,
}: {
  attemptId: string
  userCode: string
  verificationUrl: string
  expiresIn: number
  t: SettingsText
}) {
  const [secondsLeft, setSecondsLeft] = useState(expiresIn)
  const { data: pollData } = useWatcherAuthPoll(attemptId)

  useEffect(() => {
    if (secondsLeft <= 0) return
    const timer = setInterval(() => setSecondsLeft((s) => Math.max(0, s - 1)), 1000)
    return () => clearInterval(timer)
  }, [secondsLeft])

  const copyCode = useCallback(() => {
    navigator.clipboard.writeText(userCode).catch(() => {})
  }, [userCode])

  if (pollData?.status === 'authorized') {
    return <p className="text-xs text-success">✅ {t.watcherAuthSuccess}</p>
  }
  if (pollData?.status === 'failed' || pollData?.status === 'expired') {
    return <p className="text-xs text-destructive">✗ {t.watcherAuthFailed}</p>
  }

  return (
    <div className="rounded border border-border p-3 space-y-2 text-sm">
      <p className="text-xs text-muted-foreground">
        {t.watcherAuthInstructions.replace('{url}', verificationUrl)}
      </p>
      <div className="flex items-center gap-2">
        <code className="rounded bg-muted px-2 py-0.5 text-base font-mono font-bold tracking-widest">
          {userCode}
        </code>
        <Button variant="ghost" size="sm" onClick={copyCode}>{t.watcherAuthCopyCode}</Button>
        <a href={verificationUrl} target="_blank" rel="noopener noreferrer">
          <Button variant="ghost" size="sm">{t.watcherAuthOpenLink}</Button>
        </a>
      </div>
      <p className="text-xs text-muted-foreground">
        {t.watcherAuthPending} ({secondsLeft}s)
      </p>
    </div>
  )
}

function PlayersSelector({ t }: { t: SettingsText }) {
  const { data } = useWatcherStatus()
  const updateSubs = useUpdateWatcherSubscriptions()
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const [feedback, setFeedback] = useState<string | null>(null)

  const current = data?.subscribed_players?.[0] ?? 'all'
  const isSinglePlayer = current !== 'all' && availablePlayers.some((p) => p.gamertag === current)

  const handleChange = (value: string) => {
    const payload = value === 'all' ? ['all'] : [value]
    updateSubs.mutate(payload, {
      onSuccess: () => {
        setFeedback(t.watcherSubscriptionsUpdated)
        setTimeout(() => setFeedback(null), 2000)
      },
    })
  }

  return (
    <div className="space-y-1 py-2">
      <label className="block text-xs font-medium text-muted-foreground">{t.watcherPlayersLabel}</label>
      <select
        value={isSinglePlayer ? current : 'all'}
        onChange={(e) => handleChange(e.target.value)}
        className="w-full rounded-md border border-input bg-background px-2 py-1 text-sm"
      >
        <option value="all">{t.watcherPlayersAll}</option>
        {availablePlayers.filter((p) => !p.is_demo).map((p) => (
          <option key={p.gamertag} value={p.gamertag}>{p.gamertag}</option>
        ))}
      </select>
      {feedback && <p className="text-xs text-success">{feedback}</p>}
    </div>
  )
}

function resolveStateLabel(state: string, t: SettingsText): string {
  switch (state) {
    case 'Idle':     return t.watcherStateIdle
    case 'Watching': return t.watcherStateWatching
    case 'Syncing':  return t.watcherStateSyncing
    case 'Cooling':  return t.watcherStateCooling
    default:         return state
  }
}

/**
 * Format un "vu il y a X sur Y" lisible côté UI.
 *
 * Logique :
 *  - < 60 s     → "à l'instant" / "just now"
 *  - < 60 min   → "5 min"  / "5 min ago"
 *  - < 24 h     → "3 h"    / "3 hr ago"
 *  - < 7 j      → "2 j"    / "2 days ago"
 *  - sinon      → format date absolu localisé
 *
 * Le timestamp en entrée est en RFC3339 UTC (renvoyé par l'API Go).
 * `now` est injectable pour faciliter les tests.
 */
export function formatLastSeen(
  timestamp: string,
  titleName: string,
  t: SettingsText,
  locale: 'fr' | 'en' = 'fr',
  now: Date = new Date(),
): string {
  const past = new Date(timestamp)
  if (Number.isNaN(past.getTime())) {
    return t.watcherNeverSeen
  }
  const diffMs = now.getTime() - past.getTime()
  const diffMin = Math.floor(diffMs / 60_000)
  const diffH = Math.floor(diffMs / 3_600_000)
  const diffD = Math.floor(diffMs / 86_400_000)

  let duration: string
  if (diffMin < 1) {
    duration = locale === 'fr' ? "moins d'1 min" : 'less than 1 min'
  } else if (diffMin < 60) {
    duration = `${diffMin} min`
  } else if (diffH < 24) {
    duration = locale === 'fr' ? `${diffH} h` : `${diffH} hr`
  } else if (diffD < 7) {
    duration = locale === 'fr' ? `${diffD} j` : `${diffD} day${diffD > 1 ? 's' : ''}`
  } else {
    // Format absolu pour les dates anciennes.
    const date = past.toLocaleDateString(locale === 'fr' ? 'fr-FR' : 'en-US', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    })
    return t.watcherLastSeenAbsolute
      .replace('{date}', date)
      .replace('{title}', titleName)
  }

  return t.watcherLastSeenRelative
    .replace('{duration}', duration)
    .replace('{title}', titleName)
}

function RTAStatus({ t }: { t: SettingsText }) {
  const { data } = useWatcherStatus()
  if (!data?.daemon_running) return null

  return (
    <div className="space-y-1 py-2">
      <div className="flex items-center gap-1.5 text-xs">
        <span className={`h-2 w-2 rounded-full ${data.rta_connected ? 'bg-success' : 'bg-muted-foreground'}`} />
        <span className="text-muted-foreground">
          {data.rta_connected ? t.watcherRtaConnected : t.watcherRtaDisconnected}
        </span>
      </div>
      {data.players.length > 0 && (
        <ul className="mt-1 space-y-1">
          {data.players.map((p) => (
            <li key={p.xuid} className="flex flex-col gap-0.5 text-xs text-muted-foreground">
              <div className="flex items-center gap-2">
                <span className="font-medium text-foreground">{p.gamertag}</span>
                <span className="rounded bg-muted px-1 py-0.5 text-2xs">{resolveStateLabel(p.state, t)}</span>
                {p.in_game && <span className="text-success text-2xs">{t.watcherInGame}</span>}
                {p.subscribe_error ? (
                  <span
                    className="rounded bg-destructive/15 px-1 py-0.5 text-2xs text-destructive"
                    title={p.subscribe_error}
                  >
                    ⚠ {t.watcherSubscribeError}
                  </span>
                ) : (
                  <span className="text-2xs text-success">✓</span>
                )}
              </div>
              {p.last_seen && (
                <span className="pl-1 text-2xs italic text-muted-foreground/80">
                  {formatLastSeen(p.last_seen.timestamp, p.last_seen.title_name, t)}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function WatcherCard({ enabled, onToggle, t }: WatcherCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{t.watcherTitle}</CardTitle>
          <Toggle value={enabled} onChange={onToggle} />
        </div>
      </CardHeader>
      <CardContent className={enabled ? 'space-y-1 divide-y divide-border/50' : undefined}>
        <WatcherSectionBody enabled={enabled} t={t} />
      </CardContent>
    </Card>
  )
}

export function WatcherSectionBody({ enabled, t }: { enabled: boolean; t: SettingsText }) {
  const startAuth = useStartWatcherAuth()
  const [currentAttempt, setCurrentAttempt] = useState<{
    id: string
    userCode: string
    verificationUrl: string
    expiresIn: number
  } | null>(null)

  const handleStartAuth = () => {
    startAuth.mutate(undefined, {
      onSuccess: (data) => {
        setCurrentAttempt({
          id: data.attempt_id,
          userCode: data.user_code,
          verificationUrl: data.verification_url,
          expiresIn: data.expires_in,
        })
      },
    })
  }

  if (!enabled) {
    return <p className="text-xs text-muted-foreground">{t.watcherPresenceDescription}</p>
  }

  return (
    <>
      <div className="pb-2">
        <TokenStatus t={t} onStartAuth={handleStartAuth} />
      </div>
      {currentAttempt && (
        <div className="py-2">
          <AuthFlow
            attemptId={currentAttempt.id}
            userCode={currentAttempt.userCode}
            verificationUrl={currentAttempt.verificationUrl}
            expiresIn={currentAttempt.expiresIn}
            t={t}
          />
        </div>
      )}
      <PlayersSelector t={t} />
      <RTAStatus t={t} />
    </>
  )
}
