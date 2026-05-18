/**
 * CampaignTracker — affichage permanent campagne active (sticky).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.5.5.
 *
 * Trois états visuels :
 *   - "active avec données" : snapshot + actuel lissé + delta + R4 confirmation
 *   - "données insuffisantes" : < 20 matchs post-snapshot → message d'attente
 *   - "auto-closure suggested" (R5) : plateau ou axe sorti du bottom-3 → CTA
 *
 * Actions : Pause / Resume / Close / Abandon — confirmation via AlertDialog
 * pour Close et Abandon (irréversibles).
 */
import { useState } from 'react'
import { AlertDialog } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { tokenCssVar } from '@/lib/accessibility'
import type { ImprovementCampaign } from '@/lib/playerProfile'
import { useCampaignMutations } from '../hooks/usePlayerProfile'

const MIN_MATCHES_FOR_TREND = 20

const AXIS_LABELS_FR: Record<string, string> = {
  combat: 'Combat',
  survival: 'Survie',
  support: 'Support',
  score: 'Score',
  objective: 'Objectif',
  impact: 'Impact',
  kills_vs_expected: 'Kills vs attendus',
  deaths_vs_expected: 'Morts vs attendues',
  win_factor: 'Facteur de victoire',
  damage_efficiency: 'Efficacité dégâts',
  accuracy_delta: 'Précision (delta)',
  medal_exploit: 'Exploits / médailles',
  offensive_conversion: 'Conversion offensive',
  defensive_resistance: 'Résistance défensive',
}

const CLOSURE_REASONS_FR: Record<string, string> = {
  plateau_60d: 'Plateau détecté : pas de variation significative depuis 60 jours.',
  axis_no_longer_priority:
    'Cet axe n’est plus dans tes axes prioritaires actuels.',
}

interface CampaignTrackerProps {
  playerSlug: string
  campaign: ImprovementCampaign
}

export function CampaignTracker({ playerSlug, campaign }: CampaignTrackerProps) {
  const muts = useCampaignMutations(playerSlug)
  const [confirmClose, setConfirmClose] = useState(false)
  const [confirmAbandon, setConfirmAbandon] = useState(false)

  const axisLabel = AXIS_LABELS_FR[campaign.axis] ?? campaign.axis
  const startedAt = new Date(campaign.started_at).toLocaleDateString('fr-FR', {
    day: '2-digit',
    month: 'long',
  })
  const enoughData = campaign.matches_since_start >= MIN_MATCHES_FOR_TREND
  const missing = Math.max(0, MIN_MATCHES_FOR_TREND - campaign.matches_since_start)
  const playlistLabel =
    campaign.playlist_group === 'all'
      ? 'toutes playlists'
      : campaign.playlist_group
  const delta =
    campaign.current_value_lowess !== undefined
      ? campaign.current_value_lowess - campaign.snapshot_value
      : undefined
  const linkedCount = campaign.linked_challenge_ids?.length ?? 0

  return (
    <section
      className="sticky top-2 z-10 space-y-3 rounded-lg border border-border bg-card p-4 shadow-sm"
      aria-label="campaign.tracker"
    >
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold uppercase text-muted-foreground">
            Campagne en cours
          </h2>
          <p className="text-lg font-bold">{axisLabel}</p>
          <p className="text-xs text-muted-foreground">
            Démarrée le {startedAt} · {campaign.matches_since_start} match
            {campaign.matches_since_start > 1 ? 's' : ''} sur {playlistLabel}
          </p>
        </div>
        <StatusBadge status={campaign.status} confirmed={campaign.progression_confirmed} />
      </header>

      {campaign.auto_closure_suggested && (
        <AutoClosureNotice reason={campaign.auto_closure_reason} />
      )}

      {!enoughData ? (
        <p className="rounded border border-dashed border-border bg-background p-3 text-sm text-muted-foreground">
          Joue encore <span className="font-semibold">{missing}</span> match
          {missing > 1 ? 's' : ''} sur ta playlist cible pour voir ta tendance.
        </p>
      ) : (
        <TrendBlock campaign={campaign} delta={delta} />
      )}

      <p className="text-xs text-muted-foreground">
        {linkedCount > 0
          ? `${linkedCount} défi${linkedCount > 1 ? 's' : ''} lié${linkedCount > 1 ? 's' : ''} à cette campagne.`
          : 'Aucun défi lié pour l’instant — lance-en un depuis la section progression.'}
      </p>

      <div className="flex flex-wrap gap-2">
        {campaign.status === 'active' ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => muts.pause.mutate(campaign.id)}
            disabled={muts.pause.isPending}
          >
            Pause
          </Button>
        ) : campaign.status === 'paused' ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => muts.resume.mutate(campaign.id)}
            disabled={muts.resume.isPending}
          >
            Reprendre
          </Button>
        ) : null}
        <Button size="sm" variant="outline" onClick={() => setConfirmClose(true)}>
          Clore
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setConfirmAbandon(true)}
        >
          Abandonner
        </Button>
      </div>

      <AlertDialog
        open={confirmClose}
        onOpenChange={setConfirmClose}
        title="Clore cette campagne ?"
        description="La campagne sera marquée comme complétée. Tu pourras en démarrer une nouvelle sur un autre axe."
        confirmLabel="Clore"
        cancelLabel="Annuler"
        busy={muts.close.isPending}
        onConfirm={() =>
          muts.close.mutate(campaign.id, {
            onSuccess: () => setConfirmClose(false),
          })
        }
      />
      <AlertDialog
        open={confirmAbandon}
        onOpenChange={setConfirmAbandon}
        title="Abandonner cette campagne ?"
        description="Pas de pénalité — la campagne sera simplement marquée comme abandonnée."
        confirmLabel="Abandonner"
        cancelLabel="Annuler"
        destructive
        busy={muts.abandon.isPending}
        onConfirm={() =>
          muts.abandon.mutate(campaign.id, {
            onSuccess: () => setConfirmAbandon(false),
          })
        }
      />
    </section>
  )
}

