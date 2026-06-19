/**
 * JoinPage — page de jonction à un groupe via lien d'invitation.
 *
 * Atteinte via /join?invite=CODE (lien partagé par un membre). L'invité se connecte
 * avec Xbox ; le code voyage dans la session et la LinkStrategy l'ajoute au groupe
 * après le login (cf. XboxSSOLinkStrategy). Pas de compte mot de passe : le login
 * Xbox SSO fait tout. Un champ manuel sert de repli si l'invité n'a que le code.
 */
import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { API_BASE_URL } from '@/lib/api/client'
import { Route } from '@/routes/join'

export function JoinPage() {
  const { invite } = Route.useSearch()
  const locale = useAppShellStore((s) => s.locale)
  const en = locale === 'en'
  const [code, setCode] = useState(invite ?? '')

  const trimmed = code.trim().toUpperCase()
  function handleJoin() {
    if (!trimmed) return
    // Redirect plein écran : le backend pose le code en session puis redirige vers
    // Microsoft. Le callback finalise le login + l'ajout au groupe.
    window.location.assign(`${API_BASE_URL}/auth/xbox/login?invite=${encodeURIComponent(trimmed)}`)
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardContent className="space-y-5 pt-6 text-center">
            <h2 className="text-lg font-semibold">
              {en ? 'Join a group' : 'Rejoindre un groupe'}
            </h2>
            <p className="text-sm text-muted-foreground">
              {en
                ? 'You have been invited to join a group. Sign in with Xbox to join — your account is created automatically.'
                : 'Vous avez été invité à rejoindre un groupe. Connectez-vous avec Xbox pour rejoindre — votre compte est créé automatiquement.'}
            </p>

            {!invite && (
              <div className="text-left">
                <label htmlFor="invite-code" className="mb-1 block text-sm font-medium text-foreground">
                  {en ? 'Invitation code' : 'Code d’invitation'}
                </label>
                <input
                  id="invite-code"
                  value={code}
                  onChange={(e) => setCode(e.target.value.toUpperCase())}
                  placeholder="CODE1234"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-center font-mono text-sm tracking-widest focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
            )}

            <Button onClick={handleJoin} className="w-full" disabled={!trimmed}>
              {en ? 'Continue with Xbox' : 'Continuer avec Xbox'}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
