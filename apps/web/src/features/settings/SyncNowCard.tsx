/**
 * SyncNowCard — encadré « Synchronisation » de Réglages › Données & Médias : le bouton « Synchroniser
 * maintenant », qui lance tout de suite le même cycle que la synchronisation automatique, rattrapage de
 * l'historique compris, puis l'état de l'historique du joueur par titre suivi (voir `HistoryState`).
 * Offert par la capacité `can_run_sync` : absent de la démo.
 *
 * Trois états du bouton, lus sur le serveur (GET /sync/cycle), qu'il vienne du clic ou de la synchronisation
 * automatique : disponible ; « en cours » tant qu'un cycle tourne ; « Disponible dans 1 min 40 » pendant le
 * délai minimal qui suit un cycle, avec un compte à rebours à la seconde, après quoi il redevient actif seul.
 * Une ligne de texte sous le bouton dit où l'on en est, sans couleur d'alerte. Un refus « trop tôt » (429) n'est
 * pas une erreur : la ligne du délai prend sa place. L'état de l'historique se relit tout seul pendant un cycle
 * et pendant un rattrapage, et reste figé au repos.
 */
import { toast } from 'sonner'
import { apiErrorCode } from '@/lib/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { HistoryState } from './HistoryState'
import { useHistorySyncStatus, useSyncCycle, useSyncNow } from './syncNowQueries'
import { COOLDOWN_CODE, buttonLabel, manualStatusLine, remainingSeconds, startFailureText } from './syncNowLogic'
import { getSyncNowText } from './syncNowText'
import { useNowWhile } from './useNowWhile'

export function SyncNowCard() {
  const canRunSync = useAppShellStore((s) => s.capabilities?.can_run_sync ?? false)
  const locale = useAppShellStore((s) => s.locale)
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  const t = getSyncNowText(locale)
  const { data: cycle } = useSyncCycle(canRunSync)
  const syncNow = useSyncNow()
  const cycleRunning = cycle?.running === true
  const history = useHistorySyncStatus(playerSlug, locale, canRunSync, cycleRunning)
  const availableAt = cycleRunning || !cycle?.manual_available_at ? undefined : Date.parse(cycle.manual_available_at)
  const now = useNowWhile(Number.isNaN(availableAt) ? undefined : availableAt)

  if (!canRunSync) return null

  const running = syncNow.isPending || cycleRunning
  const remaining = running ? 0 : remainingSeconds(cycle?.manual_available_at, now)
  const status = manualStatusLine(t, locale, cycle, running, remaining)

  function handleClick() {
    if (running || remaining > 0) return
    syncNow.mutate(undefined, {
      onSuccess: () => toast.info(t.toastStarted),
      onError: (err) => {
        if (apiErrorCode(err) === COOLDOWN_CODE) return // la relecture de l'état dit à partir de quand
        toast.error(startFailureText(t, err))
      },
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted-foreground">{t.intro}</p>
            <Button onClick={handleClick} disabled={running || remaining > 0} className="shrink-0">
              {buttonLabel(t, running, remaining)}
            </Button>
          </div>
          {status && (
            <p className="text-xs text-muted-foreground sm:text-right" data-testid="sync-now-status" aria-live="polite">
              {status}
            </p>
          )}
        </div>
        <div className="border-t border-border/50 pt-4">
          {history.data ? (
            history.data.titles && history.data.titles.length > 0 ? (
              <HistoryState status={history.data} cycleRunning={cycleRunning} locale={locale} t={t} />
            ) : (
              <EmptyStateNotice size="compact" title={t.emptyTitle} description={t.emptyDescription} />
            )
          ) : history.isError ? (
            <p role="status" className="text-xs text-muted-foreground">
              {t.loadFailed}
            </p>
          ) : (
            <p role="status" className="text-xs text-muted-foreground">
              {t.loading}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
