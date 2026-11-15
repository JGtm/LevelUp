/**
 * Geste « reconnecter le compte Xbox » : redirection SSO en un clic quand le serveur la propose,
 * sinon page de connexion (code d'appareil). Partagé par la bannière de reconnexion et par
 * l'accueil d'un joueur dont la synchronisation attend sa connexion.
 */
import { useNavigate } from '@tanstack/react-router'
import { API_BASE_URL } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

export function useReconnectXbox(): () => void {
  const navigate = useNavigate()
  const oauthCodeFlowEnabled = useAppShellStore((s) => s.oauthCodeFlowEnabled)
  return () => {
    if (oauthCodeFlowEnabled) {
      window.location.assign(`${API_BASE_URL}/auth/xbox/login`)
      return
    }
    void navigate({ to: '/login' })
  }
}