interface StatusBadgeProps {
  status: ImprovementCampaign['status']
  confirmed: boolean
}

function StatusBadge({ status, confirmed }: StatusBadgeProps) {
  if (status !== 'active') {
    const label = status === 'paused' ? 'En pause' : status === 'completed' ? 'Clôturée' : 'Abandonnée'
    return (
      <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
        {label}
      </span>
    )
  }
  if (confirmed) {
    return (
      <span
        className="rounded-full px-2 py-0.5 text-xs font-medium"
        style={{
          backgroundColor: `color-mix(in srgb, ${tokenCssVar('outcome-win')} 20%, transparent)`,
          color: tokenCssVar('outcome-win'),
        }}
        title="p < 0.05 (Mann-Whitney U)"
      >
        ✓ Progression confirmée
      </span>
    )
  }
  return (
    <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
      En cours
    </span>
  )
}

function AutoClosureNotice({ reason }: { reason?: string }) {
  const text = reason && CLOSURE_REASONS_FR[reason] ? CLOSURE_REASONS_FR[reason] : null
  if (!text) return null
  return (
    <p
      className="rounded border border-dashed border-border p-2 text-xs"
      style={{
        backgroundColor: `color-mix(in srgb, ${tokenCssVar('outcome-loss')} 10%, transparent)`,
      }}
    >
      {text} <span className="text-muted-foreground">Tu peux clore et démarrer un nouvel axe.</span>
    </p>
  )
}

interface TrendBlockProps {
  campaign: ImprovementCampaign
  delta?: number
}

function TrendBlock({ campaign, delta }: TrendBlockProps) {
  return (
    <dl className="grid grid-cols-3 gap-3 text-sm">
      <Pair label="Snapshot" value={campaign.snapshot_value.toFixed(2)} />
      <Pair
        label="Actuel (lissé)"
        value={
          campaign.current_value_lowess !== undefined
            ? campaign.current_value_lowess.toFixed(2)
            : '—'
        }
      />
      <Pair
        label="Delta"
        value={delta !== undefined ? formatDelta(delta) : '—'}
        accent={delta !== undefined ? (delta > 0 ? 'win' : delta < 0 ? 'loss' : undefined) : undefined}
      />
    </dl>
  )
}

interface PairProps {
  label: string
  value: string
  accent?: 'win' | 'loss'
}

function Pair({ label, value, accent }: PairProps) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd
        className="font-mono text-base font-semibold"
        style={
          accent
            ? { color: tokenCssVar(accent === 'win' ? 'outcome-win' : 'outcome-loss') }
            : undefined
        }
      >
        {value}
      </dd>
    </div>
  )
}

function formatDelta(d: number): string {
  const rounded = Math.round(d * 100) / 100
  const sign = rounded > 0 ? '+' : rounded < 0 ? '−' : ''
  return `${sign}${Math.abs(rounded).toFixed(2)}`
}
