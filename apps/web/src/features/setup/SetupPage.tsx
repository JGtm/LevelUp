/**
 * SetupPage — wizard de configuration initiale piloté par ``setupState``.
 *
 * Machine d'état (source : GET /bootstrap, champ setup_state) :
 *   no_halo_link            → StepDeviceCode
 *   halo_linked_no_profile  → StepPlayer (carte de confirmation si linkedHaloIdentity)
 *   profile_ready_no_sync   → StepInitialSync
 *   ready                   → redirect vers /
 *
 * P8.4 (revue 2026-04-29) : les 3 Step* ont été extraits dans des fichiers
 * dédiés ; ce fichier ne porte plus que l'orchestrateur (~50L vs ~484L).
 */
import { useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { StepDeviceCode } from './StepDeviceCode'
import { StepPlayer } from './StepPlayer'
import { StepInitialSync } from './StepInitialSync'

export function SetupPage() {
  const navigate = useNavigate()
  const setupState = useAppShellStore((s) => s.setupState)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)

  const setupRequired = useAppShellStore((s) => s.setupRequired)

  // Rediriger vers l'accueil si le setup n'est pas requis ou est terminé
  useEffect(() => {
    if (isBootstrapped && (setupState === 'ready' || !setupRequired)) {
      navigate({ to: '/' })
    }
  }, [isBootstrapped, setupState, setupRequired, navigate])

  if (!isBootstrapped) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner size="lg" label="Vérification de la configuration…" />
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted">
      <Card className="w-full max-w-lg mx-4">
        <CardHeader>
          <div className="flex items-center gap-2">
            <img src="/logo.png" alt="LevelUp" className="h-8 w-8 rounded-full" />
            <CardTitle>Configuration de LevelUp</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          {setupState === 'no_halo_link' && <StepDeviceCode />}
          {setupState === 'halo_linked_no_profile' && <StepPlayer />}
          {setupState === 'profile_ready_no_sync' && currentPlayer && (
            <StepInitialSync playerSlug={currentPlayer.player_slug} />
          )}
          {setupState === 'profile_ready_no_sync' && !currentPlayer && (
            /* Joueur pas encore connu localement mais provisioning en cours */
            <Spinner label="Chargement du profil joueur…" />
          )}
          {setupState === 'ready' && (
            <div className="space-y-4">
              <p className="text-success font-semibold">✓ Configuration terminée !</p>
              <Button onClick={() => navigate({ to: '/' })}>Accéder à l'application</Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
