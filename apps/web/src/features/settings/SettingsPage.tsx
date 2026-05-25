/**
 * SettingsPage — page des paramètres utilisateur avec onglets.
 *
 * P8.4 (revue 2026-04-29) : tabs extraits dans des fichiers dédiés
 * (GeneralTab, SyncTab, AnalyseTab, BackfillCard) ; ToggleRow/BulletHint/TabProps
 * dans _settingsShared.tsx. Ce fichier ne porte plus que l'orchestrateur.
 */
import { useState, useEffect, useRef } from 'react'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { AccessibilityTab } from '@/features/settings/AccessibilityTab'
import { NotificationsSettingsTab } from '@/features/notifications/NotificationsSettingsTab'
import type { SettingsResponse } from '@/lib/api/types'
import type { TabProps } from './_settingsShared'
import { GeneralTab } from './GeneralTab'
import { SyncTab } from './SyncTab'
import { AnalyseTab } from './AnalyseTab'
import { BackupTab } from './BackupTab'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

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
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.instanceTitle}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <p className="text-sm text-muted-foreground">{t.instanceDescription}</p>
        <Link to="/lab">
          <Button>{t.openLabButton}</Button>
        </Link>
      </CardContent>
    </Card>
  )
}

// ─── Page principale ──────────────────────────────────────────────────────────

export function SettingsPage() {
  const { data: settings, isLoading } = useSettings()
  const mutation = useUpdateSettings()
  const canManageInstance = useAppShellStore((s) => s.capabilities?.can_manage_instance ?? false)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const locale = normalizeSettingsLocale(useAppShellStore((s) => s.locale))
  const t = getSettingsText(locale)
  const tc = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [localSettings, setLocalSettings] = useState<Partial<SettingsResponse>>({})
  const [saveStatus, setSaveStatus] = useState<'saved' | 'error' | null>(null)
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const routerState = useRouterState()
  const navigate = useNavigate()
  const activeTab =
    (new URLSearchParams(routerState.location.search).get('tab') as
      | 'general'
      | 'sync'
      | 'analyse'
      | 'lab'
      | 'users'
      | 'accessibility'
      | 'notifications'
      | 'backup'
      | null) ?? 'general'

  function setActiveTab(tab: 'general' | 'sync' | 'analyse' | 'lab' | 'users' | 'accessibility' | 'notifications' | 'backup') {
    navigate({ to: '/settings', search: { tab }, replace: true }).catch(() => {})
  }

  useEffect(() => {
    if (settings) setLocalSettings(settings)
  }, [settings])

  useEffect(() => {
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
    }
  }, [])

  function handleChange<K extends keyof SettingsResponse>(field: K, value: SettingsResponse[K]) {
    setLocalSettings((prev) => ({ ...prev, [field]: value }))
    mutation.mutate({ [field]: value } as Partial<SettingsResponse>, {
      onSuccess: () => {
        setSaveStatus('saved')
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
        saveTimerRef.current = setTimeout(() => setSaveStatus(null), 2000)
      },
      onError: () => {
        setSaveStatus('error')
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
        saveTimerRef.current = setTimeout(() => setSaveStatus(null), 4000)
      },
    })
  }

  const merged: Partial<SettingsResponse> = localSettings

  if (isLoading) return null

  return (
    <div className="relative flex flex-col">
      {saveStatus && (
        <div className="pointer-events-none absolute right-6 top-4 z-10">
          {saveStatus === 'saved' ? (
            <span className="rounded-md bg-success/10 px-2 py-1 text-sm text-success shadow-sm" role="status" aria-live="polite">
              {t.savedStatus}
            </span>
          ) : (
            <span className="rounded-md bg-destructive/10 px-2 py-1 text-sm text-destructive shadow-sm" role="alert">
              {t.errorStatus}
            </span>
          )}
        </div>
      )}

      {/* Onglets */}
      <div className="border-b border-border px-6">
        <nav className="-mb-px flex gap-4" aria-label={tc('common.settings.tabs_aria')}>
          {(
            [
              { id: 'general', label: t.tabGeneral },
              { id: 'sync', label: t.tabSync },
              { id: 'analyse', label: t.tabAnalyse },
              { id: 'accessibility', label: t.tabAccessibility },
              { id: 'notifications', label: locale === 'en' ? 'Notifications' : 'Notifications' },
              { id: 'backup', label: t.tabBackup },
              ...(canManageInstance ? [{ id: 'lab', label: t.tabLab }] : []),
              ...(isAdmin ? [{ id: 'users', label: t.tabUsers }] : []),
            ] as { id: 'general' | 'sync' | 'analyse' | 'lab' | 'users' | 'accessibility' | 'notifications' | 'backup'; label: string }[]
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
        {activeTab === 'accessibility' && <AccessibilityTab t={t} locale={locale} />}
        {activeTab === 'notifications' && <NotificationsSettingsTab />}
        {activeTab === 'backup' && <BackupTab t={t} />}
      </div>
    </div>
  )
}
