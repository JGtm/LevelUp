/**
 * StepDeviceCode — étape 1 du wizard SetupPage : Device Code Flow Microsoft.
 *
 * P8.4 (revue 2026-04-29) : extrait de SetupPage.tsx (~145L).
 */
import { useState, useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { queryKeys } from '@/lib/query/keys'
import { useStartDeviceFlow, useDeviceFlowStatus } from './queries'

export function StepDeviceCode() {
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
