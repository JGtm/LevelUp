/**
 * LoginPage — authentification par username/password.
 *
 * En mode auth_mode=password : formulaire classique.
 * En mode auth_mode=none : redirige vers / (pas de login requis).
 */
import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { useLogin } from '@/features/auth/queries'
import { queryKeys } from '@/lib/query/keys'
import type { ApiError } from '@/lib/api/client'

export function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const authMode = useAppShellStore((s) => s.authMode)
  const registrationMode = useAppShellStore((s) => s.registrationMode)
  const firstLaunch = useAppShellStore((s) => s.firstLaunch)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const login = useLogin()

  // En mode none, pas de login nécessaire
  if (authMode === 'none') {
    navigate({ to: '/' })
    return null
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    login.mutate(
      { username, password },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
          navigate({ to: '/' })
        },
        onError: (err) => {
          const apiErr = err as ApiError
          if (apiErr.code === 'invalid_credentials') {
            setError('Identifiants incorrects.')
          } else {
            setError(apiErr.message ?? 'Erreur de connexion.')
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
          <img src="/logo.png" alt="LevelUp" className="h-14 w-14 rounded-full shadow-lg" />
          <span className="text-2xl font-bold tracking-tight text-foreground">LevelUp</span>
        </div>

        <Card>
          <CardContent className="pt-6">
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="username" className="block text-sm font-medium text-foreground mb-1">
                  Identifiant
                </label>
                <input
                  id="username"
                  type="text"
                  autoComplete="username"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="Votre identifiant"
                />
              </div>
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-foreground mb-1">
                  Mot de passe
                </label>
                <input
                  id="password"
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
                {login.isPending ? 'Connexion…' : 'Se connecter'}
              </Button>
            </form>

            {canRegister && (
              <p className="mt-4 text-center text-sm text-muted-foreground">
                Pas encore de compte ?{' '}
                <Link to="/register" className="text-primary underline underline-offset-2">
                  Créer un compte
                </Link>
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
