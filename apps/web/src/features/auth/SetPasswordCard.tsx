/**
 * SetPasswordCard — définition opt-in d'un mot de passe (PR-C).
 *
 * Permet à un compte SSO Xbox de définir un mot de passe pour se reconnecter
 * rapidement (sans round-trip Microsoft) à l'expiration de la session. Affiché
 * en fin d'onboarding et dans les réglages. Skippable.
 */
import { useState, type FormEvent } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useSetPassword } from '@/features/auth/queries'
import { useAppShellStore } from '@/stores/appShellStore'
import { apiErrorMessage } from '@/lib/api/client'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const LABELS = {
  title: 'Définir un mot de passe',
  desc: 'Optionnel — permet de te reconnecter rapidement sans repasser par Xbox.',
  changeTitle: 'Changer ton mot de passe',
  submit: 'Enregistrer',
  saved: 'Mot de passe enregistré ✓',
  mismatch: 'Les mots de passe ne correspondent pas.',
  tooShort: 'Le mot de passe doit faire au moins 8 caractères.',
}

interface SetPasswordCardProps {
  /** Callback optionnel après enregistrement réussi (ex. continuer l'onboarding). */
  onSaved?: () => void
}

export function SetPasswordCard({ onSaved }: SetPasswordCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const hasPassword = useAppShellStore((s) => s.hasPassword)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const setPwd = useSetPassword()

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (password.length < 8) {
      setError(LABELS.tooShort)
      return
    }
    if (password !== confirm) {
      setError(LABELS.mismatch)
      return
    }
    setPwd.mutate(password, {
      onSuccess: () => {
        setSaved(true)
        setPassword('')
        setConfirm('')
        onSaved?.()
      },
      onError: (err) => setError(apiErrorMessage(err) ?? 'Erreur'),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{hasPassword ? LABELS.changeTitle : LABELS.title}</CardTitle>
        <CardDescription>{LABELS.desc}</CardDescription>
      </CardHeader>
      <CardContent>
        {saved ? (
          <p className="text-sm text-success">{LABELS.saved}</p>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label htmlFor="set-pwd" className="block text-sm font-medium text-foreground mb-1">
                {t('common.auth.password_label')}
              </label>
              <input
                id="set-pwd"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                maxLength={72}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label htmlFor="set-pwd-confirm" className="block text-sm font-medium text-foreground mb-1">
                {t('common.auth.confirm_password')}
              </label>
              <input
                id="set-pwd-confirm"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                maxLength={72}
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" size="sm" disabled={setPwd.isPending}>
              {LABELS.submit}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
