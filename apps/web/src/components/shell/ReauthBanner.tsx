/**
 * ReauthBanner — bannière persistante « reconnecte ton compte Xbox ».
 *
 * Affichée en tête de l'AppShell quand le refresh_token Microsoft du joueur
 * courant est mort (bootstrap.reauth_required, cf. PR-B slice 1). Le refresh des
 * données s'est arrêté ; l'utilisateur doit re-passer le SSO Xbox pour re-semer
 * des tokens valides. Le flag reauth_required est remis à false dès qu'un
 * refresh par-joueur réussit (auto-guérison, cf. registry_auth.go) ou après une
 * ré-auth interactive → la bannière disparaît au prochain bootstrap, re-fetché
 * au retour sur l'onglet (refetchOnWindowFocus, cf. __root.tsx).
 */
import { useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { API_BASE_URL } from '@/lib/api/client'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'

export function ReauthBanner() {
  const navigate = useNavigate()
  const reauthRequired = useAppShellStore((s) => s.reauthRequired)
  const oauthCodeFlowEnabled = useAppShellStore((s) => s.oauthCodeFlowEnabled)
  const locale = useAppShellStore((s) => s.locale)

  if (!reauthRequired) return null

  function handleReconnect() {
    if (oauthCodeFlowEnabled) {
      // Redirect SSO 1-clic (re-sème les tokens puis revient).
      window.location.assign(`${API_BASE_URL}/auth/xbox/login`)
      return
    }
    // Device code (SISU / pas de redirect Azure) : la page de login relance le flow.
    navigate({ to: '/login' })
  }

  return (
    <div
      role="alert"
      className="flex items-center justify-between gap-3 border-b border-warning/40 bg-warning/10 px-4 py-2 text-sm text-warning"
    >
      <span>{formatMessage(commonManifest, 'common.reauth.message', locale)}</span>
      <Button size="sm" variant="outline" onClick={handleReconnect} className="shrink-0">
        {formatMessage(commonManifest, 'common.reauth.action', locale)}
      </Button>
    </div>
  )
}
