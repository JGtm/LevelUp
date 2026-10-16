/**
 * SetPasswordCard — définition opt-in d'un mot de passe (PR-C), ou changement.
 *
 * Permet à un compte SSO Xbox de définir un mot de passe pour se reconnecter
 * rapidement (sans round-trip Microsoft) à l'expiration de la session. Affiché
 * en fin d'onboarding et dans les réglages. Skippable. Un compte qui a déjà un mot
 * de passe (`hasPassword`, bootstrap) doit saisir l'actuel pour en changer.
 */
import { useState, type FormEvent } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useSetPassword } from '@/features/auth/queries'
import { setPasswordErrorKey } from '@/features/auth/loginError'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const MIN_PASSWORD_LENGTH = 8
const MAX_PASSWORD_BYTES = 72

const INPUT_CLASS =
  'w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring'

interface SetPasswordCardProps {
  /** Callback optionnel après enregistrement réussi (ex. continuer l'onboarding). */
  onSaved?: () => void
}

export function SetPasswordCard({ onSaved }: SetPasswordCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const hasPassword = useAppShellStore((s) => s.hasPassword)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const [current, setCurrent] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const setPwd = useSetPassword()

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(t('common.auth.password_too_short'))
      return
    }
    if (password !== confirm) {
      setError(t('common.auth.register_password_mismatch'))
      return
    }
    setPwd.mutate(hasPassword ? { password, current_password: current } : { password }, {
      onSuccess: () => {
        setSaved(true)
        setCurrent('')
        setPassword('')
        setConfirm('')
        onSaved?.()
      },
      onError: (err) => setError(t(setPasswordErrorKey(err))),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">
          {hasPassword ? t('common.auth.change_password_title') : t('common.auth.set_password_title')}
        </CardTitle>
        <CardDescription>{t('common.auth.set_password_desc')}</CardDescription>
      </CardHeader>
      <CardContent>
        {saved ? (
          <p className="text-sm text-success">{t('common.auth.set_password_saved')}</p>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-3">
            {hasPassword && (
              <div>
                <label htmlFor="set-pwd-current" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.current_password')}
                </label>
                <input
                  id="set-pwd-current"
                  type="password"
                  autoComplete="current-password"
                  required
                  maxLength={MAX_PASSWORD_BYTES}
                  value={current}
                  onChange={(e) => setCurrent(e.target.value)}
                  className={INPUT_CLASS}
                />
              </div>
            )}
            <div>
              <label htmlFor="set-pwd" className="block text-sm font-medium text-foreground mb-1">
                {hasPassword ? t('common.auth.new_password') : t('common.auth.password_label')}
              </label>
              <input
                id="set-pwd"
                type="password"
                autoComplete="new-password"
                required
                minLength={MIN_PASSWORD_LENGTH}
                maxLength={MAX_PASSWORD_BYTES}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={INPUT_CLASS}
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
                minLength={MIN_PASSWORD_LENGTH}
                maxLength={MAX_PASSWORD_BYTES}
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                className={INPUT_CLASS}
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" size="sm" disabled={setPwd.isPending}>
              {t('common.auth.set_password_submit')}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
