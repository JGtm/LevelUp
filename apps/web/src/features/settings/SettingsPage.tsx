/**
 * SettingsPage — page des paramètres utilisateur avec onglets.
 *
 * Onglets (préférences utilisateur) : Apparence & Accessibilité · Jeux ·
 * Analyse · Notifications (+ Discord) · Données & Médias · Compte. Cartes dans
 * _settingsCards.tsx / *Tab.tsx ; ToggleRow/BulletHint/TabProps dans
 * _settingsShared.tsx. Ce fichier ne porte que l'orchestrateur. La Sauvegarde
 * (ops d'instance) vit désormais dans Admin · Système.
 */
import { useState, useEffect, useRef } from 'react'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'

import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { AccessibilityTab } from '@/features/settings/AccessibilityTab'
import { SetPasswordCard } from '@/features/auth/SetPasswordCard'
import { NotificationsSettingsTab } from '@/features/notifications/NotificationsSettingsTab'
import type { SettingsResponse } from '@/lib/api/types'
import { AnalyseTab } from './AnalyseTab'
import { TitlesTab } from './TitlesTab'
import { InterfaceCard, DiscordCard, MediaCard } from './_settingsCards'
import { resolveSettingsTab, type SettingsTab } from './tabs'
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
  const activeTab = resolveSettingsTab(
    new URLSearchParams(routerState.location.search).get('tab'),
  )

  function setActiveTab(tab: SettingsTab) {
    navigate({ to: '/settings', search: { tab }, replace: true }).catch(() => {})
  }

  // Copie éditable des réglages serveur : resync quand la requête livre un nouvel
  // objet (ajustement pendant le rendu, pattern React « valeur précédente », au
  // lieu d'un effet). Comportement inchangé : un refetch réaligne l'état local.
  const [prevSettings, setPrevSettings] = useState(settings)
  if (settings && settings !== prevSettings) {
    setPrevSettings(settings)
    setLocalSettings(settings)
  }

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
              { id: 'appearance', label: t.tabAppearance },
              { id: 'titles', label: t.tabTitles },
              { id: 'analyse', label: t.tabAnalyse },
              { id: 'notifications', label: t.tabNotifications },
              { id: 'data', label: t.tabData },
              { id: 'account', label: t.tabAccount },
              // « Sauvegarde » a migré vers Admin · Système ; « Synchronisation » et
              // « Utilisateurs » vers Admin · Sync & Jobs / Accès.
            ] as { id: SettingsTab; label: string }[]
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
        {activeTab === 'appearance' && (
          <>
            <InterfaceCard merged={merged} handleChange={handleChange} t={t} frozen={demoMode} />
            <AccessibilityTab t={t} locale={locale} />
          </>
        )}
        {activeTab === 'titles' && <TitlesTab t={t} frozen={demoMode} />}
        {activeTab === 'analyse' && (
          <AnalyseTab merged={merged} handleChange={handleChange} t={t} frozen={demoMode} />
        )}
        {activeTab === 'notifications' && (
          <>
            <NotificationsSettingsTab />
            {/* Discord = canal d'alerte externe, regroupé avec les notifications. */}
            <DiscordCard merged={merged} handleChange={handleChange} t={t} frozen={demoMode} />
          </>
        )}
        {activeTab === 'data' && (
          <MediaCard merged={merged} handleChange={handleChange} t={t} frozen={demoMode} />
        )}
        {/* Compte — mot de passe opt-in (re-login rapide sans Microsoft) +
            accès à la gestion des groupes/familles (page /groups, auparavant
            accessible uniquement par URL directe — retour utilisateur 2026-07-24). */}
        {activeTab === 'account' && (
          <>
            <SetPasswordCard />
            <div className="rounded-lg border border-border bg-card p-4">
              <h3 className="text-sm font-semibold text-foreground">{t.groupsCardTitle}</h3>
              <p className="mt-1 text-xs text-muted-foreground">{t.groupsCardDescription}</p>
              <Link
                to="/groups"
                className="mt-3 inline-block rounded-md border border-input px-3 py-1.5 text-sm text-foreground hover:bg-accent"
              >
                {t.groupsCardOpen}
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
