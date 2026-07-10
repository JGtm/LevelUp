/**
 * RegisterPage — inscription (premier user = admin automatique).
 */
import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRegister } from '@/features/auth/queries'
import { queryKeys } from '@/lib/query/keys'
import type { ApiError } from '@/lib/api/client'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const authMode = useAppShellStore((s) => s.authMode)
  const registrationMode = useAppShellStore((s) => s.registrationMode)
  const instanceLocked = useAppShellStore((s) => s.instanceLocked)
  const firstLaunch = useAppShellStore((s) => s.firstLaunch)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [inviteCode, setInviteCode] = useState('')
  const [error, setError] = useState<string | null>(null)

  const register = useRegister()

  // D3 cohabitation : en mode xbox, register password est réservé au bootstrap
  // admin initial (firstLaunch=true). Hors bootstrap, on redirige vers /login
  // pour que l'user utilise le flow SSO Xbox.
  if (authMode === 'xbox' && !firstLaunch) {
    navigate({ to: '/login' })
    return null
  }

  // Instance fermée (lockdown) : aucune nouvelle inscription hors bootstrap du
  // premier admin (firstLaunch). On présente un écran « fermé » plutôt que le form.
  const lockedOut = instanceLocked && !firstLaunch
  const instanceClosedLabel = t('common.auth.register_instance_closed')

  const needsInvite = !firstLaunch && registrationMode === 'invite'

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    if (password !== confirmPassword) {
      setError(t('common.auth.register_password_mismatch'))
      return
    }

    register.mutate(
      {
        username,
        password,
        invite_code: needsInvite ? inviteCode : undefined,
      },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
          navigate({ to: '/' })
        },
        onError: (err) => {
          const apiErr = err as unknown as ApiError
          const messages: Record<string, string> = {
            user_exists: t('common.auth.register_err_user_exists'),
            invite_required: t('common.auth.register_err_invite_required'),
            invalid_invite: t('common.auth.register_err_invalid_invite'),
            registration_closed: t('common.auth.register_err_closed'),
            instance_locked: t('common.auth.register_instance_closed'),
            validation_error: apiErr.message,
          }
          setError(messages[apiErr.code] ?? apiErr.message ?? t('common.auth.register_err_generic'))
        },
      },
    )
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        {/* Logo */}
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardContent className="pt-6">
            {lockedOut ? (
              <div className="space-y-4 text-center">
                <p className="text-sm font-medium text-foreground">{instanceClosedLabel}</p>
                <p className="text-sm text-muted-foreground">
                  {t('common.auth.already_account')}{' '}
                  <Link to="/login" className="text-primary underline underline-offset-2">
                    {t('common.auth.login_action')}
                  </Link>
                </p>
              </div>
            ) : (
            <>
            {firstLaunch && (
              <div className="mb-4 rounded-md bg-primary/10 px-3 py-2 text-sm text-primary">
                {t('common.auth.first_launch_admin')}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="reg-username" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.username_label')}
                </label>
                <input
                  id="reg-username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  required
                  minLength={3}
                  maxLength={30}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder={t('common.auth.username_placeholder_range')}
                />
              </div>

              <div>
                <label htmlFor="reg-password" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.password_label')}
                </label>
                <input
                  id="reg-password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder={t('common.auth.password_placeholder_min')}
                />
              </div>

              <div>
                <label htmlFor="reg-confirm" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.confirm_password')}
                </label>
                <input
                  id="reg-confirm"
                  name="confirm-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="••••••••"
                />
              </div>

              {needsInvite && (
                <div>
                  <label htmlFor="reg-invite" className="block text-sm font-medium text-foreground mb-1">
                    {t('common.auth.invitation_code')}
                  </label>
                  <input
                    id="reg-invite"
                    type="text"
                    required
                    value={inviteCode}
                    onChange={(e) => setInviteCode(e.target.value.toUpperCase())}
                    className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono tracking-wider ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    placeholder="CODE1234"
                  />
                </div>
              )}

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              <Button type="submit" className="w-full" disabled={register.isPending}>
                {register.isPending ? t('common.auth.register_pending') : t('common.auth.register_submit')}
              </Button>
            </form>

            <p className="mt-4 text-center text-sm text-muted-foreground">
              {t('common.auth.already_account')}{' '}
              <Link to="/login" className="text-primary underline underline-offset-2">
                {t('common.auth.login_action')}
              </Link>
            </p>
            </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
