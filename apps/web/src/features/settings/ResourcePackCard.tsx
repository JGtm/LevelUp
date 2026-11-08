/**
 * ResourcePackCard — la carte des Paramètres (onglet Application) : où en est le paquet de ressources (images,
 * sons, données de référence, ffmpeg) livré à part du moteur. Sous la capacité `can_update_app` seulement ; elle
 * s'efface quand le serveur ne gère aucun paquet (réponse 503 : build de développement).
 */
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { canRetryResourcePack, resourcePackPercent, resourcePackView, type ResourcePackView } from './resourcePackLogic'
import { useResourcePackRetry, useResourcePackStatus, type ResourcePackStatus } from './resourcePackQueries'
import { RESOURCE_PACK_TEXT, resourcePackErrorKey, type ResourcePackText } from './resourcePackText'

function viewLine(t: ResourcePackText, view: ResourcePackView, status: ResourcePackStatus): string {
  switch (view) {
    case 'upToDate':
      return t.upToDate
    case 'mismatch':
      return t.mismatch
    case 'waitingGame':
      return t.waitingGame
    case 'downloading':
      return t.downloading.replace('{percent}', String(resourcePackPercent(status)))
    case 'verifying':
      return t.verifying
    case 'ready':
      return t.ready
    case 'failed':
      return t.errors[resourcePackErrorKey(status.code)]
  }
}

export function ResourcePackCard() {
  const locale = useAppShellStore((s) => s.locale)
  const t = RESOURCE_PACK_TEXT[locale]
  const enabled = useDeploymentCapability('can_update_app')
  const { data: status } = useResourcePackStatus(enabled)
  const retry = useResourcePackRetry()
  if (!enabled || !status) return null
  const view = resourcePackView(status)
  const percent = resourcePackPercent(status)
  const failed = view === 'failed'
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3" data-testid="resource-pack-card">
        <p className="text-sm text-muted-foreground">{t.intro}</p>
        <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
          <p role={failed ? 'alert' : 'status'} className={failed ? 'text-destructive' : 'text-foreground'}>
            {viewLine(t, view, status)}
          </p>
          {canRetryResourcePack(view) && (
            <Button size="sm" loading={retry.isPending} onClick={() => retry.mutate()}>
              {t.retry}
            </Button>
          )}
        </div>
        {view === 'downloading' && (
          <div
            role="progressbar"
            aria-label={t.progressLabel}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={percent}
            className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
          >
            <div className="h-full bg-primary transition-all" style={{ width: `${percent}%` }} />
          </div>
        )}
        {retry.isError && (
          <p role="alert" className="text-xs text-destructive">
            {t.retryFailed}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
