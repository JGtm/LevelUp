/**
 * ReauthBanner — bannière persistante « reconnecte ton compte Xbox ».
 *
 * Affichée en tête de l'AppShell quand le refresh_token Microsoft du joueur
 * courant est mort (bootstrap.reauth_required, cf. PR-B slice 1). Le refresh des
 * données s'est arrêté ; l'utilisateur doit re-passer le SSO Xbox pour re-semer
 * des tokens valides. Une ré-auth réussie remet reauth_required à false → la
 * bannière disparaît au prochain bootstrap.
 */
import { useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { API_BASE_URL } from '@/lib/api/client'

// Textes FR rendus en expression (pas en littéral JSX) pour rester i18n-lint-clean
// tant qu'aucune clé manifest n'existe (cohérent avec les autres écrans auth).
const REAUTH_MESSAGE = 'Ta connexion Xbox a expiré — la synchronisation de tes données est en pause.'
const REAUTH_ACTION = 'Reconnecter'

export function ReauthBanner() {
  const navigate = useNavigate()
  const reauthRequired = useAppShellStore((s) => s.reauthRequired)
  const oauthCodeFlowEnabled = useAppShellStore((s) => s.oauthCodeFlowEnabled)

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
      <span>{REAUTH_MESSAGE}</span>
      <Button size="sm" variant="outline" onClick={handleReconnect} className="shrink-0">
        {REAUTH_ACTION}
      </Button>
    </div>
  )
}
