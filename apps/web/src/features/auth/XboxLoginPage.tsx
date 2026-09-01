/**
 * XboxLoginPage — authentification SSO Xbox via Device Code Flow.
 *
 * Affichée par LoginPage quand auth_mode='xbox'. Permet à un user de se connecter
 * via son compte Xbox Live (login.live.com/devicelogin). Côté backend, la session
 * est créée par XboxSSOLinkStrategy (PR 2) après l'échange OAuth.
 *
 * En mode xbox, le login password est réservé aux admins (PR 1, D3). Cette page
 * propose un toggle "Connexion admin (mot de passe)" pour le fallback.
 */
import { useState, useEffect, useRef, type FormEvent } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { AppFooter } from '@/components/shell/AppFooter'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { useStartDeviceFlow, useDeviceFlowStatus } from '@/features/setup/queries'
import { useLogin } from '@/features/auth/queries'
import { storePasswordCredential } from '@/features/auth/credentials'
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'
import { api, API_BASE_URL, apiErrorCode, type ApiError } from '@/lib/api/client'
import type { BootstrapResponse } from '@/lib/api/types'
import { postLoginDestination } from '@/features/auth/postLoginDestination'
import { verificationLinkLabel } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

// Relances AUTO max sur attempt_not_found avant bascule sur retry manuel.
const MAX_AUTO_RECOVERY = 3

export function XboxLoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const oauthCodeFlowEnabled = useAppShellStore((s) => s.oauthCodeFlowEnabled)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const [showAdminLogin, setShowAdminLogin] = useState(false)
  const [forceDeviceCode, setForceDeviceCode] = useState(false)

  if (showAdminLogin) {
    return <AdminPasswordPanel onBack={() => setShowAdminLogin(false)} />
  }

  // PR 4 — préférer Authorization Code Flow (redirect SSO) si configuré
  // côté backend. Le user peut basculer sur Device Code via le toggle.
  const useRedirectFlow = oauthCodeFlowEnabled && !forceDeviceCode

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardContent className="pt-6">
            {useRedirectFlow ? (
              <RedirectFlowPanel onUseDeviceCode={() => setForceDeviceCode(true)} />
            ) : (
              <XboxFlowPanel
                onAuthorized={async () => {
                  // Recharge le bootstrap frais (post-login autoritatif) pour
                  // décider la destination : un joueur DÉJÀ établi (données
                  // synchronisées) va direct au dashboard ; seul un vrai nouveau
                  // joueur voit la page d'onboarding OpenSpartan. fetchQuery
                  // repeuple aussi le cache lu par __root (pas de double fetch).
                  //
                  // staleTime:0 IMPÉRATIF : sans lui, fetchQuery hérite du défaut
                  // global (5 min, queryClient.ts) et RENVOIE le bootstrap ANONYME
                  // mis en cache avant le login (pré-device-flow) → un joueur établi
                  // reste coincé sur l'onboarding. On force une lecture réseau.
                  let boot: BootstrapResponse | null = null
                  try {
                    boot = await queryClient.fetchQuery({
                      queryKey: queryKeys.bootstrap,
                      queryFn: () => api.get<BootstrapResponse>('/bootstrap'),
                      staleTime: 0,
                    })
                  } catch {
                    // Bootstrap injoignable : ne pas bloquer le login abouti —
                    // l'onboarding reste un atterrissage sûr (CTA « Continuer »).
                    await queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
                  }
                  navigate({ to: postLoginDestination(boot) })
                }}
              />
            )}

            <p className="mt-6 text-center text-xs text-muted-foreground">
              <button
                type="button"
                onClick={() => setShowAdminLogin(true)}
                className="underline underline-offset-2 hover:text-foreground transition-colors"
              >
                {t('common.auth.xbox_admin_login')}
              </button>
            </p>
          </CardContent>
        </Card>

        <AppFooter variant="minimal" />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Authorization Code Flow panel (PR 4 — redirect SSO)
// ---------------------------------------------------------------------------

interface RedirectFlowPanelProps {
  onUseDeviceCode: () => void
}

