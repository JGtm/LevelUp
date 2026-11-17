/**
 * ReplayGenerateAction — générer le rejeu d'un match qui n'en a pas, sur le poste local (capacité
 * `can_generate_replays`). UN composant, deux surfaces :
 *
 * - `panel` : le bloc de la page du rejeu quand le match n'en a pas. Il garde sa bordure et sa phrase d'état vide
 *   (`emptyLabel`, que la page fournit) ; sous la capacité, il y ajoute l'état de la génération. L'attente reprend
 *   l'indicateur « analyse en cours » de la vue du match.
 * - `inline` : l'en-tête de la page du match. Bouton contour avec son libellé, puis pastille compacte pendant
 *   l'attente ; rien dans les autres états. Sans la capacité, rien.
 *
 * Un état sans action possible n'affiche aucun bouton (`generationView`). Aucun texte ne conseille de recapturer
 * un jeton.
 */
import { Link } from '@tanstack/react-router'

import { AnalysisPendingSlot, AnalysisPendingSpinner } from '@/components/ui/analysis-pending-slot'
import { Button } from '@/components/ui/button'
import { useTitleSlug } from '@/lib/title-routing'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { useAppShellStore } from '@/stores/appShellStore'

import { generationView, requestErrorKey } from './logic'
import { useReplayGenerationStatus, useRequestReplayGeneration } from './queries'
import { generationMessage, REPLAY_GENERATION_TEXT } from './text'

type ReplayGenerateActionProps = { playerSlug: string; matchId: string } & (
  | { variant: 'panel'; emptyLabel: string }
  | { variant: 'inline' }
)

const OUTLINE_LINK_CLASS =
  'inline-flex h-9 items-center justify-center rounded-md border border-border bg-transparent px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'

export function ReplayGenerateAction(props: ReplayGenerateActionProps) {
  const { playerSlug, matchId } = props
  const locale = useAppShellStore((s) => s.locale)
  const t = REPLAY_GENERATION_TEXT[locale]
  const titleSlug = useTitleSlug()
  const enabled = useDeploymentCapability('can_generate_replays')
  const { data: status } = useReplayGenerationStatus(playerSlug, matchId, enabled)
  const request = useRequestReplayGeneration(playerSlug, matchId)

  const view = enabled ? generationView(status) : generationView(undefined)
  const message = view.messageKey ? generationMessage(t, view.messageKey, status?.position ?? 0) : null
  const errorKey = requestErrorKey(request.error)
  const inline = props.variant === 'inline'

  const action =
    view.action === 'open' ? (
      <Link
        to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay"
        params={{ titleSlug, playerSlug, matchId }}
        className={OUTLINE_LINK_CLASS}
      >
        {t.open}
      </Link>
    ) : view.action ? (
      <Button
        variant={inline ? 'outline' : 'default'}
        loading={request.isPending}
        title={inline && message ? message : undefined}
        onClick={() => request.mutate()}
      >
        {view.action === 'retry' ? t.retry : t.generate}
      </Button>
    ) : null

  if (inline) {
    if (view.pending) {
      return (
        <span
          role="status"
          title={message ?? undefined}
          className="inline-flex h-9 items-center gap-2 rounded-md border border-dashed border-info/40 bg-muted/40 px-2.5 text-xs italic text-muted-foreground"
        >
          <AnalysisPendingSpinner />
          <span>{message}</span>
        </span>
      )
    }
    return (
      <>
        {errorKey && (
          <span role="alert" className="text-xs text-destructive">
            {t[errorKey]}
          </span>
        )}
        {action}
      </>
    )
  }

  return (
    <div
      data-testid="replay-generation-panel"
      className="space-y-3 rounded-lg border border-border bg-card px-3 py-12 text-center"
    >
      <p className="text-sm text-muted-foreground">{props.emptyLabel}</p>
      {view.pending && message && <AnalysisPendingSlot label={message} className="mx-auto max-w-md" />}
      {!view.pending && message && <p className="text-sm text-muted-foreground">{message}</p>}
      {errorKey && (
        <p role="alert" className="text-sm text-destructive">
          {t[errorKey]}
        </p>
      )}
      {action}
    </div>
  )
}
