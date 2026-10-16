/**
 * OAuthRefusalBanner — motif du refus d'une reconnexion Xbox, en tête du shell.
 *
 * Un utilisateur déjà connecté dont la reconnexion Xbox (bannière de reconnexion, carte
 * d'apparence) est refusée revient sur /login?oauth_error=<code> ; la racine le renvoie à
 * l'accueil et garde le code (appShellStore.oauthRefusal). Le code est traduit en message
 * localisé (loginError.oauthErrorKey), jamais affiché brut, jusqu'à fermeture.
 */
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { oauthErrorKey } from '@/features/auth/loginError'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'

export function OAuthRefusalBanner() {
  const code = useAppShellStore((s) => s.oauthRefusal)
  const setOauthRefusal = useAppShellStore((s) => s.setOauthRefusal)
  const locale = useAppShellStore((s) => s.locale)
  const key = oauthErrorKey(code)
  if (!key) return null
  return (
    <div
      role="alert"
      className="flex items-center justify-between gap-3 border-b border-destructive/40 bg-destructive/10 px-4 py-2 text-sm text-destructive"
    >
      <span>{formatMessage(commonManifest, key, locale)}</span>
      <Button size="sm" variant="outline" onClick={() => setOauthRefusal(null)} className="shrink-0">
        {formatMessage(commonManifest, 'common.shell.dismiss', locale)}
      </Button>
    </div>
  )
}
