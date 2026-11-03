/**
 * DesktopIntegrationCard — l'intégration de l'application de bureau avec le poste (lot L21) : détection
 * du jeu en cours, et les réglages qui touchent à Windows. Chaque ligne est offerte par sa capacité de
 * déploiement (`can_watch_game_process`, …) : sur le site, l'onglet n'existe pas.
 *
 * Chaque changement part tout de suite en PATCH /settings, comme le reste des Réglages, et se confirme par
 * le badge d'enregistrement de la page. Un refus du serveur arrive rédigé dans la langue de la requête et
 * s'affiche tel quel sous les réglages, en plus du badge d'échec.
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { apiErrorMessage } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import type { UpdateSettingsRequest } from '@/lib/api/types'
import { useSettings, useUpdateSettings } from './queries'
import { useDesktopStatus } from './desktopIntegrationQueries'
import { getDesktopIntegrationText, type DesktopIntegrationText } from './desktopIntegrationText'
import { ToggleRow } from './_settingsShared'
import { flashSaveStatus } from './saveFeedbackStore'

/** Le texte de l'état de la tâche planifiée ; l'erreur dit sa cause par le code fixe du serveur. */
function taskStatusText(t: DesktopIntegrationText, task: { state: string; error_code?: string }): string {
  switch (task.state) {
    case 'present':
      return t.taskPresent
    case 'absent':
      return t.taskAbsent
    case 'error':
      return task.error_code === 'unsupported_path' ? t.taskErrorPath : t.taskError
    default:
      return t.taskUnknown
  }
}

export function DesktopIntegrationCard() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getDesktopIntegrationText(locale)
  const frozen = !useDeploymentCapability('can_edit_settings')
  const canWatchGame = useDeploymentCapability('can_watch_game_process')
  const canIntegrateShell = useDeploymentCapability('can_integrate_shell')
  const canScheduleRefresh = useDeploymentCapability('can_schedule_refresh')
  const qc = useQueryClient()
  const { data: settings } = useSettings()
  const { data: status } = useDesktopStatus(canWatchGame || canScheduleRefresh)
  const update = useUpdateSettings()
  const [error, setError] = useState<string | null>(null)

  function save(req: UpdateSettingsRequest) {
    setError(null)
    update.mutate(req, {
      onSuccess: () => {
        flashSaveStatus('saved')
        void qc.invalidateQueries({ queryKey: queryKeys.desktopStatus })
      },
      onError: (err) => {
        flashSaveStatus('error')
        setError(apiErrorMessage(err) ?? t.saveFailed)
      },
    })
  }

  const gameWatch = settings?.game_process_watch ?? true
  const nativeNotifications = settings?.native_notifications ?? false
  const closeToTray = settings?.close_to_tray ?? false
  const startWithWindows = settings?.start_with_windows ?? false
  const scheduledRefresh = settings?.scheduled_refresh ?? false
  const task = status?.scheduled_refresh

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{t.intro}</p>
        <fieldset disabled={frozen} className="m-0 min-w-0 space-y-3 border-0 p-0">
          {canWatchGame && (
            <div>
              <ToggleRow
                label={t.gameWatchLabel}
                value={gameWatch}
                onChange={(v) => save({ game_process_watch: v })}
              />
              <p className="text-xs text-muted-foreground">{t.gameWatchHint}</p>
              {gameWatch && status && (
                <p className="mt-1 text-xs text-foreground" role="status">
                  {status.game_running ? t.gameRunning : t.gameNotRunning}
                </p>
              )}
            </div>
          )}
          {canIntegrateShell && (
            <div>
              <ToggleRow
                label={t.nativeNotificationsLabel}
                value={nativeNotifications}
                onChange={(v) => save({ native_notifications: v })}
              />
              <p className="text-xs text-muted-foreground">{t.nativeNotificationsHint}</p>
            </div>
          )}
          {canIntegrateShell && (
            <div>
              <ToggleRow
                label={t.closeToTrayLabel}
                value={closeToTray}
                onChange={(v) => save({ close_to_tray: v })}
              />
              <p className="text-xs text-muted-foreground">{t.closeToTrayHint}</p>
            </div>
          )}
          {canIntegrateShell && (
            <div>
              <ToggleRow
                label={t.startWithWindowsLabel}
                value={startWithWindows}
                onChange={(v) => save({ start_with_windows: v })}
              />
              <p className="text-xs text-muted-foreground">{t.startWithWindowsHint}</p>
            </div>
          )}
          {canScheduleRefresh && (
            <div>
              <ToggleRow
                label={t.scheduledRefreshLabel}
                value={scheduledRefresh}
                onChange={(v) => save({ scheduled_refresh: v })}
              />
              <p className="text-xs text-muted-foreground">{t.scheduledRefreshHint}</p>
              {task && (
                <p
                  className={`mt-1 text-xs ${task.state === 'error' ? 'text-destructive' : 'text-foreground'}`}
                  role={task.state === 'error' ? 'alert' : 'status'}
                >
                  {taskStatusText(t, task)}
                </p>
              )}
            </div>
          )}
        </fieldset>
        {error && (
          <p role="alert" className="text-xs text-destructive">
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
