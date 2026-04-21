/**
 * SettingsPage — page des paramètres utilisateur avec onglets.
 */
import { useState, useEffect } from 'react'
import { Link } from '@tanstack/react-router'

import { PageHeader } from '@/components/shell/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { useStartSyncAll, useJobStatus } from '@/features/setup/queries'
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { WatcherCard } from '@/features/settings/WatcherCard'
import type { SettingsResponse } from '@/lib/api/types'

function ToggleRow({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between py-2">
      <span className="text-sm text-foreground">{label}</span>
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
    </div>
  )
}

export function SettingsPage() {
  const { data: settings, isLoading } = useSettings()
  const mutation = useUpdateSettings()
  const canManageInstance = useAppShellStore((s) => s.capabilities?.can_manage_instance ?? false)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const locale = normalizeSettingsLocale(useAppShellStore((s) => s.locale))
  const t = getSettingsText(locale)

  const [localSettings, setLocalSettings] = useState<Partial<SettingsResponse>>({})
  const [saveStatus, setSaveStatus] = useState<'saved' | 'error' | null>(null)
  const saveTimerRef = useState<ReturnType<typeof setTimeout> | null>(null)
  const [activeTab, setActiveTab] = useState<'general' | 'sync' | 'lab' | 'users'>('general')

  useEffect(() => {
    if (settings) setLocalSettings(settings)
  }, [settings])

  useEffect(() => {
    return () => {
      if (saveTimerRef[0]) clearTimeout(saveTimerRef[0])
    }
  }, [])

  function handleChange<K extends keyof SettingsResponse>(field: K, value: SettingsResponse[K]) {
    setLocalSettings((prev) => ({ ...prev, [field]: value }))
    mutation.mutate({ [field]: value } as Partial<SettingsResponse>, {
      onSuccess: () => {
        setSaveStatus('saved')
        if (saveTimerRef[0]) clearTimeout(saveTimerRef[0])
        saveTimerRef[0] = setTimeout(() => setSaveStatus(null), 2000)
      },
      onError: () => {
        setSaveStatus('error')
        if (saveTimerRef[0]) clearTimeout(saveTimerRef[0])
        saveTimerRef[0] = setTimeout(() => setSaveStatus(null), 4000)
      },
    })
  }

  const merged: Partial<SettingsResponse> = localSettings

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label={t.loading} />
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      <PageHeader
        title={t.pageTitle}
        subtitle={t.pageSubtitle}
        actions={
          saveStatus === 'saved' ? (
            <span className="text-sm text-success" role="status" aria-live="polite">
              {t.savedStatus}
            </span>
          ) : saveStatus === 'error' ? (
            <span className="text-sm text-destructive" role="alert">
              {t.errorStatus}
            </span>
          ) : undefined
        }
      />

      {/* Onglets */}
      <div className="border-b border-border px-6">
        <nav className="-mb-px flex gap-4" aria-label="Onglets paramètres">
          {(
            [
              { id: 'general', label: t.tabGeneral },
              { id: 'sync', label: t.tabSync },
              ...(canManageInstance ? [{ id: 'lab', label: t.tabLab }] : []),
              ...(isAdmin ? [{ id: 'users', label: t.tabUsers }] : []),
            ] as { id: 'general' | 'sync' | 'lab' | 'users'; label: string }[]
          ).map(({ id, label }) => (
            <button
              key={id}
              onClick={() => setActiveTab(id)}
              className={[
                'whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors',
                activeTab === id
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground',
              ].join(' ')}
              aria-selected={activeTab === id}
              role="tab"
            >
              {label}
            </button>
          ))}
        </nav>
      </div>

      <div className="space-y-6 p-6">
        {activeTab === 'general' && (
          <GeneralTab merged={merged} handleChange={handleChange} t={t} />
        )}
        {activeTab === 'sync' && (
          <SyncTab merged={merged} handleChange={handleChange} t={t} />
        )}
        {activeTab === 'lab' && <LabTab t={t} />}
        {activeTab === 'users' && <UsersTab t={t} />}
      </div>
    </div>
  )
}

// ─── Types partagés entre onglets ─────────────────────────────────────────────

interface TabProps {
  merged: Partial<SettingsResponse>
  handleChange: <K extends keyof SettingsResponse>(field: K, value: SettingsResponse[K]) => void
  t: ReturnType<typeof getSettingsText>
}

// ─── Onglet Général ───────────────────────────────────────────────────────────

