/**
 * LoginPage — authentification par username/password.
 *
 * En mode auth_mode=password : formulaire classique.
 * En mode auth_mode=xbox    : délègue à XboxLoginPage (Device Code Flow + toggle admin).
 * En mode auth_mode=none    : redirige vers / (pas de login requis).
 */
import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { AppFooter } from '@/components/shell/AppFooter'
import { useAppShellStore } from '@/stores/appShellStore'
import { useLogin } from '@/features/auth/queries'
import { storePasswordCredential } from '@/features/auth/credentials'
import { XboxLoginPage } from '@/features/auth/XboxLoginPage'
import { queryKeys } from '@/lib/query/keys'
import type { ApiError } from '@/lib/api/client'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const authMode = useAppShellStore((s) => s.authMode)
  const registrationMode = useAppShellStore((s) => s.registrationMode)
  const firstLaunch = useAppShellStore((s) => s.firstLaunch)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const login = useLogin()

  // Tant que le store n'est pas hydraté depuis /bootstrap, authMode vaut sa
  // valeur par défaut 'none'. Rediriger ici (vers '/') AVANT hydratation faisait
  // rebondir /login -> / pendant la fenêtre pré-hydratation (race avec le gate de
  // __root), d'où l'écran "Aucun joueur configuré" et la disparition du SSO Xbox.
  // On attend donc l'hydratation avant toute décision de routage.
  if (!isBootstrapped) {
    return null
  }

  // En mode none, pas de login nécessaire
  if (authMode === 'none') {
    navigate({ to: '/' })
    return null
  }

  // En mode xbox, déléguer au composant SSO (toggle admin password intégré).
  if (authMode === 'xbox') {
    return <XboxLoginPage />
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    login.mutate(
      { username, password },
      {
        onSuccess: async () => {
          // Propose au navigateur d'enregistrer le mot de passe (login en fetch).
          storePasswordCredential(username, password)
          await queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
          navigate({ to: '/' })
        },
        onError: (err) => {
          const apiErr = err as unknown as ApiError
          if (apiErr.code === 'invalid_credentials') {
            setError(t('common.auth.invalid_credentials'))
          } else {
            setError(apiErr.message ?? t('common.auth.connection_error'))
          }
        },
      },
    )
  }

  const canRegister = firstLaunch || registrationMode !== 'closed'

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        {/* Logo */}
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardContent className="pt-6">
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="username" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.username_label')}
                </label>
                <input
                  id="username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder={t('common.auth.username_placeholder')}
                />
              </div>
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.password_label')}
                </label>
                <input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="••••••••"
                />
              </div>

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              <Button type="submit" className="w-full" disabled={login.isPending}>
                {login.isPending ? t('common.auth.login_pending') : t('common.auth.login_action')}
              </Button>
            </form>

            {canRegister && (
              <p className="mt-4 text-center text-sm text-muted-foreground">
                {t('common.auth.no_account_prompt')}{' '}
                <Link to="/register" className="text-primary underline underline-offset-2">
                  {t('common.auth.create_account')}
                </Link>
              </p>
            )}
          </CardContent>
        </Card>

        <AppFooter variant="minimal" />
      </div>
    </div>
  )
}
