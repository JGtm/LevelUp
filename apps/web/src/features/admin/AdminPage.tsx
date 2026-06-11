/**
 * AdminPage — gestion des utilisateurs, invitations, monitoring (contention DB,
 * santé des tokens) et intégrité des données.
 */
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  useAdminUsers,
  useDeleteUser,
  useChangeRole,
  useResetPassword,
  useAdminInvites,
  useGenerateInvite,
  useRevokeInvite,
  adminKeys,
} from '@/features/auth/queries'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAdminInvariants, useAdminDBContention, useAdminTokenHealth } from './queries'
import { credentialSourceParts, hasLegacyCredentialSource, TOKEN_ERROR_KEY } from './tokenHealthDisplay'
import type { AdminInvariantViolation, TokenStatus } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import {
  SHARED_SCOPE_KEY,
  buildInvariantsSnapshot,
  invariantDelta,
  readInvariantsSnapshot,
  writeInvariantsSnapshot,
  type InvariantsSnapshot,
} from './invariantsTrend'

type T = (key: CommonManifestKey) => string

function useT(): T {
  const locale = useAppShellStore((s) => s.locale)
  return (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
}

function useDateLocale(): string {
  const locale = useAppShellStore((s) => s.locale)
  return locale === 'fr' ? 'fr-FR' : 'en-US'
}

export function AdminPage() {
  const navigate = useNavigate()
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const t = useT()

  if (!isAdmin) {
    navigate({ to: '/' })
    return null
  }

  return (
    <div className="p-6 space-y-8">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-foreground">{t('common.admin.page_title')}</h1>
        <Button variant="outline" onClick={() => navigate({ to: '/' })}>
          {t('common.admin.back')}
        </Button>
      </div>

      <UsersSection currentUsername={currentUsername} />
      <InvitesSection />
      <DBContentionSection />
      <TokenHealthSection />
      <InvariantsSection />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section Utilisateurs
// ---------------------------------------------------------------------------

function UsersSection({ currentUsername }: { currentUsername: string | null | undefined }) {
  const queryClient = useQueryClient()
  const { data: users, isLoading } = useAdminUsers()
  const deleteUser = useDeleteUser()
  const changeRole = useChangeRole()
  const resetPassword = useResetPassword()
  const t = useT()

  const [resetTarget, setResetTarget] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState('')

  function handleDelete(username: string) {
    if (!confirm(`${t('common.admin.delete_user_confirm')} "${username}" ?`)) return
    deleteUser.mutate(username, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: adminKeys.users }),
    })
  }

  function handleToggleRole(username: string, currentRole: string) {
    const newRole = currentRole === 'admin' ? 'user' : 'admin'
    changeRole.mutate(
      { username, role: newRole as 'admin' | 'user' },
      { onSuccess: () => queryClient.invalidateQueries({ queryKey: adminKeys.users }) },
    )
  }

  function handleResetPassword(username: string) {
    if (!newPassword || newPassword.length < 8) return
    resetPassword.mutate(
      { username, newPassword },
      {
        onSuccess: () => {
          setResetTarget(null)
          setNewPassword('')
        },
      },
    )
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">{t('common.admin.users_section')}</h2>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : !users?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.no_users')}</p>
        ) : (
          <div className="space-y-3">
            {users.map((u) => (
              <div key={u.username} className="flex items-center justify-between rounded-md border px-4 py-3">
                <div>
                  <span className="font-medium text-foreground">{u.username}</span>
                  <span className="ml-2 rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                    {u.role}
                  </span>
                  {u.gamertag && (
                    <span className="ml-2 text-xs text-muted-foreground">({u.gamertag})</span>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {u.username !== currentUsername && (
                    <>
                      <Button size="sm" variant="outline" onClick={() => handleToggleRole(u.username, u.role)}>
                        {u.role === 'admin'
                          ? t('common.admin.demote_to_user')
                          : t('common.admin.promote_to_admin')}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setResetTarget(resetTarget === u.username ? null : u.username)}
                      >
                        {t('common.admin.password_btn')}
                      </Button>
                      <Button size="sm" variant="destructive" onClick={() => handleDelete(u.username)}>
                        {t('common.admin.delete')}
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ))}

            {/* Inline reset password form */}
            {resetTarget && (
              <div className="flex items-center gap-2 rounded-md border border-dashed px-4 py-3">
                <span className="text-sm text-muted-foreground">
                  {t('common.admin.new_password_for')} <strong>{resetTarget}</strong> :
                </span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-40 rounded-md border border-input bg-background px-2 py-1 text-sm"
                  placeholder={t('common.auth.password_placeholder_short')}
                />
                <Button size="sm" onClick={() => handleResetPassword(resetTarget)}>
                  {t('common.admin.ok')}
                </Button>
                <Button size="sm" variant="outline" onClick={() => { setResetTarget(null); setNewPassword('') }}>
                  {t('common.admin.cancel')}
                </Button>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Section Invitations
// ---------------------------------------------------------------------------

function InvitesSection() {
  const queryClient = useQueryClient()
  const { data: invites, isLoading } = useAdminInvites()
  const generateInvite = useGenerateInvite()
  const revokeInvite = useRevokeInvite()
  const t = useT()
  const dateLocale = useDateLocale()

  function handleGenerate() {
    generateInvite.mutate(7, {
      onSuccess: (data) => {
        queryClient.invalidateQueries({ queryKey: adminKeys.invites })
        toast.success(`${t('common.admin.invite_generated')} ${data.code}`)
      },
      onError: (err) => {
        // Sans ce feedback, un échec (403 non-admin, réseau…) donnait
        // l'impression que le bouton « ne fait rien ».
        toast.error(err instanceof Error ? err.message : t('common.admin.invite_generate_failed'))
      },
    })
  }

  function handleRevoke(code: string) {
    revokeInvite.mutate(code, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: adminKeys.invites }),
    })
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">{t('common.admin.invitation_codes_section')}</h2>
          <Button size="sm" onClick={handleGenerate} disabled={generateInvite.isPending}>
            {generateInvite.isPending ? t('common.admin.generating') : t('common.admin.generate_code')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : !invites?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.no_invitation_codes')}</p>
        ) : (
          <div className="space-y-2">
            {invites.map((inv) => (
              <div
                key={inv.code}
                className="flex items-center justify-between rounded-md border px-4 py-3"
              >
                <div className="space-y-0.5">
                  <span className="font-mono font-bold tracking-wider text-foreground select-all">
                    {inv.code}
                  </span>
                  <div className="flex gap-3 text-xs text-muted-foreground">
                    <span>{t('common.admin.invite_by')} {inv.created_by}</span>
                    <span>{t('common.admin.invite_expires')} {new Date(inv.expires_at).toLocaleDateString(dateLocale)}</span>
                    {inv.used_by && (
                      <span className="text-primary">{t('common.admin.invite_used_by')} {inv.used_by}</span>
                    )}
                  </div>
                </div>
                {inv.valid && !inv.used_by && (
                  <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.code)}>
                    {t('common.admin.revoke')}
                  </Button>
                )}
                {!inv.valid && !inv.used_by && (
                  <span className="text-xs text-muted-foreground">{t('common.admin.expired')}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Section Intégrité des données (invariants sync — plan SYNC_INVARIANTS_GATE)
// ---------------------------------------------------------------------------

function InvariantsSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminInvariants()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Baseline ROULANTE : au 1er run de la session, comparaison au snapshot
  // localStorage (inter-sessions) ; ensuite chaque nouveau run (generated_at
  // différent) compare au run PRÉCÉDENT — pas au snapshot figé au mount
  // (sinon un refetch intra-session masquerait une régression revenue au
  // niveau pré-mount).
  const [previous, setPrevious] = useState<InvariantsSnapshot>(() => readInvariantsSnapshot())
  const lastRunRef = useRef<{ generatedAt: string; snapshot: InvariantsSnapshot } | null>(null)
  useEffect(() => {
    if (!data) return
    const snap = buildInvariantsSnapshot(data)
    const last = lastRunRef.current
    if (last && last.generatedAt !== data.generated_at) {
      setPrevious(last.snapshot)
    }
    lastRunRef.current = { generatedAt: data.generated_at, snapshot: snap }
    writeInvariantsSnapshot(snap)
  }, [data])

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              {t('common.admin.invariants_section')}
            </h2>
            {data?.generated_at && (
              <p className="text-xs text-muted-foreground">
                {t('common.admin.invariants_generated_at')}{' '}
                {new Date(data.generated_at).toLocaleString(locale === 'fr' ? 'fr-FR' : 'en-US')}
              </p>
            )}
          </div>
          <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? t('common.admin.invariants_loading') : t('common.admin.invariants_refresh')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.invariants_loading')}</p>
        ) : isError ? (
          <p className="text-sm text-destructive">{t('common.admin.invariants_load_failed')}</p>
        ) : !data?.reports?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.invariants_empty')}</p>
        ) : (
          <div className="space-y-3">
            <InvariantsCard
              title={t('common.admin.invariants_shared_scope')}
              scope={SHARED_SCOPE_KEY}
              checkError={data.shared_check_error}
              failCount={data.shared_fail_count}
              warnCount={data.shared_warn_count}
              violations={data.shared_violations ?? []}
              previous={previous}
              t={t}
            />
            {data.reports.map((r) => (
              <InvariantsCard
                key={r.player_slug || r.gamertag}
                title={r.gamertag}
                scope={r.player_slug}
                checkError={r.check_error}
                failCount={r.fail_count}
                warnCount={r.warn_count}
                violations={r.violations}
                previous={previous}
                t={t}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function InvariantsCard({
  title,
  scope,
  checkError,
  failCount,
  warnCount,
  violations,
  previous,
  t,
}: {
  title: string
  scope: string
  checkError?: string
  failCount: number
  warnCount: number
  violations: AdminInvariantViolation[]
  previous: InvariantsSnapshot
  t: (key: CommonManifestKey) => string
}) {
  const healthy = !checkError && failCount === 0 && warnCount === 0
  return (
    <div className="rounded-md border px-4 py-3">
      <div className="flex items-center justify-between">
        <span className="font-medium text-foreground">{title}</span>
        <div className="flex items-center gap-2 text-xs">
          {checkError ? (
            <span className="rounded bg-muted px-2 py-0.5 text-destructive">
              {t('common.admin.invariants_check_error')}
            </span>
          ) : (
            <>
              {failCount > 0 && (
                <span className="rounded bg-muted px-2 py-0.5 font-semibold text-destructive">
                  {failCount} FAIL
                </span>
              )}
              {warnCount > 0 && (
                <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">
                  {warnCount} WARN
                </span>
              )}
              {healthy && (
                <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">OK</span>
              )}
            </>
          )}
        </div>
      </div>

      {checkError && <p className="mt-2 text-xs text-muted-foreground">{checkError}</p>}

      {!checkError && violations.length === 0 && (
        <p className="mt-2 text-xs text-muted-foreground">{t('common.admin.invariants_all_ok')}</p>
      )}

      {violations.length > 0 && (
        <ul className="mt-2 space-y-1.5">
          {violations.map((v) => {
            const delta = invariantDelta(previous, scope, v.key, v.count)
            return (
              <li key={v.key} className="text-xs">
                <span
                  className={
                    v.severity === 'fail'
                      ? 'font-mono font-semibold text-destructive'
                      : 'font-mono text-muted-foreground'
                  }
                >
                  [{v.severity}] {v.key}
                </span>{' '}
                <span className="text-foreground">×{v.count}</span>
                {delta !== undefined && (
                  <span
                    className={
                      delta > 0
                        ? 'ml-1 font-semibold text-destructive'
                        : 'ml-1 text-muted-foreground'
                    }
                  >
                    ({delta > 0 ? '+' : ''}
                    {delta})
                  </span>
                )}
                <span className="ml-1 text-muted-foreground">— {v.description}</span>
                {(v.sample ?? []).length > 0 && (
                  <div className="mt-0.5 truncate font-mono text-muted-foreground/70">
                    {(v.sample ?? []).join(', ')}
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section Contention DB (B-swap shared — diagnostic du stall pendant le sync)
// ---------------------------------------------------------------------------

function DBContentionSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminDBContention()
  const t = useT()

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{t('common.admin.contention_section')}</h2>
            <p className="max-w-xl text-xs text-muted-foreground">{t('common.admin.contention_desc')}</p>
          </div>
          <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? t('common.admin.loading') : t('common.admin.refresh')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : isError || !data ? (
          <p className="text-sm text-destructive">{t('common.admin.contention_unavailable')}</p>
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Metric label={t('common.admin.contention_swaps')} value={String(data.swaps)} />
            <Metric label={t('common.admin.contention_acquire')} value={`${data.avg_acquire_ms} ms`} />
            <Metric label={t('common.admin.contention_release')} value={`${data.avg_release_ms} ms`} />
            <Metric label={t('common.admin.contention_drain')} value={`${data.drain_ms_total} ms`} />
            <Metric label={t('common.admin.contention_blocked_avg')} value={`${data.avg_blocked_ms} ms`} />
            <Metric
              label={t('common.admin.contention_blocked_max')}
              value={`${data.max_blocked_ms} ms`}
              alert={data.max_blocked_ms >= 1000}
            />
            <Metric
              label={t('common.admin.contention_503')}
              value={String(data.reads_rejected)}
              alert={data.reads_rejected > 0}
            />
            <Metric
              label={t('common.admin.contention_failures')}
              value={String(data.swap_failures)}
              alert={data.swap_failures > 0}
            />
            <Metric label={t('common.admin.contention_readers')} value={String(data.readers_in_use)} />
            <Metric label={t('common.admin.contention_state')} value={data.state} />
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function Metric({ label, value, alert }: { label: string; value: string; alert?: boolean }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className="text-lg font-semibold text-foreground"
        style={alert ? { color: tokenCssVar('destructive') } : undefined}
      >
        {value}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section Santé des tokens (MSAL / XSTS / Refresh par joueur — ADR 0023)
// ---------------------------------------------------------------------------

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

/** Source de credentials du dernier scan — toute source hors store canonique = dette ADR-0023 (warning). */
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

function TokenHealthSection() {
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
                        <TokenBadge kind={t('common.admin.token_refresh')} status={p.refresh} t={t} />
                        <TokenBadge kind={t('common.admin.token_msal')} status={p.msal} t={t} />
                        <TokenBadge kind={t('common.admin.token_xsts')} status={p.xsts} t={t} />
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
