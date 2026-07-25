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
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'
import { useStartDeviceFlow, useDeviceFlowStatus } from './queries'
import { apiErrorCode, type ApiError } from '@/lib/api/client'
import { verificationLinkLabel } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

// Nombre max de relances AUTOMATIQUES sur attempt_not_found (tentative balayée /
// backend redémarré). Au-delà, on bascule sur un retry manuel pour éviter une
// boucle de relances si la tentative disparaît systématiquement.
const MAX_AUTO_RECOVERY = 3

export function StepDeviceCode() {
  const currentAttemptId = useSetupFlowStore((s) => s.currentAttemptId)
  const setCurrentAttemptId = useSetupFlowStore((s) => s.setCurrentAttemptId)
  const deviceFlowUserCode = useSetupFlowStore((s) => s.deviceFlowUserCode)
  const deviceFlowVerificationUri = useSetupFlowStore((s) => s.deviceFlowVerificationUri)
  const deviceFlowExpiresAt = useSetupFlowStore((s) => s.deviceFlowExpiresAt)
  const setDeviceFlowCodes = useSetupFlowStore((s) => s.setDeviceFlowCodes)
  const queryClient = useQueryClient()
  const startFlow = useStartDeviceFlow()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Countdown
  const [secondsLeft, setSecondsLeft] = useState<number | null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Récupération gracieuse : compteur de relances auto + bascule manuelle.
  const recoveryCountRef = useRef(0)
  const [recoveryExhausted, setRecoveryExhausted] = useState(false)

  // Échec du démarrage du flow (POST /device-flow/start en 500/503) : sans état
  // dédié, un start en erreur laissait la garde spinner tourner à l'infini
  // (l'erreur était avalée). On la surface vers l'UI d'erreur + « Réessayer ».
  const [startError, setStartError] = useState<string | null>(null)

  // Lance (ou relance) le Device Code Flow. Centralise onSuccess + onError pour
  // les 3 points d'appel (montage, récupération auto, retry manuel). Ne remet
  // PAS startError à zéro ici : ce serait un setState synchrone dans l'effet de
  // montage (cascading render) — le reset se fait dans handleRetry (event).
  function startDeviceFlow() {
    startFlow.mutate(undefined, {
      onSuccess: (data) => {
        setCurrentAttemptId(data.attempt_id)
        setDeviceFlowCodes(
          data.user_code,
          data.verification_uri,
          data.expires_in ? Date.now() + data.expires_in * 1000 : null,
        )
      },
      onError: (err) => {
        const apiErr = err as unknown as ApiError
        setStartError(
          apiErr.code === 'demo_mode'
            ? t('common.xbox_login.err_demo')
            : t('common.setup.device_start_failed'),
        )
      },
    })
  }

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

  const { data: status, error } = useDeviceFlowStatus(currentAttemptId ?? '', !!currentAttemptId)

  // Démarrer le flow au montage si pas encore en cours
  useEffect(() => {
    if (!currentAttemptId) {
      startDeviceFlow()
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Récupération : si le polling renvoie attempt_not_found (tentative balayée /
  // backend redémarré), relancer automatiquement un nouveau flow plutôt que de
  // rester bloqué jusqu'à l'expiration du compte à rebours.
  useEffect(() => {
    if (!currentAttemptId || apiErrorCode(error) !== 'attempt_not_found') return
    if (recoveryCountRef.current >= MAX_AUTO_RECOVERY) {
      setRecoveryExhausted(true)
      return
    }
    recoveryCountRef.current += 1
    setCurrentAttemptId(null)
    startDeviceFlow()
  }, [error, currentAttemptId]) // eslint-disable-line react-hooks/exhaustive-deps

  // Quand le flow réussit : invalider le bootstrap pour que setupState avance
  useEffect(() => {
    if (status?.status === 'authorized' || status?.status === 'provisioned') {
      queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    }
  }, [status?.status, queryClient])

  function handleRetry() {
    recoveryCountRef.current = 0
    setRecoveryExhausted(false)
    setStartError(null)
    setCurrentAttemptId(null)
    startDeviceFlow()
  }

  // Échec du start (500/503) : surfacer AVANT la garde spinner, sinon
  // `!status && !deviceFlowUserCode` maintiendrait le spinner indéfiniment.
  if (startError) {
    return (
      <div className="space-y-3">
        <p className="text-destructive font-medium">{startError}</p>
        <Button onClick={handleRetry}>{t('common.xbox_login.retry')}</Button>
      </div>
    )
  }

  if (startFlow.isPending || (!status && !deviceFlowUserCode)) {
    return <Spinner label={t('common.setup.device_starting')} />
  }

  // Codes d'erreur structurés
  const errorCode = status?.status === 'failed' ? status.error_code ?? null : null

  // recoveryExhausted : tentative balayée en boucle (backend instable) → on
  // route vers la même UI d'erreur + bouton « Réessayer » que les autres échecs.
  if (status?.status === 'failed' || status?.status === 'expired' || (secondsLeft !== null && secondsLeft <= 0) || recoveryExhausted) {
    const errorMessage: Record<string, string> = {
      device_flow_denied: t('common.device_code.err_denied'),
      device_flow_error: t('common.device_code.err_ms'),
      halo_exchange_failed: t('common.device_code.err_halo'),
      identity_resolution_failed: t('common.device_code.err_identity'),
    }
    return (
      <div className="space-y-3">
        <p className="text-destructive font-medium">
          {status?.status === 'expired' || (secondsLeft !== null && secondsLeft <= 0)
            ? t('common.device_code.err_expired')
            : (errorCode && errorMessage[errorCode]) ?? t('common.device_code.err_generic')}
        </p>
        <Button onClick={handleRetry}>{t('common.xbox_login.retry')}</Button>
      </div>
    )
  }

  if (status?.status === 'authorized' || status?.status === 'provisioned') {
    return (
      <div className="space-y-2">
        <p className="text-success font-semibold">{t('common.setup.device_auth_success')}</p>
        {status.gamertag && (
          <div className="rounded-lg border border-success/30 bg-success/10 p-3">
            <p className="text-xs text-muted-foreground">{t('common.setup.microsoft_account_identified')}</p>
            <p className="mt-0.5 text-lg font-bold text-success">{status.gamertag}</p>
          </div>
        )}
        <Spinner size="sm" label={t('common.setup.profile_loading_short')} />
      </div>
    )
  }

  const uri = deviceFlowVerificationUri ?? 'https://microsoft.com/devicelogin'
  const mins = secondsLeft != null ? Math.floor(secondsLeft / 60) : null
  const secs = secondsLeft != null ? secondsLeft % 60 : null

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">{t('common.setup.microsoft_connection_title')}</h2>
      <p className="text-sm text-muted-foreground">
        {t('common.setup.go_to')}{' '}
        {/* Jamais l'URL brute (query params PKCE illisibles + overflow) : libellé
            court host/chemin — le domaine reste visible (anti-phishing), l'URL
            complète est portée par le href. break-all = défense anti-overflow. */}
        <a href={uri} target="_blank" rel="noopener noreferrer" className="text-primary underline break-all">
          {verificationLinkLabel(uri)}
        </a>{' '}
        {t('common.setup.enter_this_code')}
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
          {t('common.setup.code_valid_still')}{' '}
          <span className={secondsLeft < 60 ? 'text-warning font-semibold' : ''}>
            {mins}:{String(secs).padStart(2, '0')}
          </span>
        </p>
      )}
      <p className="text-xs text-muted-foreground text-center animate-pulse">
        {t('common.setup.waiting_auth')}
      </p>
    </div>
  )
}
