/**
 * SettingsPage — page des paramètres utilisateur.
 */
import { useState, useEffect } from 'react'
import { Link } from '@tanstack/react-router'

import { PageHeader } from '@/components/shell/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
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

  // État local pour les mises à jour optimistes (feedback immédiat sans attendre le refetch)
  const [localSettings, setLocalSettings] = useState<Partial<SettingsResponse>>({})
  // Indicateur éphémère : 'saved' | 'error' | null
  const [saveStatus, setSaveStatus] = useState<'saved' | 'error' | null>(null)
  const saveTimerRef = useState<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (settings) setLocalSettings(settings)
  }, [settings])

  // Nettoyage du timer au démontage
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
        <Spinner size="lg" label="Chargement des paramètres…" />
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Paramètres"
        subtitle="Configuration de l'application"
        actions={
          saveStatus === 'saved' ? (
            <span className="text-sm text-success" role="status" aria-live="polite">
              ✓ Enregistré
            </span>
          ) : saveStatus === 'error' ? (
            <span className="text-sm text-destructive" role="alert">
              ✗ Erreur lors de la sauvegarde
            </span>
          ) : undefined
        }
      />

      <div className="space-y-6 p-6">
        {canManageInstance && (
          <Card className="border-border bg-card">
            <CardContent className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Instance</p>
                <p className="mt-2 text-lg font-semibold text-foreground">Lab interne</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Ouvrir l&apos;explorateur interne des métadonnées Waypoint, du diff OpenAPI et des diagnostics locaux.
                </p>
              </div>
              <Link to="/lab">
                <Button>Ouvrir le Lab</Button>
              </Link>
            </CardContent>
          </Card>
        )}

        {/* Langue & Interface */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Interface</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-border/50">
            <div className="flex items-center justify-between py-2">
              <span className="text-sm text-foreground">Langue</span>
              <select
                value={merged.lang ?? 'fr'}
                onChange={(e) => handleChange('lang', e.target.value)}
                className="rounded border border-input px-2 py-1 text-sm"
              >
                <option value="fr">Français</option>
                <option value="en">English</option>
              </select>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm text-foreground">Fuseau horaire</span>
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
            <ToggleRow
              label="Afficher les records"
              value={merged.show_records ?? false}
              onChange={(v) => handleChange('show_records', v)}
            />
            <ToggleRow
              label="Normaliser les libellés de modes"
              value={merged.normalize_mode_labels ?? true}
              onChange={(v) => handleChange('normalize_mode_labels', v)}
            />
            <ToggleRow
              label="Exclure BTB du classement carrière"
              value={merged.career_top_exclude_btb ?? false}
              onChange={(v) => handleChange('career_top_exclude_btb', v)}
            />
            <ToggleRow
              label="Vider les caches à l'actualisation"
              value={merged.refresh_clears_caches ?? false}
              onChange={(v) => handleChange('refresh_clears_caches', v)}
            />
          </CardContent>
        </Card>

        {/* Notifications Discord */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Notifications Discord</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-border/50">
            <ToggleRow
              label="Activer les notifications"
              value={merged.discord_notifications_enabled ?? false}
              onChange={(v) => handleChange('discord_notifications_enabled', v)}
            />
            <ToggleRow
              label="Notifier à la synchronisation"
              value={merged.discord_notify_sync ?? false}
              onChange={(v) => handleChange('discord_notify_sync', v)}
            />
            <ToggleRow
              label="Notifier au backfill"
              value={merged.discord_notify_backfill ?? false}
              onChange={(v) => handleChange('discord_notify_backfill', v)}
            />
            <ToggleRow
              label="Notifier pour les nouveaux médias"
              value={merged.discord_notify_new_media ?? false}
              onChange={(v) => handleChange('discord_notify_new_media', v)}
            />
          </CardContent>
        </Card>

        {/* Médias */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Médias</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-border/50">
            <ToggleRow
              label="Surveillance automatique des médias"
              value={merged.media_watcher_enabled ?? false}
              onChange={(v) => handleChange('media_watcher_enabled', v)}
            />
            <div className="flex items-center justify-between py-2">
              <span className="text-sm text-foreground">Tolérance association (min)</span>
              <input
                type="number"
                value={merged.media_tolerance_minutes ?? 10}
                onChange={(e) => handleChange('media_tolerance_minutes', Number(e.target.value))}
                className="w-20 rounded border border-input px-2 py-1 text-sm text-right"
                min={1}
                max={60}
              />
            </div>
          </CardContent>
        </Card>

        {/* Synchronisation SPNKr */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Synchronisation SPNKr</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-border/50">
            <ToggleRow
              label="Lancer un backfill après chaque synchronisation"
              value={merged.spnkr_refresh_with_backfill ?? false}
              onChange={(v) => handleChange('spnkr_refresh_with_backfill', v)}
            />
          </CardContent>
        </Card>

        {/* Backfill — éléments à inclure */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Données à inclure dans le backfill</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-1 py-1">
              {(
                [
                  ['spnkr_refresh_backfill_medals', 'Médailles'],
                  ['spnkr_refresh_backfill_skill', 'Classement (CSR/MMR)'],
                  ['spnkr_refresh_backfill_aliases', 'Alias gamertag'],
                  ['spnkr_refresh_backfill_personal_scores', 'Scores personnels'],
                  ['spnkr_refresh_backfill_performance_scores', 'Scores performance'],
                  ['spnkr_refresh_backfill_lusr', 'LUSR'],
                  ['spnkr_refresh_backfill_events', 'Événements'],
                  ['spnkr_refresh_backfill_weapons', 'Armes'],
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

        {/* Avertissements de cohérence */}
        {merged.discord_notifications_enabled && !merged.discord_webhook_url_present && (
          <div className="rounded-lg border border-warning bg-warning/10 px-4 py-3 text-sm text-warning">
            ⚠️ Les notifications Discord sont activées mais aucun webhook URL n'est configuré.
          </div>
        )}
        {merged.media_watcher_enabled && !merged.media_captures_base_dir && (
          <div className="rounded-lg border border-warning bg-warning/10 px-4 py-3 text-sm text-warning">
            ⚠️ La surveillance des médias est activée mais aucun dossier source n'est défini.
          </div>
        )}

      </div>
    </div>
  )
}
