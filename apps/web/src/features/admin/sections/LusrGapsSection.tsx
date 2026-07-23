/**
 * LusrGapsSection — garde-fou « trous d'intérieur » LUSR : matchs éligibles sans
 * note LUSR, sous le watermark de leur groupe (notes définitivement absentes →
 * réparables par replay chronologique). Un panneau par titre actif : couverture,
 * agrégats, santé du garde-fou, joueurs impactés + bouton « Recalculer » (replay).
 */
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { apiErrorMessage } from '@/lib/api/client'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { AdminActionButton } from '../components/AdminActionButton'
import {
  useLusrGaps,
  type AdminLusrGaps,
  type AdminLusrGapPlayer,
  type AdminLusrGuardrailHealth,
} from '../monitoring/queries'
import { useRecomputeLusrGaps } from '../monitoring/mutations'
import { useAdminTitles } from '../titles/queries'
import { useAdminT, type TAdmin } from '../useAdminText'
import type { Locale } from '@/lib/i18n/locale'

export function LusrGapsSection() {
  const tA = useAdminT()
  const { data: titles } = useAdminTitles()
  const active = (titles?.titles ?? []).filter((t) => t.status === 'active')
  if (active.length === 0) return null
  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground">{tA('admin.lusr.hint')}</p>
      {active.map((t) => (
        <TitleLusrGapsCard key={t.slug} slug={t.slug} name={t.name} tA={tA} />
      ))}
    </div>
  )
}

function TitleLusrGapsCard({ slug, name, tA }: { slug: string; name: string; tA: TAdmin }) {
  const locale = useAppShellStore((s) => s.locale)
  const { data, isLoading, isError, refetch, isFetching } = useLusrGaps(slug)
  const recompute = useRecomputeLusrGaps(slug)

  async function handleRecompute(player: string, gamertag: string): Promise<string | null> {
    try {
      const res = await recompute.mutateAsync(player)
      toast.success(`${tA('admin.lusr.recompute_done')} — ${gamertag} : ${res.updated}`)
    } catch (err) {
      toast.error(apiErrorMessage(err) ?? tA('admin.lusr.recompute_failed'))
    }
    return null
  }

  return (
    <Card className="overflow-hidden">
      <CardContent className="space-y-3 pt-6">
        <div className="flex items-center justify-between gap-2">
          <span className="text-sm font-semibold text-foreground">{name}</span>
          <div className="flex items-center gap-3">
            {data && (
              <span className="text-xs tabular-nums text-muted-foreground">
                {tA('admin.lusr.coverage')} · {data.coverage_percent.toFixed(0)} %
              </span>
            )}
            <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
              {isFetching ? tA('admin.lusr.loading') : tA('admin.lusr.refresh')}
            </Button>
          </div>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{tA('admin.lusr.loading')}</p>
        ) : isError || !data ? (
          <p className="text-sm text-destructive">{tA('admin.lusr.load_failed')}</p>
        ) : data.eligible_total === 0 ? (
          <p className="text-sm text-muted-foreground">{tA('admin.lusr.empty')}</p>
        ) : (
          <div className="space-y-3">
            <LusrCoverageBar data={data} />
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <Stat label={tA('admin.lusr.eligible')} value={data.eligible_total} />
              <Stat label={tA('admin.lusr.rated')} value={data.rated_total} accent="success" />
              <Stat
                label={tA('admin.lusr.interior_gaps')}
                value={data.interior_gaps_total}
                accent={data.interior_gaps_total > 0 ? 'destructive' : 'success'}
              />
              <Stat label={tA('admin.lusr.pending_recent')} value={data.pending_recent_total} />
            </div>
            <GuardrailLine g={data.guardrail} locale={locale} tA={tA} />
            <ImpactedPlayers players={data.players} tA={tA} onRecompute={handleRecompute} />
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/** Barre empilée rated (résolu) / pending (attente) / interior (trou) sur éligibles. */
function LusrCoverageBar({ data }: { data: AdminLusrGaps }) {
  const total = Math.max(1, data.eligible_total)
  const seg = (n: number, token: SemanticToken) =>
    n > 0 ? (
      <div key={token} style={{ width: `${(n / total) * 100}%`, backgroundColor: tokenCssVar(token) }} />
    ) : null
  return (
    <div className="flex h-2 w-full overflow-hidden rounded-sm bg-muted">
      {seg(data.rated_total, 'success')}
      {seg(data.pending_recent_total, 'info')}
      {seg(data.interior_gaps_total, 'destructive')}
    </div>
  )
}

function Stat({ label, value, accent }: { label: string; value: number; accent?: SemanticToken }) {
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className="text-lg font-semibold tabular-nums"
        style={accent ? { color: tokenCssVar(accent) } : undefined}
      >
        {value.toLocaleString()}
      </div>
    </div>
  )
}

function GuardrailLine({
  g,
  locale,
  tA,
}: {
  g: AdminLusrGuardrailHealth
  locale: Locale
  tA: TAdmin
}) {
  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
      <span>
        {tA('admin.lusr.guardrail_held')} :{' '}
        <span className="tabular-nums text-foreground">{g.held_watermark}</span>
      </span>
      <span>
        {tA('admin.lusr.guardrail_owner_missing')} :{' '}
        <span className="tabular-nums text-foreground">{g.owner_missing}</span>
      </span>
      <span>
        {tA('admin.lusr.guardrail_last_audit')} :{' '}
        {g.last_audit_at
          ? new Date(g.last_audit_at).toLocaleString(intlLocale(locale))
          : tA('admin.lusr.guardrail_never')}
      </span>
    </div>
  )
}