function GeneralTab({ merged, handleChange, t }: TabProps) {
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.interfaceTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-foreground">{t.langLabel}</span>
            <select
              value={merged.lang ?? 'fr'}
              onChange={(e) => handleChange('lang', e.target.value)}
              className="rounded border border-input px-2 py-1 text-sm"
            >
              <option value="fr">{t.langFr}</option>
              <option value="en">{t.langEn}</option>
            </select>
          </div>
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-foreground">{t.timezoneLabel}</span>
            <select
              value={merged.user_timezone ?? 'Europe/Paris'}
              onChange={(e) => handleChange('user_timezone', e.target.value)}
              className="rounded border border-input px-2 py-1 text-sm"
            >
              <option value="Europe/Paris">Europe/Paris</option>
              <option value="Europe/London">Europe/London</option>
              <option value="America/New_York">America/New_York</option>
              <option value="America/Los_Angeles">America/Los_Angeles</option>
              <option value="America/Chicago">America/Chicago</option>
              <option value="Asia/Tokyo">Asia/Tokyo</option>
              <option value="UTC">UTC</option>
            </select>
          </div>
          <ToggleRow label={t.showRecords} value={merged.show_records ?? false} onChange={(v) => handleChange('show_records', v)} />
          <ToggleRow label={t.normalizeModeLabels} value={merged.normalize_mode_labels ?? true} onChange={(v) => handleChange('normalize_mode_labels', v)} />
          <ToggleRow label={t.excludeBTB} value={merged.career_top_exclude_btb ?? false} onChange={(v) => handleChange('career_top_exclude_btb', v)} />
          <ToggleRow label={t.refreshClearsCaches} value={merged.refresh_clears_caches ?? false} onChange={(v) => handleChange('refresh_clears_caches', v)} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.discordTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <ToggleRow label={t.discordEnabled} value={merged.discord_notifications_enabled ?? false} onChange={(v) => handleChange('discord_notifications_enabled', v)} />
          <ToggleRow label={t.discordNotifySync} value={merged.discord_notify_sync ?? false} onChange={(v) => handleChange('discord_notify_sync', v)} />
          <ToggleRow label={t.discordNotifyBackfill} value={merged.discord_notify_backfill ?? false} onChange={(v) => handleChange('discord_notify_backfill', v)} />
          <ToggleRow label={t.discordNotifyNewMedia} value={merged.discord_notify_new_media ?? false} onChange={(v) => handleChange('discord_notify_new_media', v)} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.mediaTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <ToggleRow label={t.mediaWatcherEnabled} value={merged.media_watcher_enabled ?? false} onChange={(v) => handleChange('media_watcher_enabled', v)} />
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-foreground">{t.mediaToleranceLabel}</span>
            <input
              type="number"
              value={merged.media_tolerance_minutes ?? 10}
              onChange={(e) => handleChange('media_tolerance_minutes', Number(e.target.value))}
              className="w-20 rounded border border-input px-2 py-1 text-right text-sm"
              min={1}
              max={60}
            />
          </div>
        </CardContent>
      </Card>

      {merged.discord_notifications_enabled && !merged.discord_webhook_url_present && (
        <div className="rounded-lg border border-warning bg-warning/10 px-4 py-3 text-sm text-warning">
          ⚠️ {t.discordNoWebhook}
        </div>
      )}
      {merged.media_watcher_enabled && !merged.media_captures_base_dir && (
        <div className="rounded-lg border border-warning bg-warning/10 px-4 py-3 text-sm text-warning">
          ⚠️ {t.mediaNoBaseDir}
        </div>
      )}
    </>
  )
}

// ─── Onglet Utilisateurs ────────────────────────────────────────────────────

function UsersTab({ t }: { t: ReturnType<typeof getSettingsText> }) {
  return (
    <Card className="border-border bg-card">
      <CardContent className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="mt-2 text-lg font-semibold text-foreground">{t.usersTitle}</p>
          <p className="mt-1 text-sm text-muted-foreground">{t.usersDescription}</p>
        </div>
        <Link to="/admin">
          <Button>{t.openUsersButton}</Button>
        </Link>
      </CardContent>
    </Card>
  )
}

// ─── Onglet Lab ──────────────────────────────────────────────────────────────

function LabTab({ t }: { t: ReturnType<typeof getSettingsText> }) {
  return (
    <Card className="border-border bg-card">
      <CardContent className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t.instanceLabel}</p>
          <p className="mt-2 text-lg font-semibold text-foreground">{t.instanceTitle}</p>
          <p className="mt-1 text-sm text-muted-foreground">{t.instanceDescription}</p>
        </div>
        <Link to="/lab">
          <Button>{t.openLabButton}</Button>
        </Link>
      </CardContent>
    </Card>
  )
}

// ─── Onglet Synchronisation ──────────────────────────────────────────────────

