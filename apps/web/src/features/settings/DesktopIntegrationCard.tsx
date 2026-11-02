/**
 * DesktopIntegrationCard — l'intégration de l'application de bureau avec le poste (lot L21) : détection
 * du jeu en cours, et les réglages qui touchent à Windows. Chaque ligne est offerte par sa capacité de
 * déploiement (`can_watch_game_process`, …) : sur le site, l'onglet n'existe pas.
 *
 * Chaque changement part tout de suite en PATCH /settings, comme le reste des Réglages. Un refus du
 * serveur arrive rédigé dans la langue de la requête et s'affiche tel quel.
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
import { getDesktopIntegrationText } from './desktopIntegrationText'
import { ToggleRow } from './_settingsShared'

export function DesktopIntegrationCard() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getDesktopIntegrationText(locale)
  const frozen = !useDeploymentCapability('can_edit_settings')
  const canWatchGame = useDeploymentCapability('can_watch_game_process')
  const canIntegrateShell = useDeploymentCapability('can_integrate_shell')
  const qc = useQueryClient()
  const { data: settings } = useSettings()
  const { data: status } = useDesktopStatus(canWatchGame)
  const update = useUpdateSettings()
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null)

  function save(req: UpdateSettingsRequest) {
    setMessage(null)
    update.mutate(req, {
      onSuccess: () => {
        setMessage({ ok: true, text: t.saved })
        void qc.invalidateQueries({ queryKey: queryKeys.desktopStatus })
      },
      onError: (err) => setMessage({ ok: false, text: apiErrorMessage(err) ?? t.saveFailed }),
    })
  }

  const gameWatch = settings?.game_process_watch ?? true
  const nativeNotifications = settings?.native_notifications ?? false

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
        </fieldset>
        {message && (
          <p
            role={message.ok ? 'status' : 'alert'}
            className={`text-xs ${message.ok ? 'text-success' : 'text-destructive'}`}
          >
            {message.text}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
