/**
 * Store du wizard de configuration initiale (Setup Flow).
 *
 * Gère l'état local de la navigation dans le wizard :
 * - Mode sélectionné (refresh_token vs device_code)
 * - Attempt ID du Device Code Flow en cours
 * - Job ID du smoke test en cours
 *
 * Le store ne reçoit pas le statut réel du setup (SetupStatusResponse) —
 * celui-ci est géré par TanStack Query avec la clé ['setup-status'].
 *
 * Cycle de vie recommandé :
 *   useQuery(['setup-status']) → reading next_blocking_step
 *   → useSetupFlowStore.setSelectedMode()
 *   → POST /auth/device-flow/start → setCurrentAttemptId()
 *   → poll ['device-flow', attemptId] until status = "provisioned"
 *   → POST /setup/players
 *   → POST /setup/smoke-test → setCurrentJobId()
 *   → poll ['job', jobId] until status = "succeeded"
 *   → invalidate ['setup-status']
 */

import { create } from 'zustand'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SetupAuthMode = 'refresh_token' | 'device_code'

interface SetupFlowState {
  /** Mode d'authentification choisi par l'utilisateur */
  selectedMode: SetupAuthMode | null

  /** attempt_id du Device Code Flow courant (null si aucun en cours) */
  currentAttemptId: string | null

  /** Code affiché à l'utilisateur (ex: "ABCD-1234") — depuis DeviceFlowStartResponse */
  deviceFlowUserCode: string | null

  /** URL de vérification (https://microsoft.com/devicelogin) */
  deviceFlowVerificationUri: string | null

  /** Timestamp (ms) d'expiration du code (Date.now() + expires_in * 1000) */
  deviceFlowExpiresAt: number | null

  /** Gamertag résolu après succès du Device Code Flow */
  resolvedGamertag: string | null

  /** XUID résolu après succès du Device Code Flow */
  resolvedXuid: string | null

  /** job_id du job courant (smoke test ou initial sync) */
  currentJobId: string | null

  /** Titres sélectionnés à l'onboarding (slugs cochés). Multi-titre. */
  selectedTitleSlugs: string[]

  /** Nb de matchs initiaux à synchroniser par titre (slug → nombre). */
  maxMatchesByTitle: Record<string, number>

  // --- Actions ---
  setSelectedMode: (mode: SetupAuthMode) => void
  setCurrentAttemptId: (id: string | null) => void
  setDeviceFlowCodes: (userCode: string, verificationUri: string, expiresAt: number | null) => void
  setResolvedIdentity: (gamertag: string, xuid: string | null) => void
  setCurrentJobId: (id: string | null) => void
  setSelectedTitles: (slugs: string[], maxByTitle: Record<string, number>) => void
  reset: () => void
}

// ---------------------------------------------------------------------------
// Store initial
// ---------------------------------------------------------------------------

const INITIAL_STATE = {
  selectedMode: null as SetupAuthMode | null,
  currentAttemptId: null as string | null,
  deviceFlowUserCode: null as string | null,
  deviceFlowVerificationUri: null as string | null,
  deviceFlowExpiresAt: null as number | null,
  resolvedGamertag: null as string | null,
  resolvedXuid: null as string | null,
  currentJobId: null as string | null,
  selectedTitleSlugs: [] as string[],
  maxMatchesByTitle: {} as Record<string, number>,
}

// ---------------------------------------------------------------------------
// Zustand store
// ---------------------------------------------------------------------------

export const useSetupFlowStore = create<SetupFlowState>((set) => ({
  ...INITIAL_STATE,

  setSelectedMode: (mode) => set({ selectedMode: mode }),

  setCurrentAttemptId: (id) => set({ currentAttemptId: id }),

  setDeviceFlowCodes: (userCode, verificationUri, expiresAt) =>
    set({ deviceFlowUserCode: userCode, deviceFlowVerificationUri: verificationUri, deviceFlowExpiresAt: expiresAt }),

  setResolvedIdentity: (gamertag, xuid) =>
    set({ resolvedGamertag: gamertag, resolvedXuid: xuid }),

  setCurrentJobId: (id) => set({ currentJobId: id }),

  setSelectedTitles: (slugs, maxByTitle) =>
    set({ selectedTitleSlugs: slugs, maxMatchesByTitle: maxByTitle }),

  reset: () => set(INITIAL_STATE),
}))
