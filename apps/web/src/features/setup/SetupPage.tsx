/**
 * SetupPage — wizard de configuration initiale piloté par ``setupState``.
 *
 * Machine d'état (source : GET /bootstrap, champ setup_state) :
 *   no_halo_link            → StepDeviceCode
 *   halo_linked_no_profile  → StepPlayer (carte de confirmation si linkedHaloIdentity)
 *   profile_ready_no_sync   → StepInitialSync
 *   ready                   → redirect vers /
 */
import { useState, useEffect, useRef } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import type { ApiError } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import {
  useStartDeviceFlow,
  useDeviceFlowStatus,
  useCreatePlayer,
  useStartInitialSync,
  useJobStatus,
} from './queries'

function getApiErrorMessage(error: unknown, fallback: string): string {
  if (
    error &&
    typeof error === 'object' &&
    'message' in error &&
    typeof (error as ApiError).message === 'string'
  ) {
    return (error as ApiError).message
  }
  return fallback
}

// ---------------------------------------------------------------------------
// Étape 1 — Device Code Flow (Halo Link)
// ---------------------------------------------------------------------------
function StepDeviceCode() {
  const currentAttemptId = useSetupFlowStore((s) => s.currentAttemptId)
  const setCurrentAttemptId = useSetupFlowStore((s) => s.setCurrentAttemptId)
  const deviceFlowUserCode = useSetupFlowStore((s) => s.deviceFlowUserCode)
  const deviceFlowVerificationUri = useSetupFlowStore((s) => s.deviceFlowVerificationUri)
  const deviceFlowExpiresAt = useSetupFlowStore((s) => s.deviceFlowExpiresAt)
  const setDeviceFlowCodes = useSetupFlowStore((s) => s.setDeviceFlowCodes)
  const queryClient = useQueryClient()
  const startFlow = useStartDeviceFlow()

  // Countdown
  const [secondsLeft, setSecondsLeft] = useState<number | null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    if (!deviceFlowExpiresAt) return
    const update = () => {
      const diff = Math.max(0, Math.floor((deviceFlowExpiresAt - Date.now()) / 1000))
      setSecondsLeft(diff)
      if (diff <= 0 && timerRef.current) clearInterval(timerRef.current)
    }
    update()
    timerRef.current = setInterval(update, 1000)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [deviceFlowExpiresAt])

  const { data: status } = useDeviceFlowStatus(currentAttemptId ?? '', !!currentAttemptId)

  // Démarrer le flow au montage si pas encore en cours
  useEffect(() => {
    if (!currentAttemptId) {
      startFlow.mutate(undefined, {
        onSuccess: (data) => {
          setCurrentAttemptId(data.attempt_id)
          setDeviceFlowCodes(
            data.user_code,
            data.verification_uri,
            data.expires_in ? Date.now() + data.expires_in * 1000 : null,
          )
        },
      })
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Quand le flow réussit : invalider le bootstrap pour que setupState avance
  useEffect(() => {
    if (status?.status === 'authorized' || status?.status === 'provisioned') {
      queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    }
  }, [status?.status, queryClient])

  function handleRetry() {
    setCurrentAttemptId(null)
    startFlow.mutate(undefined, {
      onSuccess: (data) => {
        setCurrentAttemptId(data.attempt_id)
        setDeviceFlowCodes(
          data.user_code,
          data.verification_uri,
          data.expires_in ? Date.now() + data.expires_in * 1000 : null,
        )
      },
    })
  }

  if (startFlow.isPending || (!status && !deviceFlowUserCode)) {
    return <Spinner label="Démarrage du Device Code Flow…" />
  }

  // Codes d'erreur structurés
  const errorCode = status?.status === 'failed' ? status.error?.code ?? null : null

  if (status?.status === 'failed' || status?.status === 'expired' || (secondsLeft !== null && secondsLeft <= 0)) {
    const errorMessage: Record<string, string> = {
      device_flow_denied: "Vous avez refusé ou annulé la demande Microsoft.",
      device_flow_error: "Erreur lors de l'authentification Microsoft.",
      halo_exchange_failed: "Impossible d'obtenir un accès Halo. Veuillez réessayer.",
      identity_resolution_failed: "Impossible de résoudre votre Gamertag. Réessayez dans quelques instants.",
    }
    return (
      <div className="space-y-3">
        <p className="text-destructive font-medium">
          {status?.status === 'expired' || (secondsLeft !== null && secondsLeft <= 0)
            ? 'Le code a expiré.'
            : (errorCode && errorMessage[errorCode]) ?? "Échec de l'authentification."}
        </p>
        <Button onClick={handleRetry}>Réessayer</Button>
      </div>
    )
  }

  if (status?.status === 'authorized' || status?.status === 'provisioned') {
    return (
      <div className="space-y-2">
        <p className="text-success font-semibold">✓ Authentification réussie !</p>
        {status.gamertag && (
          <div className="rounded-lg border border-success/30 bg-success/10 p-3">
            <p className="text-xs text-muted-foreground">Compte Microsoft identifié :</p>
            <p className="mt-0.5 text-lg font-bold text-success">{status.gamertag}</p>
          </div>
        )}
        <Spinner size="sm" label="Chargement du profil…" />
      </div>
    )
  }

  const uri = deviceFlowVerificationUri ?? 'https://microsoft.com/devicelogin'
  const mins = secondsLeft != null ? Math.floor(secondsLeft / 60) : null
  const secs = secondsLeft != null ? secondsLeft % 60 : null

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Connexion Microsoft</h2>
      <p className="text-sm text-muted-foreground">
        Rendez-vous sur{' '}
        <a href={uri} target="_blank" rel="noopener noreferrer" className="text-primary underline">
          {uri.replace('https://', '')}
        </a>{' '}
        et entrez ce code :
      </p>
      <div className="rounded-lg bg-card px-6 py-4 text-center">
        {deviceFlowUserCode ? (
          <span className="text-3xl font-mono font-bold tracking-widest text-foreground select-all">
            {deviceFlowUserCode}
          </span>
        ) : (
          <Spinner size="sm" />
        )}
      </div>
      {secondsLeft != null && secondsLeft > 0 && (
        <p className="text-center text-xs text-muted-foreground">
          Code valide encore{' '}
          <span className={secondsLeft < 60 ? 'text-warning font-semibold' : ''}>
            {mins}:{String(secs).padStart(2, '0')}
          </span>
        </p>
      )}
      <p className="text-xs text-muted-foreground text-center animate-pulse">
        En attente de l'authentification…
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Étape 2 — Création du profil joueur
// ---------------------------------------------------------------------------
function StepPlayer() {
  const linkedHaloIdentity = useAppShellStore((s) => s.linkedHaloIdentity)
  const [gamertagInput, setGamertagInput] = useState('')
  const queryClient = useQueryClient()
  const createPlayer = useCreatePlayer()
  const gamertag = linkedHaloIdentity?.gamertag ?? gamertagInput

  function handleCreate() {
    if (!gamertag.trim()) return
    createPlayer.mutate(
      {
        gamertag: gamertag.trim(),
        xuid: linkedHaloIdentity?.xuid ?? undefined,
      },
      {
        onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap }),
      },
    )
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Créer votre profil joueur</h2>

      {linkedHaloIdentity ? (
        /* Carte de confirmation — identité résolue depuis la session */
        <div className="rounded-lg border border-primary/30 bg-primary/10 p-4">
          <p className="text-xs text-muted-foreground">Identité Halo liée à cette session :</p>
          <p className="mt-1 text-2xl font-bold text-primary">
            {linkedHaloIdentity.gamertag}
          </p>
          <p className="text-xs text-muted-foreground mt-0.5 font-mono">
            XUID {linkedHaloIdentity.xuid}
          </p>
          <p className="mt-2 text-xs text-muted-foreground">
            Un profil local sera créé pour ce compte.
          </p>
        </div>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">
            Entrez votre Gamertag Xbox pour créer votre profil.
          </p>
          <Input
            value={gamertagInput}
            onChange={(e) => setGamertagInput(e.target.value)}
            placeholder="MonGamertag"
            onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
          />
        </>
      )}

      {createPlayer.isError && (
        <p className="text-destructive text-sm">
          {getApiErrorMessage(createPlayer.error, 'Erreur lors de la création du profil.')}
        </p>
      )}

      <Button
        onClick={handleCreate}
        disabled={!gamertag.trim() || createPlayer.isPending || createPlayer.isSuccess}
      >
        {createPlayer.isPending
          ? 'Création…'
          : linkedHaloIdentity
          ? 'Confirmer et créer mon profil'
          : 'Ajouter'}
      </Button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Étape 3 — Synchronisation initiale
// ---------------------------------------------------------------------------
function StepInitialSync({ playerSlug }: { playerSlug: string }) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const currentJobId = useSetupFlowStore((s) => s.currentJobId)
  const setCurrentJobId = useSetupFlowStore((s) => s.setCurrentJobId)
  const startSync = useStartInitialSync()
  const queryClient = useQueryClient()

  // Reprendre depuis le job actif connu (session serveur) ou le job local du store
  const resolvedJobId = activeSyncJobId ?? currentJobId

  const { data: job } = useJobStatus(resolvedJobId ?? '', !!resolvedJobId)

  function handleStart() {
    startSync.mutate(
      { player_slug: playerSlug, max_matches: 200 },
      {
        onSuccess: (j) => {
          setCurrentJobId(j.job_id)
          queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
        },
      },
    )
  }

  // Quand la sync réussit : invalider le bootstrap pour passer à "ready"
  useEffect(() => {
    if (job?.status === 'succeeded') {
      queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    }
  }, [job?.status, queryClient])

  const errorMessages: Record<string, string> = {
    sync_auth_expired: "Votre session Halo a expiré. Relancez pour renouveler l'authentification.",
    sync_halo_api_error: "L'API Halo est temporairement indisponible. Veuillez réessayer.",
    sync_db_error: "Erreur interne lors de l'enregistrement des données. Contactez le support.",
    sync_aborted: "La synchronisation a été interrompue. Vous pouvez relancer.",
    internal_error: "Erreur inattendue. Veuillez réessayer.",
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Synchronisation initiale</h2>
      <p className="text-sm text-muted-foreground">
        Nous allons télécharger vos matchs Halo Infinite et calculer vos statistiques.
        Cela prend environ 2–4 minutes selon votre historique.
      </p>

      {!resolvedJobId && (
        <Button onClick={handleStart} disabled={startSync.isPending}>
          {startSync.isPending ? 'Démarrage…' : 'Lancer la synchronisation'}
        </Button>
      )}

      {job && (
        <div className="space-y-3">
          {/* Barre de progression */}
          {job.progress_pct != null && (
            <div className="space-y-1">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{job.phase_label ?? job.current_step ?? '…'}</span>
                <span>{job.progress_pct} %</span>
              </div>
              <div className="h-2 w-full rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-primary transition-all duration-500"
                  style={{ width: `${job.progress_pct}%` }}
                />
              </div>
            </div>
          )}

          {/* Compteurs métier */}
          {job.matches_done != null && job.matches_total != null && (
            <p className="text-sm text-muted-foreground">
              {job.matches_done} / {job.matches_total} matchs récupérés
            </p>
          )}

          {/* ETA */}
          {job.eta_seconds != null && job.status === 'running' && (
            <p className="text-xs text-muted-foreground">
              Temps restant estimé : environ {Math.ceil(job.eta_seconds / 60)} min
            </p>
          )}

          {/* Warnings */}
          {job.warnings.length > 0 && (
            <ul className="text-xs text-warning space-y-0.5">
              {job.warnings.map((w) => <li key={w}>⚠️ {w}</li>)}
            </ul>
          )}

          {/* Résultat réussi */}
          {job.status === 'succeeded' && (
            <div className="space-y-2">
              <p className="text-success font-medium">
                ✓ Synchronisation terminée&thinsp;!
                {job.result?.matches_imported != null && (
                  <span className="font-normal text-muted-foreground">
                    {' '}{Number(job.result.matches_imported)} matchs importés.
                  </span>
                )}
              </p>
              <Button onClick={() => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })}>
                Ouvrir l'application
              </Button>
            </div>
          )}

          {/* Interrompu (redémarrage serveur) */}
          {job.status === 'interrupted' && (
            <div className="space-y-2">
              <p className="text-warning font-medium">
                ⚡ Synchronisation interrompue (redémarrage serveur).
              </p>
              <Button
                variant="outline"
                onClick={() => {
                  setCurrentJobId(null)
                  handleStart()
                }}
              >
                Reprendre la synchronisation
              </Button>
            </div>
          )}

          {/* Erreur */}
          {job.status === 'failed' && (
            <div className="space-y-2">
              <p className="text-destructive font-medium">✗ Échec de la synchronisation.</p>
              {job.error?.code && (
                <p className="text-sm text-muted-foreground">
                  {errorMessages[job.error.code] ?? job.error.message}
                </p>
              )}
              {job.error?.retryable !== false && (
                <Button
                  variant="outline"
                  onClick={() => {
                    setCurrentJobId(null)
                    handleStart()
                  }}
                >
                  Réessayer
                </Button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// SetupPage — orchestrateur piloté par setupState (source : /bootstrap)
// ---------------------------------------------------------------------------
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
