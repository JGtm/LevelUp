/**
 * SettingsPage — page des paramètres utilisateur avec onglets.
 *
 * P8.4 (revue 2026-04-29) : tabs extraits dans des fichiers dédiés
 * (GeneralTab, SyncTab, AnalyseTab, BackfillCard) ; ToggleRow/BulletHint/TabProps
 * dans _settingsShared.tsx. Ce fichier ne porte plus que l'orchestrateur.
 */
import { useState, useEffect, useRef } from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'

import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { AccessibilityTab } from '@/features/settings/AccessibilityTab'
import { SetPasswordCard } from '@/features/auth/SetPasswordCard'
import { NotificationsSettingsTab } from '@/features/notifications/NotificationsSettingsTab'
import type { SettingsResponse } from '@/lib/api/types'
import { GeneralTab } from './GeneralTab'
import { AnalyseTab } from './AnalyseTab'
import { BackupTab } from './BackupTab'
import { TitlesTab } from './TitlesTab'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

// ─── Page principale ──────────────────────────────────────────────────────────

export function SettingsPage() {
  const { data: settings, isLoading } = useSettings()
  const mutation = useUpdateSettings()
  const demoMode = useAppShellStore((s) => s.demoMode)
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
      | 'titles'
      | 'sync'
      | 'analyse'
      | 'lab'
      | 'users'
      | 'accessibility'
      | 'notifications'
      | 'backup'
      | null) ?? 'general'

  function setActiveTab(tab: 'general' | 'titles' | 'sync' | 'analyse' | 'lab' | 'users' | 'accessibility' | 'notifications' | 'backup') {
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
      {demoMode && (
        <div
          className="mx-6 mt-4 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm text-muted-foreground"
          role="note"
        >
          {locale === 'en'
            ? 'Demo mode: settings are frozen. Only language and accessibility can be changed (this session only).'
            : 'Mode démo : les paramètres sont figés. Seules la langue et l’accessibilité sont modifiables (le temps de la session).'}
        </div>
      )}
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
              { id: 'titles', label: t.tabTitles },
              { id: 'analyse', label: t.tabAnalyse },
              { id: 'accessibility', label: t.tabAccessibility },
              { id: 'notifications', label: locale === 'en' ? 'Notifications' : 'Notifications' },
              { id: 'backup', label: t.tabBackup },
              // « Synchronisation » et « Utilisateurs » ont migré vers la page Admin
              // (Admin · Sync & Jobs / Accès). Cf. accès direct « Administration » dans le menu.
            ] as { id: 'general' | 'titles' | 'sync' | 'analyse' | 'lab' | 'users' | 'accessibility' | 'notifications' | 'backup'; label: string }[]
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
          <>
            <GeneralTab merged={merged} handleChange={handleChange} t={t} frozen={demoMode} />
            {/* Mon compte — mot de passe opt-in (re-login rapide sans Microsoft).
                Self-service conservé ici depuis le retrait de l'onglet « Comptes ». */}
            <SetPasswordCard />
          </>
        )}
        {activeTab === 'titles' && <TitlesTab t={t} frozen={demoMode} />}
        {activeTab === 'analyse' && (
          <AnalyseTab merged={merged} handleChange={handleChange} t={t} frozen={demoMode} />
        )}
        {activeTab === 'accessibility' && <AccessibilityTab t={t} locale={locale} />}
        {activeTab === 'notifications' && <NotificationsSettingsTab />}
        {activeTab === 'backup' && <BackupTab t={t} frozen={demoMode} />}
      </div>
    </div>
  )
}