function SyncTab({ merged, handleChange, t }: TabProps) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const setActiveSyncJobId = useAppShellStore((s) => s.setActiveSyncJobId)
  const startSyncAll = useStartSyncAll()
  const { data: jobStatus } = useJobStatus(activeSyncJobId ?? '', !!activeSyncJobId)

  const syncRunning =
    !!activeSyncJobId &&
    jobStatus?.status !== 'succeeded' &&
    jobStatus?.status !== 'failed' &&
    jobStatus?.status !== 'cancelled' &&
    jobStatus?.status !== 'interrupted'

  useEffect(() => {
    if (
      activeSyncJobId &&
      (jobStatus?.status === 'succeeded' ||
        jobStatus?.status === 'failed' ||
        jobStatus?.status === 'cancelled' ||
        jobStatus?.status === 'interrupted')
    ) {
      setActiveSyncJobId(null)
    }
  }, [jobStatus?.status, activeSyncJobId, setActiveSyncJobId])

  function handleSync() {
    if (syncRunning) return
    startSyncAll.mutate(undefined, {
      onSuccess: (job) => setActiveSyncJobId(job.job_id),
    })
  }

  return (
    <>
      {/* Synchronisation manuelle */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.manualSyncTitle}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">{t.manualSyncDescription}</p>
          <Button
            onClick={handleSync}
            disabled={syncRunning || startSyncAll.isPending}
            className="shrink-0"
          >
            {syncRunning ? t.manualSyncRunning : t.manualSyncButton}
          </Button>
        </CardContent>
      </Card>

      {/* Synchronisation périodique */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.spnkrTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <ToggleRow label={t.spnkrAutoSync} value={merged.spnkr_auto_sync_enabled ?? false} onChange={(v) => handleChange('spnkr_auto_sync_enabled', v)} />
          <div className="flex items-center justify-between py-3 text-sm">
            <span>{t.spnkrAutoSyncIntervalMinutes}</span>
            <div className="flex items-center gap-1.5">
              <input
                type="number"
                min={5}
                max={1440}
                className="w-20 rounded border border-border bg-background px-2 py-1 text-right text-sm"
                value={merged.spnkr_auto_sync_interval_minutes ?? 360}
                onChange={(e) => handleChange('spnkr_auto_sync_interval_minutes', parseInt(e.target.value, 10) || 360)}
              />
              <span className="text-muted-foreground">{t.spnkrAutoSyncIntervalMinutesUnit}</span>
            </div>
          </div>
          <ToggleRow label={t.spnkrRefreshWithBackfill} value={merged.spnkr_refresh_with_backfill ?? false} onChange={(v) => handleChange('spnkr_refresh_with_backfill', v)} />
        </CardContent>
      </Card>

      {/* Détection de présence */}
      <WatcherCard
        enabled={merged.watcher_presence_enabled ?? false}
        onToggle={(v) => handleChange('watcher_presence_enabled', v)}
        t={t}
      />

      {/* Backfill */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.backfillTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-6 gap-y-1 py-1 sm:grid-cols-3">
            {(
              [
                ['spnkr_refresh_backfill_medals', t.backfillMedals],
                ['spnkr_refresh_backfill_skill', t.backfillSkill],
                ['spnkr_refresh_backfill_aliases', t.backfillAliases],
                ['spnkr_refresh_backfill_personal_scores', t.backfillPersonalScores],
                ['spnkr_refresh_backfill_performance_scores', t.backfillPerfScores],
                ['spnkr_refresh_backfill_lusr', t.backfillLUSR],
                ['spnkr_refresh_backfill_events', t.backfillEvents],
                ['spnkr_refresh_backfill_weapons', t.backfillWeapons],
              ] as const
            ).map(([field, label]) => (
              <ToggleRow
                key={field}
                label={label}
                value={(merged as Record<string, boolean>)[field] ?? false}
                onChange={(v) => handleChange(field, v)}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Escouade — amis par défaut */}
      <FriendGamertagesSection
        value={(merged.friend_gamertags as string[] | undefined) ?? []}
        onChange={(v) => handleChange('friend_gamertags', v)}
      />
    </>
  )
}

// ─── Section amis par défaut ─────────────────────────────────────────────────

function FriendGamertagesSection({
  value,
  onChange,
}: {
  value: string[]
  onChange: (v: string[]) => void
}) {
  const [draft, setDraft] = useState(value.join(', '))

  const handleBlur = () => {
    const parsed = draft
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    onChange(parsed)
    setDraft(parsed.join(', '))
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Mon escouade (amis par défaut)</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground mb-3">
          Gamertags des coéquipiers présélectionnés à l'ouverture de la page Escouade.
          Séparer par des virgules.
        </p>
        <input
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={handleBlur}
          placeholder="Pseudo1, Pseudo2, Pseudo3"
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        />
      </CardContent>
    </Card>
  )
}
