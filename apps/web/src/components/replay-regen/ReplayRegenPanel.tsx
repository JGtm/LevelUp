/**
 * ReplayRegenPanel — l'état de la régénération des rejeux et ses boutons, UN composant pour les deux surfaces :
 * le bandeau de la page de rejeu (`variant="banner"`, avec « Plus tard ») et la carte des Paramètres
 * (`variant="row"`, qui dit aussi « tout est à jour »). Aucune copie : texte, boutons et barre de progression
 * viennent d'ici.
 *
 * Les boutons suivent la phase (`regenPhase`) : « Régénérer » à l'offre, « Pause » en cours, « Reprendre » en
 * pause. Un clic envoie la commande ; le nouvel état revient dans la réponse.
 */
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { regenPercent, regenPhase, regenPhaseActive, type RegenPhase } from './logic'
import { useReplayRegenCommand, type ReplayRegenStatus } from './queries'
import { REPLAY_REGEN_TEXT, type ReplayRegenText } from './text'

interface PanelProps {
  status: ReplayRegenStatus | undefined
  variant: 'banner' | 'row'
  /** « Plus tard » (bandeau seulement) : masque l'offre pour la session. */
  onLater?: () => void
}

function fill(template: string, values: Record<string, number>): string {
  return Object.entries(values).reduce((out, [k, v]) => out.replace(`{${k}}`, String(v)), template)
}

/** La phrase d'état de la phase courante. */
function phaseLine(t: ReplayRegenText, phase: RegenPhase, s: ReplayRegenStatus | undefined): string {
  const done = s?.done ?? 0
  const total = s?.total ?? 0
  switch (phase) {
    case 'checking':
      return t.checking
    case 'upToDate':
      return t.upToDate
    case 'offer':
      if ((s?.stale ?? 0) === 0) return s?.raw === 1 ? t.rawOne : fill(t.rawMany, { count: s?.raw ?? 0 })
      return s?.stale === 1 ? t.staleOne : fill(t.staleMany, { count: s?.stale ?? 0 })
    case 'preparing':
      return t.preparing
    case 'running':
      return fill(t.running, { done, total })
    case 'compressing':
      return fill(t.compressing, { count: s?.compressed ?? 0 })
    case 'paused':
      return fill(t.paused, { done, total })
    case 'waitingGame':
      return fill(t.waitingGame, { done, total })
    case 'compressionPaused':
      return t.compressionPaused
    case 'compressionWaitingGame':
      return t.compressionWaitingGame
  }
}

export function ReplayRegenPanel({ status, variant, onLater }: PanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = REPLAY_REGEN_TEXT[locale]
  const command = useReplayRegenCommand()
  const phase = regenPhase(status)
  const active = regenPhaseActive(phase)
  const rawOnly = phase === 'offer' && (status?.stale ?? 0) === 0
  const failed = status?.failed ?? 0
  const compressed = status?.compressed ?? 0
  const compressFailed = status?.compress_failed ?? 0

  return (
    <div
      className={
        variant === 'banner'
          ? 'flex flex-col gap-2 rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm'
          : 'flex flex-col gap-2 text-sm'
      }
      data-testid={`replay-regen-${variant}`}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p role="status" className="text-foreground">
          {phaseLine(t, phase, status)}
        </p>
        <div className="flex shrink-0 items-center gap-2">
          {phase === 'offer' && (
            <Button size="sm" loading={command.isPending} onClick={() => command.mutate('start')}>
              {rawOnly ? t.compress : t.regenerate}
            </Button>
          )}
          {phase === 'offer' && variant === 'banner' && onLater && (
            <Button size="sm" variant="outline" onClick={onLater}>
              {t.later}
            </Button>
          )}
          {(phase === 'running' ||
            phase === 'compressing' ||
            phase === 'waitingGame' ||
            phase === 'compressionWaitingGame') && (
            <Button size="sm" variant="outline" disabled={command.isPending} onClick={() => command.mutate('pause')}>
              {t.pause}
            </Button>
          )}
          {(phase === 'paused' || phase === 'compressionPaused') && (
            <Button size="sm" disabled={command.isPending} onClick={() => command.mutate('resume')}>
              {t.resume}
            </Button>
          )}
        </div>
      </div>
      {phase === 'offer' && <p className="text-xs text-muted-foreground">{rawOnly ? t.rawHint : t.hint}</p>}
      {active && status && status.total > 0 && (
        <div
          role="progressbar"
          aria-label={t.progressLabel}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={regenPercent(status)}
          className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
        >
          <div className="h-full bg-primary transition-all" style={{ width: `${regenPercent(status)}%` }} />
        </div>
      )}
      {failed > 0 && !active && (
        <p role="status" className="text-xs text-muted-foreground">
          {failed === 1 ? t.failedOne : fill(t.failedMany, { count: failed })}
        </p>
      )}
      {compressed > 0 && !active && (
        <p role="status" className="text-xs text-muted-foreground">
          {compressed === 1 ? t.compressedOne : fill(t.compressedMany, { count: compressed })}
        </p>
      )}
      {compressFailed > 0 && !active && (
        <p role="status" className="text-xs text-muted-foreground">
          {compressFailed === 1 ? t.compressFailedOne : fill(t.compressFailedMany, { count: compressFailed })}
        </p>
      )}
      {command.isError && (
        <p role="alert" className="text-xs text-destructive">
          {t.commandFailed}
        </p>
      )}
    </div>
  )
}
