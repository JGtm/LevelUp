/**
 * Route racine — layout partagé par toutes les pages.
 *
 * Au montage, déclenche le bootstrap et hydrate l'AppShellStore.
 * Bloque le rendu tant que bootstrap n'a pas répondu.
 * En mode setup_required, redirige vers /setup.
 */

import { createRootRouteWithContext, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import type { RouterContext } from '@/app/router'
import { useEffect, useRef, useState } from 'react'
import {
  createBootstrapRetryPolicy,
  isServerStartingError,
  serverStartingStepKey,
} from '@/app/serverStartup'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { resolvePageTitle } from '@/lib/pageTitle'
import { useAppShellStore } from '@/stores/appShellStore'
import { AppShell } from '@/components/shell/AppShell'
import { log } from '@/components/shell/_logger'
import { isAnonymousPath } from '@/components/shell/shellNavigation'
import {
  decideOnAuthRequired,
  sessionStorageOrNull,
  type AuthRequiredVerdict,
} from '@/components/shell/authRequiredGuard'
import { needsOwnProfile, SETUP_PATH, setupRedirectPath } from '@/features/setup/setupRouting'
import type { BootstrapResponse } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export function RootLayout() {
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const hydrateFromBootstrap = useAppShellStore((s) => s.hydrateFromBootstrap)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const setupRequired = useAppShellStore((s) => s.setupRequired)
  const authMode = useAppShellStore((s) => s.authMode)
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const firstLaunch = useAppShellStore((s) => s.firstLaunch)
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  // Politique de rejeu à état (compteur cumulé de TanStack) : une par instance, stable.
  const [bootstrapRetry] = useState(() => createBootstrapRetryPolicy())

  const { data, isLoading, isError, failureReason } = useQuery({
    queryKey: queryKeys.bootstrap,
    queryFn: () => api.get<BootstrapResponse>('/bootstrap'),
    staleTime: 2 * 60 * 1000,
    // reauth_required (bannière de reconnexion Xbox) est la seule donnée volatile
    // du bootstrap : le back l'efface dès qu'un refresh par-joueur réussit. On
    // re-fetche au retour sur l'onglet pour que la bannière disparaisse sans
    // rechargement dur. Les redirections du useEffect([data]) sont idempotentes.
    //
    // NEUTRALISÉ PENDANT UNE BASCULE DE TITRE (anti-fuite cross-titre V72-29) :
    // hydrateFromBootstrap réécrit currentTitleSlug/header/currentPlayer depuis la
    // SESSION. Un refetch window-focus déclenché dans la fenêtre d'applyActiveTitle
    // (session/header transitoires) rejouerait l'ancien titre par-dessus la bascule
    // en cours → identité/rang d'un autre titre affichés. Tant que isTitleSwitching
    // est vrai, on ne refetch pas au focus ; applyActiveTitle fait lui-même le
    // re-bootstrap final avec le bon header.
    refetchOnWindowFocus: () => !useAppShellStore.getState().isTitleSwitching,
    // Serveur qui démarre (réseau, 502, 503 server_starting) : réinterrogation chaque
    // seconde jusqu'au plafond, puis l'écran « API injoignable » (cf. serverStartup.ts).
    retry: bootstrapRetry.retry,
    retryDelay: bootstrapRetry.retryDelay,
  })

  // Mécanisme UNIQUE de titre d'onglet (I18) : keyé sur [pathname, locale] — un
  // changement de langue dans Paramètres (sans navigation) met aussi à jour l'onglet.
  // resolvePageTitle est locale-aware (table Record<Locale,string> par route) ; les
  // anciens effets locaux dupliqués (MedalsPage/ComparePage/UnifiedCitationsPage) sont
  // supprimés — ils étaient de toute façon écrasés par CET effet, rejoué à chaque
  // navigation après les effets enfants.
  useEffect(() => {
    document.title = resolvePageTitle(pathname, locale)
  }, [pathname, locale])

  // Vraie expiration de session (TTL 7 j atteint, logout dans un autre onglet) :
  // le client HTTP dispatche `levelup:auth-required` sur un 401 `auth_required`
  // hors /bootstrap (cf. lib/api/client.ts). Sans ce listener, le garde
  // anti-éjection ci-dessous — qui préserve l'état authentifié sur un bootstrap
  // anonyme transitoire — piégerait l'utilisateur dans un shell authentifié mort
  // dont chaque appel API retombe en 401. La décision (authRequiredGuard.ts) relit
  // d'abord /bootstrap : connecté → aucune éjection (route secondaire en 401) ;
  // anonyme → rechargement plein (même mécanique que LogoutButton), plafonné par
  // fenêtre dans sessionStorage ; plafond atteint → /login sans recharger.
  const authCheckInFlightRef = useRef(false)
  useEffect(() => {
    async function handleAuthRequired() {
      // Store déjà anonyme (ex. /login, mauvais mot de passe → 401) : ne rien
      // faire. Recharger ici bouclerait sur /login. Seule une session qu'on
      // CROYAIT authentifiée (currentUsername truthy) justifie une vérification.
      if (!useAppShellStore.getState().currentUsername) return
      // Anti-rafale : une salve de 401 ne déclenche qu'une vérification à la fois.
      if (authCheckInFlightRef.current) return
      authCheckInFlightRef.current = true
      const verdict = await decideOnAuthRequired({
        fetchBootstrap: () => api.get<BootstrapResponse>('/bootstrap'),
        storage: sessionStorageOrNull(),
        now: Date.now,
      })
      applyAuthVerdict(verdict)
      if (verdict.kind !== 'reload') authCheckInFlightRef.current = false
    }
    function applyAuthVerdict(verdict: AuthRequiredVerdict) {
      switch (verdict.kind) {
        case 'still_authenticated':
          log.warn('session:route_401', 'route secondaire en 401, /bootstrap connecté : aucune éjection')
          return
        case 'unknown':
          log.warn('auth:bootstrap_unreachable', '401 auth_required, /bootstrap injoignable — aucune éjection', verdict.error)
          return
        case 'reload':
          log.warn('auth:session_expired', 'session expirée (401 auth_required) — rechargement plein vers /login')
          window.location.assign('/')
          return
        case 'reload_blocked':
          log.error('auth:reload_loop_blocked', 'rechargements pour 401 plafonnés — /login sans recharger')
          hydrateFromBootstrap(verdict.bootstrap)
          navigate({ to: '/login' })
          return
      }
    }
    const listener = () => void handleAuthRequired()
    window.addEventListener('levelup:auth-required', listener)
    return () => window.removeEventListener('levelup:auth-required', listener)
  }, [hydrateFromBootstrap, navigate])

  useEffect(() => {
    if (!data) return

    // Garde anti-éjection transitoire. Un refetch (focus d'onglet, navigation
    // multi-requêtes) qui renvoie un /bootstrap ANONYME alors que le store porte
    // déjà un utilisateur authentifié est une rétrogradation SUSPECTE — vestige
    // possible d'un torn read backend (cf. fix persistance atomique des sessions).
    // On NE ré-hydrate PAS (hydrater une réponse anonyme rabattrait aussi
    // currentPlayer/availablePlayers/isAdmin → état mi-anonyme incohérent) et on NE
    // redirige PAS vers /login : on préserve l'état authentifié complet.
    // La VRAIE expiration de session (TTL atteint, logout dans un autre onglet)
    // n'est PAS gérée ici — un /bootstrap anonyme ne réinitialise jamais
    // currentUsername, donc ce garde retomberait en boucle à chaque refetch. Elle
    // est couverte par le listener `levelup:auth-required` (effet ci-dessus) qui
    // recharge la page en plein sur un 401 `auth_required`. La déconnexion explicite
    // recharge aussi la page (store vidé → wasAuthenticated null → chemin
    // autoritaire ci-dessous), donc /login reste atteignable au logout.
    const wasAuthenticated = useAppShellStore.getState().currentUsername
    if (!data.current_username && wasAuthenticated) {
      log.warn(
        'bootstrap:anon_downgrade',
        'bootstrap anonyme transitoire ignoré — session authentifiée préservée',
      )
      return
    }

    hydrateFromBootstrap(data)

    // Auth locale : rediriger si pas connecté (modes password ET xbox).
    // En mode xbox, /register reste accessible UNIQUEMENT pour le bootstrap admin
    // initial (firstLaunch=true) — RegisterPage redirige vers /login sinon.
    if (data.auth_mode === 'password' || data.auth_mode === 'xbox') {
      const path = window.location.pathname
      // Pages consultables sans compte (confidentialité) : ne jamais les
      // éjecter vers /login, le pied de page de l'écran de connexion y renvoie.
      // Le gate `setup_required` plus bas continue de s'appliquer : une instance
      // non configurée n'a rien à servir.
      const anonymous = isAnonymousPath(path)
      if (data.first_launch && !anonymous && path !== '/register') {
        navigate({ to: '/register' })
        return
      }
      if (!data.current_username && !anonymous && path !== '/login' && path !== '/register') {
        navigate({ to: '/login' })
        return
      }
      // Déjà connecté mais sur une page d'auth (lien direct, ancien onglet, retour
      // arrière…) → renvoyer vers le dashboard plutôt que d'afficher le formulaire
      // de login « quoi qu'il arrive ».
      if (data.current_username && (path === '/login' || path === '/register')) {
        navigate({ to: '/' })
        return
      }
    }

    if (data.setup_required) {
      navigate({ to: '/setup' })
      return
    }

    // ADR 0035 D3 : un compte connecté qui n'a AUCUN profil accessible n'a rien à
    // consulter, et depuis l'ADR plus rien ne tourne pour lui tant qu'aucun profil
    // n'existe (ni poller, ni sync). On le conduit au wizard, seule sortie de cet
    // état. `setup_required` ci-dessus ne couvre que l'instance VIDE : sur une
    // instance déjà peuplée, ce compte atterrissait sur « on synchronise tes
    // derniers matchs » et n'en sortait jamais.
    const setupPath = setupRedirectPath(
      {
        authMode: data.auth_mode ?? 'none',
        currentUsername: data.current_username ?? null,
        isAdmin: data.is_admin ?? false,
        availablePlayerCount: data.available_players?.length ?? 0,
      },
      window.location.pathname,
      isAnonymousPath,
    )
    if (setupPath) {
      navigate({ to: setupPath })
    }
  }, [data, hydrateFromBootstrap, navigate])

  if (isLoading) {
    const starting = isServerStartingError(failureReason)
    const stepKey = starting ? serverStartingStepKey(failureReason) : undefined
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-1">
        <span className="text-sm text-muted-foreground animate-pulse">
          {starting ? t('common.root.server_starting') : t('common.root.loading_app')}
        </span>
        {stepKey && <span className="text-xs text-muted-foreground">{t(stepKey)}</span>}
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-center space-y-2">
          <p className="font-semibold text-destructive">{t('common.root.api_unreachable')}</p>
          <p className="text-sm text-muted-foreground">
            {t('common.root.api_check_server_prefix')}
            {/* Nom de commande shell technique, non traduit */}
            {/* eslint-disable-next-line @levelup/no-hardcoded-strings */}
            <code>make go-api-run</code>
            {t('common.root.api_check_server_suffix')}
          </p>
          <button
            className="text-sm underline text-primary"
            onClick={() => window.location.reload()}
          >
            {t('common.root.api_retry')}
          </button>
        </div>
      </div>
    )
  }

  // Setup en cours → pas de shell. Idem pour le wizard d'un compte sans profil à
  // lui (ADR 0035 D3) : le shell n'a ni joueur ni titre à afficher pour lui.
  if (!isBootstrapped || setupRequired) {
    return <Outlet />
  }
  const ownProfileMissing = needsOwnProfile({
    authMode,
    currentUsername,
    isAdmin,
    availablePlayerCount: availablePlayers.length,
  })
  if (pathname === SETUP_PATH && ownProfileMissing) {
    return <Outlet />
  }

  // Auth locale non connectée → pages login/register sans shell (password ET xbox)
  if ((authMode === 'password' || authMode === 'xbox') && !currentUsername) {
    return <Outlet />
  }

  // Premier lancement → page register sans shell
  if ((authMode === 'password' || authMode === 'xbox') && firstLaunch) {
    return <Outlet />
  }

  return <AppShell />
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
})
