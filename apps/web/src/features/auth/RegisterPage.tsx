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

export function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const registrationMode = useAppShellStore((s) => s.registrationMode)
  const firstLaunch = useAppShellStore((s) => s.firstLaunch)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [inviteCode, setInviteCode] = useState('')
  const [error, setError] = useState<string | null>(null)

  const register = useRegister()

  const needsInvite = !firstLaunch && registrationMode === 'invite'

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    if (password !== confirmPassword) {
      setError('Les mots de passe ne correspondent pas.')
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
          const apiErr = err as ApiError
          const messages: Record<string, string> = {
            user_exists: 'Ce nom d\'utilisateur est déjà pris.',
            invite_required: 'Un code d\'invitation est requis.',
            invalid_invite: 'Code d\'invitation invalide ou expiré.',
            registration_closed: 'Les inscriptions sont fermées.',
            validation_error: apiErr.message,
          }
          setError(messages[apiErr.code] ?? apiErr.message ?? 'Erreur lors de l\'inscription.')
        },
      },
    )
  }

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
            {firstLaunch && (
              <div className="mb-4 rounded-md bg-primary/10 px-3 py-2 text-sm text-primary">
                Premier lancement — votre compte sera administrateur.
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="reg-username" className="block text-sm font-medium text-foreground mb-1">
                  Identifiant
                </label>
                <input
                  id="reg-username"
                  type="text"
                  autoComplete="username"
                  required
                  minLength={3}
                  maxLength={30}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="3-30 caractères"
                />
              </div>

              <div>
                <label htmlFor="reg-password" className="block text-sm font-medium text-foreground mb-1">
                  Mot de passe
                </label>
                <input
                  id="reg-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="8 caractères minimum"
                />
              </div>

              <div>
                <label htmlFor="reg-confirm" className="block text-sm font-medium text-foreground mb-1">
                  Confirmer le mot de passe
                </label>
                <input
                  id="reg-confirm"
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
                    Code d'invitation
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
                {register.isPending ? 'Inscription…' : 'Créer le compte'}
              </Button>
            </form>

            <p className="mt-4 text-center text-sm text-muted-foreground">
              Déjà un compte ?{' '}
              <Link to="/login" className="text-primary underline underline-offset-2">
                Se connecter
              </Link>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
