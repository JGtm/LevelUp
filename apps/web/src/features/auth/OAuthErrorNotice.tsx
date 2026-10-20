/**
 * OAuthErrorNotice — message d'un refus du callback de connexion Xbox.
 *
 * Le serveur renvoie le navigateur sur /login?oauth_error=<code> quand il refuse une
 * connexion Microsoft (instance fermée, invitation périmée, échange échoué…). Le code est
 * traduit en message localisé (loginError.oauthErrorKey), jamais affiché brut. Rien sans
 * refus.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { oauthErrorKey } from '@/features/auth/loginError'

export function OAuthErrorNotice({ code }: { code: unknown }) {
  const locale = useAppShellStore((s) => s.locale)
  const key = oauthErrorKey(code)
  if (!key) return null
  return (
    <p role="alert" className="mb-4 text-center text-sm text-destructive">
      {formatMessage(commonManifest, key, locale)}
    </p>
  )
}
