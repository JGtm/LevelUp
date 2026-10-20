/**
 * SetupPage — wizard de configuration initiale piloté par ``setupState``.
 *
 * Machine d'état (source : GET /bootstrap, champ setup_state) :
 *   no_halo_link            → StepDeviceCode
 *   halo_linked_no_profile  → StepPlayer (carte de confirmation si linkedHaloIdentity)
 *   profile_ready_no_sync   → StepInitialSync
 *   ready                   → redirect vers /
 *
 * ADR 0035 D3 (2026-09-15) : ces états décrivent l'INSTANCE. Un utilisateur
 * connecté qui n'a AUCUN profil à lui sur une instance déjà peuplée passe avant
 * eux — il va à l'étape « profil » et ne sort pas du wizard tant qu'il n'en a pas
 * déclaré un. La décision est pure et vit dans setupRouting.ts.
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
import { needsOwnProfile, resolveSetupStep, shouldLeaveSetup } from './setupRouting'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export function SetupPage() {
  const navigate = useNavigate()
  const setupState = useAppShellStore((s) => s.setupState)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const setupRequired = useAppShellStore((s) => s.setupRequired)
  const authMode = useAppShellStore((s) => s.authMode)
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const linkedHaloIdentity = useAppShellStore((s) => s.linkedHaloIdentity)

  const needsOwn = needsOwnProfile({
    authMode,
    currentUsername,
    isAdmin,
    availablePlayerCount: availablePlayers.length,
  })
  const step = resolveSetupStep(setupState, needsOwn, !!linkedHaloIdentity)

  // Rediriger vers l'accueil si le setup n'est pas requis ou est terminé — sauf
  // pour un utilisateur sans profil à lui, qui n'a rien à voir ailleurs.
  useEffect(() => {
    if (isBootstrapped && shouldLeaveSetup(setupState, setupRequired, needsOwn)) {
      navigate({ to: '/' })
    }
  }, [isBootstrapped, setupState, setupRequired, needsOwn, navigate])

  if (!isBootstrapped) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner size="lg" label={t('common.setup.config_loading')} />
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted">
      <Card className="w-full max-w-lg mx-4">
        <CardHeader>
          <div className="flex items-center gap-2">
            <img src="/logo.png" alt="LevelUp" className="h-8 w-8 rounded-full" />
            <CardTitle>{t('common.setup.config_title')}</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          {step === 'device_code' && <StepDeviceCode />}
          {step === 'player' && <StepPlayer />}
          {step === 'initial_sync' && currentPlayer && (
            <StepInitialSync playerSlug={currentPlayer.player_slug} />
          )}
          {step === 'initial_sync' && !currentPlayer && (
            /* Joueur pas encore connu localement mais provisioning en cours */
            <Spinner label={t('common.setup.profile_loading')} />
          )}
          {step === 'done' && (
            <div className="space-y-4">
              <p className="text-success font-semibold">{t('common.setup.config_done')}</p>
              <Button onClick={() => navigate({ to: '/' })}>{t('common.setup.access_app')}</Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
