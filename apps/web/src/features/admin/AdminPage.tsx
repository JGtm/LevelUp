/**
 * AdminPage — gestion des utilisateurs et des invitations.
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
import { useAdminInvariants } from './queries'
import type { AdminInvariantViolation } from '@/lib/api/types'
import {
  SHARED_SCOPE_KEY,
  buildInvariantsSnapshot,
  invariantDelta,
  readInvariantsSnapshot,
  writeInvariantsSnapshot,
  type InvariantsSnapshot,
} from './invariantsTrend'

export function AdminPage() {
  const navigate = useNavigate()
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const currentUsername = useAppShellStore((s) => s.currentUsername)

  if (!isAdmin) {
    navigate({ to: '/' })
    return null
  }

  return (
    <div className="p-6 space-y-8">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-foreground">Administration</h1>
        <Button variant="outline" onClick={() => navigate({ to: '/' })}>
          Retour
        </Button>
      </div>

      <UsersSection currentUsername={currentUsername} />
      <InvitesSection />
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
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [resetTarget, setResetTarget] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState('')

  function handleDelete(username: string) {
    if (!confirm(`Supprimer l'utilisateur "${username}" ?`)) return
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
        <h2 className="text-lg font-semibold text-foreground mb-4">Utilisateurs</h2>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Chargement…</p>
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
                        {u.role === 'admin' ? '→ user' : '→ admin'}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setResetTarget(resetTarget === u.username ? null : u.username)}
                      >
                        MDP
                      </Button>
                      <Button size="sm" variant="destructive" onClick={() => handleDelete(u.username)}>
                        Supprimer
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
                  OK
                </Button>
                <Button size="sm" variant="outline" onClick={() => { setResetTarget(null); setNewPassword('') }}>
                  Annuler
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
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

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
            {generateInvite.isPending ? 'Génération…' : 'Générer un code'}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">Chargement…</p>
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
                    <span>par {inv.created_by}</span>
                    <span>expire {new Date(inv.expires_at).toLocaleDateString('fr-FR')}</span>
                    {inv.used_by && <span className="text-primary">utilisé par {inv.used_by}</span>}
                  </div>
                </div>
                {inv.valid && !inv.used_by && (
                  <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.code)}>
                    Révoquer
                  </Button>
                )}
                {!inv.valid && !inv.used_by && (
                  <span className="text-xs text-muted-foreground">expiré</span>
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
