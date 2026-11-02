/**
 * BrowserLoginPanel — connexion Microsoft par le navigateur par défaut (application de bureau, capacité
 * `can_login_via_browser`), sans code à recopier.
 *
 * Le clic demande au serveur une adresse d'autorisation (GET /auth/xbox/login) et l'ouvre par un lien
 * `target="_blank"` : la coquille l'envoie au navigateur du système, jamais dans sa fenêtre. Le retour de
 * Microsoft arrive dans le navigateur ; ici, le sondage de la tentative (le même que celui du code
 * d'appareil) apprend qu'elle a abouti et fait avancer l'assistant par le bootstrap. Le code d'appareil
 * reste le secours (`onUseDeviceCode`).
 */
import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'
import { type ApiError } from '@/lib/api/client'
import { isDemoRefusal } from '@/lib/api/demoReadOnly'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { deviceFlowFailureKey, isTooManyLoginAttempts } from '@/features/auth/loginError'
import { isDeviceFlowSucceeded, useDeviceFlowStatus, useStartBrowserLogin } from './queries'

/**
 * Ouvre une adresse par un lien `target="_blank"` : la coquille intercepte la demande de nouvelle
 * fenêtre et l'envoie au navigateur du système. Le lien est créé dans la foulée du clic (l'activation
 * de l'utilisateur vaut encore) ; le même lien reste affiché à l'écran en repli.
 */
function openInBrowser(url: string): void {
  const link = document.createElement('a')
  link.href = url
  link.target = '_blank'
  link.rel = 'noopener noreferrer'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

type Translate = (key: CommonManifestKey) => string

/** L'état de la connexion par le navigateur : la tentative ouverte, son sondage, le démarrage. */
function useBrowserLogin(t: Translate) {
  const queryClient = useQueryClient()
  const start = useStartBrowserLogin()
  const [attempt, setAttempt] = useState<{ id: string; url: string } | null>(null)
  const [startError, setStartError] = useState<string | null>(null)
  const { data: status } = useDeviceFlowStatus(attempt?.id ?? '', !!attempt)
  const advancedRef = useRef(false)

  // La tentative a abouti : le bootstrap avance l'assistant (étape du profil), comme au code d'appareil.
  useEffect(() => {
    if (isDeviceFlowSucceeded(status?.status) && !advancedRef.current) {
      advancedRef.current = true
      void queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    }
  }, [status?.status, queryClient])

  function handleStart() {
    setStartError(null)
    setAttempt(null)
    advancedRef.current = false
    start.mutate(undefined, {
      onSuccess: (data) => {
        setAttempt({ id: data.attempt_id, url: data.authorize_url })
        openInBrowser(data.authorize_url)
      },
      onError: (err) => {
        const apiErr = err as unknown as ApiError
        setStartError(
          isDemoRefusal(apiErr)
            ? t('common.xbox_login.err_demo')
            : isTooManyLoginAttempts(apiErr)
              ? t('common.xbox_login.err_busy')
              : t('common.setup.device_start_failed'),
        )
      },
    })
  }

  return { attempt, startError, status, isStarting: start.isPending, handleStart }
}

interface BrowserLoginPanelProps {
  onUseDeviceCode: () => void
}

function UseCodeLink({ onUseDeviceCode, t }: BrowserLoginPanelProps & { t: Translate }) {
  return (
    <p className="text-center text-xs text-muted-foreground">
      <button
        type="button"
        onClick={onUseDeviceCode}
        className="underline underline-offset-2 transition-colors hover:text-foreground"
      >
        {t('common.setup.browser_login_use_code')}
      </button>
    </p>
  )
}

export function BrowserLoginPanel({ onUseDeviceCode }: BrowserLoginPanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t: Translate = (key) => formatMessage(commonManifest, key, locale)
  const { attempt, startError, status, isStarting, handleStart } = useBrowserLogin(t)
  const useCodeLink = <UseCodeLink onUseDeviceCode={onUseDeviceCode} t={t} />

  if (startError || status?.status === 'failed' || status?.status === 'expired') {
    const message = startError ?? t(deviceFlowFailureKey(status?.status ?? 'failed', status?.error_code))
    return (
      <div className="space-y-3">
        <p role="alert" className="font-medium text-destructive">
          {message}
        </p>
        <Button onClick={handleStart}>{t('common.xbox_login.retry')}</Button>
        {useCodeLink}
      </div>
    )
  }

  if (isDeviceFlowSucceeded(status?.status)) {
    return (
      <div className="space-y-2">
        <p className="font-semibold text-success">{t('common.setup.device_auth_success')}</p>
        {status?.gamertag && (
          <div className="rounded-lg border border-success/30 bg-success/10 p-3">
            <p className="text-xs text-muted-foreground">{t('common.setup.microsoft_account_identified')}</p>
            <p className="mt-0.5 text-lg font-bold text-success">{status.gamertag}</p>
          </div>
        )}
        <Spinner size="sm" label={t('common.setup.profile_loading_short')} />
      </div>
    )
  }

  if (isStarting) return <Spinner label={t('common.setup.browser_login_opening')} />

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">{t('common.setup.microsoft_connection_title')}</h2>
      {attempt ? (
        <>
          <p className="animate-pulse text-sm text-muted-foreground">{t('common.setup.browser_login_waiting')}</p>
          <p className="text-xs text-muted-foreground">
            {t('common.setup.browser_login_reopen_hint')}{' '}
            <a href={attempt.url} target="_blank" rel="noopener noreferrer" className="text-primary underline">
              {t('common.setup.browser_login_reopen_link')}
            </a>
          </p>
        </>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">{t('common.setup.browser_login_lead')}</p>
          <Button onClick={handleStart} className="w-full">
            {t('common.setup.browser_login_button')}
          </Button>
        </>
      )}
      {useCodeLink}
    </div>
  )
}