function RedirectFlowPanel({ onUseDeviceCode }: RedirectFlowPanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  function handleClick() {
    // Redirect plein-page vers le backend qui génère le state CSRF + redirect
    // vers Microsoft /authorize. Microsoft renvoie l'user vers /auth/xbox/callback
    // qui finalise la session puis redirect vers "/".
    window.location.assign(`${API_BASE_URL}/auth/xbox/login`)
  }

  return (
    <div className="space-y-5 text-center">
      <h2 className="text-lg font-semibold">{t('common.xbox_login.title')}</h2>
      <p className="text-sm text-muted-foreground">
        {t('common.auth.xbox_connect_microsoft')}
      </p>
      <Button onClick={handleClick} className="w-full">
        {t('common.auth.xbox_sign_in')}
      </Button>
      <p className="text-xs text-muted-foreground">
        <button
          type="button"
          onClick={onUseDeviceCode}
          className="underline underline-offset-2 hover:text-foreground transition-colors"
        >
          {t('common.auth.xbox_use_9char_code')}
        </button>
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Xbox Device Code Flow panel
// ---------------------------------------------------------------------------

interface XboxFlowPanelProps {
  onAuthorized: () => void | Promise<void>
}

function XboxFlowPanel({ onAuthorized }: XboxFlowPanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const [attemptId, setAttemptId] = useState<string | null>(null)
  const [userCode, setUserCode] = useState<string | null>(null)
  const [verificationUri, setVerificationUri] = useState<string | null>(null)
  const [expiresAt, setExpiresAt] = useState<number | null>(null)
  const [secondsLeft, setSecondsLeft] = useState<number | null>(null)
  const [startError, setStartError] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const recoveryCountRef = useRef(0)

  const startFlow = useStartDeviceFlow()
  const { data: status, error } = useDeviceFlowStatus(attemptId ?? '', !!attemptId)

  // Démarrer le flow au montage.
  useEffect(() => {
    if (!attemptId && !startFlow.isPending && !startError) {
      startFlow.mutate(undefined, {
        onSuccess: (data) => {
          setAttemptId(data.attempt_id)
          setUserCode(data.user_code)
          setVerificationUri(data.verification_uri)
          setExpiresAt(data.expires_in ? Date.now() + data.expires_in * 1000 : null)
        },
        onError: (err) => {
          const apiErr = err as unknown as ApiError
          if (apiErr.code === 'demo_mode') {
            setStartError(t('common.xbox_login.err_demo'))
          } else {
            setStartError(apiErr.message ?? t('common.xbox_login.err_start'))
          }
        },
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Countdown.
  useEffect(() => {
    if (!expiresAt) return
    const update = () => {
      const diff = Math.max(0, Math.floor((expiresAt - Date.now()) / 1000))
      setSecondsLeft(diff)
      if (diff <= 0 && timerRef.current) clearInterval(timerRef.current)
    }
    update()
    timerRef.current = setInterval(update, 1000)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [expiresAt])

  // Redirect quand authorized.
  useEffect(() => {
    if (status?.status === 'authorized' || status?.status === 'provisioned') {
      void onAuthorized()
    }
  }, [status?.status, onAuthorized])

  // Récupération : attempt_not_found (tentative balayée / backend redémarré) →
  // relancer un nouveau flow plutôt que de rester bloqué.
  useEffect(() => {
    if (!attemptId || apiErrorCode(error) !== 'attempt_not_found') return
    if (recoveryCountRef.current >= MAX_AUTO_RECOVERY) {
      setStartError(t('common.xbox_login.err_interrupted'))
      return
    }
    recoveryCountRef.current += 1
    setAttemptId(null)
    setUserCode(null)
    setVerificationUri(null)
    setExpiresAt(null)
    setSecondsLeft(null)
    startFlow.mutate(undefined, {
      onSuccess: (data) => {
        setAttemptId(data.attempt_id)
        setUserCode(data.user_code)
        setVerificationUri(data.verification_uri)
        setExpiresAt(data.expires_in ? Date.now() + data.expires_in * 1000 : null)
      },
      onError: (err) => {
        const apiErr = err as unknown as ApiError
        setStartError(apiErr.message ?? t('common.xbox_login.err_restart'))
      },
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [error, attemptId])

  function handleRetry() {
    recoveryCountRef.current = 0
    setAttemptId(null)
    setUserCode(null)
    setVerificationUri(null)
    setExpiresAt(null)
    setSecondsLeft(null)
    setStartError(null)
    startFlow.mutate(undefined, {
      onSuccess: (data) => {
        setAttemptId(data.attempt_id)
        setUserCode(data.user_code)
        setVerificationUri(data.verification_uri)
        setExpiresAt(data.expires_in ? Date.now() + data.expires_in * 1000 : null)
      },
      onError: (err) => {
        const apiErr = err as unknown as ApiError
        setStartError(apiErr.message ?? t('common.xbox_login.err_restart'))
      },
    })
  }

  // États d'erreur / chargement.
  if (startError) {
    return (
      <div className="space-y-3 text-center">
        <p className="text-destructive font-medium">{startError}</p>
        <Button onClick={handleRetry}>{t('common.xbox_login.retry')}</Button>
      </div>
    )
  }

  if (startFlow.isPending || !attemptId) {
    return <Spinner label={t('common.auth.xbox_preparing')} />
  }

  if (status?.status === 'failed' || status?.status === 'expired' || (secondsLeft !== null && secondsLeft <= 0)) {
    const expired = status?.status === 'expired' || (secondsLeft !== null && secondsLeft <= 0)
    return (
      <div className="space-y-3 text-center">
        <p className="text-destructive font-medium">
          {expired ? 'Le code a expiré.' : status?.error_detail ?? 'Échec de l\'authentification.'}
        </p>
        <Button onClick={handleRetry}>{t('common.xbox_login.retry')}</Button>
      </div>
    )
  }

  if (status?.status === 'authorized' || status?.status === 'provisioned') {
    return (
      <div className="space-y-3 text-center">
        <p className="text-success font-semibold">{t('common.auth.xbox_auth_success')}</p>
        {status.gamertag && (
          <p className="text-sm text-muted-foreground">
            {formatMessage(commonManifest, 'common.xbox_login.welcome', locale, {
              gamertag: status.gamertag,
            })}
          </p>
        )}
        <Spinner size="sm" label={t('common.empty.loading')} />
      </div>
    )
  }

  // En attente du user_code (rare — flow démarré mais user_code pas encore propagé).
  if (!userCode) {
    return <Spinner label={t('common.auth.xbox_generating_code')} />
  }

  const uri = verificationUri ?? 'https://microsoft.com/devicelogin'
  const mins = secondsLeft != null ? Math.floor(secondsLeft / 60) : null
  const secs = secondsLeft != null ? secondsLeft % 60 : null

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-center">{t('common.xbox_login.title')}</h2>
      <p className="text-sm text-muted-foreground text-center">
        {t('common.setup.go_to')}{' '}
        {/* Jamais l'URL brute (query params PKCE illisibles + overflow) : libellé
            court host/chemin — le domaine reste visible (anti-phishing), l'URL
            complète est portée par le href. break-all = défense anti-overflow. */}
        <a href={uri} target="_blank" rel="noopener noreferrer" className="text-primary underline break-all">
          {verificationLinkLabel(uri)}
        </a>
      </p>
      <div className="rounded-lg bg-card border px-6 py-4 text-center">
        <p className="mb-2 text-xs text-muted-foreground">{t('common.auth.xbox_code_to_enter')}</p>
        <span className="text-3xl font-mono font-bold tracking-widest text-foreground select-all">
          {userCode}
        </span>
      </div>
      {secondsLeft != null && secondsLeft > 0 && (
        <p className="text-center text-xs text-muted-foreground">
          {t('common.xbox_login.valid_for')}{' '}
          <span className={secondsLeft < 60 ? 'text-warning font-semibold' : ''}>
            {mins}:{String(secs).padStart(2, '0')}
          </span>
        </p>
      )}
      <p className="text-xs text-muted-foreground text-center animate-pulse">
        {t('common.setup.waiting_auth')}
      </p>

      {/* Disclaimer anti-phishing (cf. SPRINT_XBOX_SSO §8 piège 9). */}
      <div className="rounded-md bg-muted/40 border border-muted px-3 py-2 text-xs text-muted-foreground">
        {t('common.auth.xbox_only_if_you_clicked')}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Admin password fallback panel (D3 cohabitation)
// ---------------------------------------------------------------------------

interface AdminPasswordPanelProps {
  onBack: () => void
}

function AdminPasswordPanel({ onBack }: AdminPasswordPanelProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const login = useLogin()

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    login.mutate(
      { username, password },
      {
        onSuccess: async () => {
          // Propose au navigateur d'enregistrer le mot de passe (login en fetch).
          storePasswordCredential(username, password)
          await queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
          navigate({ to: '/' })
        },
        onError: (err) => {
          const apiErr = err as unknown as ApiError
          if (apiErr.code === 'invalid_credentials') {
            setError('Identifiants incorrects.')
          } else if (apiErr.code === 'password_login_admin_only') {
            // D3 : utilisateur valide mais pas admin en mode xbox.
            setError('En mode SSO Xbox, le login par mot de passe est réservé aux administrateurs. Utilisez la connexion Xbox.')
          } else {
            setError(apiErr.message ?? 'Erreur de connexion.')
          }
        },
      },
    )
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardContent className="pt-6">
            <div className="mb-4 rounded-md bg-warning/10 border border-warning/30 px-3 py-2 text-xs text-warning">
              {t('common.auth.xbox_admin_only')}
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="admin-username" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.username_label')}
                </label>
                <input
                  id="admin-username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
              <div>
                <label htmlFor="admin-password" className="block text-sm font-medium text-foreground mb-1">
                  {t('common.auth.password_label')}
                </label>
                <input
                  id="admin-password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>

              {error && <p className="text-sm text-destructive">{error}</p>}

              <Button type="submit" className="w-full" disabled={login.isPending}>
                {login.isPending ? t('common.auth.login_pending') : t('common.auth.login_action')}
              </Button>
            </form>

            <p className="mt-4 text-center text-sm text-muted-foreground">
              <button
                type="button"
                onClick={onBack}
                className="underline underline-offset-2 hover:text-foreground transition-colors"
              >
                {t('common.auth.xbox_back_to_xbox')}
              </button>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