function ImpactedPlayers({
  players,
  tA,
  onRecompute,
}: {
  players: AdminLusrGapPlayer[]
  tA: TAdmin
  onRecompute: (player: string, gamertag: string) => Promise<string | null>
}) {
  const impacted = players.filter((p) => p.interior_gaps > 0 || p.check_error)
  if (impacted.length === 0) {
    return (
      <p className="text-xs" style={{ color: tokenCssVar('success') }}>
        {tA('admin.lusr.no_gaps')}
      </p>
    )
  }
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-muted-foreground">
        {tA('admin.lusr.players_impacted')}
      </div>
      <ul className="divide-y divide-border/60 rounded-sm border border-border/60">
        {impacted.map((p) => (
          <PlayerRow key={p.player_slug || p.gamertag} p={p} tA={tA} onRecompute={onRecompute} />
        ))}
      </ul>
    </div>
  )
}

function PlayerRow({
  p,
  tA,
  onRecompute,
}: {
  p: AdminLusrGapPlayer
  tA: TAdmin
  onRecompute: (player: string, gamertag: string) => Promise<string | null>
}) {
  return (
    <li className="px-2 py-1.5 text-xs">
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium text-foreground">{p.gamertag}</span>
        <div className="flex items-center gap-2">
          {p.check_error ? (
            <span className="rounded bg-muted px-2 py-0.5 text-destructive">
              {tA('admin.lusr.check_error')}
            </span>
          ) : (
            <span className="tabular-nums" style={{ color: tokenCssVar('destructive') }}>
              {p.interior_gaps} {tA('admin.lusr.interior_gaps')}
            </span>
          )}
          {p.interior_gaps > 0 && (
            <AdminActionButton
              label={tA('admin.lusr.recompute')}
              busyLabel={tA('admin.lusr.recompute_busy')}
              confirmMessage={tA('admin.lusr.recompute_confirm')}
              onRun={() => onRecompute(p.player_slug || p.gamertag, p.gamertag)}
            />
          )}
        </div>
      </div>
      {p.check_error && <p className="mt-1 text-muted-foreground">{p.check_error}</p>}
      {!p.check_error && p.top_gaps.length > 0 && (
        <div className="mt-1 truncate font-mono text-muted-foreground/70">
          {p.top_gaps.map((g) => g.match_id.slice(0, 8)).join(', ')}
        </div>
      )}
    </li>
  )
}
