/**
 * useBrowserLogin — l'état de la connexion Microsoft par le navigateur : la tentative ouverte, son
 * sondage, le démarrage.
 *
 * Le hook vit dans l'étape de connexion (StepDeviceCode), PAS dans le panneau : l'utilisateur peut
 * ouvrir le navigateur, basculer vers le code d'appareil, puis finir quand même la connexion dans
 * l'onglet. Le sondage reste donc actif tant que l'étape est montée, quelle que soit la méthode
 * affichée : c'est lui qui remet la tentative autorisée à la session (le serveur ne la réclame qu'au
 * sondage) et qui fait avancer l'assistant.
 */
import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'
import { type ApiError } from '@/lib/api/client'
import { isDemoRefusal } from '@/lib/api/demoReadOnly'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { isTooManyLoginAttempts } from '@/features/auth/loginError'
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

export function useBrowserLogin() {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
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

/** Ce que rend useBrowserLogin au panneau d'affichage. */
export type BrowserLogin = ReturnType<typeof useBrowserLogin>
