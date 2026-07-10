/**
 * FreshnessPanel — fraîcheur des données par joueur suivi et par titre actif
 * (A4, seuils DC-3). Rendu sur l'onglet État : chaque titre = une sous-section
 * listant les joueurs (dernier match, dernier sync, statut), + l'âge du dernier
 * backup. Best-effort : erreur de chargement = message, jamais de crash.
 * Couleurs exclusivement via tokens sémantiques.
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { PlayerFreshness, TitleFreshnessReport } from '@/lib/api/types'
import { useMonitoringFreshness } from '../monitoring/queries'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'

/** Token d'accent par statut de fraîcheur (unknown = neutre). */
function freshnessToken(status: string): SemanticToken | undefined {
  switch (status) {
    case 'ok':
      return 'success'
    case 'warn':
      return 'warning'
    case 'critical':
      return 'destructive'
    default:
      return undefined
  }
}

export function FreshnessPanel() {
  const { data, isError } = useMonitoringFreshness()
  const tA = useAdminT()
  const locale = useAdminLocale()

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        {tA('admin.freshness.section')}
      </h3>
      {isError ? (
        <p className="text-sm text-destructive">{tA('admin.freshness.unavailable')}</p>
      ) : !data ? (
        <p className="text-sm text-muted-foreground">…</p>
      ) : (
        <div className="space-y-4">
          {(data.titles ?? []).map((t) => (
            <TitleFreshnessSection key={t.title_slug} report={t} />
          ))}
          {data.backup && (
            <p className="text-xs text-muted-foreground">
              {tA('admin.freshness.backup_label')}{' '}
              {data.backup.last_backup_at ? (
                <span title={adminAbsoluteTime(data.backup.last_backup_at, locale)}>
                  {adminRelativeTime(data.backup.last_backup_at, locale)}
                </span>
              ) : (
                <span style={{ color: tokenCssVar('warning') }}>
                  {tA('admin.freshness.backup_never')}
                </span>
              )}
              {!data.backup.enabled && <span> ({tA('admin.freshness.backup_disabled')})</span>}
            </p>
          )}
        </div>
      )}
    </section>
  )
}

function TitleFreshnessSection({ report }: { report: TitleFreshnessReport }) {
  const tA = useAdminT()
  return (
    <div className="space-y-2">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {report.title_slug}
      </h4>
      {report.note ? (
        <p className="text-xs text-muted-foreground">{report.note}</p>
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-3 py-2 font-medium">{tA('admin.freshness.col_player')}</th>
                <th className="px-3 py-2 font-medium">{tA('admin.freshness.col_last_match')}</th>
                <th className="px-3 py-2 font-medium">{tA('admin.freshness.col_last_sync')}</th>
                <th className="px-3 py-2 font-medium">{tA('admin.freshness.col_status')}</th>
              </tr>
            </thead>
            <tbody>
              {(report.players ?? []).map((p) => (
                <FreshnessRow key={p.xuid || p.gamertag} player={p} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function FreshnessRow({ player }: { player: PlayerFreshness }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  const token = freshnessToken(player.status)
  const color = token ? tokenCssVar(token) : undefined
  return (
    <tr className="border-b last:border-b-0 hover:bg-muted/30">
      <td className="px-3 py-2 font-medium text-foreground">{player.gamertag}</td>
      <td className="px-3 py-2 text-xs text-muted-foreground" title={adminAbsoluteTime(player.last_match_at, locale)}>
        {player.last_match_at ? adminRelativeTime(player.last_match_at, locale) : tA('admin.freshness.never')}
      </td>
      <td className="px-3 py-2 text-xs text-muted-foreground" title={adminAbsoluteTime(player.last_sync_ok_at, locale)}>
        {player.last_sync_ok_at ? adminRelativeTime(player.last_sync_ok_at, locale) : tA('admin.freshness.sync_unknown')}
      </td>
      <td className="px-3 py-2">
        <span
          className="inline-flex items-center gap-1.5 rounded-sm bg-muted px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground"
          style={color ? { color } : undefined}
          title={player.reason || player.check_error || undefined}
        >
          {color && <span aria-hidden className="inline-block h-2 w-2 flex-none" style={{ backgroundColor: color }} />}
          {tA(`admin.freshness.status_${player.status}` as Parameters<typeof tA>[0])}
        </span>
      </td>
    </tr>
  )
}
