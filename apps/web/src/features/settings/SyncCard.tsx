/**
 * SyncCard — synchronisation du joueur affiché (Réglages, onglet Données & Médias) : le bouton qui
 * lance `POST /players/{slug}/sync`, et l'état de l'historique (rattrapage des matchs anciens en cours,
 * ou historique complet), tels que les dit le serveur (`GET /players/{slug}/sync/status`).
 *
 * Aucune action impossible : sans droit d'écriture sur le joueur, ou titre en pause pour lui (la phrase dit
 * où le réactiver), pas de bouton ; pendant un job, le bouton le dit et ne relance rien ; pendant le délai
 * du lancement manuel, il est désactivé et le délai restant est dit. Au repos l'état est relu toutes les 15 s : une
 * synchronisation lancée ailleurs (automatique, accueil) désactive le bouton sans attendre un clic refusé. En démo la carte est figée et n'affiche pas l'état de l'historique (le serveur n'en a
 * pas de réel à servir).
 */
import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { apiErrorCode } from '@/lib/api/client'
import { fillTemplate } from '@/lib/i18n/template'
import { usePlayerSyncStatus } from '@/lib/players/playerSyncStatus'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { useStartDeltaSync } from '@/features/setup/queries'
import { jobEnded, resolveSyncCardView, type HistoryState } from './syncCardLogic'
import { getSyncCardText, type SyncCardText } from './syncCardText'

const HISTORY_TEXT: Record<Exclude<HistoryState, 'hidden'>, (t: SyncCardText) => string> = {
  complete: (t) => t.historyComplete,
  catching_up: (t) => t.historyCatchingUp,
  paused_incomplete: (t) => t.historyPausedIncomplete,
}

export function SyncCard() {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const setActiveSyncJobId = useAppShellStore((s) => s.setActiveSyncJobId)
  const frozen = !useDeploymentCapability('can_edit_settings')
  const playerSlug = currentPlayer?.player_slug ?? ''
  const canWrite = useCanWritePlayer(playerSlug)
  const startSync = useStartDeltaSync()
  const queryClient = useQueryClient()
  const [starting, setStarting] = useState(false)
  const t = getSyncCardText(locale)

  // Démo : aucun état à lire (le serveur n'y sert pas d'historique réel), la requête reste éteinte.
  const { data: served, isError } = usePlayerSyncStatus(frozen ? '' : playerSlug, { pollIdle: true })
  // Une réponse d'un autre joueur (changement de joueur en cours de route) ne se lit jamais.
  const status = served?.player_slug === playerSlug ? served : undefined

  // Fin du job lancé d'ici : l'état se relit (historique atteint ou non), et le job n'est plus « actif »
  // pour le reste de l'application.
  const ended = jobEnded(status, activeSyncJobId)
  useEffect(() => {
    if (!ended) return
    setActiveSyncJobId(null)
    void queryClient.invalidateQueries({ queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug) })
  }, [ended, playerSlug, titleSlug, queryClient, setActiveSyncJobId])

  if (!currentPlayer) return null

  const view = resolveSyncCardView({
    status,
    canWrite,
    syncEnabled: currentPlayer.sync_enabled,
    frozen,
    starting,
  })
  const refreshStatus = () => queryClient.invalidateQueries({ queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug) })

  async function startNow() {
    if (!view.canStart) return
    setStarting(true)
    try {
      const started = await startSync.mutateAsync(playerSlug)
      setActiveSyncJobId(started.job_id)
      toast.info(t.toastStarted)
    } catch (err) {
      const code = apiErrorCode(err)
      if (code === 'sync_cooldown') toast.error(t.toastCooldown)
      else if (code === 'sync_already_active') toast.error(t.toastAlreadyActive)
      else toast.error(t.toastFailed)
    } finally {
      // Le refus, comme le succès, change l'état réel (délai, job en vol) : on le relit avant de rendre la main.
      await refreshStatus()
      setStarting(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {canWrite && view.paused && (
          <p role="status" className="text-sm text-muted-foreground" data-testid="sync-paused">
            {t.paused}
          </p>
        )}
        {canWrite && !view.paused && (
          <fieldset disabled={frozen} className="m-0 min-w-0 space-y-3 border-0 p-0">
            <p className="text-sm text-muted-foreground">
              {fillTemplate(t.intro, { player: currentPlayer.gamertag })}
            </p>
            <Button onClick={() => void startNow()} disabled={!view.canStart}>
              {view.running ? t.running : t.start}
            </Button>
            {!view.running && view.cooldownMinutes > 0 && (
              <p role="status" className="text-xs text-muted-foreground" data-testid="sync-cooldown">
                {fillTemplate(t.cooldown, { minutes: view.cooldownMinutes })}
              </p>
            )}
          </fieldset>
        )}
        {view.history !== 'hidden' && (
          <p className="text-sm text-foreground" data-testid="sync-history">
            {HISTORY_TEXT[view.history](t)}
          </p>
        )}
        {isError && !status && !frozen && (
          <p role="status" className="text-xs text-muted-foreground">
            {t.statusUnavailable}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
