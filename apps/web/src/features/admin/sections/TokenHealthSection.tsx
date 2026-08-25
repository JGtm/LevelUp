/**
 * TokenHealthSection — Santé des tokens (Accès / XSTS / Refresh par joueur,
 * ADR 0023). Extraction 1:1 depuis l'ancienne AdminPage.
 *
 * La famille « Accès » s'appelait « MSAL » avant ADR 0023 Phase 5 (2026-08-25) :
 * elle mesure l'expiration de l'access_token Microsoft persisté, pas un cache
 * MSAL — lequel n'existe plus depuis le retrait de MSAL (2026-07-15).
 */
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { TokenStatus } from '@/lib/api/types'
import type { CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAdminTokenHealth } from '../queries'
import { credentialSourceParts, hasLegacyCredentialSource, TOKEN_ERROR_KEY } from '../tokenHealthDisplay'
import { useT, useDateLocale, type T } from '../useAdminText'

const TOKEN_STATUS_KEY: Record<TokenStatus, CommonManifestKey> = {
  ok: 'common.admin.token_status_ok',
  expiring: 'common.admin.token_status_expiring',
  expired: 'common.admin.token_status_expired',
  absent: 'common.admin.token_status_absent',
  reauth: 'common.admin.token_status_reauth',
}

function tokenStatusColor(status: TokenStatus): string | undefined {
  switch (status) {
    case 'ok':
      return tokenCssVar('success')
    case 'expiring':
      return tokenCssVar('warning')
    case 'expired':
    case 'reauth':
      return tokenCssVar('destructive')
    default:
      return undefined // absent → neutre (text-muted-foreground)
  }
}

function TokenBadge({ kind, status, t }: { kind: string; status: TokenStatus; t: T }) {
  const color = tokenStatusColor(status)
  return (
    <span className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
      {kind}:{' '}
      <span className={color ? 'font-semibold' : undefined} style={color ? { color } : undefined}>
        {t(TOKEN_STATUS_KEY[status])}
      </span>
    </span>
  )
}

/**
 * Source de credentials du dernier scan. Depuis ADR 0023 Phase 5, seul « store »
 * est possible ; toute autre valeur signale une régression (affichée en warning).
 */
function CredentialSourceChip({ source, t }: { source?: string; t: T }) {
  if (!source) return null
  const unknown = source === 'unknown'
  const parts = unknown ? [] : credentialSourceParts(source)
  const legacy = hasLegacyCredentialSource(parts)
  return (
    <span className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground" title={unknown ? undefined : source}>
      {t('common.admin.token_source')}:{' '}
      <span
        className={legacy ? 'font-semibold' : undefined}
        style={legacy ? { color: tokenCssVar('warning') } : undefined}
      >
        {unknown ? t('common.admin.token_source_unknown') : parts.join('+')}
      </span>
    </span>
  )
}

export function TokenHealthSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminTokenHealth()
  const t = useT()
  const dateLocale = useDateLocale()

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{t('common.admin.tokens_section')}</h2>
            <p className="max-w-xl text-xs text-muted-foreground">{t('common.admin.tokens_desc')}</p>
          </div>
          <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? t('common.admin.loading') : t('common.admin.refresh')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : isError ? (
          <p className="text-sm text-destructive">{t('common.admin.tokens_unavailable')}</p>
        ) : data?.store_unavailable ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.tokens_store_unavailable')}</p>
        ) : !data?.players?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.no_tracked_players')}</p>
        ) : (
          <div className="space-y-3">
            {data.players.map((p) => (
              <div key={p.xuid || p.gamertag} className="rounded-md border px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="font-medium text-foreground">{p.gamertag || p.xuid}</span>
                  <div className="flex flex-wrap items-center justify-end gap-2">
                    {p.load_error ? (
                      <span className="rounded bg-muted px-2 py-0.5 text-xs text-destructive">
                        {p.load_error}
                      </span>
                    ) : (
                      <>
                        <CredentialSourceChip source={p.credential_source} t={t} />
                        <TokenBadge kind={t('common.admin.token_refresh')} status={p.refresh as TokenStatus} t={t} />
                        <TokenBadge kind={t('common.admin.token_access')} status={p.access as TokenStatus} t={t} />
                        <TokenBadge kind={t('common.admin.token_xsts')} status={p.xsts as TokenStatus} t={t} />
                      </>
                    )}
                  </div>
                </div>
                {p.xsts_expires_at && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t('common.admin.xsts_expires_on')}{' '}
                    {new Date(p.xsts_expires_at).toLocaleString(dateLocale)}
                  </p>
                )}
                {p.last_auth_error_class ? (
                  <p
                    className="mt-1 text-xs font-medium"
                    style={{
                      color: tokenCssVar(
                        p.last_auth_error_class === 'transient' ? 'warning' : 'destructive',
                      ),
                    }}
                    title={p.last_auth_error}
                  >
                    {t(TOKEN_ERROR_KEY[p.last_auth_error_class] ?? 'common.admin.token_error_transient')}
                    {p.last_auth_error_at
                      ? ` — ${t('common.admin.token_error_at')} : ${new Date(p.last_auth_error_at).toLocaleString(dateLocale)}`
                      : null}
                  </p>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
