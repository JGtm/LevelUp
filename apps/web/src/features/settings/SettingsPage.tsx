/**
 * SettingsPage — page des paramètres utilisateur avec onglets.
 */
import { useState, useEffect } from 'react'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings, useUpdateSettings, useScanMedia, useRecalculateSessions, useStartBackfill } from '@/features/settings/queries'
import { useStartSyncAll, useJobStatus } from '@/features/setup/queries'
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import type { HintBullets } from '@/features/settings/i18n'
import { WatcherSectionBody } from '@/features/settings/WatcherCard'
import type { SettingsResponse } from '@/lib/api/types'

function ToggleRow({ label, value, onChange, disabled }: { label: string; value: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <div className={`flex items-center justify-between py-2 ${disabled ? 'opacity-40' : ''}`}>
      <span className="text-sm text-foreground">{label}</span>
      <button
        onClick={() => !disabled && onChange(!value)}
        disabled={disabled}
        className={`relative inline-flex h-5 w-9 flex-shrink-0 rounded-full border-2 border-transparent transition-colors ${
          disabled ? 'cursor-not-allowed' : 'cursor-pointer'
        } ${value && !disabled ? 'bg-primary' : 'bg-muted'}`}
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

function BulletHint({ hint }: { hint: HintBullets }) {
  return (
    <div className="mt-1 space-y-1 text-xs text-muted-foreground">
      <p>{hint.intro}</p>
      <ul className="ml-3 list-disc space-y-0.5">
        {hint.items.map((item, i) => (
          <li key={i}>{item}</li>
        ))}
      </ul>
      {hint.outro && <p>{hint.outro}</p>}
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

  const routerState = useRouterState()
  const navigate = useNavigate()
  const activeTab =
    (new URLSearchParams(routerState.location.search).get('tab') as
      | 'general'
      | 'sync'
      | 'analyse'
      | 'lab'
      | 'users'
      | null) ?? 'general'

  function setActiveTab(tab: 'general' | 'sync' | 'analyse' | 'lab' | 'users') {
    navigate({ to: '/settings', search: { tab }, replace: true }).catch(() => {})
  }

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
      {saveStatus && (
        <div className="flex justify-end px-6 pt-4">
          {saveStatus === 'saved' ? (
            <span className="text-sm text-success" role="status" aria-live="polite">
              {t.savedStatus}
            </span>
          ) : (
            <span className="text-sm text-destructive" role="alert">
              {t.errorStatus}
            </span>
          )}
        </div>
      )}

      {/* Onglets */}
      <div className="border-b border-border px-6">
        <nav className="-mb-px flex gap-4" aria-label="Onglets paramètres">
          {(
            [
              { id: 'general', label: t.tabGeneral },
              { id: 'sync', label: t.tabSync },
              { id: 'analyse', label: t.tabAnalyse },
              ...(canManageInstance ? [{ id: 'lab', label: t.tabLab }] : []),
              ...(isAdmin ? [{ id: 'users', label: t.tabUsers }] : []),
            ] as { id: 'general' | 'sync' | 'analyse' | 'lab' | 'users'; label: string }[]
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
        {activeTab === 'analyse' && (
          <AnalyseTab merged={merged} handleChange={handleChange} t={t} />
        )}
        {activeTab === 'lab' && <LabTab t={t} />}
        {activeTab === 'users' && <UsersTab merged={merged} handleChange={handleChange} t={t} />}
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
  const scanMedia = useScanMedia()
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
          <ToggleRow label={t.discordNotifySync} value={merged.discord_notify_sync ?? false} onChange={(v) => handleChange('discord_notify_sync', v)} disabled={!merged.discord_notifications_enabled} />
          <ToggleRow label={t.discordNotifyBackfill} value={merged.discord_notify_backfill ?? false} onChange={(v) => handleChange('discord_notify_backfill', v)} disabled={!merged.discord_notifications_enabled} />
          <ToggleRow label={t.discordNotifyNewMedia} value={merged.discord_notify_new_media ?? false} onChange={(v) => handleChange('discord_notify_new_media', v)} disabled={!merged.discord_notifications_enabled} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.mediaTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <ToggleRow label={t.mediaWatcherEnabled} value={merged.media_watcher_enabled ?? false} onChange={(v) => handleChange('media_watcher_enabled', v)} />
          <div className="flex flex-col gap-1 py-2">
            <span className="text-sm text-foreground">{t.mediaBaseDirLabel}</span>
            <input
              type="text"
              value={merged.media_captures_base_dir ?? ''}
              onChange={(e) => handleChange('media_captures_base_dir', e.target.value)}
              placeholder={t.mediaBaseDirPlaceholder}
              className="w-full rounded border border-input px-2 py-1 text-sm font-mono"
            />
            <span className="text-xs text-muted-foreground">{t.mediaBaseDirHint}</span>
          </div>
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
          <div className="flex items-center justify-between py-2">
            <Button
              variant="outline"
              size="sm"
              disabled={scanMedia.isPending}
              onClick={() => scanMedia.mutate()}
            >
              {scanMedia.isPending
                ? t.mediaScanRunning
                : scanMedia.isSuccess
                  ? t.mediaScanDone
                  : scanMedia.isError
                    ? t.mediaScanError
                    : t.mediaScanButton}
            </Button>
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

function UsersTab({ merged, handleChange, t }: TabProps) {
  return (
    <>
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

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.authProviderTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 divide-y divide-border/50">
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-foreground">{t.authProviderLabel}</span>
            <select
              value={merged.auth_provider ?? 'msal'}
              onChange={(e) => handleChange('auth_provider', e.target.value)}
              className="rounded border border-input px-2 py-1 text-sm"
            >
              <option value="msal">{t.authProviderMsal}</option>
              <option value="sisu">{t.authProviderSisu}</option>
            </select>
          </div>
          <p className="pt-2 text-xs text-muted-foreground">{t.authProviderHint}</p>
        </CardContent>
      </Card>
    </>
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

// ─── Card Backfill (recalcul rétroactif) ─────────────────────────────────────

interface BackfillCardProps {
  t: ReturnType<typeof getSettingsText>
}

function BackfillCard({ t }: BackfillCardProps) {
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const realPlayers = availablePlayers.filter((p) => !p.is_demo)
  const firstSlug = realPlayers[0]?.player_slug ?? ''

  const [scope, setScope] = useState({
    medals: false,
    skill: false,
    aliases: false,
    personal_scores: false,
    performance_scores: false,
    lusr: false,
    events: false,
    weapons: false,
  })
  const [selectedSlug, setSelectedSlug] = useState<string>(firstSlug)
  const [forceRescan, setForceRescan] = useState(false)
  const [showForceConfirm, setShowForceConfirm] = useState(false)
  const [activeJobId, setActiveJobId] = useState<string | null>(null)

  // Si la liste de joueurs arrive après le premier render, initialiser selectedSlug.
  useEffect(() => {
    if (!selectedSlug && firstSlug) setSelectedSlug(firstSlug)
  }, [firstSlug, selectedSlug])

  const startBackfill = useStartBackfill()
  const { data: jobStatus } = useJobStatus(activeJobId ?? '', !!activeJobId)

  const running =
    !!activeJobId &&
    jobStatus?.status !== 'succeeded' &&
    jobStatus?.status !== 'failed' &&
    jobStatus?.status !== 'cancelled' &&
    jobStatus?.status !== 'interrupted'

  const anyChecked = Object.values(scope).some(Boolean)
  const canRun = anyChecked && !!selectedSlug && !running && !startBackfill.isPending

  function runBackfill(force: boolean) {
    startBackfill.mutate(
      { player_slug: selectedSlug, ...scope, force_rescan: force },
      { onSuccess: (job) => setActiveJobId(job.job_id) },
    )
  }

  function handleRunClick() {
    if (!canRun) return
    if (forceRescan) {
      setShowForceConfirm(true)
    } else {
      runBackfill(false)
    }
  }

  function confirmForce() {
    setShowForceConfirm(false)
    runBackfill(true)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.backfillTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Sélection des types de données */}
        <div className="grid grid-cols-2 gap-x-6 gap-y-1 py-1 sm:grid-cols-3">
          {(
            [
              ['medals', t.backfillMedals],
              ['skill', t.backfillSkill],
              ['aliases', t.backfillAliases],
              ['personal_scores', t.backfillPersonalScores],
              ['performance_scores', t.backfillPerfScores],
              ['lusr', t.backfillLUSR],
              ['events', t.backfillEvents],
              ['weapons', t.backfillWeapons],
            ] as const
          ).map(([field, label]) => (
            <ToggleRow
              key={field}
              label={label}
              value={scope[field]}
              onChange={(v) => setScope((s) => ({ ...s, [field]: v }))}
              disabled={running}
            />
          ))}
        </div>

        {/* Joueur */}
        <div className="flex items-center justify-between gap-3 border-t border-border/50 pt-3 text-sm">
          <span>{t.backfillPlayerLabel}</span>
          <select
            value={selectedSlug}
            onChange={(e) => setSelectedSlug(e.target.value)}
            disabled={running || realPlayers.length === 0}
            className="rounded-md border border-input bg-background px-2 py-1 text-sm disabled:cursor-not-allowed disabled:opacity-50"
          >
            {realPlayers.map((p) => (
              <option key={p.player_slug} value={p.player_slug}>{p.gamertag}</option>
            ))}
          </select>
        </div>

        {/* Forcer */}
        <ToggleRow
          label={t.backfillForceLabel}
          value={forceRescan}
          onChange={setForceRescan}
          disabled={running}
        />

        {/* Bouton + hint */}
        <div className="flex flex-col gap-2">
          <Button
            onClick={handleRunClick}
            disabled={!canRun}
            className="self-start"
          >
            {running ? t.backfillRunningLabel : t.backfillRunButton}
          </Button>
          {!anyChecked && (
            <p className="text-xs text-muted-foreground">{t.backfillNoScopeHint}</p>
          )}
        </div>

        {/* Confirmation forcer */}
        {showForceConfirm && (
          <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
            <p className="font-medium text-foreground">{t.backfillForceConfirmTitle}</p>
            <p className="mt-1 text-xs text-muted-foreground">{t.backfillForceConfirmBody}</p>
            <div className="mt-3 flex gap-2">
              <Button variant="destructive" size="sm" onClick={confirmForce}>
                {t.backfillForceConfirmOk}
              </Button>
              <Button variant="outline" size="sm" onClick={() => setShowForceConfirm(false)}>
                {t.backfillForceConfirmCancel}
              </Button>
            </div>
          </div>
        )}

        {/* Warnings retournés par le backend après exécution */}
        {jobStatus?.status === 'succeeded' && jobStatus.warnings && jobStatus.warnings.length > 0 && (
          <div className="rounded-md border border-amber-500/40 bg-amber-500/5 p-3 text-sm">
            <p className="font-medium text-amber-700 dark:text-amber-400">{t.backfillWarningsHeader}</p>
            <ul className="mt-1 list-disc pl-5 text-xs text-muted-foreground">
              {jobStatus.warnings.map((w, i) => (
                <li key={i}>{w}</li>
              ))}
            </ul>
          </div>
        )}
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

      {/* Synchronisation automatique (planifiée + watcher fusionnés) */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.autoSyncTitle}</CardTitle>
        </CardHeader>
        <CardContent className="divide-y divide-border/50">
          {/* Sous-section : Synchronisation planifiée */}
          <div className="pb-4">
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t.schedulerSectionTitle}
            </h3>
            <div className="space-y-1 divide-y divide-border/30">
              <ToggleRow
                label={t.spnkrAutoSync}
                value={merged.spnkr_auto_sync_enabled ?? false}
                onChange={(v) => handleChange('spnkr_auto_sync_enabled', v)}
              />
              <div
                className={`flex items-center justify-between py-3 text-sm ${
                  merged.spnkr_auto_sync_enabled ? '' : 'opacity-40'
                }`}
              >
                <span>{t.spnkrAutoSyncIntervalMinutes}</span>
                <div className="flex items-center gap-1.5">
                  <input
                    type="number"
                    min={5}
                    max={1440}
                    disabled={!merged.spnkr_auto_sync_enabled}
                    className="w-20 rounded border border-border bg-background px-2 py-1 text-right text-sm disabled:cursor-not-allowed"
                    value={merged.spnkr_auto_sync_interval_minutes ?? 360}
                    onChange={(e) => handleChange('spnkr_auto_sync_interval_minutes', parseInt(e.target.value, 10) || 360)}
                  />
                  <span className="text-muted-foreground">{t.spnkrAutoSyncIntervalMinutesUnit}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Sous-section : Détection de présence Xbox (watcher RTA/XSTS) */}
          <div className="pt-4">
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t.watcherSectionTitle}
            </h3>
            <div className="space-y-1 divide-y divide-border/30">
              <ToggleRow
                label={t.watcherPresenceEnabled}
                value={merged.watcher_presence_enabled ?? false}
                onChange={(v) => handleChange('watcher_presence_enabled', v)}
              />
              <div className="pt-1">
                <WatcherSectionBody enabled={merged.watcher_presence_enabled ?? false} t={t} />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Backfill — Recalcul rétroactif */}
      <BackfillCard t={t} />

      {/* Escouade — amis par défaut */}
      <FriendGamertagesSection
        value={(merged.friend_gamertags as string[] | undefined) ?? []}
        onChange={(v) => handleChange('friend_gamertags', v)}
      />
    </>
  )
}

// ─── Onglet Analyse ──────────────────────────────────────────────────────────

function AnalyseTab({ merged, handleChange, t }: TabProps) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const syncRunning = !!activeSyncJobId
  const recalculate = useRecalculateSessions()
  const [showRecalcConfirm, setShowRecalcConfirm] = useState(false)

  return (
    <>
      {/* Card : Regroupement de sessions */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.sessionGroupingTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 divide-y divide-border/50">
          {/* Délai entre matchs */}
          <div className="flex flex-col gap-1 py-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-foreground">{t.sessionGapLabel}</span>
              <div className="flex items-center gap-1.5">
                <input
                  type="number"
                  min={0}
                  max={1440}
                  className="w-20 rounded border border-border bg-background px-2 py-1 text-right text-sm"
                  value={merged.session_gap_minutes ?? 120}
                  onChange={(e) =>
                    handleChange('session_gap_minutes', parseInt(e.target.value, 10) || 120)
                  }
                />
                <span className="text-sm text-muted-foreground">{t.sessionGapUnit}</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{t.sessionGapHint}</p>
          </div>

          {/* Composition de l'équipe */}
          <div className="flex flex-col gap-1 py-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-foreground">{t.sessionTeamChangeLabel}</span>
              <select
                value={merged.session_team_change_mode ?? 'friends'}
                onChange={(e) =>
                  handleChange(
                    'session_team_change_mode',
                    e.target.value as 'ignore' | 'group' | 'friends',
                  )
                }
                className="rounded border border-input px-2 py-1 text-sm"
              >
                <option value="ignore">{t.sessionTeamChangeIgnore}</option>
                <option value="group">{t.sessionTeamChangeGroup}</option>
                <option value="friends">{t.sessionTeamChangeFriends}</option>
              </select>
            </div>
            <BulletHint hint={t.sessionTeamChangeHint} />
          </div>

          {/* Couper si passage ranked ↔ social */}
          <div className="flex flex-col gap-1 py-2">
            <ToggleRow
              label={t.sessionSplitRankedLabel}
              value={merged.session_split_on_ranked_change ?? false}
              onChange={(v) => handleChange('session_split_on_ranked_change', v)}
            />
            <p className="text-xs text-muted-foreground">{t.sessionSplitRankedHint}</p>
          </div>

          {/* Bouton recalcul */}
          <div className="pt-2">
            {!showRecalcConfirm ? (
              <Button
                variant="outline"
                size="sm"
                disabled={syncRunning || recalculate.isPending}
                onClick={() => setShowRecalcConfirm(true)}
                title={syncRunning ? t.sessionRecalcRunning : undefined}
              >
                {recalculate.isPending ? t.sessionRecalcRunning : t.sessionRecalcButton}
              </Button>
            ) : (
              <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
                <p className="font-medium text-foreground">{t.sessionRecalcConfirmTitle}</p>
                <p className="mt-1 text-xs text-muted-foreground">{t.sessionRecalcConfirmBody}</p>
                <div className="mt-3 flex gap-2">
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => { recalculate.mutate(); setShowRecalcConfirm(false) }}
                  >
                    {t.sessionRecalcConfirmOk}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowRecalcConfirm(false)}
                  >
                    {t.sessionRecalcConfirmCancel}
                  </Button>
                </div>
              </div>
            )}
            {syncRunning && (
              <p className="mt-1 text-xs text-muted-foreground">{t.sessionRecalcPending}</p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Card : Badges de performance */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.badgesTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 divide-y divide-border/50">
          {/* Sensibilité */}
          <div className="flex flex-col gap-2 py-2">
            <span className="text-sm text-foreground">{t.badgeSensitivityLabel}</span>
            <div className="flex gap-4">
              {(
                [
                  ['relaxed', t.badgeSensitivityRelaxed],
                  ['standard', t.badgeSensitivityStandard],
                  ['strict', t.badgeSensitivityStrict],
                ] as const
              ).map(([val, label]) => (
                <label key={val} className="flex cursor-pointer items-center gap-1.5 text-sm">
                  <input
                    type="radio"
                    name="badge_sensitivity"
                    value={val}
                    checked={(merged.outcome_badge_sensitivity ?? 'standard') === val}
                    onChange={() => handleChange('outcome_badge_sensitivity', val)}
                  />
                  {label}
                </label>
              ))}
            </div>
            <BulletHint hint={t.badgeSensitivityHint} />
          </div>

          {/* Exclure bots des badges */}
          <div className="flex flex-col gap-1 py-2">
            <ToggleRow
              label={t.badgeExcludeBotsFromBadgesLabel}
              value={merged.outcome_exclude_bot_matches_from_badges ?? true}
              onChange={(v) => handleChange('outcome_exclude_bot_matches_from_badges', v)}
            />
            <p className="text-xs text-muted-foreground">{t.badgeExcludeBotsFromBadgesHint}</p>
          </div>

          {/* Exclure bots des records */}
          <div className="flex flex-col gap-1 py-2">
            <ToggleRow
              label={t.badgeExcludeBotsFromRecordsLabel}
              value={merged.outcome_exclude_bot_matches_from_records ?? false}
              onChange={(v) => handleChange('outcome_exclude_bot_matches_from_records', v)}
            />
            <p className="text-xs text-muted-foreground">{t.badgeExcludeBotsFromRecordsHint}</p>
          </div>
        </CardContent>
      </Card>
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
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Mon escouade (amis par défaut)</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground mb-3">
          Gamertags des coéquipiers présélectionnés à l'ouverture de la page Escouade.
          Les joueurs déjà configurés dans l'app apparaissent en priorité.
        </p>
        <GamertagCombobox
          selected={value}
          onChange={onChange}
          allowFreeInput={true}
          placeholder="Rechercher ou saisir un gamertag…"
        />
      </CardContent>
    </Card>
  )
}
