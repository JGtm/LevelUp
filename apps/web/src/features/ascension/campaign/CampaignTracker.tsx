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
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import type { ImprovementCampaign } from '@/lib/playerProfile'
import { useCampaignMutations } from '@/features/ascension/profile/queries'
import { useProfileI18n } from '@/features/ascension/profile/useProfileI18n'
import { intlLocale } from '@/lib/formatters'

const MIN_MATCHES_FOR_TREND = 20

interface CampaignTrackerProps {
  playerSlug: string
  campaign: ImprovementCampaign
}

export function CampaignTracker({ playerSlug, campaign }: CampaignTrackerProps) {
  const { t, locale } = useProfileI18n()
  const muts = useCampaignMutations(playerSlug)
  const [confirmClose, setConfirmClose] = useState(false)
  const [confirmAbandon, setConfirmAbandon] = useState(false)

  const axisKey = `profile.axis.${campaign.axis}` as ProfileManifestKey
  const lusrKey = `profile.lusr.${campaign.axis}` as ProfileManifestKey
  const axisLabel = t(campaign.axis_kind === 'radar' ? axisKey : lusrKey)
  const startedAt = new Date(campaign.started_at).toLocaleDateString(
    intlLocale(locale),
    { day: '2-digit', month: 'long' },
  )
  const enoughData = campaign.matches_since_start >= MIN_MATCHES_FOR_TREND
  const missing = Math.max(0, MIN_MATCHES_FOR_TREND - campaign.matches_since_start)
  const playlistLabel =
    campaign.playlist_group === 'all'
      ? t('campaign.tracker.playlist_all')
      : campaign.playlist_group
  const delta =
    campaign.current_value_lowess !== undefined
      ? campaign.current_value_lowess - campaign.snapshot_value
      : undefined
  const linkedCount = campaign.linked_challenge_ids?.length ?? 0

  return (
    <section className="sticky top-2 z-10 space-y-3 rounded-lg border border-border bg-card p-4 shadow-sm">
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold uppercase text-muted-foreground">
            {t('campaign.tracker.title')}
          </h2>
          <p className="text-lg font-bold">{axisLabel}</p>
          <p className="text-xs text-muted-foreground">
            {t('campaign.tracker.subtitle', {
              date: startedAt,
              n: campaign.matches_since_start,
              playlist: playlistLabel,
            })}
          </p>
        </div>
        <StatusBadge status={campaign.status} confirmed={campaign.progression_confirmed} />
      </header>

      {campaign.auto_closure_suggested && (
        <AutoClosureNotice reason={campaign.auto_closure_reason} />
      )}

      {!enoughData ? (
        <p className="rounded border border-dashed border-border bg-background p-3 text-sm text-muted-foreground">
          {t('campaign.tracker.insufficient', { missing })}
        </p>
      ) : (
        <TrendBlock campaign={campaign} delta={delta} />
      )}

      <p className="text-xs text-muted-foreground">
        {linkedCount > 0
          ? t('campaign.tracker.linked_count', { n: linkedCount })
          : t('campaign.tracker.linked_none')}
      </p>

      <div className="flex flex-wrap gap-2">
        {campaign.status === 'active' ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => muts.pause.mutate(campaign.id)}
            disabled={muts.pause.isPending}
          >
            {t('campaign.action.pause')}
          </Button>
        ) : campaign.status === 'paused' ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => muts.resume.mutate(campaign.id)}
            disabled={muts.resume.isPending}
          >
            {t('campaign.action.resume')}
          </Button>
        ) : null}
        <Button size="sm" variant="outline" onClick={() => setConfirmClose(true)}>
          {t('campaign.action.close')}
        </Button>
        <Button size="sm" variant="ghost" onClick={() => setConfirmAbandon(true)}>
          {t('campaign.action.abandon')}
        </Button>
      </div>

      <AlertDialog
        open={confirmClose}
        onOpenChange={setConfirmClose}
        title={t('campaign.confirm.close.title')}
        description={t('campaign.confirm.close.description')}
        confirmLabel={t('campaign.action.close')}
        cancelLabel={t('campaign.confirm.cancel')}
        busy={muts.close.isPending}
        onConfirm={() =>
          muts.close.mutate(campaign.id, { onSuccess: () => setConfirmClose(false) })
        }
      />
      <AlertDialog
        open={confirmAbandon}
        onOpenChange={setConfirmAbandon}
        title={t('campaign.confirm.abandon.title')}
        description={t('campaign.confirm.abandon.description')}
        confirmLabel={t('campaign.action.abandon')}
        cancelLabel={t('campaign.confirm.cancel')}
        destructive
        busy={muts.abandon.isPending}
        onConfirm={() =>
          muts.abandon.mutate(campaign.id, { onSuccess: () => setConfirmAbandon(false) })
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
  const { t } = useProfileI18n()
  if (status !== 'active') {
    const key = `campaign.status.${status}` as ProfileManifestKey
    return (
      <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
        {t(key)}
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
        title={t('campaign.status.active_confirmed_tooltip')}
      >
        {t('campaign.status.active_confirmed')}
      </span>
    )
  }
  return (
    <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
      {t('campaign.status.active')}
    </span>
  )
}

function AutoClosureNotice({ reason }: { reason?: string }) {
  const { t } = useProfileI18n()
  if (!reason) return null
  const key = `campaign.auto_closure.${reason}` as ProfileManifestKey
  return (
    <p
      className="rounded border border-dashed border-border p-2 text-xs"
      style={{
        backgroundColor: `color-mix(in srgb, ${tokenCssVar('outcome-loss')} 10%, transparent)`,
      }}
    >
      {t(key)}{' '}
      <span className="text-muted-foreground">{t('campaign.auto_closure.cta')}</span>
    </p>
  )
}

interface TrendBlockProps {
  campaign: ImprovementCampaign
  delta?: number
}

function TrendBlock({ campaign, delta }: TrendBlockProps) {
  const { t } = useProfileI18n()
  return (
    <dl className="grid grid-cols-3 gap-3 text-sm">
      <Pair
        label={t('campaign.tracker.kpi_snapshot')}
        value={campaign.snapshot_value.toFixed(2)}
      />
      <Pair
        label={t('campaign.tracker.kpi_current')}
        value={
          campaign.current_value_lowess !== undefined
            ? campaign.current_value_lowess.toFixed(2)
            : '—'
        }
      />
      <Pair
        label={t('campaign.tracker.kpi_delta')}
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
