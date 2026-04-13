/**
 * SetupPage — wizard de configuration initiale.
 *
 * Machine d'état : choose_mode → auth → player → smoke_test → done
 */
import { useState, useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { queryKeys } from '@/lib/query/keys'
import {
  useSetupStatus,
  useStartDeviceFlow,
  useDeviceFlowStatus,
  useCreatePlayer,
  useStartSmokeTest,
  useJobStatus,
} from './queries'

// ---------------------------------------------------------------------------
// Étape 1 — Choix du mode auth
// ---------------------------------------------------------------------------
function StepChooseMode() {
  const setSelectedMode = useSetupFlowStore((s) => s.setSelectedMode)
  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Mode d'authentification</h2>
      <p className="text-sm text-gray-500">
        Comment souhaitez-vous vous connecter à l'API Halo ?
      </p>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <Card
          className="cursor-pointer hover:border-purple-400 transition-colors"
          onClick={() => setSelectedMode('device_code')}
        >
          <CardContent className="py-4">
            <p className="font-medium">Device Code Flow</p>
            <p className="text-xs text-gray-400 mt-1">
              Connexion via Microsoft en scannant un QR code (recommandé)
            </p>
          </CardContent>
        </Card>
        <Card
          className="cursor-pointer hover:border-purple-400 transition-colors"
          onClick={() => setSelectedMode('refresh_token')}
        >
          <CardContent className="py-4">
            <p className="font-medium">Refresh Token</p>
            <p className="text-xs text-gray-400 mt-1">
              Entrez un refresh token existant (avancé)
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Étape 2 — Device Code Flow
// ---------------------------------------------------------------------------
function StepDeviceCode() {
  const currentAttemptId = useSetupFlowStore((s) => s.currentAttemptId)
  const setCurrentAttemptId = useSetupFlowStore((s) => s.setCurrentAttemptId)
  const deviceFlowUserCode = useSetupFlowStore((s) => s.deviceFlowUserCode)
  const deviceFlowVerificationUri = useSetupFlowStore((s) => s.deviceFlowVerificationUri)
  const setDeviceFlowCodes = useSetupFlowStore((s) => s.setDeviceFlowCodes)
  const setResolvedIdentity = useSetupFlowStore((s) => s.setResolvedIdentity)
  const queryClient = useQueryClient()
  const startFlow = useStartDeviceFlow()

  const { data: status } = useDeviceFlowStatus(
    currentAttemptId ?? '',
    !!currentAttemptId,
  )

  // Démarrer le flow au montage si pas encore en cours
  useEffect(() => {
    if (!currentAttemptId) {
      startFlow.mutate(undefined, {
        onSuccess: (data) => {
          setCurrentAttemptId(data.attempt_id)
          setDeviceFlowCodes(data.user_code, data.verification_uri)
        },
      })
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Quand le flow réussit : enregistrer l'identité et avancer
  useEffect(() => {
    if (status?.status === 'authorized' || status?.status === 'provisioned') {
      if (status.gamertag) {
        setResolvedIdentity(status.gamertag, status.xuid ?? null)
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus })
    }
  }, [status?.status]) // eslint-disable-line react-hooks/exhaustive-deps

  function handleRetry() {
    setCurrentAttemptId(null)
    startFlow.mutate(undefined, {
      onSuccess: (data) => {
        setCurrentAttemptId(data.attempt_id)
        setDeviceFlowCodes(data.user_code, data.verification_uri)
      },
    })
  }

  if (startFlow.isPending || (!status && !deviceFlowUserCode)) {
    return <Spinner label="Démarrage du Device Code Flow…" />
  }

  if (status?.status === 'failed' || status?.status === 'expired') {
    return (
      <div className="space-y-3">
        <p className="text-red-600 font-medium">
          {status.status === 'expired' ? 'Le code a expiré.' : "Échec de l'authentification."}
        </p>
        <Button onClick={handleRetry}>Réessayer</Button>
      </div>
    )
  }

  if (status?.status === 'authorized' || status?.status === 'provisioned') {
    return (
      <div className="space-y-2">
        <p className="text-green-600 font-semibold">✓ Authentification réussie !</p>
        {status.gamertag && (
          <p className="text-sm text-gray-600">
            Connecté en tant que : <strong>{status.gamertag}</strong>
          </p>
        )}
        <Spinner size="sm" label="Finalisation du profil…" />
      </div>
    )
  }

  const uri = deviceFlowVerificationUri ?? 'https://microsoft.com/devicelogin'

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Connexion Microsoft</h2>
      <p className="text-sm text-gray-500">
        Rendez-vous sur{' '}
        <a href={uri} target="_blank" rel="noopener noreferrer" className="text-purple-600 underline">
          {uri.replace('https://', '')}
        </a>{' '}
        et entrez ce code :
      </p>
      <div className="rounded-lg bg-gray-900 px-6 py-4 text-center">
        {deviceFlowUserCode ? (
          <span className="text-3xl font-mono font-bold tracking-widest text-white select-all">
            {deviceFlowUserCode}
          </span>
        ) : (
          <Spinner size="sm" />
        )}
      </div>
      <p className="text-xs text-gray-400 text-center animate-pulse">
        En attente de l'authentification…
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Étape 3 — Ajout joueur
// ---------------------------------------------------------------------------
function StepPlayer() {
  const resolvedGamertag = useSetupFlowStore((s) => s.resolvedGamertag)
  const resolvedXuid = useSetupFlowStore((s) => s.resolvedXuid)
  const [gamertag, setGamertag] = useState(resolvedGamertag ?? '')
  const queryClient = useQueryClient()
  const createPlayer = useCreatePlayer()

  function handleCreate() {
    if (!gamertag.trim()) return
    createPlayer.mutate(
      { gamertag: gamertag.trim(), xuid: resolvedXuid ?? undefined },
      {
        onSuccess: () =>
          queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus }),
      },
    )
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Ajouter votre joueur</h2>

      {resolvedGamertag ? (
        <div className="rounded-lg border border-green-200 bg-green-50 p-4">
          <p className="text-xs text-gray-500">Identité Halo détectée :</p>
          <p className="mt-1 text-xl font-bold text-green-700">{resolvedGamertag}</p>
          {resolvedXuid && (
            <p className="mt-0.5 text-xs text-gray-400">XUID : {resolvedXuid}</p>
          )}
          <p className="mt-2 text-xs text-gray-500">
            Un profil local va être créé pour ce compte.
          </p>
        </div>
      ) : (
        <>
          <p className="text-sm text-gray-500">
            Entrez votre Gamertag Xbox pour créer votre profil.
          </p>
          <Input
            value={gamertag}
            onChange={(e) => setGamertag(e.target.value)}
            placeholder="MonGamertag"
            onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
          />
        </>
      )}

      <Button
        onClick={handleCreate}
        loading={createPlayer.isPending}
        disabled={!gamertag.trim()}
      >
        {resolvedGamertag ? 'Confirmer et créer mon profil' : 'Ajouter'}
      </Button>

      {createPlayer.isSuccess && (
        <p className="text-green-600 text-sm">
          ✓ Joueur <strong>{createPlayer.data.player.gamertag}</strong> ajouté.
        </p>
      )}
      {createPlayer.isError && (
        <p className="text-red-600 text-sm">Erreur lors de la création du profil.</p>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Étape 4 — Smoke test
// ---------------------------------------------------------------------------
function StepSmokeTest({ playerSlug }: { playerSlug: string }) {
  const currentJobId = useSetupFlowStore((s) => s.currentJobId)
  const setCurrentJobId = useSetupFlowStore((s) => s.setCurrentJobId)
  const startTest = useStartSmokeTest()

  const { data: job } = useJobStatus(currentJobId ?? '', !!currentJobId)

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Test de connexion</h2>
      <p className="text-sm text-gray-500">
        Vérifie que LevelUp peut récupérer vos données Halo.
      </p>

      {!currentJobId && (
        <Button
          onClick={() => {
            startTest.mutate(
              { player_slug: playerSlug, max_matches: 5 },
              { onSuccess: (j) => setCurrentJobId(j.job_id) },
            )
          }}
          loading={startTest.isPending}
        >
          Lancer le test
        </Button>
      )}

      {job && (
        <div className="space-y-2">
          {job.progress_pct != null && (
            <div className="h-2 w-full rounded-full bg-gray-100">
              <div
                className="h-full rounded-full bg-purple-500 transition-all"
                style={{ width: `${job.progress_pct}%` }}
              />
            </div>
          )}
          <p className="text-sm text-gray-600">{job.current_step ?? '…'}</p>
          {job.status === 'succeeded' && (
            <p className="text-green-600 font-medium">✓ Test réussi ! LevelUp est prêt.</p>
          )}
          {job.status === 'failed' && (
            <p className="text-red-600 font-medium">✗ Échec du test. {job.error?.message}</p>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// SetupPage orchestrateur
// ---------------------------------------------------------------------------
export function SetupPage() {
  const navigate = useNavigate()
  const { data: status, isLoading } = useSetupStatus()
  const selectedMode = useSetupFlowStore((s) => s.selectedMode)

  useEffect(() => {
    if (status && !status.needs_setup) {
      navigate({ to: '/' })
    }
  }, [status, navigate])

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner size="lg" label="Vérification de la configuration…" />
      </div>
    )
  }

  const step = status?.next_blocking_step ?? 'choose_mode'

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <Card className="w-full max-w-lg mx-4">
        <CardHeader>
          <div className="flex items-center gap-2">
            <span className="text-2xl">⚔️</span>
            <CardTitle>Configuration de LevelUp</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          {step === 'choose_mode' && <StepChooseMode />}
          {step === 'auth' && selectedMode === 'device_code' && <StepDeviceCode />}
          {step === 'auth' && selectedMode === 'refresh_token' && (
            <div>
              <p className="text-sm text-gray-500">
                Entrez votre refresh token dans <code>.env.local</code> puis redémarrez l'application.
              </p>
            </div>
          )}
          {step === 'auth' && !selectedMode && <StepChooseMode />}
          {step === 'player' && <StepPlayer />}
          {step === 'smoke_test' && status?.player.default_player_slug && (
            <StepSmokeTest playerSlug={status.player.default_player_slug} />
          )}
          {step === 'done' && (
            <div className="space-y-4">
              <p className="text-green-600 font-semibold">✓ Configuration terminée !</p>
              <Button onClick={() => navigate({ to: '/' })}>Accéder à l'application</Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
