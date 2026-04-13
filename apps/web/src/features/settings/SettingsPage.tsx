/**
 * SettingsPage — page des paramètres utilisateur.
 */
import { PageHeader } from '@/components/shell/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { useSettings, useUpdateSettings } from '@/features/setup/queries'
import type { SettingsResponse } from '@/lib/api/types'

function ToggleRow({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between py-2">
      <span className="text-sm text-gray-700">{label}</span>
      <button
        onClick={() => onChange(!value)}
        className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
          value ? 'bg-purple-600' : 'bg-gray-200'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform ${
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
  const { dirtyFields, setDirtyField, clearDirtyFields, setLastSavedAt } = useSettingsDraftStore()

  function handleSave() {
    mutation.mutate(dirtyFields, {
      onSuccess: () => {
        clearDirtyFields()
        setLastSavedAt(new Date().toISOString())
      },
    })
  }

  const merged: Partial<SettingsResponse> = { ...settings, ...dirtyFields }
  const hasDirty = Object.keys(dirtyFields).length > 0

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
          hasDirty ? (
            <div className="flex gap-2">
              <Button variant="ghost" size="sm" onClick={clearDirtyFields}>
                Annuler
              </Button>
              <Button size="sm" onClick={handleSave} loading={mutation.isPending}>
                Enregistrer
              </Button>
            </div>
          ) : undefined
        }
      />

      <div className="space-y-6 p-6">
        {/* Langue */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Interface</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-gray-50">
            <div className="flex items-center justify-between py-2">
              <span className="text-sm text-gray-700">Langue</span>
              <select
                value={merged.lang ?? 'fr'}
                onChange={(e) => setDirtyField('lang', e.target.value)}
                className="rounded border border-gray-300 px-2 py-1 text-sm"
              >
                <option value="fr">Français</option>
                <option value="en">English</option>
              </select>
            </div>
            <ToggleRow
              label="Afficher les records"
              value={merged.show_records ?? false}
              onChange={(v) => setDirtyField('show_records', v)}
            />
            <ToggleRow
              label="Normaliser les libellés de modes"
              value={merged.normalize_mode_labels ?? true}
              onChange={(v) => setDirtyField('normalize_mode_labels', v)}
            />
          </CardContent>
        </Card>

        {/* Notifications Discord */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Notifications Discord</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-gray-50">
            <ToggleRow
              label="Activer les notifications"
              value={merged.discord_notifications_enabled ?? false}
              onChange={(v) => setDirtyField('discord_notifications_enabled', v)}
            />
            <ToggleRow
              label="Notifier à la synchronisation"
              value={merged.discord_notify_sync ?? false}
              onChange={(v) => setDirtyField('discord_notify_sync', v)}
            />
            <ToggleRow
              label="Notifier au backfill"
              value={merged.discord_notify_backfill ?? false}
              onChange={(v) => setDirtyField('discord_notify_backfill', v)}
            />
            <ToggleRow
              label="Notifier pour les nouveaux médias"
              value={merged.discord_notify_new_media ?? false}
              onChange={(v) => setDirtyField('discord_notify_new_media', v)}
            />
          </CardContent>
        </Card>

        {/* Médias */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Médias</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 divide-y divide-gray-50">
            <ToggleRow
              label="Surveillance automatique des médias"
              value={merged.media_watcher_enabled ?? false}
              onChange={(v) => setDirtyField('media_watcher_enabled', v)}
            />
            <div className="flex items-center justify-between py-2">
              <span className="text-sm text-gray-700">Tolérance association (min)</span>
              <input
                type="number"
                value={merged.media_tolerance_minutes ?? 10}
                onChange={(e) => setDirtyField('media_tolerance_minutes', Number(e.target.value))}
                className="w-20 rounded border border-gray-300 px-2 py-1 text-sm text-right"
                min={1}
                max={60}
              />
            </div>
          </CardContent>
        </Card>

        {mutation.isError && (
          <p className="text-sm text-red-600">
            Erreur lors de la sauvegarde. Veuillez réessayer.
          </p>
        )}
        {mutation.isSuccess && (
          <p className="text-sm text-green-600">✓ Paramètres enregistrés.</p>
        )}
      </div>
    </div>
  )
}
