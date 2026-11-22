/**
 * SyncNowCard — encadré « Synchronisation » de Réglages › Données & Médias : le bouton « Synchroniser
 * maintenant », qui lance tout de suite le même cycle que la synchronisation automatique, rattrapage de
 * l'historique compris, puis l'état de l'historique du joueur par titre suivi (voir `HistoryState`).
 * Offert par la capacité `can_run_sync` : absent de la démo.
 *
 * Le bouton reste « en cours » tant que le serveur dit qu'un cycle tourne, qu'il vienne du clic ou de la
 * synchronisation automatique. L'état se relit tout seul pendant un cycle et pendant un rattrapage, et reste
 * figé au repos. Le refus d'un lancement et l'échec d'une lecture se disent par un texte écrit ici dans chaque
 * langue.
 */
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { HistoryState } from './HistoryState'
import { useHistorySyncStatus, useSyncCycle, useSyncNow } from './syncNowQueries'
import { startFailureText } from './syncNowLogic'
import { getSyncNowText } from './syncNowText'

export function SyncNowCard() {
  const canRunSync = useAppShellStore((s) => s.capabilities?.can_run_sync ?? false)
  const locale = useAppShellStore((s) => s.locale)
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  const t = getSyncNowText(locale)
  const { data: cycle } = useSyncCycle(canRunSync)
  const syncNow = useSyncNow()
  const cycleRunning = cycle?.running === true
  const history = useHistorySyncStatus(playerSlug, locale, canRunSync, cycleRunning)

  if (!canRunSync) return null

  const running = syncNow.isPending || cycleRunning

  function handleClick() {
    if (running) return
    syncNow.mutate(undefined, {
      onSuccess: () => toast.info(t.toastStarted),
      onError: (err) => toast.error(startFailureText(t, err)),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">{t.intro}</p>
          <Button onClick={handleClick} disabled={running} className="shrink-0">
            {running ? t.buttonRunning : t.button}
          </Button>
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
